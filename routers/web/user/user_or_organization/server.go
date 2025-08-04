package user_or_organization

import (
	gocontext "context"

	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/setting"
	context_service "code.gitea.io/gitea/services/context"

	"code.gitea.io/gitea/routers/web/user/accesser"
)

type orgRequestAccesser interface {
	IsReadAccessGranted(ctx gocontext.Context, request accesser.OrgAccessRequest) (bool, error)
	IsAccessGranted(ctx gocontext.Context, request accesser.OrgAccessRequest) (bool, error)
}

type userRequestAccesser interface {
	IsReadAccessGranted(ctx gocontext.Context, request accesser.UserAccessRequest) (bool, error)
}

type repoRequestAccessor interface {
	AccessesByCustomPrivileges(ctx gocontext.Context, request accesser.RepoAccessRequest) (bool, error)
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

type Server struct {
	isSourceControlTenantsAndRoleModelEnabled bool

	orgRequestAccesser
	userRequestAccesser
	repoRequestAccessor
}

func NewServer(
	orgAccesser orgRequestAccesser,
	userAccesser userRequestAccesser,
	isSourceControlTenantsEnabled bool,
	repoRequestAccessor repoRequestAccessor,
) Server {
	return Server{
		orgRequestAccesser: orgAccesser, userRequestAccesser: userAccesser,
		isSourceControlTenantsAndRoleModelEnabled: isSourceControlTenantsEnabled,
		repoRequestAccessor:                       repoRequestAccessor,
	}
}

func (s Server) handleUserOrOrganizationRequest(ctx *context.Context) {
	context_service.UserAssignmentWeb()(ctx)

	openProfilePage := func() {
		if !ctx.Written() {
			ctx.Data["EnableFeed"] = setting.Other.EnableFeed
			s.Profile(ctx)
		}
	}

	isGranted, err := s.isAccessGranted(ctx)
	if err != nil {
		s.handleServerErrors(ctx, err)
		return
	}

	if !isGranted {
		ctx.NotFound("Access denied", nil)
		return
	}

	openProfilePage()
}
