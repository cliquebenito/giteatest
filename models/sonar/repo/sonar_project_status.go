package repo

import (
	"context"

	"xorm.io/builder"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/modules/timeutil"
)

// ScSonarProjectStatus структура полей в бд для таблицы ScSonarProjectStatus
type ScSonarProjectStatus struct {
	ID          string             `xorm:"pk uuid"`
	SonarServer string             `xorm:"NOT NULL"`
	ProjectKey  string             `xorm:"NOT NULL"`
	Branch      string             `xorm:"NOT NULL"`
	AnalysedAt  timeutil.TimeStamp `xorm:"updated not null"`
	Status      string             `xorm:"NOT NULL DEFAULT 0"`
}

func init() {
	db.RegisterModel(new(ScSonarProjectStatus))
}

// UpsertSonarProjectStatus обновляем информации о проекте в базе
func UpsertSonarProjectStatus(ctx context.Context, sonarProjectStatus *ScSonarProjectStatus) (*ScSonarProjectStatus, error) {
	var sonarProjectStatusExist ScSonarProjectStatus
	has, err := db.GetEngine(ctx).
		Where(builder.And(
			builder.Eq{"sonar_server": sonarProjectStatus.SonarServer},
			builder.Eq{"project_key": sonarProjectStatus.ProjectKey},
			builder.Eq{"branch": sonarProjectStatus.Branch}),
		).Get(&sonarProjectStatusExist)
	if err != nil {
		return nil, err
	} else if !has {
		return sonarProjectStatus, db.Insert(ctx, sonarProjectStatus)
	} else {
		ss := &ScSonarProjectStatus{
			SonarServer: sonarProjectStatus.SonarServer,
			ProjectKey:  sonarProjectStatus.ProjectKey,
			Branch:      sonarProjectStatus.Branch,
			AnalysedAt:  sonarProjectStatus.AnalysedAt,
			Status:      sonarProjectStatus.Status,
		}
		_, errUpdateSonarProject := db.GetEngine(ctx).ID(sonarProjectStatusExist.ID).Update(ss)
		if errUpdateSonarProject != nil {
			return nil, errUpdateSonarProject
		}
		sonarProjectStatus.ID = sonarProjectStatusExist.ID
	}
	return sonarProjectStatus, nil
}

// GetSonarProjectStatusBySettings получаем информации из таблицы sc_sonar_projects_status по настройкам из sonar_settings
func GetSonarProjectStatusBySettings(settings *repo.ScSonarSettings, branchName string) ([]ScSonarProjectStatus, error) {
	var sonarSettings []ScSonarProjectStatus
	ses := db.GetEngine(db.DefaultContext).
		Where(builder.And(builder.Eq{"sonar_server": settings.URL}, builder.Eq{"project_key": settings.ProjectKey}))
	if branchName != "" {
		ses.Where(builder.Eq{"branch": branchName})
	}
	if err := ses.Find(&sonarSettings); err != nil {
		return nil, err
	}
	return sonarSettings, nil
}
