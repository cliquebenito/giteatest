package repo_server

import (
	"context"

	"code.gitea.io/gitea/models/git/protected_branch"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/routers/web/user/accesser"

	"github.com/gobwas/glob"
)

type orgRequestAccessor interface {
	IsReadAccessGranted(ctx context.Context, request accesser.OrgAccessRequest) (bool, error)
	IsAccessGranted(ctx context.Context, request accesser.OrgAccessRequest) (bool, error)
}

type userRequestAccessor interface {
	IsReadAccessGranted(ctx context.Context, request accesser.UserAccessRequest) (bool, error)
}

type repoRequestAccessor interface {
	AccessesByCustomPrivileges(ctx context.Context, request accesser.RepoAccessRequest) (bool, error)
	GrantCustomPrivilege(request accesser.RepoAccessRequest) error
	CheckCustomPrivilegesForUser(request accesser.RepoAccessRequest) bool
	RemoveCustomPrivilege(request accesser.RepoAccessRequest) error
	RemoveCustomPrivilegesForTeam(request accesser.RepoAccessRequest) error
	CheckGroupingPolicyByParams(request accesser.RepoCustomParamsRequest) bool
	RemoveCustomPrivilegesByTenantAndOrgID(request accesser.RepoAccessRequest) error
	RemoveCustomPrivilegesByOldPrivileges(removeCustomPrivileges [][]string) error
	AddCustomGroupingPrivileges(name string, groupingPolicies []string) error
	ExpandNewCustomPrivileges(newPolicies [][]string) error
	RemoveCustomPrivilegesByFieldIdxAndName(fieldIndex int, fieldName string) error
}

type protectedBranchManager interface {
	GetProtectedFilePatterns(ctx context.Context, protectBranch protected_branch.ProtectedBranch) []glob.Glob
	CheckUserCanPush(ctx context.Context, protectBranch protected_branch.ProtectedBranch, user *user_model.User) bool
	IsProtectedFile(ctx context.Context, protectBranch protected_branch.ProtectedBranch, patterns []glob.Glob, path string) bool
	GetMergeMatchProtectedBranchRule(ctx context.Context, repoID int64, branchName string) (*protected_branch.ProtectedBranch, error)
}

type Server struct {
	orgRequestAccessor
	userRequestAccessor
	repoRequestAccessor
	protectedBranchManager
}

func NewRepoServer(
	orgAccessor orgRequestAccessor,
	userAccessor userRequestAccessor,
	repoAccessor repoRequestAccessor,
	protectedBranchManager protectedBranchManager,
) *Server {
	return &Server{
		orgAccessor,
		userAccessor,
		repoAccessor,
		protectedBranchManager,
	}
}
