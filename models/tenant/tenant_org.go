package tenant

import (
	"context"
	"fmt"

	"xorm.io/builder"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/modules/log"
)

func init() {
	db.RegisterModel(new(ScTenantOrganizations))
}

// ScTenantOrganizations структура полей смежной таблицы sc_tenant_organizations
type ScTenantOrganizations struct {
	ID             string `xorm:"pk uuid"`
	TenantID       string `xorm:"UNIQUE(s)"`
	OrganizationID int64  `xorm:"UNIQUE(s)"`
	OrgKey         string `xorm:"VARCHAR(50) UNIQUE(s)"`
	ProjectKey     string `xorm:"VARCHAR(50) UNIQUE(s)"`
}

// GetTenantOrganizations извлекаем все organizations для tenant по tenant_id
func GetTenantOrganizations(ctx context.Context, tenantID string) ([]*ScTenantOrganizations, error) {
	var tenantOrganizations []*ScTenantOrganizations
	return tenantOrganizations, db.GetEngine(ctx).
		Where(fmt.Sprintf("tenant_id = '%s'", tenantID)).
		Find(&tenantOrganizations)
}

// GetTenantOrganizationsByOrgId извлекаем тенант для организации
func GetTenantOrganizationsByOrgId(ctx context.Context, organizationID int64) (*ScTenantOrganizations, error) {
	tenantOrganization := &ScTenantOrganizations{OrganizationID: organizationID}
	has, err := db.GetEngine(ctx).Get(tenantOrganization)
	if err != nil {
		return nil, err
	} else if !has {
		return nil, ErrTenantOrganizationNotExists{OrgID: organizationID}
	}
	return tenantOrganization, nil
}

// GetTenantByOrgIdOrDefault получаем идентификатор тенанта в которой находится организация, если не получается, то возвращаем дефолтный тенант
func GetTenantByOrgIdOrDefault(ctx context.Context, organizationID int64) (string, error) {
	tenantOrganization, err := GetTenantOrganizationsByOrgId(ctx, organizationID)
	if err != nil {
		log.Debug("Error has occurred while getting tenant by orgId: %d. Error: %v", organizationID, err)
		log.Debug("Trying get default tenant")
		defaultTenant, err := GetDefaultTenant(ctx)
		if err != nil {
			return "", err
		}
		return defaultTenant.ID, nil
	}
	return tenantOrganization.TenantID, nil
}

// GetTenantOrganizationsByKeys извлечение организаций тенанта по ключам
func GetTenantOrganizationsByKeys(ctx context.Context, orgKey, projectKey string) (*ScTenantOrganizations, error) {
	tenantOrganization := new(ScTenantOrganizations)
	has, err := db.GetEngine(ctx).
		Where(builder.Eq{"org_key": orgKey, "project_key": projectKey}).
		Get(tenantOrganization)
	if err != nil {
		return nil, err
	} else if !has {
		return nil, ErrTenantOrganizationsNotExists{OrgKey: orgKey, ProjectKey: projectKey}
	}
	return tenantOrganization, nil
}

// GetTenantOrganizationsByProjectKey извлечение организаций тенанта по ключу проекта
func GetTenantOrganizationsByProjectKey(ctx context.Context, projectKey string) (*ScTenantOrganizations, bool, error) {
	tenantOrganization := &ScTenantOrganizations{ProjectKey: projectKey}
	has, err := db.GetEngine(ctx).Get(tenantOrganization)
	if err != nil {
		return nil, false, err
	}
	return tenantOrganization, has, nil
}

// InsertTenantOrganization добавление связи tenant_id c organization_id
func InsertTenantOrganization(ctx context.Context, tenantOrganization *ScTenantOrganizations) error {
	err := db.Insert(ctx, tenantOrganization)
	if err != nil {
		return err
	}
	return nil
}

// DeleteTenantOrganization удаление organization у tenant по organization_id
func DeleteTenantOrganization(ctx context.Context, tenantID string, organizationIDs []int64) error {
	_, err := db.GetEngine(ctx).
		Where(builder.Eq{"tenant_id": tenantID}.
			And(builder.In("organization_id", organizationIDs))).
		Delete(&ScTenantOrganizations{})
	if err != nil {
		return err
	}
	return nil
}

// DeleteTenantOrg удаление organization у tenant
func DeleteTenantOrg(ctx context.Context, tenantOrganization *ScTenantOrganizations) error {
	_, err := db.GetEngine(ctx).Delete(tenantOrganization)
	if err != nil {
		return err
	}
	return nil
}

// TenantOrganizationIsExist проверяет наличие связи тенанта и организации
func TenantOrganizationIsExist(ctx context.Context, tenantID string, orgID int64) (bool, error) {
	return db.GetEngine(ctx).
		Where(
			builder.Eq{
				"tenant_id":       tenantID,
				"organization_id": orgID,
			},
		).
		Exist(&ScTenantOrganizations{})
}
