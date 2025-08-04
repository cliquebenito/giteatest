package protected_brancher

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"code.gitea.io/gitea/models/git/protected_branch"
	repo_model "code.gitea.io/gitea/models/repo"
	protectd_branch_mocks "code.gitea.io/gitea/services/protected_branch/mocks"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var manager ProtectedBranchManager
var mockDB *protectd_branch_mocks.ManagerProtectedBranchDB

func init() {
	checker = NewProtectedBranchChecker()
	getter = NewProtectedBranchGetter()
	merger = NewProtectedBranchMerger()
	updater = NewProtectedBranchUpdater()
	mockUpdater = new(protectd_branch_mocks.ProtectedBranchUpdater)
	mockDB = new(protectd_branch_mocks.ManagerProtectedBranchDB)
	manager = NewProtectedBranchManager(getter, checker, merger, mockUpdater, mockDB)
}

func TestGetFirstMatched(t *testing.T) {
	cases := []struct {
		name           string
		rules          protected_branch.ProtectedBranchRules
		branchName     string
		expectedResult *protected_branch.ProtectedBranch
	}{
		{
			name:           "rules is empty",
			rules:          protected_branch.ProtectedBranchRules{},
			branchName:     "branch1",
			expectedResult: nil,
		},
		{
			name: "rules is not empty, branchName does not match any rule",
			rules: protected_branch.ProtectedBranchRules{
				&protected_branch.ProtectedBranch{RuleName: "pattern1"},
				&protected_branch.ProtectedBranch{RuleName: "pattern2"},
			},
			branchName:     "branch1",
			expectedResult: nil,
		},
		{
			name: "rules is not empty, branchName matches one of the rules",
			rules: protected_branch.ProtectedBranchRules{
				&protected_branch.ProtectedBranch{RuleName: "pattern1"},
				&protected_branch.ProtectedBranch{RuleName: "pattern2"},
			},
			branchName:     "pattern1",
			expectedResult: &protected_branch.ProtectedBranch{RuleName: "pattern1"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := manager.GetFirstMatched(context.TODO(), tc.rules, tc.branchName)
			require.Equal(t, tc.expectedResult, result)
		})
	}
}

func TestGetMatchProtectedBranchRules(t *testing.T) {
	cases := []struct {
		name           string
		rules          protected_branch.ProtectedBranchRules
		branchName     string
		expectedResult protected_branch.ProtectedBranchRules
	}{
		{
			name:           "rules is empty",
			rules:          protected_branch.ProtectedBranchRules{},
			branchName:     "branch1",
			expectedResult: protected_branch.ProtectedBranchRules{},
		},
		{
			name: "rules is not empty, branchName does not match any rule",
			rules: protected_branch.ProtectedBranchRules{
				&protected_branch.ProtectedBranch{RuleName: "pattern1"},
				&protected_branch.ProtectedBranch{RuleName: "pattern2"},
			},
			branchName:     "branch1",
			expectedResult: protected_branch.ProtectedBranchRules{},
		},
		{
			name: "rules is not empty, branchName matches one of the rules",
			rules: protected_branch.ProtectedBranchRules{
				&protected_branch.ProtectedBranch{RuleName: "pattern1"},
				&protected_branch.ProtectedBranch{RuleName: "pattern2"},
			},
			branchName: "pattern1",
			expectedResult: protected_branch.ProtectedBranchRules{
				&protected_branch.ProtectedBranch{RuleName: "pattern1"},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := manager.GetMatchProtectedBranchRules(context.TODO(), tc.rules, tc.branchName)
			require.Equal(t, tc.expectedResult, result)
		})
	}
}

func TestGetProtectedBranchRuleByName(t *testing.T) {
	ctx := context.Background()
	repoID := int64(123)
	ruleName := "main"

	expectedRule := &protected_branch.ProtectedBranch{
		RepoID:   repoID,
		RuleName: ruleName,
	}

	t.Run("successfully finds rule", func(t *testing.T) {
		mockDB.On("GetProtectedBranchRuleByName", ctx, repoID, ruleName).
			Return(expectedRule, nil).Once()

		rule, err := manager.GetProtectedBranchRuleByName(ctx, repoID, ruleName)
		require.NoError(t, err)
		require.Equal(t, expectedRule, rule)
		mockDB.AssertExpectations(t)
	})

	t.Run("rule not found", func(t *testing.T) {
		mockDB.On("GetProtectedBranchRuleByName", ctx, repoID, ruleName).
			Return(nil, NewProtectedBranchNotFoundError()).Once()

		rule, err := manager.GetProtectedBranchRuleByName(ctx, repoID, ruleName)
		require.True(t, IsProtectedBranchNotFoundError(err))
		require.Nil(t, rule)
		mockDB.AssertExpectations(t)
	})

	t.Run("other DB error", func(t *testing.T) {
		dbErr := errors.New("Err: get protected branch rule by name: db connection failed")

		mockDB.On("GetProtectedBranchRuleByName", ctx, repoID, ruleName).
			Return(nil, dbErr).Once()

		rule, _ := manager.GetProtectedBranchRuleByName(ctx, repoID, ruleName)
		require.Nil(t, rule)
		mockDB.AssertExpectations(t)
	})
}

