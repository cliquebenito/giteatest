package task_tracker_sender

import (
	"code.gitea.io/gitea/models/unit_links"
	"context"
	"errors"
	"fmt"
	"strconv"

	"code.gitea.io/gitea/models/unit_links_sender"
	"code.gitea.io/gitea/models/unit_links_sender/unit_links_sender_db"
	"code.gitea.io/gitea/modules/json"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/sbt/audit"
)

type taskSender interface {
	SendAddPullRequestLinks(
		ctx context.Context,
		unitLinks unit_links.AllPayloadToAddOrDeletePr,
		userName string,
		pullRequestID int64,
		pullRequestURL string,
	) error
	SendDeletePullRequestLinks(
		ctx context.Context,
		unitLinks unit_links.AllPayloadToAddOrDeletePr,
		userName string,
		pullRequestID int64,
	) error
	SendUpdatePullRequestStatus(
		ctx context.Context,
		payloads unit_links.AllPayloadToAddOrDeletePr,
		userName string,
		pullRequestID int64,
	) error
}

type taskDB interface {
	LockTask(ctx context.Context, taskID int64) error
	UnlockTask(ctx context.Context, taskID int64) error
	UnlockTaskWithSuccess(ctx context.Context, taskID int64) error
	GetPullRequestLinksTask(ctx context.Context) ([]unit_links_sender.UnitLinksSenderTasks, error)
}

type taskTrackerSender struct {
	taskSender
	taskDB
}

func NewTaskTrackerSender(sender taskSender, taskPuller taskDB) taskTrackerSender {
	return taskTrackerSender{taskSender: sender, taskDB: taskPuller}
}

func (s taskTrackerSender) SendNewPrTasksToTaskTracker(ctx context.Context) error {
	auditParams := make(map[string]string)

	tasks, err := s.taskDB.GetPullRequestLinksTask(ctx)
	if err != nil {
		auditParams["error"] = "Error has occurred while getting pull request links task"
		audit.CreateAndSendEvent(audit.UnitLinksRequestCreateEvent, audit.EmptyRequiredField, audit.EmptyRequiredField, audit.StatusFailure, audit.EmptyRequiredField, auditParams)
		return fmt.Errorf("get tasks: %w", err)
	}

	if tasks == nil || len(tasks) == 0 {
		return nil
	}

	for _, task := range tasks {
		auditParams["task_id"] = strconv.FormatInt(task.ID, 10)
		auditParams["task_pull_request_id"] = strconv.FormatInt(task.PullRequestID, 10)
		auditParams["task_pull_request_url"] = task.PullRequestURL

		if err = s.taskDB.LockTask(ctx, task.ID); err != nil {
			if handledErr := new(unit_links_sender_db.TaskAlreadyLockedError); errors.As(err, &handledErr) {
				auditParams["error"] = "Error has occurred while locking task"
				audit.CreateAndSendEvent(audit.UnitTaskLockEvent, audit.EmptyRequiredField, audit.EmptyRequiredField, audit.StatusFailure, audit.EmptyRequiredField, auditParams)
				log.Debug("lock task id: '%d' err: %s", handledErr.Error())
			}

			continue
		}

		var links unit_links.AllPayloadToAddOrDeletePr

		if err = json.Unmarshal([]byte(task.Payload), &links); err != nil {
			log.Error("task_tracker_sender: unmarshal task payload: %s, %v", task.Payload, err)
			s.handleErr(ctx, task.ID)
			auditParams["error"] = "Error has occurred while unmarshalling"
			audit.CreateAndSendEvent(audit.PullRequestLinksAddEvent, audit.EmptyRequiredField, audit.EmptyRequiredField, audit.StatusFailure, audit.EmptyRequiredField, auditParams)

			continue
		}

		switch task.Action {
		case unit_links_sender.SendDeletePullRequestLinksAction:
			if senderErr := s.SendDeletePullRequestLinks(
				ctx,
				links,
				task.UserName,
				task.PullRequestID,
			); senderErr != nil {
				auditParams["error"] = "Error has occurred while deleting the link"
				audit.CreateAndSendEvent(audit.PullRequestLinksDeleteEvent, audit.EmptyRequiredField, audit.EmptyRequiredField, audit.StatusFailure, audit.EmptyRequiredField, auditParams)
				log.Error("send the delete event: %s, %v", task.Payload, senderErr)
				s.handleErr(ctx, task.ID)

				continue
			}

		case unit_links_sender.SendAddPullRequestLinksAction:
			if senderErr := s.SendAddPullRequestLinks(
				ctx,
				links,
				task.UserName,
				task.PullRequestID,
				task.PullRequestURL,
			); senderErr != nil {
				auditParams["error"] = "Error has occurred while sending the link"
				audit.CreateAndSendEvent(audit.PullRequestLinksAddEvent, audit.EmptyRequiredField, audit.EmptyRequiredField, audit.StatusFailure, audit.EmptyRequiredField, auditParams)
				log.Error("send the add event to task tracker: %s, %v", task.Payload, senderErr)
				s.handleErr(ctx, task.ID)

				continue
			}
		case unit_links_sender.SendUpdatePullRequestStatusAction:
			if senderErr := s.SendUpdatePullRequestStatus(ctx, links, task.UserName, task.PullRequestID); senderErr != nil {
				auditParams["error"] = "Error has occurred while sending the status of an updating pull request"
				audit.CreateAndSendEvent(audit.PullRequestsUpdateEvent, audit.EmptyRequiredField, audit.EmptyRequiredField, audit.StatusFailure, audit.EmptyRequiredField, auditParams)
				log.Error("send the update event to task tracker: %s, %v", task.Payload, senderErr)
				s.handleErr(ctx, task.ID)

				continue
			}

		default:
			log.Error("send the event to task tracker: %s: unknown action: %s", task.Payload, task.Action)
			s.handleErr(ctx, task.ID)

			continue
		}

		if unlockErr := s.taskDB.UnlockTaskWithSuccess(ctx, task.ID); unlockErr != nil {
			auditParams["error"] = "Error has occurred while unlocking and changing status task"
			audit.CreateAndSendEvent(audit.UnitTaskUnlockEvent, audit.EmptyRequiredField, audit.EmptyRequiredField, audit.StatusFailure, audit.EmptyRequiredField, auditParams)

			s.handleErr(ctx, task.ID)
		}

		log.Debug("send the event to task tracker: %s, %v: success", task.Payload, task.Action)
		audit.CreateAndSendEvent(audit.PullRequestLinksAddEvent, audit.EmptyRequiredField, audit.EmptyRequiredField, audit.StatusSuccess, audit.EmptyRequiredField, auditParams)

	}

	return nil
}

func (s taskTrackerSender) handleErr(ctx context.Context, taskID int64) {
	if err := s.taskDB.UnlockTask(ctx, taskID); err != nil {
		if handledErr := new(unit_links_sender_db.TaskAlreadyLockedError); errors.As(err, &handledErr) {
			log.Debug("unlock task id: '%d' err: %s", handledErr.Error())
		}
	}
}
