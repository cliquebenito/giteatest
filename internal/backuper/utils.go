package backuper

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/afero"

	"code.gitea.io/gitea/internal/models"
)

const gitalyBackupCLIName = "gitaly-backup"

const gitalyBackupDefaultNumberOfThreads = 4

// //go:generate mockery --name=runner --exported
type runner interface {
	Run(ctx context.Context, command string, args ...string) error
}

type binaryFinder func(name string) (string, error)

func createTempFile(fs afero.Fs, config models.BackupConfig) (string, error) {
	const tmpDir = "/tmp/sc-gitaly-backup"

	if err := fs.RemoveAll(tmpDir); err != nil {
		if !os.IsNotExist(err) {
			return "", fmt.Errorf("remove temp dir: %w", err)
		}
	}

	if err := fs.MkdirAll(tmpDir, os.ModePerm); err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}

	file, err := afero.TempFile(fs, tmpDir, "backup-")
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}

	backupConfigFullPath := file.Name()

	backupConfigBody, err := config.JSON()
	if err != nil {
		return "", fmt.Errorf("marshal config: %w", err)
	}

	if err = afero.WriteFile(fs, backupConfigFullPath, backupConfigBody, os.ModePerm); err != nil {
		return "", fmt.Errorf("write config: %w", err)
	}

	return file.Name(), nil
}
