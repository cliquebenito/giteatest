package code_hub_counter_task_db

import (
	"context"
	"fmt"

	"xorm.io/builder"

	"code.gitea.io/gitea/models/code_hub_counter_task"
	"code.gitea.io/gitea/models/db"
)

func (c codeHubCounterTasksDB) LockTask(ctx context.Context, taskID int64) error {
	lockTask := func(ctx context.Context) error {
		tasks := make([]code_hub_counter_task.CodeHubCounterTasks, 0)

		if err := c.engine.
			Where(builder.Eq{"id": taskID}, builder.Eq{"status": code_hub_counter_task.StatusUnlocked}).
			Find(&tasks); err != nil {
			return fmt.Errorf("get unlocked task: %w", err)
		}

		if len(tasks) == 0 {
			return NewTaskAlreadyLockedError(taskID)
		}

		_, err := c.engine.
			Where(builder.Eq{"id": taskID}, builder.Eq{"status": code_hub_counter_task.StatusUnlocked}).
			Cols("status").
			Update(&code_hub_counter_task.CodeHubCounterTasks{Status: code_hub_counter_task.StatusLocked})
		if err != nil {
			return fmt.Errorf("update unlocked task: %w", err)
		}

		return nil
	}

	if err := db.WithTx(ctx, lockTask); err != nil {
		return fmt.Errorf("lock task: %w", err)
	}

	return nil
}

func (c codeHubCounterTasksDB) UnlockTask(ctx context.Context, taskID int64) error {
	unlockTask := func(ctx context.Context) error {
		tasks := make([]code_hub_counter_task.CodeHubCounterTasks, 0)

		if err := c.engine.
			Where(builder.Eq{"id": taskID}).
			Find(&tasks); err != nil {
			return fmt.Errorf("get locked task: %w", err)
		}

		if len(tasks) == 0 {
			return nil
		}

		_, err := c.engine.
			Where(builder.Eq{"id": taskID}).
			Cols("status").
			Update(&code_hub_counter_task.CodeHubCounterTasks{Status: code_hub_counter_task.StatusUnlocked})
		if err != nil {
			return fmt.Errorf("unlock task: %w", err)
		}

		return nil
	}

	if err := db.WithTx(ctx, unlockTask); err != nil {
		return fmt.Errorf("unlock task tx: %w", err)
	}

	return nil
}

func (c codeHubCounterTasksDB) UnlockTaskWithSuccess(ctx context.Context, taskID int64) error {
	unlockTask := func(ctx context.Context) error {
		tasks := make([]code_hub_counter_task.CodeHubCounterTasks, 0)

		if err := c.engine.
			Where(builder.Eq{"id": taskID}, builder.Eq{"status": code_hub_counter_task.StatusLocked}).
			Find(&tasks); err != nil {
			return fmt.Errorf("get locked task: %w", err)
		}

		if len(tasks) == 0 {
			return nil
		}

		_, err := c.engine.
			Where(builder.Eq{"id": taskID}, builder.Eq{"status": code_hub_counter_task.StatusLocked}).
			Cols("status").
			Update(&code_hub_counter_task.CodeHubCounterTasks{Status: code_hub_counter_task.StatusDone})
		if err != nil {
			return fmt.Errorf("unlock task: %w", err)
		}

		return nil
	}

	if err := db.WithTx(ctx, unlockTask); err != nil {
		return fmt.Errorf("unlock task with success tx: %w", err)
	}

	return nil
}
