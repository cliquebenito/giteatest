package privileges

import (
	"context"
	"fmt"

	"github.com/casbin/casbin/v2"

	"code.gitea.io/gitea/models/db"
	org_model "code.gitea.io/gitea/models/organization"
	"code.gitea.io/gitea/models/role_model"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/sbt/audit"
	audit2 "code.gitea.io/gitea/modules/sbt/audit/utils"
	v2 "code.gitea.io/gitea/routers/api/v2/cache"
	"code.gitea.io/gitea/services/casbingormadapter"
	"code.gitea.io/gitea/services/forms"
)

type PrivilegesProcessor interface {
	RevokePrivileges(ctx context.Context, request forms.ApplyPrivilegeRequest, cache v2.RequestCache, auditInfo audit2.AuditRequiredParams) (forms.ApplyPrivilegesResponse, error)
	GrantPrivileges(ctx context.Context, request forms.ApplyPrivilegeRequest, cache v2.RequestCache, auditInfo audit2.AuditRequiredParams) (forms.ApplyPrivilegesResponse, error)
	GrantAndRevokeTx(ctx context.Context, request forms.ApplyPrivilegeGroups, cache v2.RequestCache, auditInfo audit2.AuditRequiredParams) (forms.ApplyPrivilegesResponse, error)
	ApplyPrivilegesRequest(ctx context.Context, request forms.ApplyPrivilegeRequest, auditInfo audit2.AuditRequiredParams) (forms.ApplyPrivilegesResponse, error)
	GetPrivilegesRequest(ctx context.Context, privileges forms.GetPrivilegesRequest) (forms.ResponsePrivilegesGet, error)
}

type privilegeService struct {
	engine   db.Engine
	enforcer casbin.IEnforcer
	adapter  *casbingormadapter.Adapter
}

func NewPrivilege(engine db.Engine, enforcer casbin.IEnforcer) (*privilegeService, error) {
	rawAdapter := enforcer.GetAdapter()
	adapter, ok := rawAdapter.(*casbingormadapter.Adapter)
	if !ok {
		log.Error("Error has occured while getting casbin adapter: can not cast %T to gorm adapter", adapter)
		return nil, fmt.Errorf("enforcer get adapter: can not cast %T to gorm adapter", adapter)
	}
	return &privilegeService{
		adapter:  adapter,
		engine:   engine,
		enforcer: enforcer,
	}, nil
}

// ApplyPrivilegesRequest метод usecase для работы с привилегиями
func (p *privilegeService) ApplyPrivilegesRequest(ctx context.Context, request forms.ApplyPrivilegeRequest, auditInfo audit2.AuditRequiredParams) (forms.ApplyPrivilegesResponse, error) {
	var (
		response forms.ApplyPrivilegesResponse
		err      error
	)
	cache := v2.NewRequestCache(p.engine)

	switch {
	case request.HasOnlyRevokeAndGrant():
		response, err = p.GrantAndRevokeTx(ctx, request.Action, cache, auditInfo)
		if err != nil {
			log.Error("Error has occurred while granting and revoking privileges. Error: %v", err)
			return response, fmt.Errorf("grant and revoke: %w", err)
		}
	case request.HasOnlyRevoke():
		response, err = p.RevokePrivileges(ctx, request, cache, auditInfo)
		if err != nil {
			log.Error("Error has occurred while revoking privileges. Error: %v", err)
			return response, fmt.Errorf("revoke: %w", err)
		}
	case request.HasOnlyGrant():
		response, err = p.GrantPrivileges(ctx, request, cache, auditInfo)
		if err != nil {
			log.Error("Error has occurred while granting privileges. Error: %v", err)
			return response, fmt.Errorf("grant: %w", err)
		}
	}
	log.Debug("Apply Privileges success")
	return response, nil
}

