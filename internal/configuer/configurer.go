package configuer

import (
	"context"

	"code.gitea.io/gitea/internal/models"
)

type Configuer interface {
	Create(ctx context.Context) (models.BackupConfig, error)
}
