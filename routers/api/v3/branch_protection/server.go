package protected_branch

import (
	"context"

	"code.gitea.io/gitea/models/git/protected_branch"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/routers/api/v3/models"
)

type Server struct {
	protectedBranchManager    protectedBranchManager
	branchProtectionConverter branchProtectionConverter
	auditConverter            auditConverter
}

type auditConverter interface {
	Convert(protectBranch protected_branch.ProtectedBranch) protected_branch.AuditProtectedBranch
}

type branchProtectionConverter interface {
	ToBranchProtectionBody(rule protected_branch.ProtectedBranch) models.BranchProtectionBody
	ToBranchProtectionRulesBody(rules protected_branch.ProtectedBranchRules) []models.BranchProtectionBody
	ToProtectedBranch(ctx context.Context, protectedBranchRequest models.BranchProtectionBody) *protected_branch.ProtectedBranch
}

type protectedBranchManager interface {
	FindRepoProtectedBranchRules(ctx context.Context, repoID int64) (protected_branch.ProtectedBranchRules, error)
	GetProtectedBranchRuleByName(ctx context.Context, repoID int64, ruleName string) (*protected_branch.ProtectedBranch, error)
	CreateProtectedBranch(ctx context.Context, repo *repo_model.Repository, protectedBranch *protected_branch.ProtectedBranch) (*protected_branch.ProtectedBranch, error)
	UpdateProtectedBranch(ctx context.Context, repo *repo_model.Repository, protectedBranch *protected_branch.ProtectedBranch, ruleName string) (*protected_branch.ProtectedBranch, error)
	DeleteProtectedBranchByRuleName(ctx context.Context, repo *repo_model.Repository, ruleName string) error
}

func NewBranchProtectionServer(protectedBranchManager protectedBranchManager, branchProtectionConverter branchProtectionConverter, auditConverter auditConverter) Server {
	return Server{
		protectedBranchManager:    protectedBranchManager,
		branchProtectionConverter: branchProtectionConverter,
		auditConverter:            auditConverter,
	}
}
