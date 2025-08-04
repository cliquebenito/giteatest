package v_32

import (
	"xorm.io/xorm"

	"code.gitea.io/gitea/models/role_model"
)

// CreateScTeamCustomPrivileges создаем таблицу sc_custom_privileges_group
func CreateScCustomPrivileges(x *xorm.Engine) error {
	return x.Sync(new((role_model.ScCustomPrivilegesGroup)))
}
