package protected_brancher

import (
	"testing"

	"code.gitea.io/gitea/models/git/protected_branch"

	"github.com/stretchr/testify/require"
)

var (
	baseRule = &protected_branch.ProtectedBranch{
		RuleName:                  "**",
		EnableDeleterWhitelist:    true,
		DeleterWhitelistUserIDs:   []int64{1, 2},
		EnableApprovalsWhitelist:  true,
		ApprovalsWhitelistUserIDs: []int64{3},
		RequiredApprovals:         1,
	}

	baseReleaseRule = &protected_branch.ProtectedBranch{
		RuleName:              "release/**",
		EnableMergeWhitelist:  true,
		MergeWhitelistUserIDs: []int64{2, 3},
	}

	releaseRule1 = &protected_branch.ProtectedBranch{
		RuleName:              "release/1.0",
		IsPlainName:           true,
		EnableWhitelist:       true,
		WhitelistUserIDs:      []int64{3, 4},
		EnableMergeWhitelist:  true,
		MergeWhitelistUserIDs: []int64{4, 5},
	}

	mainRule = &protected_branch.ProtectedBranch{
		RuleName:                  "main",
		IsPlainName:               true,
		EnableWhitelist:           true,
		WhitelistUserIDs:          []int64{},
		EnableMergeWhitelist:      true,
		MergeWhitelistUserIDs:     []int64{1},
		EnableApprovalsWhitelist:  true,
		ApprovalsWhitelistUserIDs: []int64{1, 2},
		RequiredApprovals:         2,
	}

	merger *ProtectedBranchMerger
)

func init() {
	merger = NewProtectedBranchMerger()
}

func TestMergeProtectedBranchRules_NoRules(t *testing.T) {
	rules := protected_branch.ProtectedBranchRules{}

	result := merger.MergeProtectedBranchRules(nil, rules)

	require.Nil(t, result)
}

func TestMergeProtectedBranchRules_OverrideRule(t *testing.T) {
	rules := protected_branch.ProtectedBranchRules{
		baseRule,
		mainRule,
	}

	result := merger.MergeProtectedBranchRules(nil, rules)

	require.Equal(t, "main", result.RuleName)
	require.True(t, result.IsPlainName)

	require.True(t, result.EnableWhitelist)
	require.Empty(t, result.WhitelistUserIDs)

	require.True(t, result.EnableMergeWhitelist)
	require.ElementsMatch(t, []int64{1}, result.MergeWhitelistUserIDs)

	require.True(t, result.EnableApprovalsWhitelist)
	require.ElementsMatch(t, []int64{1, 2, 3}, result.ApprovalsWhitelistUserIDs)

	require.True(t, result.EnableDeleterWhitelist)
	require.ElementsMatch(t, []int64{1, 2}, result.DeleterWhitelistUserIDs)

	require.Equal(t, int64(3), result.RequiredApprovals)

	require.False(t, result.RequireSignedCommits)
	require.False(t, result.EnableForcePushWhitelist)
	require.False(t, result.EnableStatusCheck)
	require.False(t, result.BlockOnRejectedReviews)
	require.False(t, result.BlockOnOutdatedBranch)
	require.False(t, result.EnableSonarQube)
}

func TestMergeProtectedBranchRules_MergeMultipleRules(t *testing.T) {
	rules := protected_branch.ProtectedBranchRules{
		baseRule,
		releaseRule1,
		baseReleaseRule,
	}

	result := merger.MergeProtectedBranchRules(nil, rules)

	require.Equal(t, "release/1.0", result.RuleName)
	require.True(t, result.IsPlainName)

	require.True(t, result.EnableWhitelist)
	require.ElementsMatch(t, []int64{3, 4}, result.WhitelistUserIDs)

	require.True(t, result.EnableMergeWhitelist)
	require.ElementsMatch(t, []int64{2, 3, 4, 5}, result.MergeWhitelistUserIDs)

	require.True(t, result.EnableApprovalsWhitelist)
	require.ElementsMatch(t, []int64{3}, result.ApprovalsWhitelistUserIDs)

	require.True(t, result.EnableDeleterWhitelist)
	require.ElementsMatch(t, []int64{1, 2}, result.DeleterWhitelistUserIDs)

	require.Equal(t, int64(1), result.RequiredApprovals)

	require.False(t, result.RequireSignedCommits)
	require.False(t, result.EnableForcePushWhitelist)
	require.False(t, result.BlockOnRejectedReviews)
	require.False(t, result.BlockOnOutdatedBranch)
	require.False(t, result.EnableSonarQube)
}
