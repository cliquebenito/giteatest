package external_metric_counter_db

import (
	"context"
	"fmt"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/modules/timeutil"
)

// UpsertCounter обновить значения счетчика уникальных использований для репозитория по его id
func (c externalMetricCounterDB) UpsertCounter(ctx context.Context, repoID int64, counter int, description string) error {
	createCounter := func(ctx context.Context) error {
		if err := c.upsertCounter(ctx, repoID, counter, description); err != nil {
			return fmt.Errorf("upsert external metric counter: %w", err)
		}
		return nil
	}

	if err := db.WithTx(ctx, createCounter); err != nil {
		return fmt.Errorf("upsert external metric counter with tx: %w", err)
	}

	return nil
}

func (c externalMetricCounterDB) upsertCounter(_ context.Context, repoID int64, counter int, description string) error {

	sql := `
INSERT INTO external_metric_counter (repo_id, metric_value, text, updated_at, created_at)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT (repo_id)
DO UPDATE SET
	metric_value = EXCLUDED.metric_value,
	text = EXCLUDED.text,
	updated_at = EXCLUDED.updated_at;`

	now := timeutil.TimeStampNow()

	_, err := c.engine.Exec(sql,
		repoID,
		counter,
		description,
		now,
		now,
	)
	if err != nil {
		return fmt.Errorf("upsert from external metric counter: %w", err)
	}

	return nil
}
