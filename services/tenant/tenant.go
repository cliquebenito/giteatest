package tenant

import (
	"context"

	"code.gitea.io/gitea/models/role_model"
	tenat_model "code.gitea.io/gitea/models/tenant"
)

// GetTenants получение все tenants
func GetTenants(ctx context.Context) ([]*tenat_model.ScTenant, error) {
	tenants, err := tenat_model.GetTenants(ctx)
	if err != nil {
		return nil, err
	}
	return tenants, err
}

// AddTenant добавление tenant
func AddTenant(ctx context.Context, tenant *tenat_model.ScTenant) error {
	_, err := tenat_model.InsertTenant(ctx, tenant)
	if err != nil {
		return err
	}
	return nil
}

// RemoveTenantByID удаление tenant пo tenant_id
func RemoveTenantByID(ctx context.Context, tenantID string, orgIDs []int64) error {
	if err := role_model.RemoveExistingPrivilegesByTenant(tenantID); err != nil {
		return err
	}
	if err := tenat_model.DeleteTenant(ctx, tenantID, orgIDs); err != nil {
		return err
	}
	return nil
}

// UpdateTenant обновление данных о tenant
func UpdateTenant(ctx context.Context, tenant *tenat_model.ScTenant) error {
	err := tenat_model.UpdateTenant(ctx, tenant)
	if err != nil {
		return err
	}
	return nil
}

// TenantByID получени информации о конкретном tenant по его tenant_id
func TenantByID(ctx context.Context, tenantID string) (*tenat_model.ScTenant, error) {
	ten, err := tenat_model.GetTenantByID(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	return ten, nil
}
