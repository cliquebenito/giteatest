package v1_22

import (
	"code.gitea.io/gitea/models/tenant"
	userModel "code.gitea.io/gitea/models/user"
	"github.com/google/uuid"
	"xorm.io/xorm"
)

// CreateTenantOrganizationTable функция для создания миграции таблицы sc_tenant_organizations
func CreateTenantOrganizationTable(x *xorm.Engine) error {
	var organizations []*userModel.User
	if err := x.Table("user").Where("type = 1").Find(&organizations); err != nil {
		return err
	}
	if err := x.Sync(new(tenant.ScTenantOrganizations)); err != nil {
		return err
	}
	if len(organizations) == 0 {
		return nil
	}
	var tenants []*tenant.ScTenant
	if err := x.Table("sc_tenant").Find(&tenants); err != nil {
		return err
	}
	tenantOrganizations := make([]*tenant.ScTenantOrganizations, len(organizations))
	for idx, project := range organizations {
		tenantOrganizations[idx] = &tenant.ScTenantOrganizations{
			ID:             uuid.NewString(),
			TenantID:       tenants[0].ID,
			OrganizationID: project.ID,
		}
	}
	if _, err := x.Insert(&tenantOrganizations); err != nil {
		return err
	}
	return nil
}
