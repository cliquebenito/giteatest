package tenant

import (
	"github.com/google/uuid"

	"code.gitea.io/gitea/models/organization"
	tenant_model "code.gitea.io/gitea/models/tenant"
	"code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/context"
)

// TenantOrganizations извлекаем все organizations у tenant
func TenantOrganizations(ctx *context.Context, tenantID string) ([]*tenant_model.ScTenantOrganizations, error) {
	tenantProjects, err := tenant_model.GetTenantOrganizations(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	return tenantProjects, nil
}

// CreateRelationTenantOrganization добавление organization к tenant
func CreateRelationTenantOrganization(ctx *context.Context, tenantProject *tenant_model.ScTenantOrganizations) error {
	return tenant_model.InsertTenantOrganization(ctx, tenantProject)
}

// RemoveTenantOrganization удаление organization у tenant
func RemoveTenantOrganization(ctx *context.Context, tenantID string, organizationID int64) error {
	err := tenant_model.DeleteTenantOrganization(ctx, tenantID, []int64{organizationID})
	if err != nil {
		return err
	}

	return nil
}

// CreateOrg создание проекта и tenantOrganization
func CreateOrg(ctx *context.Context, org *organization.Organization, orgKey, projectKey string) (*organization.Organization, error) {
	tenant, has, err := tenant_model.GetTenantByOrgKey(ctx, orgKey)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, tenant_model.ErrTenantByKeysNotExists{OrgKey: orgKey, ProjectKey: projectKey}
	}

	// todo
	userAdmin, err := user.GetAdminUser()
	if err != nil {
		return nil, err
	}
	err = organization.CreateOrganization(org, userAdmin)
	if err != nil {
		return nil, err
	}
	tenantOrganization := &tenant_model.ScTenantOrganizations{
		ID:             uuid.NewString(),
		TenantID:       tenant.ID,
		OrganizationID: org.ID,
		OrgKey:         orgKey,
		ProjectKey:     projectKey,
	}
	err = tenant_model.InsertTenantOrganization(ctx, tenantOrganization)
	if err != nil {
		return nil, err
	}

	return org, nil
}

// RemoveOrg удаляет проект и tenantOrganization
func RemoveOrg(ctx *context.Context, tenantOrganization *tenant_model.ScTenantOrganizations) error {
	org := &organization.Organization{
		ID:   tenantOrganization.OrganizationID,
		Type: user.UserTypeOrganization,
	}

	err := organization.DeleteOrganization(ctx, org)
	if err != nil {
		return err
	}

	err = tenant_model.DeleteTenantOrg(ctx, tenantOrganization)
	return err
}
