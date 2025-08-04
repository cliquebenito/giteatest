package pull_request_task_creator

import (
	"context"

	"code.gitea.io/gitea/models/pull_request_sender"
)

// //go:generate mockery --name=taskTrackerDB --exported
type taskTrackerDB interface {
	PullRequestStatusUpdate(ctx context.Context, req pull_request_sender.UpdatePullRequestStatusOptions) error
	IsActiveOfPullRequestStatus(ctx context.Context, pullRequestID int64) (bool, error)
}

// PullRequestTaskCreator отправитель событий об изменении статусов pr
type PullRequestTaskCreator struct {
	taskTrackerDB
}

// NewPullRequestTaskCreator создаем экземпляр pullRequestSender
func NewPullRequestTaskCreator(
	taskTrackerDB taskTrackerDB,
) PullRequestTaskCreator {
	return PullRequestTaskCreator{
		taskTrackerDB: taskTrackerDB,
	}
}
