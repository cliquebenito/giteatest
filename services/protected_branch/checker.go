package protected_brancher

import (
	"context"

	git_model "code.gitea.io/gitea/models/git"
	"code.gitea.io/gitea/models/git/protected_branch"
	access_model "code.gitea.io/gitea/models/perm/access"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/base"

	"github.com/gobwas/glob"
)

type protectedBranchChecker interface {
	CheckUserCanPush(context.Context, protected_branch.ProtectedBranch, *user_model.User) bool
	CheckUserCanDeleteBranch(context.Context, protected_branch.ProtectedBranch, *user_model.User) bool
	CheckUserCanForcePush(context.Context, protected_branch.ProtectedBranch, *user_model.User) bool
	IsUserMergeWhitelisted(context.Context, protected_branch.ProtectedBranch, int64, access_model.Permission) bool
	IsUserOfficialReviewer(context.Context, protected_branch.ProtectedBranch, *user_model.User) bool
	IsProtectedFile(context.Context, protected_branch.ProtectedBranch, []glob.Glob, string) bool
	IsUnprotectedFile(context.Context, protected_branch.ProtectedBranch, []glob.Glob, string) bool
	MergeBlockedByProtectedFiles(context.Context, protected_branch.ProtectedBranch, []string) bool
}

type ProtectedBranchChecker struct{}

func NewProtectedBranchChecker() *ProtectedBranchChecker {
	return &ProtectedBranchChecker{}
}

// CheckUserCanPush returns if some user could push to this protected branch
func (p ProtectedBranchChecker) CheckUserCanPush(ctx context.Context, protectBranch protected_branch.ProtectedBranch, user *user_model.User) bool {
	return git_model.CanUserPush(ctx, protectBranch, user)
}

// CheckUserCanDeleteBranch returns if some user could delete protected branch
func (p ProtectedBranchChecker) CheckUserCanDeleteBranch(_ context.Context, protectBranch protected_branch.ProtectedBranch, user *user_model.User) bool {
	if protectBranch.EnableDeleterWhitelist {
		return base.Int64sContains(protectBranch.DeleterWhitelistUserIDs, user.ID)
	}

	return true
}

// CheckUserCanForcePush returns if some user could force push to this protected branch
func (p ProtectedBranchChecker) CheckUserCanForcePush(_ context.Context, protectBranch protected_branch.ProtectedBranch, user *user_model.User) bool {
	if protectBranch.EnableForcePushWhitelist {
		return base.Int64sContains(protectBranch.ForcePushWhitelistUserIDs, user.ID)
	}

	return true
}

// IsUserMergeWhitelisted checks if some user is whitelisted to merge to this branch
func (p ProtectedBranchChecker) IsUserMergeWhitelisted(ctx context.Context, protectBranch protected_branch.ProtectedBranch, userID int64, permissionInRepo access_model.Permission) bool {
	return git_model.IsUserMergeWhitelisted(ctx, protectBranch, userID, permissionInRepo)
}

// IsUserOfficialReviewer check if user is official reviewer for the branch (counts towards required approvals)
func (p ProtectedBranchChecker) IsUserOfficialReviewer(ctx context.Context, protectBranch protected_branch.ProtectedBranch, user *user_model.User) bool {
	return git_model.IsUserOfficialReviewer(ctx, protectBranch, user)
}

// IsProtectedFile return if path is protected
func (p ProtectedBranchChecker) IsProtectedFile(_ context.Context, protectBranch protected_branch.ProtectedBranch, patterns []glob.Glob, path string) bool {
	return git_model.IsProtectedFile(protectBranch, patterns, path)
}

// IsUnprotectedFile return if path is unprotected
func (p ProtectedBranchChecker) IsUnprotectedFile(_ context.Context, protectBranch protected_branch.ProtectedBranch, patterns []glob.Glob, path string) bool {
	return git_model.IsUnprotectedFile(protectBranch, patterns, path)
}

// MergeBlockedByProtectedFiles returns true if merge is blocked by protected files change
func (p ProtectedBranchChecker) MergeBlockedByProtectedFiles(_ context.Context, protectBranch protected_branch.ProtectedBranch, changedProtectedFiles []string) bool {
	return git_model.MergeBlockedByProtectedFiles(protectBranch, changedProtectedFiles)
}
