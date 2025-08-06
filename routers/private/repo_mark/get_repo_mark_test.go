package repo_mark

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"code.gitea.io/gitea/models/repo_marks"
	"code.gitea.io/gitea/routers/private/repo_mark/mocks"
)

var (
	testCtx = mock.AnythingOfType("context.backgroundCtx")
)

type testMark struct {
	label string
	key   string
}

func (c testMark) Label() string {
	return c.label
}

func (c testMark) Key() string {
	return c.key
}

func TestGet(t *testing.T) {
	var repoId int64 = 1
	ctx := context.Background()
	mockRepoMarks := mocks.NewRepoMarksDB(t)
	testMark1 := testMark{label: "Test", key: "test"}
	testMarks1 := repo_marks.RepoMarks{RepoID: repoId, MarkKey: "test"}
	mockGetter := NewRepoMarksGetter(mockRepoMarks, []repo_marks.RepoMark{testMark1})
	mockRepoMarks.
		On("GetRepoMarks", testCtx, repoId).
		Return([]repo_marks.RepoMarks{testMarks1}, nil)

	resp, err := mockGetter.GetRepoMarks(ctx, repoId)

	require.NoError(t, err)
	require.Equal(t, 1, len(resp.Marks))
	assert.Equal(t, "Test", resp.Marks[0].Label)
}

func TestGetEmptyDefault(t *testing.T) {
	var repoId int64 = 1
	ctx := context.Background()
	mockRepoMarks := mocks.NewRepoMarksDB(t)
	testMarks1 := repo_marks.RepoMarks{RepoID: repoId, MarkKey: "test"}
	mockGetter := NewRepoMarksGetter(mockRepoMarks, []repo_marks.RepoMark{})
	mockRepoMarks.
		On("GetRepoMarks", testCtx, repoId).
		Return([]repo_marks.RepoMarks{testMarks1}, nil)

	resp, err := mockGetter.GetRepoMarks(ctx, repoId)

	require.NoError(t, err)
	require.Equal(t, 0, len(resp.Marks))
}

func TestGetMismatchDefault(t *testing.T) {
	var repoId int64 = 1
	ctx := context.Background()
	mockRepoMarks := mocks.NewRepoMarksDB(t)
	testMark1 := testMark{label: "Test", key: "test_test"}
	testMarks1 := repo_marks.RepoMarks{RepoID: repoId, MarkKey: "test"}
	mockGetter := NewRepoMarksGetter(mockRepoMarks, []repo_marks.RepoMark{testMark1})
	mockRepoMarks.
		On("GetRepoMarks", testCtx, repoId).
		Return([]repo_marks.RepoMarks{testMarks1}, nil)

	resp, err := mockGetter.GetRepoMarks(ctx, repoId)

	require.NoError(t, err)
	assert.Equal(t, 0, len(resp.Marks))
}

func TestGetDoubleDefault(t *testing.T) {
	var repoId int64 = 1
	ctx := context.Background()
	mockRepoMarks := mocks.NewRepoMarksDB(t)
	testMark1 := testMark{label: "Test-1", key: "test_1"}
	testMark2 := testMark{label: "Test-2", key: "test_2"}
	testMarks1 := repo_marks.RepoMarks{RepoID: repoId, MarkKey: "test_2"}
	mockGetter := NewRepoMarksGetter(mockRepoMarks, []repo_marks.RepoMark{testMark1, testMark2})
	mockRepoMarks.
		On("GetRepoMarks", testCtx, repoId).
		Return([]repo_marks.RepoMarks{testMarks1}, nil)

	resp, err := mockGetter.GetRepoMarks(ctx, repoId)

	require.NoError(t, err)
	require.Equal(t, 1, len(resp.Marks))
	assert.Equal(t, "Test-2", resp.Marks[0].Label)
}
