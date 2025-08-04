package external_metric_counter_db

import (
	"context"
	"fmt"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/external_metric_counter"
)

// DeleteCounter удалить значения счетчика уникальных использований для репозитория по его id
func (c externalMetricCounterDB) DeleteCounter(ctx context.Context, repoID int64) error {
	deleteCounter := func(ctx context.Context) error {
		if err := c.deleteCounter(ctx, repoID); err != nil {
			return fmt.Errorf("delete external metric counter: %w", err)
		}
		return nil
	}

	if err := db.WithTx(ctx, deleteCounter); err != nil {
		return fmt.Errorf("delete external metric counter with tx: %w", err)
	}

	return nil
}

func (c externalMetricCounterDB) deleteCounter(_ context.Context, repoID int64) error {
	codeHubCounter := &external_metric_counter.ExternalMetricCounter{
		RepoID: repoID,
	}
	_, err := c.engine.Delete(codeHubCounter)

	if err != nil {
		return fmt.Errorf("delete external metric counter: %w", err)
	}

	return nil
}
