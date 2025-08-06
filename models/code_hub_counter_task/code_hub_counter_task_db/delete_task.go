package code_hub_counter_task_db

import (
	"context"
	"fmt"

	"code.gitea.io/gitea/models/code_hub_counter_task"
	"code.gitea.io/gitea/models/db"
)

// DeleteTask удалить таску счетчика CodeHub после ее успешной обработки
func (c codeHubCounterTasksDB) DeleteTask(ctx context.Context, taskID int64) error {
	deleteTask := func(ctx context.Context) error {
		if err := c.deleteTask(ctx, taskID); err != nil {
			return fmt.Errorf("delete code hub counter task: %w", err)
		}
		return nil
	}

	if err := db.WithTx(ctx, deleteTask); err != nil {
		return fmt.Errorf("delete codehub counter task with tx: %w", err)
	}

	return nil
}

func (c codeHubCounterTasksDB) deleteTask(_ context.Context, taskID int64) error {
	codeHubCounterTask := code_hub_counter_task.CodeHubCounterTasks{ID: taskID}
	_, err := c.engine.Delete(codeHubCounterTask)

	if err != nil {
		return fmt.Errorf("delete codehub counter task: %w", err)
	}

	return nil
}
