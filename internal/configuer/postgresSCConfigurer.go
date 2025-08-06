package configuer

import (
	"context"
	"fmt"

	"code.gitea.io/gitea/internal/models"
	repo_model "code.gitea.io/gitea/models/repo"
	"github.com/jackc/pgx/v5"
	"gitlab.com/gitlab-org/gitaly/v16/proto/go/gitalypb"
)

type pgSCConfiguer struct {
	conn              *pgx.Conn
	gitalyStorageName string
	postgresSchema    string
}

func NewPgSCConfiguer(conn *pgx.Conn, gitalyStorageName, postgresSchema string) pgSCConfiguer {
	return pgSCConfiguer{conn: conn, gitalyStorageName: gitalyStorageName, postgresSchema: postgresSchema}
}

func (c pgSCConfiguer) Create(ctx context.Context) (models.BackupConfig, error) {
	query := fmt.Sprintf("SELECT id, owner_name, name FROM %s.repository", c.postgresSchema)

	rows, err := c.conn.Query(ctx, query)
	if err != nil {
		return models.BackupConfig{}, fmt.Errorf("execute query: %w", err)
	}

	var backupConfig models.BackupConfig

	for rows.Next() {
		repo := new(repo_model.Repository)

		if err = rows.Scan(&repo.ID, &repo.OwnerName, &repo.Name); err != nil {
			return models.BackupConfig{}, fmt.Errorf("scan row: %w", err)
		}

		repoConfig := &gitalypb.Repository{
			GlProjectPath: repo.OwnerName,
			GlRepository:  repo.Name,
			RelativePath:  repo.RepoPath(),
			StorageName:   c.gitalyStorageName,
		}

		backupConfig.RepoConfigs = append(backupConfig.RepoConfigs, repoConfig)
	}

	if err = rows.Err(); err != nil {
		return models.BackupConfig{}, fmt.Errorf("execute rows: %w", err)
	}

	return backupConfig, nil
}
