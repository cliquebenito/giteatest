//go:build !correct

// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package git

import (
	"testing"

	"code.gitea.io/gitea/models/git/protected_branch"
	"github.com/stretchr/testify/require"
)

func TestJoinPatterns_BothNonEmpty(t *testing.T) {
	result := joinPatterns("foo", "bar")
	require.Equal(t, "foo;bar", result)
}

func TestJoinPatterns_AIsEmpty(t *testing.T) {
	result := joinPatterns("", "bar")
	require.Equal(t, "bar", result)
}

func TestJoinPatterns_BIsEmpty(t *testing.T) {
	result := joinPatterns("foo", "")
	require.Equal(t, "foo", result)
}

func TestJoinPatterns_BothEmpty(t *testing.T) {
	result := joinPatterns("", "")
	require.Equal(t, "", result)
}

func TestMergeWhiteLists_FlagFalse(t *testing.T) {
	result := mergeWhiteLists(false, []int64{1, 2}, []int64{3})
	require.Empty(t, result)
}

func TestMergeWhiteLists_EmptyLists(t *testing.T) {
	result := mergeWhiteLists(true, []int64{}, []int64{})
	require.Empty(t, result)
}

func TestMergeWhiteLists_UniqueValues(t *testing.T) {
	result := mergeWhiteLists(true, []int64{1, 2}, []int64{3, 4})
	require.ElementsMatch(t, []int64{1, 2, 3, 4}, result)
}

func TestMergeWhiteLists_WithDuplicates(t *testing.T) {
	result := mergeWhiteLists(true, []int64{1, 2, 2}, []int64{2, 3})
	require.ElementsMatch(t, []int64{1, 2, 3}, result)
}

func TestMergeWhiteLists_NilInput(t *testing.T) {
	result := mergeWhiteLists(true, nil, []int64{1})
	require.ElementsMatch(t, []int64{1}, result)
}

func TestMergeStringLists_FlagFalse(t *testing.T) {
	result := mergeStringLists(false, []string{"a"}, []string{"b"})
	require.Empty(t, result)
}

func TestMergeStringLists_EmptyLists(t *testing.T) {
	result := mergeStringLists(true, []string{}, []string{})
	require.Empty(t, result)
}

func TestMergeStringLists_Unique(t *testing.T) {
	result := mergeStringLists(true, []string{"a"}, []string{"b"})
	require.ElementsMatch(t, []string{"a", "b"}, result)
}

func TestMergeStringLists_Duplicates(t *testing.T) {
	result := mergeStringLists(true, []string{"a", "a"}, []string{"a", "b"})
	require.ElementsMatch(t, []string{"a", "b"}, result)
}

func TestMergeStringLists_CaseSensitivity(t *testing.T) {
	result := mergeStringLists(true, []string{"A"}, []string{"a"})
	require.ElementsMatch(t, []string{"A", "a"}, result)
}

func TestMergeStringLists_NilInput(t *testing.T) {
	result := mergeStringLists(true, nil, []string{"a"})
	require.ElementsMatch(t, []string{"a"}, result)
}

func TestMergeProtectedBranch_NilInput(t *testing.T) {
	pb := &protected_branch.ProtectedBranch{ID: 1, RuleName: "main"}
	result := MergeProtectedBranch(pb, nil)
	require.Equal(t, pb, result)
}

func TestMergeProtectedBranch_IsPlainNameOverridesNew(t *testing.T) {
	pb := &protected_branch.ProtectedBranch{
		ID:               10,
		RepoID:           100,
		RuleName:         "pb-rule",
		IsPlainName:      true,
		CreatedUnix:      123,
		UpdatedUnix:      456,
		EnableWhitelist:  true,
		WhitelistUserIDs: []int64{1, 2},
	}
	newPB := &protected_branch.ProtectedBranch{
		ID:               99,
		RepoID:           999,
		RuleName:         "new-rule",
		IsPlainName:      false,
		CreatedUnix:      789,
		UpdatedUnix:      987,
		EnableWhitelist:  true,
		WhitelistUserIDs: []int64{2, 3},
	}

	merged := MergeProtectedBranch(pb, newPB)
	require.Equal(t, int64(10), merged.ID)
	require.Equal(t, "pb-rule", merged.RuleName)
	require.True(t, merged.EnableWhitelist)
	require.ElementsMatch(t, []int64{1, 2, 3}, merged.WhitelistUserIDs)
}

