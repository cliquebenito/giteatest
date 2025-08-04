package protected_branch_db

import (
	"context"

	"code.gitea.io/gitea/models/git/protected_branch"
	repo_model "code.gitea.io/gitea/models/repo"

	"xorm.io/xorm"
)

type dbEngine interface {
	Where(interface{}, ...interface{}) *xorm.Session
	Delete(beans ...interface{}) (int64, error)
	Insert(beans ...interface{}) (int64, error)
	ID(interface{}) *xorm.Session
	Get(beans ...interface{}) (bool, error)
}

// //go:generate mockery --name=protectedBranchDB --exported
type protectedBranchDB interface {
	DeleteProtectedBranch(ctx context.Context, repoID, id int64) error
	RemoveUserIDFromProtectedBranch(ctx context.Context, protectBranch *protected_branch.ProtectedBranch, userID int64) error
	GetProtectedBranchRuleByName(ctx context.Context, repoID int64, ruleName string) (*protected_branch.ProtectedBranch, error)
	GetProtectedBranchRuleByID(ctx context.Context, repoID, ruleID int64) (*protected_branch.ProtectedBranch, error)
	FindRepoProtectedBranchRules(ctx context.Context, repoID int64) (protected_branch.ProtectedBranchRules, error)
	UpsertProtectBranch(ctx context.Context, repo *repo_model.Repository, protectBranch *protected_branch.ProtectedBranch, opts protected_branch.WhitelistOptions) error
	UpdateProtectBranch(ctx context.Context, repo *repo_model.Repository, protectedBranch *protected_branch.ProtectedBranch) (*protected_branch.ProtectedBranch, error)
	CreateProtectedBranch(ctx context.Context, protectedBranch *protected_branch.ProtectedBranch) (*protected_branch.ProtectedBranch, error)
}

type ProtectedBranchDB struct {
	engine dbEngine
}

func NewProtectedBranchDB(engine dbEngine) ProtectedBranchDB {
	return ProtectedBranchDB{engine: engine}
}
