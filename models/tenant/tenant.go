package tenant

import (
	"context"

	"xorm.io/builder"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/modules/timeutil"
)

func init() {
	db.RegisterModel(new(ScTenant))
}

// ScTenant структура полей для таблицы tenant
type ScTenant struct {
	ID        string             `xorm:"pk uuid"`
	Name      string             `xorm:"VARCHAR(50) UNIQUE"`
	OrgKey    string             `xorm:"VARCHAR(50) UNIQUE"`
	Default   bool               `xorm:"NOT NULL DEFAULT true"`
	IsActive  bool               `xorm:"NOT NULL DEFAULT true"`
	CreatedAt timeutil.TimeStamp `xorm:"created"`
	UpdatedAt timeutil.TimeStamp `xorm:"updated"`
}

// GetTenants извлечение всех tenants
func GetTenants(ctx context.Context) ([]*ScTenant, error) {
	var tenants []*ScTenant
	return tenants, db.GetEngine(ctx).
		Table("sc_tenant").
		Find(&tenants)
}

// GetTenantByID извлечение tenant по tenant_id
func GetTenantByID(ctx context.Context, tenantID string) (*ScTenant, error) {
	tenant := new(ScTenant)
	has, err := db.GetEngine(ctx).
		Table("sc_tenant").
		ID(tenantID).
		Get(tenant)
	if err != nil {
		return nil, err
	} else if !has {
		return nil, ErrorTenantDoesntExists{TenantID: tenantID}
	}
	return tenant, nil
}

// GetTenantByID извлечение tenant по tenant_name
func GetTenantByName(ctx context.Context, engine db.Engine, tenantName string) (*ScTenant, error) {
	tenant := &ScTenant{Name: tenantName}
	has, err := engine.Table("sc_tenant").Get(tenant)
	if err != nil {
		return nil, err
	} else if !has {
		return nil, ErrorTenantDoesntExists{TenantID: tenantName}
	}
	return tenant, nil
}

// GetTenantByName извлечение tenant по name
func GetTenantByNameWithFlag(ctx context.Context, name string) (*ScTenant, bool, error) {
	tenant := new(ScTenant)
	has, err := db.GetEngine(ctx).Where(builder.Eq{"name": name}).Get(tenant)
	if err != nil {
		return nil, false, err
	}
	return tenant, has, nil
}

// GetTenantsByNameOrOrgKey извлечение всех tenants по name или orgKey
func GetTenantsByNameOrOrgKey(ctx context.Context, name, orgKey string) ([]*ScTenant, error) {
	var tenants []*ScTenant
	return tenants, db.GetEngine(ctx).
		Where(builder.Eq{"name": name}).
		Or(builder.Eq{"org_key": orgKey}).
		Find(&tenants)
}

// GetTenantByName извлечение tenant по orgKey
func GetTenantByOrgKey(ctx context.Context, orgKey string) (*ScTenant, bool, error) {
	tenant := new(ScTenant)
	has, err := db.GetEngine(ctx).Where(builder.Eq{"org_key": orgKey}).Get(tenant)
	if err != nil {
		return nil, false, err
	}
	return tenant, has, nil
}

// GetDefaultTenant получение дефолтного тенанта
func GetDefaultTenant(ctx context.Context) (*ScTenant, error) {
	tenant := &ScTenant{Default: true}
	has, err := db.GetEngine(ctx).Get(tenant)
	if err != nil {
		return nil, err
	} else if !has {
		return nil, ErrorTenantDoesntExists{TenantID: tenant.ID}
	}
	return tenant, nil
}

// InsertTenant вставка записи о tenant в базу данных
func InsertTenant(ctx context.Context, tenant *ScTenant) (*ScTenant, error) {
	err := db.Insert(ctx, tenant)
	if err != nil {
		return nil, err
	}
	return tenant, nil
}

// UpdateTenant обновление информации о tenant
func UpdateTenant(ctx context.Context, tenant *ScTenant) error {
	_, err := db.GetEngine(ctx).ID(tenant.ID).Cols("name", "is_active").Update(tenant)
	if err != nil {
		return err
	}
	return nil
}

// DeleteTenant удаление tenant и связанных с ним organization из таблицы tenant_project
func DeleteTenant(ctx context.Context, tenantID string, organizationIDs []int64) error {
	_, err := db.GetEngine(ctx).ID(tenantID).Delete(&ScTenant{})
	if err != nil {
		return err
	}
	err = DeleteTenantOrganization(ctx, tenantID, organizationIDs)
	if err != nil {
		return err
	}
	return nil
}