func TestFindRepoProtectedBranchRules(t *testing.T) {
	ctx := context.Background()
	repoID := int64(123)

	t.Run("successfully finds rules", func(t *testing.T) {
		expectedRules := protected_branch.ProtectedBranchRules{
			baseRule,
			baseReleaseRule,
			releaseRule1,
			mainRule,
		}

		mockDB.On("FindRepoProtectedBranchRules", ctx, repoID).
			Return(expectedRules, nil).Once()

		rules, err := manager.FindRepoProtectedBranchRules(ctx, repoID)
		require.NoError(t, err)
		require.Equal(t, expectedRules, rules)
		mockDB.AssertExpectations(t)
	})

	t.Run("no rules found", func(t *testing.T) {
		mockDB.On("FindRepoProtectedBranchRules", ctx, repoID).
			Return(protected_branch.ProtectedBranchRules{}, nil).Once()

		rules, _ := manager.FindRepoProtectedBranchRules(ctx, repoID)
		require.Empty(t, rules)
		mockDB.AssertExpectations(t)
	})

	t.Run("database error", func(t *testing.T) {
		dbErr := errors.New("Err: find repo protected branch rules: database operation failed")

		mockDB.On("FindRepoProtectedBranchRules", ctx, repoID).
			Return(nil, dbErr).Once()

		rules, _ := manager.FindRepoProtectedBranchRules(ctx, repoID)
		require.Empty(t, rules)
		mockDB.AssertExpectations(t)
	})
}

func TestCreateProtectedBranch_Success(t *testing.T) {
	mockDB.ExpectedCalls = nil
	mockUpdater.ExpectedCalls = nil

	ctx := context.Background()
	repo := &repo_model.Repository{ID: 1}
	newBranch := &protected_branch.ProtectedBranch{
		RuleName:         "new-feature",
		WhitelistUserIDs: []int64{100},
	}
	expectedBranch := &protected_branch.ProtectedBranch{
		RepoID:           1,
		RuleName:         "new-feature",
		WhitelistUserIDs: []int64{100},
	}

	mockDB.On("GetProtectedBranchRuleByName", ctx, int64(1), "new-feature").
		Return(nil, NewProtectedBranchNotFoundError()).Once()

	mockUpdater.On("UpdateWhitelistOptions", ctx, repo, newBranch,
		protected_branch.WhitelistOptions{
			UserIDs:          []int64{100},
			DeleteUserIDs:    []int64{},
			ForcePushUserIDs: []int64{},
		}).Return(nil).Once()

	mockDB.On("CreateProtectedBranch", ctx, newBranch).
		Return(expectedBranch, nil).Once()

	result, err := manager.CreateProtectedBranch(ctx, repo, newBranch)

	require.NoError(t, err)
	require.Equal(t, expectedBranch, result)

	require.Equal(t, int64(1), result.RepoID)
	require.Equal(t, "new-feature", result.RuleName)
	require.Equal(t, []int64{100}, result.WhitelistUserIDs)

	mockDB.AssertCalled(t, "GetProtectedBranchRuleByName", ctx, int64(1), "new-feature")

	mockUpdater.AssertCalled(t, "UpdateWhitelistOptions", ctx, repo,
		mock.MatchedBy(func(b *protected_branch.ProtectedBranch) bool {
			return b.RepoID == 1 && b.RuleName == "new-feature"
		}),
		mock.MatchedBy(func(opts protected_branch.WhitelistOptions) bool {
			return reflect.DeepEqual(opts.UserIDs, []int64{100})
		}))

	mockDB.AssertCalled(t, "CreateProtectedBranch", ctx,
		mock.MatchedBy(func(b *protected_branch.ProtectedBranch) bool {
			return b.RepoID == 1 && b.RuleName == "new-feature"
		}))

	mockDB.AssertExpectations(t)
	mockUpdater.AssertExpectations(t)
}

