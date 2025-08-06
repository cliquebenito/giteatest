package pull_request_task_creator

import (
	"context"
	"fmt"

	"code.gitea.io/gitea/models/pull_request_sender"
	"code.gitea.io/gitea/modules/log"
)

// UpdateStatusOfPullRequest позволяет отправить информацию об обновлении статуса pr
func (p PullRequestTaskCreator) UpdateStatusOfPullRequest(ctx context.Context, request PullRequestUpdateStatus) error {
	if err := request.Validate(); err != nil {
		log.Error("Error has occurred while validating request: %v", err)
		return fmt.Errorf("validate request: %w", err)
	}

	if err := p.taskTrackerDB.PullRequestStatusUpdate(ctx, pull_request_sender.UpdatePullRequestStatusOptions{
		UserName:          request.UserName,
		PullRequestURL:    request.PullRequestURL,
		FromUnitID:        request.PullRequestID,
		PullRequestStatus: request.PullRequestStatus,
	}); err != nil {
		log.Error("Error has occurred while updating request: %v", err)
		return fmt.Errorf("update pull request status, id: '%d': %w", request.PullRequestID, err)
	}
	return nil
}
