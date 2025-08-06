package protected_branch_db

import (
	"context"

	git_model "code.gitea.io/gitea/models/git"
	"code.gitea.io/gitea/models/git/protected_branch"
	repo_model "code.gitea.io/gitea/models/repo"
)

// UpdateProtectBranch saves branch protection options of repository.
// If ID is 0, it creates a new record. Otherwise, updates existing record.
// This function also performs check if whitelist user and team's IDs have been changed
// to avoid unnecessary whitelist delete and regenerate.
func (p ProtectedBranchDB) UpsertProtectBranch(ctx context.Context, repo *repo_model.Repository, protectBranch *protected_branch.ProtectedBranch, opts protected_branch.WhitelistOptions) error {
	return git_model.UpdateProtectBranch(ctx, repo, protectBranch, opts)
}

// RemoveUserIDFromProtectedBranch remove all user ids from protected branch options
func (p ProtectedBranchDB) RemoveUserIDFromProtectedBranch(ctx context.Context, protectBranch *protected_branch.ProtectedBranch, userID int64) error {
	return git_model.RemoveUserIDFromProtectedBranch(ctx, protectBranch, userID)
}

// UpdateProtectBranch update all columns protected branch
func (p ProtectedBranchDB) UpdateProtectBranch(ctx context.Context, repo *repo_model.Repository, protectedBranch *protected_branch.ProtectedBranch) (*protected_branch.ProtectedBranch, error) {
	_, err := p.engine.ID(protectedBranch.ID).AllCols().Update(protectedBranch)
	if err != nil {
		return nil, err
	}
	return protectedBranch, nil
}
