package protected_branch

import (
	"context"

	"code.gitea.io/gitea/models/git/protected_branch"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/modules/base"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/services/forms"
)

const (
	tplProtectedBranch base.TplName = "repo/settings/protected_branch"
	tplBranches        base.TplName = "repo/settings/branches"
)

type protectedBranchManager interface {
	FindRepoProtectedBranchRules(ctx context.Context, repoID int64) (protected_branch.ProtectedBranchRules, error)
	GetProtectedBranchRuleByName(ctx context.Context, repoID int64, ruleName string) (*protected_branch.ProtectedBranch, error)
	GetProtectedBranchRuleByID(ctx context.Context, repoID, ruleID int64) (*protected_branch.ProtectedBranch, error)
	UpsertProtectBranch(ctx context.Context, repo *repo_model.Repository, protectBranch *protected_branch.ProtectedBranch, opts protected_branch.WhitelistOptions) error
	FindAllMatchedBranches(ctx context.Context, gitRepo *git.Repository, ruleName string) ([]string, error)
	DeleteProtectedBranch(ctx context.Context, repoID, protectedBranchID int64) error
}

type formConverter interface {
	ConvertProtectBranchFormToProtectedBranchRule(form *forms.ProtectBranchForm, protectBranch *protected_branch.ProtectedBranch) (*protected_branch.ProtectedBranch, error)
}

type auditConverter interface {
	Convert(protectBranch protected_branch.ProtectedBranch) protected_branch.AuditProtectedBranch
}

type Server struct {
	protectedBranchManager protectedBranchManager
	auditConverter         auditConverter
	formConverter          formConverter
}

func NewServer(protectedBranchManager protectedBranchManager, auditConverter auditConverter, formConverter formConverter) Server {
	return Server{
		protectedBranchManager: protectedBranchManager,
		auditConverter:         auditConverter,
		formConverter:          formConverter,
	}
}
