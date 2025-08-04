package repo_marks_db

import (
	"context"
	"fmt"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/repo_marks"
	"code.gitea.io/gitea/modules/log"
)

// DeleteRepoMark отвечает за удаление метки CodeHub
func (m repoMarksDB) DeleteRepoMark(ctx context.Context, repoID int64, repoMark repo_marks.RepoMark) error {
	deleteTask := func(ctx context.Context) error {
		if _, err := m.engine.Delete(&repo_marks.RepoMarks{
			RepoID:  repoID,
			MarkKey: repoMark.Key(),
		}); err != nil {
			log.Error("Error has occurred while deleting repo mark: %v", err)
			return fmt.Errorf("delete repo mark: %w", err)
		}
		return nil
	}

	if err := db.WithTx(ctx, deleteTask); err != nil {
		log.Error("Error has occurred while deleting repo mark in a transaction: %v", err)
		return fmt.Errorf("delete repo mark in a transaction: %w", err)
	}
	return nil
}
