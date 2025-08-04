package v1_25

import (
	repo_model "code.gitea.io/gitea/models/sonar/repo"

	"xorm.io/xorm"
)

// CreateScSonarProjectStatus создаем таблицу для хранения информации о статусе проекта в sonar
func CreateScSonarProjectStatus(x *xorm.Engine) error {
	return x.Sync(new(repo_model.ScSonarProjectStatus))
}
