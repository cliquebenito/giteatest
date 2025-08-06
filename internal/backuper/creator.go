package backuper

import (
	"context"
	"fmt"

	"github.com/spf13/afero"

	"code.gitea.io/gitea/internal/models"
)

type BackupCreator struct {
	fs                  afero.Fs
	config              models.BackupConfig
	gitalyBackupCLIPath string
	shellRunner         runner
	targetPath          string

	incremental bool
}

func NewBackupCreatorWithConfig(
	fs afero.Fs,
	config models.BackupConfig,
	shellRunner runner,
	finder binaryFinder,
	targetPath string,
	incremental bool,
) (BackupCreator, error) {
	gitalyBackupCLIPath, err := finder(gitalyBackupCLIName)
	if err != nil {
		return BackupCreator{}, fmt.Errorf("miss %s binary: %w", gitalyBackupCLIName, err)
	}

	return BackupCreator{
		fs:                  fs,
		config:              config,
		shellRunner:         shellRunner,
		targetPath:          targetPath,
		gitalyBackupCLIPath: gitalyBackupCLIPath,
		incremental:         incremental,
	}, nil
}

func (b BackupCreator) createTempFile() (string, error) {
	return createTempFile(b.fs, b.config)
}

func (b BackupCreator) RunCreateBackupCommand(ctx context.Context) error {
	backupConfigPath, err := b.createTempFile()
	if err != nil {
		return fmt.Errorf("create backup config file: %w", err)
	}

	arguments := b.buildGitalyCreateBackupCLICommand(b.targetPath, backupConfigPath, gitalyBackupDefaultNumberOfThreads, b.incremental)

	if err = b.shellRunner.Run(ctx, b.gitalyBackupCLIPath, arguments...); err != nil {
		return fmt.Errorf("run `gitaly-backup create` command: %w", err)
	}

	return nil
}
