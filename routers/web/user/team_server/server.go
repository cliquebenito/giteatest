package team_server

import (
	gocontext "context"

	"code.gitea.io/gitea/routers/web/user/accesser"
	"code.gitea.io/gitea/services/custom_creator"
	"code.gitea.io/gitea/services/forms"
)

type orgRequestAccessor interface {
	IsReadAccessGranted(ctx gocontext.Context, request accesser.OrgAccessRequest) (bool, error)
	IsAccessGranted(ctx gocontext.Context, request accesser.OrgAccessRequest) (bool, error)
}

type userRequestAccessor interface {
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

type customCreator interface {
	AddUserToTeam(ctx gocontext.Context, teamID, orgID int64, tenantID string, userIDs []int64) error
	AddCustomPrivilegeToTeamUser(ctx gocontext.Context, ordID int64, teamName string, customPrivilegeForm []forms.CustomPrivileges) error
	UpdateCustomPrivilegeToTeamUser(ctx gocontext.Context, ordID int64, teamName string, customPrivilegeForm []forms.CustomPrivileges) error
	CreateOrDeleteCustomPrivileges(ctx gocontext.Context, customPrivileges custom_creator.ConfCustomPrivileges) error
	RemoveCustomPrivilegesByTeam(ctx gocontext.Context, teamName string) error
}

type Server struct {
	orgRequestAccessor
	userRequestAccessor
	repoRequestAccessor
	customCreator
}

func NewTeamServer(
	orgAccessor orgRequestAccessor,
	userAccessor userRequestAccessor,
	repoAccessor repoRequestAccessor,
	customCreator customCreator,
) *Server {
	return &Server{
		orgAccessor,
		userAccessor,
		repoAccessor,
		customCreator,
	}
}