func TestCreateProtectedBranch_AlreadyExists(t *testing.T) {
	mockDB.ExpectedCalls = nil
	mockUpdater.ExpectedCalls = nil

	ctx := context.Background()
	repo := &repo_model.Repository{ID: 1}
	existingBranch := &protected_branch.ProtectedBranch{RuleName: "existing-feature"}

	mockDB.On("GetProtectedBranchRuleByName", ctx, int64(1), "existing-feature").
		Return(existingBranch, nil).Once()

	result, err := manager.CreateProtectedBranch(ctx, repo, existingBranch)

	require.True(t, IsProtectedBranchAlreadyExistError(err))
	require.Nil(t, result)

	mockDB.AssertExpectations(t)
	mockUpdater.AssertExpectations(t)
}

func TestCreateProtectedBranch_CheckExistenceError(t *testing.T) {
	mockDB.ExpectedCalls = nil
	mockUpdater.ExpectedCalls = nil

	ctx := context.Background()
	repo := &repo_model.Repository{ID: 1}
	newBranch := &protected_branch.ProtectedBranch{RuleName: "error-feature"}
	expectedErr := errors.New("database connection failed")

	mockDB.On("GetProtectedBranchRuleByName", ctx, int64(1), "error-feature").
		Return(nil, expectedErr).Once()

	result, err := manager.CreateProtectedBranch(ctx, repo, newBranch)

	require.ErrorIs(t, err, expectedErr)
	require.Nil(t, result)

	mockDB.AssertExpectations(t)
	mockUpdater.AssertExpectations(t)
}

func TestCreateProtectedBranch_UpdateWhitelistError(t *testing.T) {
	mockDB.ExpectedCalls = nil
	mockUpdater.ExpectedCalls = nil

	ctx := context.Background()
	repo := &repo_model.Repository{ID: 1}
	newBranch := &protected_branch.ProtectedBranch{
		RuleName:         "update-error-feature",
		WhitelistUserIDs: []int64{100},
	}
	expectedErr := errors.New("update failed")

	mockDB.On("GetProtectedBranchRuleByName", ctx, int64(1), "update-error-feature").
		Return(nil, NewProtectedBranchNotFoundError()).Once()

	mockUpdater.On("UpdateWhitelistOptions", ctx, repo,
		mock.MatchedBy(func(b *protected_branch.ProtectedBranch) bool {
			return b.RepoID == 1 && b.RuleName == "update-error-feature"
		}), mock.AnythingOfType("protected_branch.WhitelistOptions")).
		Return(expectedErr).Once()

	result, err := manager.CreateProtectedBranch(ctx, repo, newBranch)

	require.ErrorIs(t, err, expectedErr)
	require.Nil(t, result)

	mockDB.AssertExpectations(t)
	mockUpdater.AssertExpectations(t)
}

func TestCreateProtectedBranch_WithFullOptions(t *testing.T) {
	mockDB.ExpectedCalls = nil
	mockUpdater.ExpectedCalls = nil

	ctx := context.Background()
	repo := &repo_model.Repository{ID: 1}
	newBranch := &protected_branch.ProtectedBranch{
		RuleName:                  "full-options-feature",
		WhitelistUserIDs:          []int64{100},
		DeleterWhitelistUserIDs:   []int64{400},
		ForcePushWhitelistUserIDs: []int64{500},
	}

	expectedBranch := &protected_branch.ProtectedBranch{
		RepoID:                    1,
		RuleName:                  "full-options-feature",
		WhitelistUserIDs:          []int64{100},
		DeleterWhitelistUserIDs:   []int64{400},
		ForcePushWhitelistUserIDs: []int64{500},
	}

	mockDB.On("GetProtectedBranchRuleByName", ctx, int64(1), "full-options-feature").
		Return(nil, NewProtectedBranchNotFoundError()).Once()

	mockUpdater.
		On("UpdateWhitelistOptions", ctx, repo, mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			br := args.Get(2).(*protected_branch.ProtectedBranch)
			opts := args.Get(3).(protected_branch.WhitelistOptions)

			br.WhitelistUserIDs = opts.UserIDs
			br.DeleterWhitelistUserIDs = opts.DeleteUserIDs
			br.ForcePushWhitelistUserIDs = opts.ForcePushUserIDs
		}).
		Return(nil).Once()

	mockDB.On("CreateProtectedBranch", ctx,
		mock.MatchedBy(func(b *protected_branch.ProtectedBranch) bool {
			return b.RepoID == 1 && b.RuleName == "full-options-feature" && reflect.DeepEqual(b.WhitelistUserIDs, []int64{100})
		})).Return(expectedBranch, nil).Once()

	result, err := manager.CreateProtectedBranch(ctx, repo, newBranch)

	require.NoError(t, err)
	require.Equal(t, expectedBranch, result)

	mockDB.AssertExpectations(t)
	mockUpdater.AssertExpectations(t)
}

