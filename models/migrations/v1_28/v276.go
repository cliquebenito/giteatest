package v1_28

import (
	"xorm.io/xorm"

	"code.gitea.io/gitea/models/repo"
)

// CreateCodeOwnersSettings функция для создания миграции таблицы code_owners
func CreateCodeOwnersSettings(x *xorm.Engine) error {
	return x.Sync(new(repo.CodeOwnersSettings))
}
