// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package git

import (
	"context"
	"sort"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/git/protected_branch"
	"code.gitea.io/gitea/modules/git"

	"github.com/gobwas/glob"
)

// Get first matched protected branch rule from rules with branch name
// Deprecated: use ProtectedBranchManager.GetFirstMatched
func GetFirstMatched(rules protected_branch.ProtectedBranchRules, branchName string) *protected_branch.ProtectedBranch {
	for _, rule := range rules {
		if rule.Match(branchName) {
			return rule
		}
	}
	return nil
}

// Merge all protected branch rules, if in rules have plane name rule return plane name
// rule with combain white lists and flags
// Deprecated: use ProtectedBranchManager.MergeProtectedBranchRules
func MergeProtectedBranchRules(rules protected_branch.ProtectedBranchRules) *protected_branch.ProtectedBranch {
	if len(rules) == 0 {
		return nil
	}
	protectedBranch := &protected_branch.ProtectedBranch{}
	for _, rule := range rules {
		protectedBranch = MergeProtectedBranch(protectedBranch, rule)
	}
	return protectedBranch
}

// Find all matched branches from rules
// Deprecated: use ProtectedBranchManager.GetMatchProtectedBranchRules
func GetMatchProtectedBranchRules(rules protected_branch.ProtectedBranchRules, branchName string) protected_branch.ProtectedBranchRules {
	protectedBranchRules := make(protected_branch.ProtectedBranchRules, 0)
	for _, rule := range rules {
		if rule.Match(branchName) {
			protectedBranchRules = append(protectedBranchRules, rule)
		}
	}
	return protectedBranchRules
}

// FindRepoProtectedBranchRules load all repository's protected rules
// Deprecated: use ProtectedBranchManager.FindRepoProtectedBranchRules
func FindRepoProtectedBranchRules(ctx context.Context, repoID int64) (protected_branch.ProtectedBranchRules, error) {
	var rules protected_branch.ProtectedBranchRules
	err := db.GetEngine(ctx).Where("repo_id = ?", repoID).Asc("created_unix").Find(&rules)
	if err != nil {
		return nil, err
	}
	rules = sortRules(rules) // to make non-glob rules have higher priority, and for same glob/non-glob rules, first created rules have higher priority
	return rules, nil
}

// FindAllMatchedBranches find all matched branches
// Deprecated: use ProtectedBranchManager.FindAllMatchedBranches
func FindAllMatchedBranches(_ context.Context, gitRepo *git.Repository, ruleName string) ([]string, error) {
	// FIXME: how many should we get?
	branches, _, err := gitRepo.GetBranchNames(0, 9999999)
	if err != nil {
		return nil, err
	}
	rule := glob.MustCompile(ruleName)
	results := make([]string, 0, len(branches))
	for _, branch := range branches {
		if rule.Match(branch) {
			results = append(results, branch)
		}
	}
	return results, nil
}

// return merged match protected branch from repo with id repoId to branch with name branchName
// Deprecated: use ProtectedBranchManager.GetMergeMatchProtectedBranchRule
func GetMergeMatchProtectedBranchRule(ctx context.Context, repoID int64, branchName string) (*protected_branch.ProtectedBranch, error) {
	rules, err := FindRepoProtectedBranchRules(ctx, repoID)
	if err != nil {
		return nil, err
	}
	rules = GetMatchProtectedBranchRules(rules, branchName)
	return MergeProtectedBranchRules(rules), nil
}

// IsBranchProtected checks if branch is protected
// Deprecated: use ProtectedBranchManager.IsBranchProtected
func IsBranchProtected(ctx context.Context, repoID int64, branchName string) (bool, error) {
	rule, err := GetMergeMatchProtectedBranchRule(ctx, repoID, branchName)
	if err != nil {
		return false, err
	}
	return rule != nil, nil
}

// Sorted rules first will be rule with IsPlaneName next older
func sortRules(rules protected_branch.ProtectedBranchRules) protected_branch.ProtectedBranchRules {
	sort.Slice(rules, func(i, j int) bool {
		rules[i].GlobRule, rules[i].IsPlainName = LoadGlob(*rules[i])
		rules[j].GlobRule, rules[j].IsPlainName = LoadGlob(*rules[j])
		if rules[i].IsPlainName != rules[j].IsPlainName {
			return rules[i].IsPlainName // plain name comes first, so plain name means "less"
		}
		return rules[i].CreatedUnix < rules[j].CreatedUnix
	})
	return rules
}
