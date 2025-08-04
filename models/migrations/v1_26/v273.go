package v1_26

import (
	"xorm.io/xorm"

	"code.gitea.io/gitea/models/organization/custom"
)

// CreateScTeamCustomPrivileges создаем таблицу sc_team_custom_privileges
func CreateScTeamCustomPrivileges(x *xorm.Engine) error {
	return x.Sync(new(custom.ScTeamCustomPrivilege))
}
