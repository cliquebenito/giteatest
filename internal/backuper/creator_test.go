package backuper

import (
	"context"
	"strconv"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"code.gitea.io/gitea/internal/backuper/mocks"
	"code.gitea.io/gitea/internal/models"
)

func Test_backuper_RunCreateBackupCommand(t *testing.T) {
	ctx := context.Background()
	fs := afero.NewMemMapFs()
	runnerMock := mocks.NewRunner(t)
	config := models.BackupConfig{}
	incremental := false

	mockedBackuper, err := NewBackupCreatorWithConfig(fs, config, runnerMock, testBinaryFinder, testTargetPath, incremental)
	assert.NoError(t, err)

	tempFilePath := mock.AnythingOfType("string")

	runnerMock.On(
		"Run",
		testCtx,
		"/usr/local/bin/gitaly-backup",
		"create",
		"-path",
		"/tmp/sc-gitaly-backup",
		"-parallel",
		strconv.Itoa(gitalyBackupDefaultNumberOfThreads),
		"<",
		tempFilePath,
	).Return(nil)

	err = mockedBackuper.RunCreateBackupCommand(ctx)
	assert.NoError(t, err)
}
