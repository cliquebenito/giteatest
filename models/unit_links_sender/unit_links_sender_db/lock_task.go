package unit_links_sender_db

import (
	"context"
	"fmt"

	"xorm.io/builder"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/unit_links_sender"
)

func (s unitLinksSenderDB) LockTask(ctx context.Context, taskID int64) error {
	lockTask := func(ctx context.Context) error {
		tasks := make([]unit_links_sender.UnitLinksSenderTasks, 0)

		if err := s.engine.
			Where(builder.Eq{"id": taskID}, builder.Eq{"status": unit_links_sender.StatusUnlocked}).
			Find(&tasks); err != nil {
			return fmt.Errorf("get unlocked task: %w", err)
		}

		if len(tasks) == 0 {
			return NewTaskAlreadyLockedError(taskID)
		}

		_, err := s.engine.
			Where(builder.Eq{"id": taskID}, builder.Eq{"status": unit_links_sender.StatusUnlocked}).
			Cols("status").
			Update(&unit_links_sender.UnitLinksSenderTasks{Status: unit_links_sender.StatusLocked})
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

func (s unitLinksSenderDB) UnlockTask(ctx context.Context, taskID int64) error {
	unlockTask := func(ctx context.Context) error {
		tasks := make([]unit_links_sender.UnitLinksSenderTasks, 0)

		if err := s.engine.
			Where(builder.Eq{"id": taskID}).
			Find(&tasks); err != nil {
			return fmt.Errorf("get locked task: %w", err)
		}

		if len(tasks) == 0 {
			return nil
		}

		_, err := s.engine.
			Where(builder.Eq{"id": taskID}).
			Cols("status").
			Update(&unit_links_sender.UnitLinksSenderTasks{Status: unit_links_sender.StatusUnlocked})
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

func (s unitLinksSenderDB) UnlockTaskWithSuccess(ctx context.Context, taskID int64) error {
	unlockTask := func(ctx context.Context) error {
		tasks := make([]unit_links_sender.UnitLinksSenderTasks, 0)

		if err := s.engine.
			Where(builder.Eq{"id": taskID}, builder.Eq{"status": unit_links_sender.StatusLocked}).
			Find(&tasks); err != nil {
			return fmt.Errorf("get locked task: %w", err)
		}

		if len(tasks) == 0 {
			return nil
		}

		_, err := s.engine.
			Where(builder.Eq{"id": taskID}, builder.Eq{"status": unit_links_sender.StatusLocked}).
			Cols("status").
			Update(&unit_links_sender.UnitLinksSenderTasks{Status: unit_links_sender.StatusDone})
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
