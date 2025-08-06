package v1_21

import (
	"code.gitea.io/gitea/models/repo"
	"xorm.io/xorm"
)

// CreateScSonarSettingsTable создание таблицы ScSonarSettings в зависимости от параметра SourceControl.Enabled
func CreateScSonarSettingsTable(x *xorm.Engine) error {
	return x.Sync(new(repo.ScSonarSettings))
}
