package code_hub_counter_task_db

import (
	"context"
	"fmt"

	"xorm.io/builder"

	"code.gitea.io/gitea/models/code_hub_counter_task"
)

func (c codeHubCounterTasksDB) GetCodeHubCounterTasks(_ context.Context) ([]code_hub_counter_task.CodeHubCounterTasks, error) {
	tasks := make([]code_hub_counter_task.CodeHubCounterTasks, 0)

	if err := c.engine.
		Where(builder.Eq{"status": code_hub_counter_task.StatusUnlocked}).
		OrderBy("created_at ASC").
		Find(&tasks); err != nil {
		return nil, fmt.Errorf("find tasks: %w", err)
	}

	return tasks, nil
}
