package protected_branch_db

import (
	"context"

	git_model "code.gitea.io/gitea/models/git"
)

// DeleteProtectedBranch removes ProtectedBranch relation between the user and repository.
func (p ProtectedBranchDB) DeleteProtectedBranch(ctx context.Context, repoID, id int64) error {
	return git_model.DeleteProtectedBranch(ctx, repoID, id)
}
