package protected_brancher

import (
	"context"

	git_models "code.gitea.io/gitea/models/git"
	"code.gitea.io/gitea/models/git/protected_branch"
	"code.gitea.io/gitea/modules/git"

	"github.com/gobwas/glob"
)

type protectedBranchGetter interface {
	GetGlob(context.Context, protected_branch.ProtectedBranch) (glob.Glob, bool)
	IsRuleNameSpecial(ruleName string) bool
	GetProtectedFilePatterns(context.Context, protected_branch.ProtectedBranch) []glob.Glob
	GetUnprotectedFilePatterns(context.Context, protected_branch.ProtectedBranch) []glob.Glob
	FindAllMatchedBranches(context.Context, *git.Repository, string) ([]string, error)
}

type ProtectedBranchGetter struct{}

func NewProtectedBranchGetter() *ProtectedBranchGetter {
	return &ProtectedBranchGetter{}
}

// get glob for protected branch
func (p ProtectedBranchGetter) GetGlob(_ context.Context, protectBranch protected_branch.ProtectedBranch) (glob.Glob, bool) {
	return git_models.LoadGlob(protectBranch)
}

// IsRuleNameSpecial return true if it contains special character
func (p ProtectedBranchGetter) IsRuleNameSpecial(ruleName string) bool {
	return git_models.IsRuleNameSpecial(ruleName)
}

// GetProtectedFilePatterns parses a semicolon separated list of protected file patterns and returns a glob.Glob slice
func (p ProtectedBranchGetter) GetProtectedFilePatterns(_ context.Context, protectBranch protected_branch.ProtectedBranch) []glob.Glob {
	return git_models.GetProtectedFilePatterns(protectBranch)
}

// GetUnprotectedFilePatterns parses a semicolon separated list of unprotected file patterns and returns a glob.Glob slice
func (p ProtectedBranchGetter) GetUnprotectedFilePatterns(_ context.Context, protectBranch protected_branch.ProtectedBranch) []glob.Glob {
	return git_models.GetUnprotectedFilePatterns(protectBranch)
}

// FindAllMatchedBranches find all matched branches
func (p ProtectedBranchGetter) FindAllMatchedBranches(ctx context.Context, gitRepo *git.Repository, ruleName string) ([]string, error) {
	return git_models.FindAllMatchedBranches(ctx, gitRepo, ruleName)
}
