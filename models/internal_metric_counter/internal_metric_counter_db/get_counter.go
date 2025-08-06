package internal_metric_counter_db

import (
	"context"
	"fmt"

	"code.gitea.io/gitea/models/internal_metric_counter"
)

// GetInternalMetricCounter получить счетчик уникальных использований для репозитория по его id и key
func (c codeHubCounterDB) GetInternalMetricCounter(_ context.Context, repoID int64, metricKey string) (*internal_metric_counter.InternalMetricCounter, error) {
	counter := new(internal_metric_counter.InternalMetricCounter)

	// тут нужен голый sql, так как ОРМка объединяет несколько запросов с помощью AND и ответ получается пустой
	query := "SELECT * FROM internal_metric_counter WHERE repo_id = ? AND metric_key = ? LIMIT 1"
	has, err := c.engine.SQL(query, repoID, metricKey).Get(counter)

	if err != nil {
		return nil, fmt.Errorf("find code hub counter: %w", err)
	}
	if !has {
		return nil, NewCodeHubCounterDoesntExistsError(repoID)
	}
	return counter, nil
}

// GetInternalMetricCountersByRepoIDs получить счетчики по слайсу id
func (c codeHubCounterDB) GetInternalMetricCountersByRepoIDs(_ context.Context, repoIDs []int64) ([]*internal_metric_counter.InternalMetricCounter, error) {
	if len(repoIDs) == 0 {
		return []*internal_metric_counter.InternalMetricCounter{}, nil
	}

	counters := make([]*internal_metric_counter.InternalMetricCounter, 0)
	err := c.engine.In("repo_id", repoIDs).Find(&counters)
	if err != nil {
		return nil, fmt.Errorf("find internal metric counters: %w", err)
	}
	return counters, nil
}

// GetInternalMetricCounters получить счетчики для всех репозиториев
func (c codeHubCounterDB) GetInternalMetricCounters(_ context.Context) ([]internal_metric_counter.InternalMetricCounter, error) {
	var counters = make([]internal_metric_counter.InternalMetricCounter, 0)

	err := c.engine.
		OrderBy("id").
		Find(&counters)
	if err != nil {
		return nil, fmt.Errorf("find code hub counters: %w", err)
	}
	return counters, nil
}
