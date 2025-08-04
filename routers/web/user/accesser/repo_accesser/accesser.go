package repo_accesser

import (
	"context"
	"fmt"
	"strconv"

	"code.gitea.io/gitea/models/organization"
	"code.gitea.io/gitea/models/role_model/custom_casbin_role_manager"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/trace"

	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/routers/web/user/accesser"
)

type casbinCustomPermissioner interface {
	CheckCustomPrivileges(ctx context.Context, privilege custom_casbin_role_manager.ConfCustomPrivilege) (bool, error)
	GrantCustomPrivilegeTeamUser(privilege custom_casbin_role_manager.GrantCustomPrivilege) error
	GetCustomPrivilegesForUser(orgID, userID, tenant, teamName string) bool
	RemoveUserFromTeamCustomPrivilege(privilege custom_casbin_role_manager.GrantCustomPrivilege) error
	RemoveCustomPrivileges(teamName string) error
	CheckGroupingPolicy(fieldIndex int, fieldName string) bool
	RemoveExistingPrivilegesByTenantAndOrgID(tenantID string, orgID int64) error
	RemoveCustomPrivilegesByArrays(removeCustomPrivileges [][]string) error
	CreateCustomGroupingPrivileges(name string, groupingPolicies []string) error
	UpdateCustomPrivileges(newPolicies [][]string) error
	RemoveCustomPrivilegesByParams(fieldIndex int, fieldName string) error
}

type requestAccessor struct {
	casbinCustomPermissioner
}

func NewRepoAccessor(casbinCustomPermissioner casbinCustomPermissioner) *requestAccessor {
	return &requestAccessor{casbinCustomPermissioner: casbinCustomPermissioner}
}

func (a requestAccessor) AccessesByCustomPrivileges(ctx context.Context, request accesser.RepoAccessRequest) (bool, error) {
	logTracer := trace.NewLogTracer()
	message := logTracer.CreateTraceMessage(ctx)
	err := logTracer.Trace(message)
	if err != nil {
		log.Error("Error has occurred while creating trace message: %v", err)
	}
	defer func() {
		err = logTracer.TraceTime(message)
		if err != nil {
			log.Error("Error has occurred while creating trace time message: %v", err)
		}
	}()

	customParams := custom_casbin_role_manager.ConfCustomPrivilege{
		Org:             &organization.Organization{ID: request.OrgID},
		User:            &user_model.User{ID: request.DoerID},
		TenantID:        request.TargetTenantID,
		RepoID:          request.RepoID,
		CustomPrivilege: request.CustomPrivilege,
	}
	allowed, err := a.CheckCustomPrivileges(ctx, customParams)
	if err != nil {
		log.Error("Error has occurred while checking permissions by custom privilege")
		return false, fmt.Errorf("check custom privileges: %w", err)
	}
	return allowed, nil
}

func (a requestAccessor) GrantCustomPrivilege(request accesser.RepoAccessRequest) error {
	customParams := custom_casbin_role_manager.GrantCustomPrivilege{
		Org:             &organization.Organization{ID: request.OrgID},
		User:            &user_model.User{ID: request.DoerID},
		TenantID:        request.TargetTenantID,
		RepoID:          request.RepoID,
		Team:            request.Team,
		CustomPrivilege: request.CustomPrivilege,
	}
	return a.GrantCustomPrivilegeTeamUser(customParams)
}

func (a requestAccessor) CheckCustomPrivilegesForUser(request accesser.RepoAccessRequest) bool {
	return a.GetCustomPrivilegesForUser(strconv.FormatInt(request.OrgID, 10), strconv.FormatInt(request.DoerID, 10), request.TargetTenantID, request.Team.Name)
}

func (a requestAccessor) RemoveCustomPrivilege(request accesser.RepoAccessRequest) error {
	customParams := custom_casbin_role_manager.GrantCustomPrivilege{
		Org:             &organization.Organization{ID: request.OrgID},
		User:            &user_model.User{ID: request.DoerID},
		TenantID:        request.TargetTenantID,
		RepoID:          request.RepoID,
		Team:            request.Team,
		CustomPrivilege: request.CustomPrivilege,
	}
	return a.RemoveUserFromTeamCustomPrivilege(customParams)
}

func (a requestAccessor) RemoveCustomPrivilegesForTeam(request accesser.RepoAccessRequest) error {
	return a.RemoveCustomPrivileges(request.Team.Name)
}

func (a requestAccessor) CheckGroupingPolicyByParams(request accesser.RepoCustomParamsRequest) bool {
	return a.CheckGroupingPolicy(request.FieldIdx, request.FieldName)
}

func (a requestAccessor) RemoveCustomPrivilegesByTenantAndOrgID(request accesser.RepoAccessRequest) error {
	return a.RemoveExistingPrivilegesByTenantAndOrgID(request.TargetTenantID, request.OrgID)
}

func (a requestAccessor) RemoveCustomPrivilegesByOldPrivileges(removeCustomPrivileges [][]string) error {
	return a.RemoveCustomPrivilegesByArrays(removeCustomPrivileges)
}

func (a requestAccessor) AddCustomGroupingPrivileges(name string, groupingPolicies []string) error {
	return a.CreateCustomGroupingPrivileges(name, groupingPolicies)
}

func (a requestAccessor) ExpandNewCustomPrivileges(newPolicies [][]string) error {
	return a.UpdateCustomPrivileges(newPolicies)
}

func (a requestAccessor) RemoveCustomPrivilegesByFieldIdxAndName(fieldIndex int, fieldName string) error {
	return a.RemoveCustomPrivilegesByParams(fieldIndex, fieldName)
}
