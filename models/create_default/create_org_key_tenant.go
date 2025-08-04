package create_default

import (
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/tenant"
	"code.gitea.io/gitea/modules/log"
	"context"
	"fmt"
)

// InsertOrgKeyForTenant обновляем org_key для существующих тенантов
func InsertOrgKeyForTenant(ctx context.Context) error {
	var tenants []*tenant.ScTenant
	err := db.GetEngine(ctx).Find(&tenants)
	if err != nil {
		log.Error("Error getting while getting tenant", err)
		return err
	}
	if len(tenants) == 0 {
		return nil
	}
	for _, tenantEntity := range tenants {
		if tenantEntity.OrgKey == "" {
			tenantEntity.OrgKey = tenantEntity.Name
			_, err := db.GetEngine(ctx).ID(tenantEntity.ID).Update(tenantEntity)
			if err != nil {
				log.Error(fmt.Sprintf("Error updating tenant with ID: %s", tenantEntity.ID), err)
				return err
			}
		}
	}
	return nil
}
