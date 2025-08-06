package repo

import (
	"context"
	"fmt"

	"xorm.io/builder"

	"code.gitea.io/gitea/models/db"
)

func init() {
	db.RegisterModel(new(ScRepoKey))
}

type RepoKeyDB struct {
	engine db.Engine
}

func NewRepoKeyDB(engine db.Engine) RepoKeyDB {
	return RepoKeyDB{engine: engine}
}

// ScRepoKey структура
type ScRepoKey struct {
	ID      int64  `xorm:"pk autoincr"`
	RepoID  string `xorm:"UNIQUE(s)"`
	RepoKey string `xorm:"VARCHAR(255) UNIQUE(s)"`
}

// GetRepoByKey извлечение репозитория по внешнему ключу
func (r RepoKeyDB) GetRepoByKey(ctx context.Context, key string) (*ScRepoKey, error) {
	repoKey := new(ScRepoKey)
	has, err := r.engine.
		Where(builder.Eq{"repo_key": key}).
		Get(repoKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get repokey by key: %w", err)
	} else if !has {
		return nil, ErrorRepoKeyDoesntExists{RepoKey: key}
	}
	return repoKey, nil
}

// GetRepoByRepoID извлечение репозитория по внутреннему ключу
func (r RepoKeyDB) GetRepoByRepoID(ctx context.Context, repoId string) (*ScRepoKey, error) {
	repoKey := new(ScRepoKey)
	has, err := r.engine.
		Where(builder.Eq{"repo_id": repoId}).
		Get(repoKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get repokey by repo id: %w", err)
	} else if !has {
		return nil, ErrorRepoKeyDoesntExists{RepoID: repoId}
	}
	return repoKey, nil
}

// InsertRepoKey добавление связи между внутренним и внешним ключами репозитория
func (r RepoKeyDB) InsertRepoKey(ctx context.Context, repoKey *ScRepoKey) error {
	_, err := r.engine.Insert(repoKey)
	if err != nil {
		return err
	}
	return nil
}

// UpdateRepoKey обновить внешний ключ в таблице
func (r RepoKeyDB) UpdateRepoKey(ctx context.Context, repoKey *ScRepoKey) error {
	_, err := r.engine.ID(repoKey.ID).Cols("repo_key").Update(repoKey)
	if err != nil {
		return err
	}
	return nil
}

// DeleteRepoKey удаление связи между внутренним и внешним ключами репозитория
func (r RepoKeyDB) DeleteRepoKey(ctx context.Context, repoKey *ScRepoKey) error {
	_, err := r.engine.Delete(repoKey)
	if err != nil {
		return err
	}
	return nil
}

// DeleteRepoKeyByRepoID удаление связи между внутренним и внешним ключами репозитория по внутреннему ключу
func (r RepoKeyDB) DeleteRepoKeyByRepoID(ctx context.Context, repoID string) error {
	_, err := r.engine.
		Where(builder.Eq{"repo_id": repoID}).
		Delete(&ScRepoKey{})
	if err != nil {
		return err
	}
	return nil
}
