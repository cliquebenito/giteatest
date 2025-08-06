package code_hub_counter

import (
	"context"
	"testing"

	"code.gitea.io/gitea/models/code_hub_counter_task"
	"code.gitea.io/gitea/routers/private/code_hub_counter/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var (
	testCtx = mock.AnythingOfType("context.backgroundCtx")
)

func TestInsert(t *testing.T) {
	var (
		repoId int64 = 1
		userId int64 = 2
	)
	ctx := context.Background()
	mockInserter := mocks.NewUsagesTasksDB(t)
	mockInserter.
		On("InsertTask", testCtx, repoId, userId, code_hub_counter_task.CloneRepositoryAction).
		Return(nil)

	err := mockInserter.InsertTask(ctx, repoId, userId, code_hub_counter_task.CloneRepositoryAction)

	require.NoError(t, err)
	mockInserter.AssertCalled(t, "InsertTask", testCtx, repoId, userId, code_hub_counter_task.CloneRepositoryAction)
}
