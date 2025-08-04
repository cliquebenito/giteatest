package hooks

import (
	"context"

	"code.gitea.io/gitea/models/git/protected_branch"
	"code.gitea.io/gitea/models/pull/pullrequestidresolver"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/routers/private/unit_linker"
	"code.gitea.io/gitea/routers/web/user/accesser"

	"github.com/gobwas/glob"
)

type Server struct {
	unitLinker            unit_linker.UnitLinker
	pullRequestIDResolver pullrequestidresolver.PullRequestIDResolver

	taskTrackerEnabled bool

	repoRequestAccessor    repoRequestAccessor
	orgRequestAccessor     orgRequestAccessor
	protectedBranchManager protectedBranchManager
}

type orgRequestAccessor interface {
	IsReadAccessGranted(ctx context.Context, request accesser.OrgAccessRequest) (bool, error)
	IsAccessGranted(ctx context.Context, request accesser.OrgAccessRequest) (bool, error)
}

type repoRequestAccessor interface {
	AccessesByCustomPrivileges(ctx context.Context, request accesser.RepoAccessRequest) (bool, error)
	UpdateCustomPrivileges(newPolicies [][]string) error
	RemoveCustomPrivilegesByOldPrivileges(removeCustomPrivileges [][]string) error
}

type protectedBranchManager interface {
	GetProtectedFilePatterns(ctx context.Context, protectBranch protected_branch.ProtectedBranch) []glob.Glob
	GetUnprotectedFilePatterns(ctx context.Context, protectBranch protected_branch.ProtectedBranch) []glob.Glob
	CheckUserCanPush(ctx context.Context, protectBranch protected_branch.ProtectedBranch, user *user_model.User) bool
	CheckUserCanForcePush(ctx context.Context, protectBranch protected_branch.ProtectedBranch, user *user_model.User) bool
	CheckUserCanDeleteBranch(ctx context.Context, protectBranch protected_branch.ProtectedBranch, user *user_model.User) bool
	GetMergeMatchProtectedBranchRule(ctx context.Context, repoID int64, branchName string) (*protected_branch.ProtectedBranch, error)
}

func NewServer(
	unitLinker unit_linker.UnitLinker,
	pullRequestIDResolver pullrequestidresolver.PullRequestIDResolver,
	taskTrackerEnabled bool,
	repoRequestAccessor repoRequestAccessor,
	orgAccessor orgRequestAccessor,
	protectedBranchManager protectedBranchManager,
) Server {
	return Server{
		unitLinker:             unitLinker,
		pullRequestIDResolver:  pullRequestIDResolver,
		taskTrackerEnabled:     taskTrackerEnabled,
		repoRequestAccessor:    repoRequestAccessor,
		orgRequestAccessor:     orgAccessor,
		protectedBranchManager: protectedBranchManager,
	}
}
