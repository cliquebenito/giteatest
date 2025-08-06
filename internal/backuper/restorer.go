package backuper

import (
	"context"
	"fmt"

	"github.com/spf13/afero"

	"code.gitea.io/gitea/internal/models"
)

type BackupRestorer struct {
	fs                  afero.Fs
	config              models.BackupConfig
	gitalyBackupCLIPath string
	shellRunner         runner
	targetPath          string
}

func NewBackupRestorerWithConfig(
	fs afero.Fs,
	config models.BackupConfig,
	shellRunner runner,
	finder binaryFinder,
	targetPath string,
) (BackupRestorer, error) {
	gitalyBackupCLIPath, err := finder(gitalyBackupCLIName)
	if err != nil {
		return BackupRestorer{}, fmt.Errorf("miss %s binary: %w", gitalyBackupCLIName, err)
	}

	return BackupRestorer{
		fs:                  fs,
		config:              config,
		shellRunner:         shellRunner,
		targetPath:          targetPath,
		gitalyBackupCLIPath: gitalyBackupCLIPath,
	}, nil
}

func (b BackupRestorer) createTempFile() (string, error) {
	return createTempFile(b.fs, b.config)
}

func (b BackupRestorer) RunRestoreBackupCommand(ctx context.Context) error {
	backupConfigPath, err := b.createTempFile()
	if err != nil {
		return fmt.Errorf("create backup config file: %w", err)
	}

	arguments := b.buildGitalyRestoreBackupCLIACommand(b.targetPath, backupConfigPath, gitalyBackupDefaultNumberOfThreads)

	if err = b.shellRunner.Run(ctx, b.gitalyBackupCLIPath, arguments...); err != nil {
		return fmt.Errorf("run `gitaly-backup restore` command: %w", err)
	}

	return nil
}
