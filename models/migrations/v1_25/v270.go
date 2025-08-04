package v1_25

import (
	"code.gitea.io/gitea/modules/setting"
	"xorm.io/xorm"
)

// UpdateScTenantTable обновление таблицы ScTenant в зависимости от параметра SourceControl.Enabled
func UpdateScTenantTable(x *xorm.Engine) error {
	if setting.SourceControl.Enabled {
		query := "ALTER TABLE sc_tenant ADD COLUMN IF NOT EXISTS org_key VARCHAR(50)"
		_, err := x.Exec(query)
		if err == nil {
			queryUnique := "ALTER TABLE sc_tenant ADD UNIQUE (org_key)"
			_, err = x.Exec(queryUnique)
		}
		return err
	}
	return nil
}
