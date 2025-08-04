package code_hub_counter

import (
	"context"
	"errors"
	"fmt"

	"code.gitea.io/gitea/models/code_hub_counter_task"
	"code.gitea.io/gitea/models/code_hub_counter_task/code_hub_counter_task_db"
	"code.gitea.io/gitea/models/internal_metric_counter"
	"code.gitea.io/gitea/modules/log"
)

const UniqueClonesMetricKey = "unique_clones"

// //go:generate mockery --name=taskDB --exported
type taskDB interface {
	LockTask(ctx context.Context, taskID int64) error
	UnlockTask(ctx context.Context, taskID int64) error
	UnlockTaskWithSuccess(ctx context.Context, taskID int64) error
	GetCodeHubCounterTasks(_ context.Context) ([]code_hub_counter_task.CodeHubCounterTasks, error)
	DeleteTask(ctx context.Context, taskID int64) error
}

// //go:generate mockery --name=uniqueUsagesDB --exported
type uniqueUsagesDB interface {
	CountUniqueUsagesByRepoID(_ context.Context, repoID int64) (int, error)
	CountUniqueUsages(_ context.Context, repoID int64, userID int64) (int, error)
	UpdateUniqueUsage(ctx context.Context, repoID int64, userID int64) error
}

// //go:generate mockery --name=counterDB --exported
type counterDB interface {
	UpdateCounter(ctx context.Context, repoID int64, counter int, metricKey string) error
	GetInternalMetricCounters(_ context.Context) ([]internal_metric_counter.InternalMetricCounter, error)
}

type codeHubCounter struct {
	taskDB
	uniqueUsagesDB
	counterDB
}

func NewCodeHubCounter(taskWorker taskDB, uniqueUsages uniqueUsagesDB, counter counterDB) codeHubCounter {
	return codeHubCounter{taskDB: taskWorker, uniqueUsagesDB: uniqueUsages, counterDB: counter}
}

// ProcessNewUsageTasks метод для обработки необработанных логов об использовании репозитория в таблице code_hub_counter_tasks
func (c codeHubCounter) ProcessNewUsageTasks(ctx context.Context) error {
	tasks, err := c.taskDB.GetCodeHubCounterTasks(ctx)

	if err != nil {
		return fmt.Errorf("getting unprocessed tasks: %w", err)
	}

	if tasks == nil || len(tasks) == 0 {
		return nil
	}

	for _, task := range tasks {
		if err = c.taskDB.LockTask(ctx, task.ID); err != nil {
			if handledErr := new(code_hub_counter_task_db.TaskAlreadyLockedError); errors.As(err, &handledErr) {
				log.Debug("error has occurred while locking task id: '%d' err: %s", task.ID, handledErr.Error())
			}
			continue
		}
		var uniqueUsagesCount int
		if uniqueUsagesCount, err = c.uniqueUsagesDB.CountUniqueUsages(ctx, task.RepoID, task.UserID); err != nil {
			log.Error("error has occurred while counting uniqueUsagesCount usages for repo %d and user %d: %v", task.RepoID, task.UserID, err)
			c.handleErr(ctx, task.ID)
			continue
		}

		// если использование уникально, заносим в таблицу уникальных использований новую запись
		if uniqueUsagesCount == 0 {
			if err = c.uniqueUsagesDB.UpdateUniqueUsage(ctx, task.RepoID, task.UserID); err != nil {
				log.Error("error has occurred while inserting uniqueUsagesCount usage for repo %d and user %d: %v", task.RepoID, task.UserID, err)
				c.handleErr(ctx, task.ID)
				continue
			}
		}

		if unlockErr := c.taskDB.UnlockTaskWithSuccess(ctx, task.ID); unlockErr != nil {
			c.handleErr(ctx, task.ID)
			continue
		}

		if deleteErr := c.taskDB.DeleteTask(ctx, task.ID); deleteErr != nil {
			log.Error("error has occurred while deleting task for repo %d and user %d: %v", task.RepoID, task.UserID, err)
			continue
		}

		log.Debug("process task: %d, %d %v: success", task.RepoID, task.UserID, task.Action)
	}
	return nil
}

// CalculateRepoCounters пересчитать статистику уникальных использований для всех репозиториев
func (c codeHubCounter) CalculateRepoCounters(ctx context.Context) error {
	counters, err := c.counterDB.GetInternalMetricCounters(ctx)
	if err != nil {
		log.Error("error has occurred while getting counters for all repos: %v", err)
		return fmt.Errorf("getting counters: %w", err)
	}

	if counters == nil || len(counters) == 0 {
		log.Debug("unique usages counters not found")
		return nil
	}

	for _, counter := range counters {
		usagesCount, err := c.uniqueUsagesDB.CountUniqueUsagesByRepoID(ctx, counter.RepoID)
		if err != nil {
			log.Error("error has occurred while counting unique usagesCount for repo %d: %v", counter.RepoID, err)
			continue
		}

		// если поменялось значение счетчика, то перезаписываем
		if counter.MetricValue != usagesCount {
			if err = c.counterDB.UpdateCounter(ctx, counter.RepoID, usagesCount, UniqueClonesMetricKey); err != nil {
				log.Error("error has occurred while updating counter for repo %d: %v", counter.RepoID, err)
				continue
			}
		}
	}
	return nil
}

func (c codeHubCounter) handleErr(ctx context.Context, taskID int64) {
	if err := c.taskDB.UnlockTask(ctx, taskID); err != nil {
		if handledErr := new(code_hub_counter_task_db.TaskAlreadyLockedError); errors.As(err, &handledErr) {
			log.Debug("error has occurred while unlocking task id: '%d' err: %s", taskID, handledErr.Error())
		}
	}
}
