package configuer

import (
	"context"
	"fmt"

	"code.gitea.io/gitea/internal/models"
	"github.com/jackc/pgx/v5"
	"gitlab.com/gitlab-org/gitaly/v16/proto/go/gitalypb"
)

type pgConfiguer struct {
	conn           *pgx.Conn
	postgresSchema string
}

func NewPgConfiguer(conn *pgx.Conn, postgresSchema string) pgConfiguer {
	return pgConfiguer{conn: conn, postgresSchema: postgresSchema}
}

func (c pgConfiguer) Create(ctx context.Context) (models.BackupConfig, error) {
	query := fmt.Sprintf("SELECT virtual_storage, relative_path, repository_id FROM %s.repositories", c.postgresSchema)

	rows, err := c.conn.Query(ctx, query)
	if err != nil {
		return models.BackupConfig{}, fmt.Errorf("execute query: %w", err)
	}

	var backupConfig models.BackupConfig

	for rows.Next() {
		repoConfig := new(gitalypb.Repository)
		var repoID int

		if err = rows.Scan(&repoConfig.StorageName, &repoConfig.RelativePath, &repoID); err != nil {
			return models.BackupConfig{}, fmt.Errorf("scan row: %w", err)
		}

		// The repository name is not stored in the table,
		// in order to be able to find the repository by logs,
		// create a header like <repo/{repo_id}>

		repoConfig.GlProjectPath = fmt.Sprintf("repo/%d", repoID)

		backupConfig.RepoConfigs = append(backupConfig.RepoConfigs, repoConfig)
	}

	if err = rows.Err(); err != nil {
		return models.BackupConfig{}, fmt.Errorf("execute rows: %w", err)
	}

	return backupConfig, nil
}
