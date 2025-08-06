package task_tracker_db

import (
	"context"
	"fmt"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/pull_request_sender"
	"code.gitea.io/gitea/modules/log"
)

// PullRequestStatusUpdate добавляем запись об изменения статуса pr
func (t taskTrackerDB) PullRequestStatusUpdate(ctx context.Context, request pull_request_sender.UpdatePullRequestStatusOptions) error {
	updateTasks := func(ctx context.Context) error {
		if err := t.insertNewTasks(ctx, request); err != nil {
			log.Error("Error has occurred while updating tasks: %v", err)
			return fmt.Errorf("updating pr statuses tasks: %w", err)
		}
		return nil
	}

	if err := db.WithTx(ctx, updateTasks); err != nil {
		log.Error("Error has occurred while updating tasks in a transaction: %v", err)
		return fmt.Errorf("updating status of pr in a transaction: %w", err)
	}
	return nil
}
