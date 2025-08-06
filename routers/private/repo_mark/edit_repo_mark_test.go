package repo_mark

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/routers/private/repo_mark/mocks"
)

func TestDelete(t *testing.T) {
	var (
		repoKey       = "repo"
		repoID  int64 = 1
	)

	ctx := context.Background()
	mockRepoKey := mocks.NewRepoKeyDB(t)
	mockEditRepoMarks := mocks.NewEditRepoMarksDB(t)
	testMark1 := testMark{label: "Test", key: "test"}
	mockEditor := NewRepoMarksEditor(mockEditRepoMarks, mockRepoKey)
	mockRepoKey.
		On("GetRepoByKey", testCtx, repoKey).
		Return(&repo.ScRepoKey{RepoKey: repoKey, RepoID: "1"}, nil)
	mockEditRepoMarks.
		On("DeleteRepoMark", testCtx, repoID, testMark1).
		Return(nil)

	err := mockEditor.DeleteRepoMark(ctx, repoKey, testMark1)

	require.NoError(t, err)
}

func TestInsert(t *testing.T) {
	var (
		repoKey        = "repo"
		repoID   int64 = 1
		expertID int64 = 2
	)

	ctx := context.Background()
	mockRepoKey := mocks.NewRepoKeyDB(t)
	mockEditRepoMarks := mocks.NewEditRepoMarksDB(t)
	testMark1 := testMark{label: "Test", key: "test"}
	mockEditor := NewRepoMarksEditor(mockEditRepoMarks, mockRepoKey)
	mockRepoKey.
		On("GetRepoByKey", testCtx, repoKey).
		Return(&repo.ScRepoKey{RepoKey: repoKey, RepoID: "1"}, nil)
	mockEditRepoMarks.
		On("InsertRepoMark", testCtx, repoID, expertID, testMark1).
		Return(nil)

	err := mockEditor.InsertRepoMark(ctx, repoKey, expertID, testMark1)

	require.NoError(t, err)
}
