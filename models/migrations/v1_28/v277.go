package v1_28

import (
	"xorm.io/xorm"

	"code.gitea.io/gitea/models/repo"
)

// CreateCodeOwners функция для создания миграции таблицы code_owners
func CreateCodeOwners(x *xorm.Engine) error {
	return x.Sync(new(repo.CodeOwners))
}