func TestCreateProtectedBranch_EmptyUserLists(t *testing.T) {
	mockDB.ExpectedCalls = nil
	mockUpdater.ExpectedCalls = nil

	ctx := context.Background()
	repo := &repo_model.Repository{ID: 1}
	newBranch := &protected_branch.ProtectedBranch{RuleName: "empty-lists-feature"}

	expectedBranch := &protected_branch.ProtectedBranch{RepoID: 1, RuleName: "empty-lists-feature"}

	mockDB.On("GetProtectedBranchRuleByName", ctx, int64(1), "empty-lists-feature").
		Return(nil, NewProtectedBranchNotFoundError()).Once()

	mockUpdater.On("UpdateWhitelistOptions", ctx, repo,
		mock.MatchedBy(func(b *protected_branch.ProtectedBranch) bool {
			return b.RepoID == 1 && b.RuleName == "empty-lists-feature" && len(b.WhitelistUserIDs) == 0
		}), protected_branch.WhitelistOptions{
			UserIDs:          []int64{},
			DeleteUserIDs:    []int64{},
			ForcePushUserIDs: []int64{},
		}).Return(nil).Once()

	mockDB.On("CreateProtectedBranch", ctx,
		mock.MatchedBy(func(b *protected_branch.ProtectedBranch) bool {
			return b.RepoID == 1 && b.RuleName == "empty-lists-feature" && len(b.WhitelistUserIDs) == 0
		})).Return(expectedBranch, nil).Once()

	result, err := manager.CreateProtectedBranch(ctx, repo, newBranch)

	require.NoError(t, err)
	require.Equal(t, expectedBranch, result)

	mockDB.AssertExpectations(t)
	mockUpdater.AssertExpectations(t)
}

func TestUpdateProtectedBranch_Success(t *testing.T) {
	mockDB.ExpectedCalls = nil
	mockUpdater.ExpectedCalls = nil

	ctx := context.Background()
	repo := &repo_model.Repository{ID: 1}
	existingBranch := &protected_branch.ProtectedBranch{ID: 1, RepoID: 1, RuleName: "existing-feature"}
	updatedBranch := &protected_branch.ProtectedBranch{
		RuleName:                  "existing-feature",
		WhitelistUserIDs:          []int64{100},
		DeleterWhitelistUserIDs:   []int64{400},
		ForcePushWhitelistUserIDs: []int64{500},
	}
	expectedResult := &protected_branch.ProtectedBranch{
		ID:                        1,
		RepoID:                    1,
		RuleName:                  "existing-feature",
		WhitelistUserIDs:          []int64{100},
		DeleterWhitelistUserIDs:   []int64{400},
		ForcePushWhitelistUserIDs: []int64{500},
	}

	mockDB.On("GetProtectedBranchRuleByName", ctx, int64(1), "existing-feature").
		Return(existingBranch, nil).Once()

	mockUpdater.On("UpdateWhitelistOptions", ctx, repo, existingBranch, protected_branch.WhitelistOptions{
		UserIDs:          []int64{100},
		DeleteUserIDs:    []int64{400},
		ForcePushUserIDs: []int64{500},
	}).Return(nil).Once()

	mockUpdater.On("UpdateModelProtectedBranch", existingBranch, updatedBranch).
		Return(expectedResult).Once()

	mockDB.On("UpdateProtectBranch", ctx, repo, expectedResult).
		Return(expectedResult, nil).Once()

	result, err := manager.UpdateProtectedBranch(ctx, repo, updatedBranch, "existing-feature")

	require.NoError(t, err)
	require.Equal(t, expectedResult, result)
	mockDB.AssertExpectations(t)
	mockUpdater.AssertExpectations(t)
}

