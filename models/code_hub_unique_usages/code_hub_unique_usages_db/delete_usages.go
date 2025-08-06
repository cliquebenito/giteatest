package code_hub_unique_usages_db

import (
	"context"
	"fmt"

	"code.gitea.io/gitea/models/code_hub_unique_usages"
	"code.gitea.io/gitea/models/db"
)

// DeleteUniqueUsages удалить записи об уникальном использовании репозитория
func (c codeHubUniqueUsagesDB) DeleteUniqueUsages(ctx context.Context, repoID int64) error {
	deleteCounter := func(ctx context.Context) error {
		if err := c.deleteUniqueUsages(ctx, repoID); err != nil {
			return fmt.Errorf("delete code hub unique usages: %w", err)
		}
		return nil
	}

	if err := db.WithTx(ctx, deleteCounter); err != nil {
		return fmt.Errorf("delete codehub unique usages: %w", err)
	}

	return nil
}

func (c codeHubUniqueUsagesDB) deleteUniqueUsages(_ context.Context, repoID int64) error {
	uniqueUsage := &code_hub_unique_usages.CodeHubUniqueUsages{RepoID: repoID}
	_, err := c.engine.Delete(uniqueUsage)

	if err != nil {
		return fmt.Errorf("delete codehub unique usages: %w", err)
	}

	return nil
}