// GrantAndRevokeTx метод usecase для назначения и удаления привилегий в транзакции
func (p *privilegeService) GrantAndRevokeTx(ctx context.Context, request forms.ApplyPrivilegeGroups, cache v2.RequestCache, auditInfo audit2.AuditRequiredParams) (forms.ApplyPrivilegesResponse, error) {
	var finalResponse forms.ApplyPrivilegesResponse

	revokeRequest := forms.ApplyPrivilegeRequest{
		Action: forms.ApplyPrivilegeGroups{
			Revoke: request.Revoke,
		},
	}
	revokeResponse, err := p.RevokePrivileges(ctx, revokeRequest, cache, auditInfo)
	if err != nil {
		log.Error("Error has occurred while revoking privileges. Error: %v", err)
		return mergeResponseByUser(nil), err
	}
	finalResponse.Error.Revoke = revokeResponse.Error.Revoke
	finalResponse.AppliedStatus.Revoke = revokeResponse.AppliedStatus.Revoke

	grantRequest := forms.ApplyPrivilegeRequest{
		Action: forms.ApplyPrivilegeGroups{
			Grant: request.Grant,
		},
	}
	grantResponse, err := p.GrantPrivileges(ctx, grantRequest, cache, auditInfo)
	if err != nil {
		log.Error("Error has occurred while granting privileges. Error: %v", err)
		return mergeResponseByUser(nil), err
	}
	finalResponse.Error.Grant = grantResponse.Error.Grant
	finalResponse.AppliedStatus.Grant = grantResponse.AppliedStatus.Grant

	return mergeResponseByUser(&finalResponse), nil
}

// GrantPrivileges метод назначения привилегий
func (p *privilegeService) GrantPrivileges(ctx context.Context, request forms.ApplyPrivilegeRequest, cache v2.RequestCache, auditInfo audit2.AuditRequiredParams) (forms.ApplyPrivilegesResponse, error) {
	log.Debug("Grant privileges started")
	var response forms.ApplyPrivilegesResponse

	for _, grant := range request.Action.Grant {
		var errorGroups []forms.PrivilegeGroupErr
		for _, group := range grant.PrivilegeGroups {
			auditParams := map[string]string{
				"role":        group.PrivilegesGroup,
				"tenant_key":  group.TenantKey,
				"project_key": group.ProjectKey,
				"user_key":    grant.UserExternalID,
			}
			role, ok := role_model.GetRoleByString(group.PrivilegesGroup)
			if !ok {
				auditParams["error"] = "Error has occurred while searching for a role"
				audit.CreateAndSendEvent(audit.PrivilegesGrantEvent, auditInfo.DoerName, auditInfo.DoerID, audit.StatusFailure, auditInfo.RemoteAddress, auditParams)
				errorGroups = appendPrivilegeGroupError(errorGroups, group.TenantKey, group.ProjectKey, group.PrivilegesGroup, ErrWrongPrivelegeGroup{Name: role.String()}.Error())
				continue
			}
			tenant, err := cache.GetTenant(ctx, group.TenantKey, group.ProjectKey)
			if err != nil {
				auditParams["error"] = "Error has occurred while getting tenant"
				audit.CreateAndSendEvent(audit.PrivilegesGrantEvent, auditInfo.DoerName, auditInfo.DoerID, audit.StatusFailure, auditInfo.RemoteAddress, auditParams)
				errorGroups = appendPrivilegeGroupError(errorGroups, group.TenantKey, group.ProjectKey, group.PrivilegesGroup, err.Error())
				continue
			}
			user, err := cache.GetUser(ctx, grant.UserExternalID)
			if err != nil {
				auditParams["error"] = "Error has occurred while getting user"
				audit.CreateAndSendEvent(audit.PrivilegesGrantEvent, auditInfo.DoerName, auditInfo.DoerID, audit.StatusFailure, auditInfo.RemoteAddress, auditParams)
				errorGroups = appendPrivilegeGroupError(errorGroups, group.TenantKey, group.ProjectKey, group.PrivilegesGroup, err.Error())
				continue
			}
			err = role_model.GrantUserPermissionToOrganizationTx(
				p.enforcer,
				&user_model.User{ID: user.ID},
				tenant.TenantID,
				&org_model.Organization{ID: tenant.OrganizationID},
				role,
			)
			if err != nil {
				auditParams["error"] = "Error has occurred while granting privileges"
				audit.CreateAndSendEvent(audit.PrivilegesGrantEvent, auditInfo.DoerName, auditInfo.DoerID, audit.StatusFailure, auditInfo.RemoteAddress, auditParams)
				errorGroups = appendPrivilegeGroupError(errorGroups, group.TenantKey, group.ProjectKey, group.PrivilegesGroup, err.Error())
				continue
			}
			audit.CreateAndSendEvent(audit.PrivilegesGrantEvent, auditInfo.DoerName, auditInfo.DoerID, audit.StatusSuccess, auditInfo.RemoteAddress, auditParams)
			response.AppliedStatus.Grant = append(response.AppliedStatus.Grant, grant)
		}

		if len(errorGroups) > 0 {
			response.Error.Grant = append(response.Error.Grant, forms.PrivilegeGroupAssignmentErr{
				UserExternalID:  grant.UserExternalID,
				PrivilegeGroups: errorGroups,
			})
		}
	}

	return mergeResponseByUser(&response), nil
}

