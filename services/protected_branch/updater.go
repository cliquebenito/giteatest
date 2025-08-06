package protected_brancher

import (
	"context"

	git_model "code.gitea.io/gitea/models/git"
	"code.gitea.io/gitea/models/git/protected_branch"
	repo_model "code.gitea.io/gitea/models/repo"
)

// //go:generate mockery --name=protectedBranchUpdater --exported
type protectedBranchUpdater interface {
	UpdateWhitelistOptions(context.Context, *repo_model.Repository, *protected_branch.ProtectedBranch, protected_branch.WhitelistOptions) error
	UpdateModelProtectedBranch(*protected_branch.ProtectedBranch, *protected_branch.ProtectedBranch) *protected_branch.ProtectedBranch
}

type ProtectedBranchUpdater struct{}

func NewProtectedBranchUpdater() *ProtectedBranchUpdater {
	return &ProtectedBranchUpdater{}
}

func (pb *ProtectedBranchUpdater) UpdateWhitelistOptions(ctx context.Context, repo *repo_model.Repository, protectBranch *protected_branch.ProtectedBranch, opts protected_branch.WhitelistOptions) error {
	return git_model.UpdateWhitelistOptions(ctx, repo, protectBranch, opts)
}

func (pb *ProtectedBranchUpdater) UpdateModelProtectedBranch(existProtectedBranch *protected_branch.ProtectedBranch, newProtectedBranch *protected_branch.ProtectedBranch) *protected_branch.ProtectedBranch {
	existProtectedBranch.RuleName = newProtectedBranch.RuleName

	existProtectedBranch.EnableWhitelist = newProtectedBranch.EnableWhitelist
	existProtectedBranch.WhitelistDeployKeys = newProtectedBranch.WhitelistDeployKeys

	existProtectedBranch.EnableForcePushWhitelist = newProtectedBranch.EnableForcePushWhitelist
	existProtectedBranch.ForcePushWhitelistDeployKeys = newProtectedBranch.ForcePushWhitelistDeployKeys

	existProtectedBranch.EnableDeleterWhitelist = newProtectedBranch.EnableDeleterWhitelist
	existProtectedBranch.DeleterWhitelistDeployKeys = newProtectedBranch.DeleterWhitelistDeployKeys

	existProtectedBranch.RequireSignedCommits = newProtectedBranch.RequireSignedCommits
	existProtectedBranch.ProtectedFilePatterns = newProtectedBranch.ProtectedFilePatterns
	existProtectedBranch.UnprotectedFilePatterns = newProtectedBranch.UnprotectedFilePatterns

	return existProtectedBranch
}
