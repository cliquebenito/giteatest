package code_hub_unique_usages_db

import (
	"context"
	"fmt"
	"time"

	"code.gitea.io/gitea/models/code_hub_unique_usages"
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/modules/timeutil"
)

// UpdateUniqueUsage вставить запись об уникальном использовании пользователем репозитория
func (c codeHubUniqueUsagesDB) UpdateUniqueUsage(ctx context.Context, repoID int64, userID int64) error {
	createUniqueUsage := func(ctx context.Context) error {
		if err := c.insertUniqueUsage(ctx, repoID, userID); err != nil {
			return fmt.Errorf("insert code hub unique usage: %w", err)
		}
		return nil
	}

	if err := db.WithTx(ctx, createUniqueUsage); err != nil {
		return fmt.Errorf("create codehub unique usage with tx: %w", err)
	}

	return nil
}

func (c codeHubUniqueUsagesDB) insertUniqueUsage(_ context.Context, repoID int64, userID int64) error {
	timeNow := timeutil.TimeStamp(time.Now().Unix())

	if _, err := c.engine.Insert(code_hub_unique_usages.CodeHubUniqueUsages{
		RepoID:    repoID,
		UserID:    userID,
		UpdatedAt: timeNow,
		CreatedAt: timeNow,
	}); err != nil {
		return fmt.Errorf("insert from codehub unique usage: %w", err)
	}

	return nil
}
