package internal_metric_counter_db

import (
	"context"
	"fmt"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/internal_metric_counter"
)

// DeleteCounter удалить значения счетчика уникальных использований для репозитория по его id
func (c codeHubCounterDB) DeleteCounter(ctx context.Context, repoID int64) error {
	deleteCounter := func(ctx context.Context) error {
		if err := c.deleteCounter(ctx, repoID); err != nil {
			return fmt.Errorf("delete code hub counter: %w", err)
		}
		return nil
	}

	if err := db.WithTx(ctx, deleteCounter); err != nil {
		return fmt.Errorf("delete codehub counter with tx: %w", err)
	}

	return nil
}

func (c codeHubCounterDB) deleteCounter(_ context.Context, repoID int64) error {
	codeHubCounter := &internal_metric_counter.InternalMetricCounter{
		RepoID: repoID,
	}
	_, err := c.engine.Delete(codeHubCounter)

	if err != nil {
		return fmt.Errorf("delete codehub counter: %w", err)
	}

	return nil
}
