package external_metric_counter_db

import (
	"context"
	"fmt"

	"code.gitea.io/gitea/models/external_metric_counter"
)

// GetExternalMetricCounter получить счетчик уникальных использований для репозитория по его id
func (c externalMetricCounterDB) GetExternalMetricCounter(_ context.Context, repoID int64) (*external_metric_counter.ExternalMetricCounter, error) {
	counter := new(external_metric_counter.ExternalMetricCounter)

	// тут нужен голый sql, так как ОРМка объединяет несколько запросов с помощью AND и ответ получается пустой
	query := "SELECT * FROM external_metric_counter WHERE repo_id = ? LIMIT 1"
	has, err := c.engine.SQL(query, repoID).Get(counter)

	if err != nil {
		return nil, fmt.Errorf("find code hub counter: %w", err)
	}
	if !has {
		return nil, NewExternalMetricCounterDoesntExistsError(repoID)
	}
	return counter, nil
}

// GetExternalMetricCountersByRepoIDs получить счетчики по слайсу id
func (c externalMetricCounterDB) GetExternalMetricCountersByRepoIDs(_ context.Context, repoIDs []int64) ([]*external_metric_counter.ExternalMetricCounter, error) {
	if len(repoIDs) == 0 {
		return []*external_metric_counter.ExternalMetricCounter{}, nil
	}

	counters := make([]*external_metric_counter.ExternalMetricCounter, 0)
	err := c.engine.In("repo_id", repoIDs).Find(&counters)
	if err != nil {
		return nil, fmt.Errorf("find external metric counters: %w", err)
	}
	return counters, nil
}

// GetExternalMetricCounters получить счетчики для всех репозиториев
func (c externalMetricCounterDB) GetExternalMetricCounters(_ context.Context) ([]external_metric_counter.ExternalMetricCounter, error) {
	var counters = make([]external_metric_counter.ExternalMetricCounter, 0)

	err := c.engine.
		OrderBy("id").
		Find(&counters)
	if err != nil {
		return nil, fmt.Errorf("find code hub counters: %w", err)
	}
	return counters, nil
}
