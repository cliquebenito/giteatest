package protected_branch_db

import (
	"context"

	git_model "code.gitea.io/gitea/models/git"
	"code.gitea.io/gitea/models/git/protected_branch"
)

// GetProtectedBranchRuleByName getting protected branch rule by name
func (p ProtectedBranchDB) GetProtectedBranchRuleByName(ctx context.Context, repoID int64, ruleName string) (*protected_branch.ProtectedBranch, error) {
	return git_model.GetProtectedBranchRuleByName(ctx, repoID, ruleName)
}

// GetProtectedBranchRuleByID getting protected branch rule by rule ID
func (p ProtectedBranchDB) GetProtectedBranchRuleByID(ctx context.Context, repoID, ruleID int64) (*protected_branch.ProtectedBranch, error) {
	return git_model.GetProtectedBranchRuleByID(ctx, repoID, ruleID)
}

// FindRepoProtectedBranchRules load all repository's protected rules
func (p ProtectedBranchDB) FindRepoProtectedBranchRules(ctx context.Context, repoID int64) (protected_branch.ProtectedBranchRules, error) {
	return git_model.FindRepoProtectedBranchRules(ctx, repoID)
}
