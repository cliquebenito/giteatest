package unit_links_sender_db

import (
	"context"
	"fmt"
	"xorm.io/builder"

	"code.gitea.io/gitea/models/unit_links_sender"
)

func (s unitLinksSenderDB) GetPullRequestLinksTask(_ context.Context) ([]unit_links_sender.UnitLinksSenderTasks, error) {
	tasks := make([]unit_links_sender.UnitLinksSenderTasks, 0)

	if err := s.engine.
		Where(builder.Eq{"status": unit_links_sender.StatusUnlocked}).
		Table("unit_links_sender_tasks").
		Find(&tasks); err != nil {
		return nil, fmt.Errorf("find tasks: %w", err)
	}

	return tasks, nil
}
