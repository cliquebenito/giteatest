package code_hub_counter

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"code.gitea.io/gitea/models/code_hub_counter_task"
	"code.gitea.io/gitea/models/internal_metric_counter"
	"code.gitea.io/gitea/routers/private/code_hub_counter/mocks"
)

func TestProcessTasksSuccess(t *testing.T) {
	ctx := context.Background()
	userID := int64(1)
	repoID := int64(2)
	mockTaskDB := mocks.NewTaskDB(t)
	mockUniqueUsagesDB := mocks.NewUniqueUsagesDB(t)
	counter := NewCodeHubCounter(mockTaskDB, mockUniqueUsagesDB, nil)
	counterTasksReturn := make([]code_hub_counter_task.CodeHubCounterTasks, 1)
	counterTasksReturn[0] = code_hub_counter_task.CodeHubCounterTasks{
		ID:     int64(1),
		UserID: userID,
		RepoID: repoID,
		Action: code_hub_counter_task.CloneRepositoryAction,
		Status: code_hub_counter_task.StatusUnlocked,
	}
	mockTaskDB.
		On("GetCodeHubCounterTasks", testCtx).
		Return(counterTasksReturn, nil)
	mockTaskDB.
		On("LockTask", testCtx, int64(1)).
		Return(nil)
	mockUniqueUsagesDB.
		On("CountUniqueUsages", testCtx, repoID, userID).
		Return(0, nil)
	mockUniqueUsagesDB.
		On("UpdateUniqueUsage", testCtx, repoID, userID).
		Return(nil)
	mockTaskDB.
		On("UnlockTaskWithSuccess", testCtx, int64(1)).
		Return(nil)
	mockTaskDB.
		On("DeleteTask", testCtx, int64(1)).
		Return(nil)

	err := counter.ProcessNewUsageTasks(ctx)
	assert.NoError(t, err)

	mockUniqueUsagesDB.AssertCalled(t, "UpdateUniqueUsage", testCtx, repoID, userID)
	mockTaskDB.AssertCalled(t, "DeleteTask", testCtx, int64(1))
}

func TestProcessTasksSuccessDuplicate(t *testing.T) {
	ctx := context.Background()
	userID := int64(1)
	repoID := int64(2)
	mockTaskDB := mocks.NewTaskDB(t)
	mockUniqueUsagesDB := mocks.NewUniqueUsagesDB(t)
	counter := NewCodeHubCounter(mockTaskDB, mockUniqueUsagesDB, nil)
	counterTasksReturn := make([]code_hub_counter_task.CodeHubCounterTasks, 1)
	counterTasksReturn[0] = code_hub_counter_task.CodeHubCounterTasks{
		ID:     int64(1),
		UserID: userID,
		RepoID: repoID,
		Action: code_hub_counter_task.CloneRepositoryAction,
		Status: code_hub_counter_task.StatusUnlocked,
	}
	mockTaskDB.
		On("GetCodeHubCounterTasks", testCtx).
		Return(counterTasksReturn, nil)
	mockTaskDB.
		On("LockTask", testCtx, int64(1)).
		Return(nil)
	// Не уникальное использование
	mockUniqueUsagesDB.
		On("CountUniqueUsages", testCtx, repoID, userID).
		Return(1, nil)
	mockTaskDB.
		On("UnlockTaskWithSuccess", testCtx, int64(1)).
		Return(nil)
	mockTaskDB.
		On("DeleteTask", testCtx, int64(1)).
		Return(nil)

	err := counter.ProcessNewUsageTasks(ctx)
	assert.NoError(t, err)

	mockUniqueUsagesDB.AssertNotCalled(t, "UpdateUniqueUsage", testCtx, mock.Anything, mock.Anything)
	mockTaskDB.AssertCalled(t, "DeleteTask", testCtx, int64(1))
}

func TestProcessTasksHandleErr(t *testing.T) {
	ctx := context.Background()
	userID := int64(1)
	repoID := int64(2)
	mockTaskDB := mocks.NewTaskDB(t)
	mockUniqueUsagesDB := mocks.NewUniqueUsagesDB(t)
	counter := NewCodeHubCounter(mockTaskDB, mockUniqueUsagesDB, nil)
	counterTasksReturn := make([]code_hub_counter_task.CodeHubCounterTasks, 1)
	counterTasksReturn[0] = code_hub_counter_task.CodeHubCounterTasks{
		ID:     int64(1),
		UserID: userID,
		RepoID: repoID,
		Action: code_hub_counter_task.CloneRepositoryAction,
		Status: code_hub_counter_task.StatusUnlocked,
	}
	mockTaskDB.
		On("GetCodeHubCounterTasks", testCtx).
		Return(counterTasksReturn, nil)
	mockTaskDB.
		On("LockTask", testCtx, int64(1)).
		Return(nil)
	mockUniqueUsagesDB.
		On("CountUniqueUsages", testCtx, repoID, userID).
		Return(0, errors.New(""))
	mockTaskDB.
		On("UnlockTask", testCtx, int64(1)).
		Return(nil)

	err := counter.ProcessNewUsageTasks(ctx)
	assert.NoError(t, err)

	mockTaskDB.AssertCalled(t, "UnlockTask", testCtx, int64(1))
}

func TestCalculateRepoCounters(t *testing.T) {
	ctx := context.Background()
	repoID := int64(1)
	mockCounterDB := mocks.NewCounterDB(t)
	mockUniqueUsagesDB := mocks.NewUniqueUsagesDB(t)
	counter := NewCodeHubCounter(nil, mockUniqueUsagesDB, mockCounterDB)
	countersReturn := make([]internal_metric_counter.InternalMetricCounter, 1)
	countersReturn[0] = internal_metric_counter.InternalMetricCounter{
		ID:     int64(1),
		RepoID: repoID,
	}
	mockCounterDB.
		On("GetInternalMetricCounters", testCtx).
		Return(countersReturn, nil)
	mockUniqueUsagesDB.
		On("CountUniqueUsagesByRepoID", testCtx, repoID).
		Return(1, nil)
	mockCounterDB.
		On("UpdateCounter", testCtx, repoID, 1, mock.Anything).
		Return(nil)

	err := counter.CalculateRepoCounters(ctx)
	assert.NoError(t, err)

	mockCounterDB.AssertCalled(t, "UpdateCounter", testCtx, repoID, 1, mock.Anything)
}
