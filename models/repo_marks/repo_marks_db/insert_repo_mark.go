package repo_marks_db

import (
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/repo_marks"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/timeutil"
	"context"
	"fmt"
)

// InsertRepoMark отвечает за вставку метки CodeHub
func (m repoMarksDB) InsertRepoMark(ctx context.Context, repoID int64, expertID int64, repoMark repo_marks.RepoMark) error {
	exist, err := m.engine.Exist(&repo_marks.RepoMarks{
		RepoID:   repoID,
		ExpertID: expertID,
		MarkKey:  repoMark.Key(),
	})
	if err != nil {
		log.Error("Error has occurred while checking repo mark existence: %v", err)
		return fmt.Errorf("check repo mark existence: %w", err)
	}
	if exist {
		log.Debug("Repo mark with key %s already exists", repoMark.Key())
		return ErrMarkAlreadyExists{MarkKey: repoMark.Key()}
	}

	insertTask := func(ctx context.Context) error {
		if _, err := m.engine.Insert(&repo_marks.RepoMarks{
			RepoID:    repoID,
			ExpertID:  expertID,
			MarkKey:   repoMark.Key(),
			CreatedAt: timeutil.TimeStampNow(),
			UpdatedAt: timeutil.TimeStampNow(),
		}); err != nil {
			log.Error("Error has occurred while inserting repo mark: %v", err)
			return fmt.Errorf("insert repo mark: %w", err)
		}
		return nil
	}

	if err := db.WithTx(ctx, insertTask); err != nil {
		log.Error("Error has occurred while inserting repo mark in a transaction: %v", err)
		return fmt.Errorf("insert repo mark in a transaction: %w", err)
	}
	return nil
}
