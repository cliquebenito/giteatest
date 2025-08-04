package task_tracker_db

import (
	"context"
	"encoding/json"
	"fmt"

	"code.gitea.io/gitea/models/pull_request_sender"
	"code.gitea.io/gitea/models/unit_links"
	"code.gitea.io/gitea/models/unit_links_sender"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/timeutil"
)

func (t taskTrackerDB) insertNewTasks(_ context.Context, opts pull_request_sender.UpdatePullRequestStatusOptions) error {
	payloadUpdate := []payloadStatusForUpdating{{
		FromUnitID:     opts.FromUnitID,
		FromUnitType:   unit_links.PullRequestFromUnitType,
		FromUnitStatus: opts.PullRequestStatus,
	}}
	payload, err := json.Marshal(payloadUpdate)
	if err != nil {
		log.Error("Error has occurred while marshaling payload: %v", err)
		return fmt.Errorf("marshal payload: %w", err)
	}
	timeNow := timeutil.TimeStampNow()

	if _, err = t.engine.Insert(unit_links_sender.UnitLinksSenderTasks{
		Payload:        string(payload),
		Action:         unit_links_sender.SendUpdatePullRequestStatusAction,
		PullRequestID:  opts.FromUnitID,
		PullRequestURL: opts.PullRequestURL,
		UserName:       opts.UserName,
		Status:         unit_links_sender.StatusUnlocked,
		CreatedAt:      timeNow,
		UpdatedAt:      timeNow,
	}); err != nil {
		log.Error("Error has occurred while inserting task: %v", err)
		return fmt.Errorf("insert from update issue statues: %w", err)
	}
	return nil
}
