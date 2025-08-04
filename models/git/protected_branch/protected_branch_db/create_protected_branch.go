package protected_branch_db

import (
	"context"
	"fmt"

	"code.gitea.io/gitea/models/git/protected_branch"
	"code.gitea.io/gitea/modules/log"
)

func (p ProtectedBranchDB) CreateProtectedBranch(ctx context.Context, protectedBranch *protected_branch.ProtectedBranch) (*protected_branch.ProtectedBranch, error) {
	_, err := p.engine.Insert(protectedBranch)
	if err != nil {
		log.Error("Err: insert protected branch db: %v", err)
		return nil, fmt.Errorf("Err: insert protected branch db: %w", err)
	}
	return protectedBranch, nil
}
