package repo_mark

import (
	"context"
	"fmt"
	"strconv"

	"code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/repo_marks"
	"code.gitea.io/gitea/modules/log"
)

// //go:generate mockery --name=editRepoMarksDB --exported
type editRepoMarksDB interface {
	InsertRepoMark(ctx context.Context, repoID int64, expertID int64, repoMark repo_marks.RepoMark) error
	DeleteRepoMark(ctx context.Context, repoID int64, repoMark repo_marks.RepoMark) error
}

// // go:generate mockery --name=repoKeyDB --exported
type repoKeyDB interface {
	GetRepoByKey(ctx context.Context, key string) (*repo.ScRepoKey, error)
}

type RepoMarksEditor struct {
	editRepoMarksDB
	repoKeyDB
}

func NewRepoMarksEditor(editRepoMarksDB editRepoMarksDB, repoKeyDB repoKeyDB) RepoMarksEditor {
	return RepoMarksEditor{editRepoMarksDB: editRepoMarksDB, repoKeyDB: repoKeyDB}
}

func (r RepoMarksEditor) DeleteRepoMark(ctx context.Context, repoKey string, repoMark repo_marks.RepoMark) error {
	var (
		scRepoKey *repo.ScRepoKey
		err       error
	)

	if scRepoKey, err = r.repoKeyDB.GetRepoByKey(ctx, repoKey); err != nil {
		log.Error("Error has occurred while getting repository by key %s: %v", repoKey, err)
		return fmt.Errorf("get repo key: %w", err)
	}
	repoId, err := strconv.ParseInt(scRepoKey.RepoID, 10, 64)
	if err != nil {
		log.Error("Error has occurred while parsing repository by id %s: %v", scRepoKey.RepoID, err)
		return fmt.Errorf("parse repo key: %w", err)
	}
	err = r.editRepoMarksDB.DeleteRepoMark(ctx, repoId, repoMark)
	if err != nil {
		log.Error("Error has occurred while deleting repo mark: %v", err)
		return fmt.Errorf("delete repo mark: %w", err)
	}
	return nil
}

func (r RepoMarksEditor) InsertRepoMark(ctx context.Context, repoKey string, expertID int64, repoMark repo_marks.RepoMark) error {
	var (
		scRepoKey *repo.ScRepoKey
		err       error
	)

	if scRepoKey, err = r.repoKeyDB.GetRepoByKey(ctx, repoKey); err != nil {
		log.Error("Error has occurred while getting repository by key %s: %v", repoKey, err)
		return fmt.Errorf("get repo key: %w", err)
	}
	repoId, err := strconv.ParseInt(scRepoKey.RepoID, 10, 64)
	if err != nil {
		log.Error("Error has occurred while parsing repository by id %s: %v", scRepoKey.RepoID, err)
		return fmt.Errorf("parse repo key: %w", err)
	}
	err = r.editRepoMarksDB.InsertRepoMark(ctx, repoId, expertID, repoMark)
	if err != nil {
		log.Error("Error has occurred while inserting repo mark: %v", err)
		return fmt.Errorf("insert repo mark: %w", err)
	}
	return nil
}
