package v1_24

import (
	"code.gitea.io/gitea/models/repo"
	"xorm.io/xorm"
)

// CreateScRepoLicenses создаем таблицу для хранения инфомрции о лицензиях из репозитория
func CreateScRepoLicenses(x *xorm.Engine) error {
	return x.Sync(new(repo.ScRepoLicenses))
}