// RevokePrivileges метод удаления привилегий
func (p *privilegeService) RevokePrivileges(ctx context.Context, request forms.ApplyPrivilegeRequest, cache v2.RequestCache, auditInfo audit2.AuditRequiredParams) (forms.ApplyPrivilegesResponse, error) {
	log.Debug("Revoke privileges started")
	var response forms.ApplyPrivilegesResponse

	for _, revoke := range request.Action.Revoke {
		var errorGroups []forms.PrivilegeGroupErr
		for _, group := range revoke.PrivilegeGroups {
			auditParams := map[string]string{
				"role":        group.PrivilegesGroup,
				"tenant_key":  group.TenantKey,
				"project_key": group.ProjectKey,
				"user_key":    revoke.UserExternalID,
			}
			role, ok := role_model.GetRoleByString(group.PrivilegesGroup)
			if !ok {
				auditParams["error"] = "Error has occurred while searching for a role"
				audit.CreateAndSendEvent(audit.PrivilegesRevokeEvent, auditInfo.DoerName, auditInfo.DoerID, audit.StatusFailure, auditInfo.RemoteAddress, auditParams)
				errorGroups = appendPrivilegeGroupError(errorGroups, group.TenantKey, group.ProjectKey, group.PrivilegesGroup, ErrWrongPrivelegeGroup{Name: role.String()}.Error())
				continue
			}
			tenant, err := cache.GetTenant(ctx, group.TenantKey, group.ProjectKey)
			if err != nil {
				auditParams["error"] = "Error has occurred while getting tenant"
				audit.CreateAndSendEvent(audit.PrivilegesRevokeEvent, auditInfo.DoerName, auditInfo.DoerID, audit.StatusFailure, auditInfo.RemoteAddress, auditParams)
				errorGroups = appendPrivilegeGroupError(errorGroups, group.TenantKey, group.ProjectKey, group.PrivilegesGroup, err.Error())
				continue
			}
			user, err := cache.GetUser(ctx, revoke.UserExternalID)
			if err != nil {
				auditParams["error"] = "Error has occurred while getting user"
				audit.CreateAndSendEvent(audit.PrivilegesRevokeEvent, auditInfo.DoerName, auditInfo.DoerID, audit.StatusFailure, auditInfo.RemoteAddress, auditParams)
				errorGroups = appendPrivilegeGroupError(errorGroups, group.TenantKey, group.ProjectKey, group.PrivilegesGroup, err.Error())
				continue
			}
			err = role_model.RevokeUserPermissionToOrganizationTx(
				p.enforcer,
				&user_model.User{ID: user.ID},
				tenant.TenantID,
				&org_model.Organization{ID: tenant.OrganizationID},
				role,
			)
			if err != nil {
				auditParams["error"] = "Error has occurred while revoking privileges"
				audit.CreateAndSendEvent(audit.PrivilegesRevokeEvent, auditInfo.DoerName, auditInfo.DoerID, audit.StatusFailure, auditInfo.RemoteAddress, auditParams)
				errorGroups = appendPrivilegeGroupError(errorGroups, group.TenantKey, group.ProjectKey, group.PrivilegesGroup, err.Error())
				continue
			}
			audit.CreateAndSendEvent(audit.PrivilegesRevokeEvent, auditInfo.DoerName, auditInfo.DoerID, audit.StatusSuccess, auditInfo.RemoteAddress, auditParams)
			response.AppliedStatus.Revoke = append(response.AppliedStatus.Revoke, revoke)
		}
		if len(errorGroups) > 0 {
			response.Error.Revoke = append(response.Error.Revoke, forms.PrivilegeGroupAssignmentErr{
				UserExternalID:  revoke.UserExternalID,
				PrivilegeGroups: errorGroups,
			})
		}
	}
	return mergeResponseByUser(&response), nil
}

