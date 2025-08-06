package v1_25

import (
	sonar_model "code.gitea.io/gitea/models/sonar/repo"

	"xorm.io/xorm"
)

// CreateScSonarProjectMetrics создаем таблицу для хранения информации o метриках для проекта sonar
func CreateScSonarProjectMetrics(x *xorm.Engine) error {
	return x.Sync(new(sonar_model.ScSonarProjectMetrics))
}
