package internal_metric_counter_db

import (
	"context"
	"fmt"
	"time"

	"xorm.io/builder"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/internal_metric_counter"
	"code.gitea.io/gitea/modules/timeutil"
)

// UpdateCounter обновить значения счетчика уникальных использований для репозитория по его id
func (c codeHubCounterDB) UpdateCounter(ctx context.Context, repoID int64, counter int, metricKey string) error {
	createCounter := func(ctx context.Context) error {
		if err := c.updateCounter(ctx, repoID, counter, metricKey); err != nil {
			return fmt.Errorf("update code hub counter: %w", err)
		}
		return nil
	}

	if err := db.WithTx(ctx, createCounter); err != nil {
		return fmt.Errorf("update codehub counter with tx: %w", err)
	}

	return nil
}

func (c codeHubCounterDB) updateCounter(_ context.Context, repoID int64, counter int, metricKey string) error {
	timeNow := time.Now().Unix()
	codeHubCounter := &internal_metric_counter.InternalMetricCounter{
		RepoID:      repoID,
		MetricValue: counter,
		UpdatedAt:   timeutil.TimeStamp(timeNow),
	}
	_, err := c.engine.
		Where(builder.And(builder.Eq{"repo_id": repoID}, builder.Eq{"metric_key": metricKey})).
		Cols("metric_value", "updated_at").
		Update(codeHubCounter)

	if err != nil {
		return fmt.Errorf("update from codehub counter: %w", err)
	}

	return nil
}