func appendPrivilegeGroupError(errorGroups []forms.PrivilegeGroupErr, tenantID string, projectKey string, privilegeGroup string, errMsg string) []forms.PrivilegeGroupErr {
	return append(errorGroups, forms.PrivilegeGroupErr{
		TenantID:       tenantID,
		ProjectKey:     projectKey,
		PrivilegeGroup: privilegeGroup,
		ErrMsg:         errMsg,
	})
}

// mergeResponseByUser метод объединения response по user_key
func mergeResponseByUser(response *forms.ApplyPrivilegesResponse) forms.ApplyPrivilegesResponse {
	if response == nil {
		log.Debug("incorrect response")
		return forms.ApplyPrivilegesResponse{}
	}
	// создаем мапы для grant и revoke
	aggregatedGrants := make(map[string]map[string]forms.PrivilegeGroup)
	aggregatedRevokes := make(map[string]map[string]forms.PrivilegeGroup)

	// проверяем наличие информации в мапе по user_key, если не найдено - добавляем
	for _, grant := range response.AppliedStatus.Grant {
		if _, exists := aggregatedGrants[grant.UserExternalID]; !exists {
			aggregatedGrants[grant.UserExternalID] = make(map[string]forms.PrivilegeGroup)
		}
		// формируем группу привилегий по конкретному user_key
		for _, group := range grant.PrivilegeGroups {
			groupKey := fmt.Sprintf("%s|%s|%s", group.TenantKey, group.ProjectKey, group.PrivilegesGroup)
			aggregatedGrants[grant.UserExternalID][groupKey] = group
		}
	}
	// проверяем наличие информации в мапе по user_key, если не найдено - добавляем
	for _, revoke := range response.AppliedStatus.Revoke {
		if _, exists := aggregatedRevokes[revoke.UserExternalID]; !exists {
			aggregatedRevokes[revoke.UserExternalID] = make(map[string]forms.PrivilegeGroup)
		}
		// формируем группу привилегий по конкретному user_key
		for _, group := range revoke.PrivilegeGroups {
			groupKey := fmt.Sprintf("%s|%s|%s", group.TenantKey, group.ProjectKey, group.PrivilegesGroup)
			aggregatedRevokes[revoke.UserExternalID][groupKey] = group
		}
	}
	// создаем массив с объединенными привилегиями
	var mergedGrants []forms.PrivilegeGroupAssignment
	for userID, groupMap := range aggregatedGrants {
		var privilegeGroups []forms.PrivilegeGroup
		for _, group := range groupMap {
			privilegeGroups = append(privilegeGroups, group)
		}
		// подготавливаем респонс для назначения привилегий
		mergedGrants = append(mergedGrants, forms.PrivilegeGroupAssignment{
			UserExternalID:  userID,
			PrivilegeGroups: privilegeGroups,
		})
	}

	var mergedRevokes []forms.PrivilegeGroupAssignment
	for userID, groupMap := range aggregatedRevokes {
		var privilegeGroups []forms.PrivilegeGroup
		for _, group := range groupMap {
			privilegeGroups = append(privilegeGroups, group)
		}
		// подготавливаем респонс для отбора привилегий
		mergedRevokes = append(mergedRevokes, forms.PrivilegeGroupAssignment{
			UserExternalID:  userID,
			PrivilegeGroups: privilegeGroups,
		})
	}
	// кладем все в единую структуру респонса
	response.AppliedStatus.Grant = mergedGrants
	response.AppliedStatus.Revoke = mergedRevokes

	return *response
}