func TestUpdateProtectedBranch_NotFound(t *testing.T) {
	mockDB.ExpectedCalls = nil
	mockUpdater.ExpectedCalls = nil

	ctx := context.Background()
	repo := &repo_model.Repository{ID: 1}
	branch := &protected_branch.ProtectedBranch{RuleName: "non-existent-feature"}

	mockDB.On("GetProtectedBranchRuleByName", ctx, int64(1), "non-existent-feature").
		Return(nil, NewProtectedBranchNotFoundError()).Once()

	result, err := manager.UpdateProtectedBranch(ctx, repo, branch, "non-existent-feature")

	require.True(t, IsProtectedBranchNotFoundError(err))
	require.Nil(t, result)
	mockDB.AssertExpectations(t)
}

func TestUpdateProtectedBranch_WhitelistUpdateError(t *testing.T) {
	mockDB.ExpectedCalls = nil
	mockUpdater.ExpectedCalls = nil

	ctx := context.Background()
	repo := &repo_model.Repository{ID: 1}
	existingBranch := &protected_branch.ProtectedBranch{ID: 1, RepoID: 1, RuleName: "error-feature"}
	updatedBranch := &protected_branch.ProtectedBranch{RuleName: "error-feature"}
	expectedErr := errors.New("whitelist update error")

	mockDB.On("GetProtectedBranchRuleByName", ctx, int64(1), "error-feature").
		Return(existingBranch, nil).Once()

	mockUpdater.On("UpdateWhitelistOptions", ctx, repo, existingBranch, mock.Anything).
		Return(expectedErr).Once()

	result, err := manager.UpdateProtectedBranch(ctx, repo, updatedBranch, "error-feature")

	require.ErrorIs(t, err, expectedErr)
	require.Nil(t, result)
	mockDB.AssertExpectations(t)
	mockUpdater.AssertExpectations(t)
}

func TestUpdateProtectedBranch_DBUpdateError(t *testing.T) {
	mockDB.ExpectedCalls = nil
	mockUpdater.ExpectedCalls = nil

	ctx := context.Background()
	repo := &repo_model.Repository{ID: 1}
	existingBranch := &protected_branch.ProtectedBranch{ID: 1, RepoID: 1, RuleName: "db-error-feature"}
	updatedBranch := &protected_branch.ProtectedBranch{RuleName: "db-error-feature"}
	expectedErr := errors.New("database error")

	mockDB.On("GetProtectedBranchRuleByName", ctx, int64(1), "db-error-feature").
		Return(existingBranch, nil).Once()

	mockUpdater.On("UpdateWhitelistOptions", ctx, repo, existingBranch, mock.Anything).
		Return(nil).Once()

	mockDB.On("UpdateProtectBranch", ctx, repo, mock.Anything).
		Return(nil, expectedErr).Once()

	mockUpdater.On("UpdateModelProtectedBranch", existingBranch, updatedBranch).
		Return(existingBranch).Once()

	result, err := manager.UpdateProtectedBranch(ctx, repo, updatedBranch, "db-error-feature")

	require.ErrorIs(t, err, expectedErr)
	require.Nil(t, result)
	mockDB.AssertExpectations(t)
	mockUpdater.AssertExpectations(t)
}

func TestDeleteProtectedBranchByRuleName(t *testing.T) {
	ctx := context.Background()
	repo := &repo_model.Repository{ID: 1}
	ruleName := "existing-feature"

	t.Run("Success", func(t *testing.T) {
		expectedBranch := &protected_branch.ProtectedBranch{ID: 1, RepoID: repo.ID, RuleName: ruleName}

		mockDB.On("GetProtectedBranchRuleByName", ctx, repo.ID, ruleName).
			Return(expectedBranch, nil).Once()
		mockDB.On("DeleteProtectedBranch", ctx, repo.ID, expectedBranch.ID).
			Return(nil).Once()

		err := manager.DeleteProtectedBranchByRuleName(ctx, repo, ruleName)

		require.NoError(t, err)
		mockDB.AssertExpectations(t)
	})

	t.Run("Branch Not Found", func(t *testing.T) {
		mockDB.On("GetProtectedBranchRuleByName", ctx, repo.ID, "non-existing-feature").
			Return(nil, NewProtectedBranchNotFoundError()).Once()

		err := manager.DeleteProtectedBranchByRuleName(ctx, repo, "non-existing-feature")

		require.True(t, IsProtectedBranchNotFoundError(err))
		mockDB.AssertNotCalled(t, "DeleteProtectedBranch")
	})
}
