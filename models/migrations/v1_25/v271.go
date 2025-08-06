package v1_25

import (
	"code.gitea.io/gitea/modules/setting"
	"xorm.io/xorm"
)

// UpdateScTenantOrganizationTable обновление таблицы sc_tenant_organizations в зависимости от параметра SourceControl.Enabled
func UpdateScTenantOrganizationTable(x *xorm.Engine) error {
	if setting.SourceControl.Enabled {
		query := "ALTER TABLE sc_tenant_organizations ADD COLUMN IF NOT EXISTS org_key VARCHAR(50)"
		_, err := x.Exec(query)
		if err == nil {
			querySecondColumn := "ALTER TABLE sc_tenant_organizations ADD COLUMN IF NOT EXISTS project_key VARCHAR(50)"
			_, err = x.Exec(querySecondColumn)
		}
		return err
	}
	return nil
}
