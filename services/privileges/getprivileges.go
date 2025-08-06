package privileges

import (
	"context"

	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/routers/api/v2/cache"
	"code.gitea.io/gitea/services/forms"
)

// GetPrivilegesRequest метод usecase для получения привилегий
func (p *privilegeService) GetPrivilegesRequest(ctx context.Context, privileges forms.GetPrivilegesRequest) (forms.ResponsePrivilegesGet, error) {
	var response = forms.ResponsePrivilegesGet{}
	cache := cache.NewRequestCache(p.engine)

	for _, request := range privileges {
		user, err := cache.GetUser(ctx, request.UserKey)
		if err != nil {
			log.Error("Error has occurred while getting user. Error: %v", err)
			return response, err
		}
		tenant, err := cache.GetTenantByKeys(ctx, request.TenantKey, request.ProjectKey)
		if err != nil {
			log.Error("Error has occurred while getting tenant. Error: %v", err)
			return response, err
		}
		role, err := cache.GetPrivileges(ctx, tenant.OrganizationID)
		if err != nil {
			log.Error("Error has occurred while getting privileges. Error: %v", err)
			return response, err
		}
		privilegeGroups := make([]string, 0, 8)
		for _, v := range role {
			if v.User.ID == user.ID && v.Org.ID == tenant.OrganizationID {
				privilegeGroups = append(privilegeGroups, v.Role.String())
			}
		}
		addGrantToResponse(&response, request.UserKey, request.TenantKey, request.ProjectKey, privilegeGroups)
	}

	return response, nil
}
func addGrantToResponse(response *forms.ResponsePrivilegesGet, userKey, tenantKey, projectKey string, privilegeGroups []string) {
	if len(privilegeGroups) > 0 {
		response.Grant = append(response.Grant, forms.PrivilegeGroupAssignmentGrant{
			UserExternalID: userKey,
			PrivilegeGroups: []forms.PrivilegeGroupGrant{
				{
					TenantKey:      tenantKey,
					ProjectKey:     projectKey,
					PrivilegeGroup: privilegeGroups,
				},
			},
		})
	}
}
