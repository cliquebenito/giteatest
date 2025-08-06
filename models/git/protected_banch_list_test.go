//go:build !correct

// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package git

import (
	"fmt"
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

	releaseRule2 = &protected_branch.ProtectedBranch{
		RuleName:                  "release/2.0",
		IsPlainName:               true,
		EnableWhitelist:           true,
		WhitelistUserIDs:          []int64{3},
		EnableForcePushWhitelist:  true,
		ForcePushWhitelistUserIDs: []int64{2, 3},
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

	signedCommitRule = &protected_branch.ProtectedBranch{
		RuleName:             "signed/*",
		RequireSignedCommits: true,
	}

	statusCheckRule = &protected_branch.ProtectedBranch{
		RuleName:            "ci/*",
		EnableStatusCheck:   true,
		StatusCheckContexts: []string{"build", "test"},
	}

	strictMergeRule = &protected_branch.ProtectedBranch{
		RuleName:               "stable",
		IsPlainName:            true,
		BlockOnRejectedReviews: true,
		BlockOnOutdatedBranch:  true,
		EnableSonarQube:        true,
	}
	allRules = protected_branch.ProtectedBranchRules{baseRule, baseReleaseRule, releaseRule1, releaseRule2, mainRule, signedCommitRule, statusCheckRule, strictMergeRule}
)

func TestBranchRuleMatchPriority(t *testing.T) {
	cases := []struct {
		Rules            []string
		BranchName       string
		ExpectedMatchIdx int
	}{
		{
			Rules:            []string{"release/*", "release/v1.17"},
			BranchName:       "release/v1.17",
			ExpectedMatchIdx: 1,
		},
		{
			Rules:            []string{"release/v1.17", "release/*"},
			BranchName:       "release/v1.17",
			ExpectedMatchIdx: 0,
		},
		{
			Rules:            []string{"release/**/v1.17", "release/test/v1.17"},
			BranchName:       "release/test/v1.17",
			ExpectedMatchIdx: 1,
		},
		{
			Rules:            []string{"release/test/v1.17", "release/**/v1.17"},
			BranchName:       "release/test/v1.17",
			ExpectedMatchIdx: 0,
		},
		{
			Rules:            []string{"release/**", "release/v1.0.0"},
			BranchName:       "release/v1.0.0",
			ExpectedMatchIdx: 1,
		},
		{
			Rules:            []string{"release/v1.0.0", "release/**"},
			BranchName:       "release/v1.0.0",
			ExpectedMatchIdx: 0,
		},
		{
			Rules:            []string{"release/**", "release/v1.0.0"},
			BranchName:       "release/v2.0.0",
			ExpectedMatchIdx: 0,
		},
		{
			Rules:            []string{"release/*", "release/v1.0.0"},
			BranchName:       "release/1/v2.0.0",
			ExpectedMatchIdx: -1,
		},
	}

	for _, testCase := range cases {
		var pbs protected_branch.ProtectedBranchRules
		for _, rule := range testCase.Rules {
			pbs = append(pbs, &protected_branch.ProtectedBranch{RuleName: rule})
		}
		pbs = sortRules(pbs)
		matchedPB := GetFirstMatched(pbs, testCase.BranchName)
		if matchedPB == nil {
			if testCase.ExpectedMatchIdx >= 0 {
				require.Error(t, fmt.Errorf("no matched rules but expected %s[%d]", testCase.Rules[testCase.ExpectedMatchIdx], testCase.ExpectedMatchIdx))
			}
		} else {
			require.EqualValues(t, testCase.Rules[testCase.ExpectedMatchIdx], matchedPB.RuleName)
		}
	}
}

func TestGetMatchProtectedBrancheRule_MatchesMain(t *testing.T) {
	result := GetMatchProtectedBranchRules(allRules, "main")

	require.Len(t, result, 2)
	require.Contains(t, result, baseRule)
	require.Contains(t, result, mainRule)
}

func TestGetMatchProtectedBrancheRule_MatchesRelease10(t *testing.T) {
	result := GetMatchProtectedBranchRules(allRules, "release/1.0")

	require.Len(t, result, 3)
	require.Contains(t, result, baseRule)
	require.Contains(t, result, baseReleaseRule)
	require.Contains(t, result, releaseRule1)
}

func TestGetMatchProtectedBrancheRule_MatchesRelease20(t *testing.T) {
	result := GetMatchProtectedBranchRules(allRules, "release/2.0")

	require.Len(t, result, 3)
	require.Contains(t, result, baseRule)
	require.Contains(t, result, baseReleaseRule)
	require.Contains(t, result, releaseRule2)
}

func TestGetMatchProtectedBrancheRule_MatchesCI(t *testing.T) {
	result := GetMatchProtectedBranchRules(allRules, "ci/build")

	require.Len(t, result, 2)
	require.Contains(t, result, baseRule)
	require.Contains(t, result, statusCheckRule)
}

func TestGetMatchProtectedBrancheRule_MatchesSigned(t *testing.T) {
	result := GetMatchProtectedBranchRules(allRules, "signed/feature")

	require.Len(t, result, 2)
	require.Contains(t, result, baseRule)
	require.Contains(t, result, signedCommitRule)
}

func TestGetMatchProtectedBrancheRule_MatchesStable(t *testing.T) {
	result := GetMatchProtectedBranchRules(allRules, "stable")

	require.Len(t, result, 2)
	require.Contains(t, result, baseRule)
	require.Contains(t, result, strictMergeRule)
}

func TestGetMatchProtectedBrancheRule_NoMatch(t *testing.T) {
	result := GetMatchProtectedBranchRules(allRules, "feature/experimental")

	require.Len(t, result, 1)
	require.Contains(t, result, baseRule)
}

func TestMergeProtectedBranchRules_SimpleMerge(t *testing.T) {
	rules := protected_branch.ProtectedBranchRules{
		baseRule,
		releaseRule1,
	}

	result := MergeProtectedBranchRules(rules)

	require.NotNil(t, result)
	require.Contains(t, result.WhitelistUserIDs, int64(3))
	require.Contains(t, result.WhitelistUserIDs, int64(4))
}

func TestMergeProtectedBranchRules_NoRules(t *testing.T) {
	rules := protected_branch.ProtectedBranchRules{}

	result := MergeProtectedBranchRules(rules)

	require.Nil(t, result)
}

func TestMergeProtectedBranchRules_OverrideRule(t *testing.T) {
	rules := protected_branch.ProtectedBranchRules{
		baseRule,
		mainRule,
	}

	result := MergeProtectedBranchRules(rules)

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

	result := MergeProtectedBranchRules(rules)

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
