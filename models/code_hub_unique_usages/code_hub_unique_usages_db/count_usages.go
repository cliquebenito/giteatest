package code_hub_unique_usages_db

import (
	"context"
	"fmt"

	"xorm.io/builder"

	"code.gitea.io/gitea/models/code_hub_unique_usages"
	"code.gitea.io/gitea/models/repo"
)

// CountUniqueUsages получить количество уникальных использований репозитория по id репозитория и пользователя
func (c codeHubUniqueUsagesDB) CountUniqueUsages(_ context.Context, repoID int64, userID int64) (int, error) {
	uniqueUsage := code_hub_unique_usages.CodeHubUniqueUsages{}
	repository := repo.Repository{}

	count, err := c.engine.Where(builder.Eq{"id": repoID}).Count(&repository)
	if err != nil {
		return 0, fmt.Errorf("find repo unique usages: %w", err)
	}

	if count == 0 {
		return 0, NewRepoNotFoundError(repoID)
	}

	usages, err := c.engine.
		Where(builder.Eq{"repo_id": repoID}, builder.Eq{"user_id": userID}).
		Count(&uniqueUsage)
	if err != nil {
		return 0, fmt.Errorf("find code hub unique usages: %w", err)
	}
	return int(usages), nil
}

// CountUniqueUsagesByRepoID получить количество уникальных использований репозитория по id репозитория
func (c codeHubUniqueUsagesDB) CountUniqueUsagesByRepoID(_ context.Context, repoID int64) (int, error) {
	uniqueUsage := code_hub_unique_usages.CodeHubUniqueUsages{}
	repository := repo.Repository{}

	count, err := c.engine.Where(builder.Eq{"id": repoID}).Count(&repository)
	if err != nil {
		return 0, fmt.Errorf("find repo unique usages: %w", err)
	}

	if count == 0 {
		return 0, NewRepoNotFoundError(repoID)
	}

	usages, err := c.engine.
		Where(builder.Eq{"repo_id": repoID}).
		Count(&uniqueUsage)
	if err != nil {
		return 0, fmt.Errorf("find code hub unique usages: %w", err)
	}
	return int(usages), nil
}
