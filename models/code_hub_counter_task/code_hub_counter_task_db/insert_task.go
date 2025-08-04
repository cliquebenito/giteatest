package code_hub_counter_task_db

import (
	"context"
	"fmt"

	"code.gitea.io/gitea/models/code_hub_counter_task"
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/timeutil"
)

// InsertTask добавляем запись о действии с репозиторием для подсчета статистики
func (c codeHubCounterTasksDB) InsertTask(ctx context.Context, repoID int64, userID int64, action code_hub_counter_task.CodeHubAction) error {
	insertTask := func(ctx context.Context) error {
		if err := c.insertNewTask(ctx, repoID, userID, action); err != nil {
			log.Error("Error has occurred while updating tasks: %v", err)
			return fmt.Errorf("updating pr statuses tasks: %w", err)
		}
		return nil
	}

	if err := db.WithTx(ctx, insertTask); err != nil {
		log.Error("Error has occurred while updating tasks in a transaction: %v", err)
		return fmt.Errorf("updating status of pr in a transaction: %w", err)
	}
	return nil
}

func (c codeHubCounterTasksDB) insertNewTask(_ context.Context, repoID int64, userID int64, action code_hub_counter_task.CodeHubAction) error {
	timeNow := timeutil.TimeStampNow()

	if _, err := c.engine.Insert(code_hub_counter_task.CodeHubCounterTasks{
		UserID:    userID,
		RepoID:    repoID,
		Action:    action,
		Status:    code_hub_counter_task.StatusUnlocked,
		CreatedAt: timeNow,
		UpdatedAt: timeNow,
	}); err != nil {
		log.Error("Error has occurred while inserting task: %v", err)
		return fmt.Errorf("insert from update issue statues: %w", err)
	}
	return nil
}
