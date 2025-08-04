package iamprivileger

import (
	"context"
	"fmt"

	"code.gitea.io/gitea/models/organization"
	"code.gitea.io/gitea/models/tenant"
	iampriveleges "code.gitea.io/gitea/modules/auth/iam/iamprivileges"
	"code.gitea.io/gitea/modules/log"
)

func (p Privileger) getTenantsByNames(ctx context.Context, tenantNames []string) ([]*tenant.ScTenant, error) {
	tenants, err := tenant.GetTenants(ctx)
	if err != nil {
		return nil, fmt.Errorf("get tenants: %w", err)
	}

	var targetTenants []*tenant.ScTenant

	for _, tenantObj := range tenants {
		log.Debug(fmt.Sprintf("tenantObj: %+v\n", tenantObj))
		for _, tenantName := range tenantNames {
			if tenantName == tenantObj.Name {
				targetTenants = append(targetTenants, tenantObj)
			}
		}
	}
	return targetTenants, nil
}

// getOrganizationsByName метод отдает организации по тенанту
func (p Privileger) getOrganizationsByName(
	ctx context.Context,
	scTenants []*tenant.ScTenant,
	privileges iampriveleges.SourceControlPrivilegesByTenant,
) (map[string][]*organization.Organization, error) {
	organizationsByTenantID := make(map[string][]*organization.Organization)

	// сперва находим все активные организации
	allOrg, err := organization.GetAllActiveOrganization(ctx)
	if err != nil {
		return nil, fmt.Errorf("get all organizations: %w", err)
	}

	log.Debug("GetAllActiveOrganization: %+v\n", allOrg)

	for _, scTenant := range scTenants {
		// поиск идет по org_key, ибо он не меняется в отличии от tenant_name
		log.Debug("getOrganizationsByName scTenant.OrgKey: %+v\n", scTenant.OrgKey)

		projectNames, err := privileges.UniqProjectNamesByTenantName(scTenant.OrgKey)
		if err != nil {
			return nil, fmt.Errorf("get privileges for tenant %s: %w", scTenant.Name, err)
		}

		log.Debug("getOrganizationsByName projectNames %+v\n", projectNames)

		for _, projectName := range projectNames {
			for _, orgObj := range allOrg {
				if orgObj.Name == projectName {
					organizationsByTenantID[scTenant.ID] = append(organizationsByTenantID[scTenant.ID], orgObj)
				}
			}
		}
	}

	log.Debug("getOrganizationsByName organizationsByTenantID %+v\n", organizationsByTenantID)

	return organizationsByTenantID, nil
}
