package repo

import (
	"context"

	"xorm.io/builder"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/modules/log"
)

// ScSonarProjectMetrics структура полей в бд для таблицы ScSonarProjectMetrics
type ScSonarProjectMetrics struct {
	ID                   string `xorm:"pk uuid"`
	SonarProjectStatusID string `xorm:"INDEX NOT NULL"`
	Key                  string `xorm:"NOT NULL"`
	Name                 string
	Type                 string
	Domain               string
	Value                string
	IsQualityGate        bool `xorm:"NOT NULL DEFAULT false"`
}

func init() {
	db.RegisterModel(new(ScSonarProjectMetrics))
}

// UpsertSonarMetrics обновляем информации о метриках в базе
func UpsertSonarMetrics(ctx context.Context, listScSonarMetrics []ScSonarProjectMetrics) error {
	has, err := db.GetEngine(ctx).Get(&ScSonarProjectMetrics{SonarProjectStatusID: listScSonarMetrics[0].SonarProjectStatusID})
	if err != nil {
		return err
	}
	if has {
		_, errDelete := db.GetEngine(ctx).Delete(&ScSonarProjectMetrics{SonarProjectStatusID: listScSonarMetrics[0].SonarProjectStatusID})
		if errDelete != nil {
			log.Error("UpsertSonarMetrics failed while deleting notes in table sc_sonar_project_metrics: %v", errDelete)
			return errDelete
		}
	}
	if err := db.Insert(ctx, listScSonarMetrics); err != nil {
		log.Error("UpsertSonarMetrics failed while adding notes in table sc_sonar_project_metrics: %v", err)
		return err
	}
	return nil
}

// GetSonarProjectMetrics получаем спиоск метрик из сонара
func GetSonarProjectMetrics(sonarProjectID string) ([]ScSonarProjectMetrics, error) {
	var getSonarMetrics []ScSonarProjectMetrics
	err := db.GetEngine(db.DefaultContext).
		Where(builder.Eq{"sonar_project_status_id": sonarProjectID}).
		Find(&getSonarMetrics)
	if err != nil {
		log.Error("GetSonarProjectMetrics failed while getting sonar_project_status_id %v: %v", sonarProjectID, err)
		return nil, err
	}
	return getSonarMetrics, nil
}
