package code_hub_counter

import (
	"context"
	"fmt"

	"code.gitea.io/gitea/models/code_hub_counter_task"
	"code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/modules/log"
)

// //go:generate mockery --name=usagesTasksDB --exported
type usagesTasksDB interface {
	InsertTask(ctx context.Context, repoID int64, userID int64, action code_hub_counter_task.CodeHubAction) error
}

type taskCreator struct {
	usagesTasksDB
	isCounterEnabled bool
}

func NewTaskCreator(uniqueUsages usagesTasksDB, counterEnabled bool) taskCreator {
	return taskCreator{usagesTasksDB: uniqueUsages, isCounterEnabled: counterEnabled}
}

// Create создать таску для подсчета статистики об использовании репозитория
func (t taskCreator) Create(ctx context.Context, repoID int64, userID int64, action code_hub_counter_task.CodeHubAction) error {
	if t.isCounterEnabled {
		if err := t.InsertTask(ctx, repoID, userID, action); err != nil {
			log.Error("error has occurred while creating usage task: %v", err)
			return fmt.Errorf("create usage task: %w", err)
		}
	}
	return nil
}

// CreateByRepoNameOwner создать таску для подсчета статистики об использовании репозитория
func (t taskCreator) CreateByRepoNameOwner(ctx context.Context, repoName string, repoOwner string, userID int64, action code_hub_counter_task.CodeHubAction) error {
	repository, err := repo.GetRepositoryByOwnerAndName(ctx, repoOwner, repoName)
	if err != nil {
		log.Error("error has occurred while getting repo by url: %v", err)
		return fmt.Errorf("get repo by url: %w", err)
	}
	return t.Create(ctx, repository.ID, userID, action)
}