func TestMergeProtectedBranch_PatternJoin(t *testing.T) {
	pb := &protected_branch.ProtectedBranch{
		ProtectedFilePatterns:   "a,b",
		UnprotectedFilePatterns: "x",
	}
	newPB := &protected_branch.ProtectedBranch{
		ProtectedFilePatterns:   "c",
		UnprotectedFilePatterns: "y,z",
	}
	merged := MergeProtectedBranch(pb, newPB)
	require.Equal(t, "a,b;c", merged.ProtectedFilePatterns)
	require.Equal(t, "x;y,z", merged.UnprotectedFilePatterns)
}

func TestMergeProtectedBranch_FlagsAndStatusCheckMerge(t *testing.T) {
	pb := &protected_branch.ProtectedBranch{
		EnableStatusCheck:             true,
		StatusCheckContexts:           []string{"a"},
		RequiredApprovals:             1,
		BlockOnRejectedReviews:        false,
		BlockOnOfficialReviewRequests: false,
		BlockOnOutdatedBranch:         true,
		EnableSonarQube:               false,
	}
	newPB := &protected_branch.ProtectedBranch{
		EnableStatusCheck:             true,
		StatusCheckContexts:           []string{"b"},
		RequiredApprovals:             2,
		BlockOnRejectedReviews:        true,
		BlockOnOfficialReviewRequests: true,
		BlockOnOutdatedBranch:         false,
		EnableSonarQube:               true,
	}
	merged := MergeProtectedBranch(pb, newPB)
	require.ElementsMatch(t, []string{"a", "b"}, merged.StatusCheckContexts)
	require.Equal(t, int64(3), merged.RequiredApprovals)
	require.True(t, merged.BlockOnRejectedReviews)
	require.True(t, merged.BlockOnOfficialReviewRequests)
	require.True(t, merged.EnableSonarQube)
}

func TestMergeProtectedBranch_MergeWhiteLists(t *testing.T) {
	pb := &protected_branch.ProtectedBranch{
		EnableWhitelist:  true,
		WhitelistUserIDs: []int64{1, 2},

		EnableForcePushWhitelist:  true,
		ForcePushWhitelistUserIDs: []int64{3, 4},

		EnableDeleterWhitelist:  true,
		DeleterWhitelistUserIDs: []int64{5, 6},

		EnableMergeWhitelist:  true,
		MergeWhitelistUserIDs: []int64{7, 8},
	}
	newPB := &protected_branch.ProtectedBranch{
		EnableWhitelist:  true,
		WhitelistUserIDs: []int64{2, 3},

		EnableForcePushWhitelist:  true,
		ForcePushWhitelistUserIDs: []int64{5, 6},

		EnableDeleterWhitelist:  true,
		DeleterWhitelistUserIDs: []int64{},

		EnableMergeWhitelist:  true,
		MergeWhitelistUserIDs: []int64{7},
	}
	merged := MergeProtectedBranch(pb, newPB)
	require.ElementsMatch(t, []int64{1, 2, 3}, merged.WhitelistUserIDs)
	require.ElementsMatch(t, []int64{3, 4, 5, 6}, merged.ForcePushWhitelistUserIDs)
	require.ElementsMatch(t, []int64{5, 6}, merged.DeleterWhitelistUserIDs)
	require.ElementsMatch(t, []int64{7, 8}, merged.MergeWhitelistUserIDs)
}
