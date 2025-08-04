package v1_22

import (
	"code.gitea.io/gitea/models/tenant"
	"code.gitea.io/gitea/modules/timeutil"
	"github.com/google/uuid"
	"xorm.io/xorm"
)

// CreateTenantTable функция для создания миграции таблицы sc_tenant
func CreateTenantTable(x *xorm.Engine) error {
	if err := x.Sync(new(tenant.ScTenant)); err != nil {
		return err
	}
	ten := &tenant.ScTenant{
		ID:        uuid.NewString(),
		Name:      "tenant",
		Default:   true,
		IsActive:  true,
		CreatedAt: timeutil.TimeStampNow(),
		UpdatedAt: timeutil.TimeStampNow(),
	}
	if _, err := x.Insert(ten); err != nil {
		return err
	}
	return nil
}
