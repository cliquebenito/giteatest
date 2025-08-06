package repo

import (
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/modules/timeutil"
	"xorm.io/builder"
)

func init() {
	db.RegisterModel(new(ScSonarSettings))
}

// ScSonarSettings таблица для хранения настроек интеграции с Sonar
type ScSonarSettings struct {
	ID     int64 `xorm:"pk autoincr"`
	RepoID int64 `xorm:"INDEX"`
	// Url для доступа к Api Sonar
	URL string `xorm:"VARCHAR(2048) not null"`
	// Токен для доступа к Api Sonar
	Token string `xorm:"VARCHAR(255) not null"`
	// Ключ проекта в Sonar
	ProjectKey string             `xorm:"VARCHAR(50) not null"`
	Updated    timeutil.TimeStamp `xorm:"updated not null"`
}

// InsertOrUpdateSonarSettings добавление или изменение настроек Sonar для репозитория (если настроек не было то они добавляются, если были то обновляются новыми значениями)
func InsertOrUpdateSonarSettings(repoId int64, url string, token string, projectKey string) error {
	var res ScSonarSettings
	has, err := db.GetEngine(db.DefaultContext).Where("repo_id = ?", repoId).Get(&res)
	if err != nil {
		return err
	} else if !has {
		return db.Insert(db.DefaultContext, ScSonarSettings{
			URL:        url,
			RepoID:     repoId,
			Token:      token,
			ProjectKey: projectKey,
		})
	} else {
		_, err = db.GetEngine(db.DefaultContext).ID(res.ID).Update(ScSonarSettings{
			URL:        url,
			Token:      token,
			ProjectKey: projectKey,
		})
	}

	return err
}

// GetSonarSettings Получение настроек Sonar репозитория
func GetSonarSettings(repoId int64) (*ScSonarSettings, error) {
	var res ScSonarSettings
	has, err := db.GetEngine(db.DefaultContext).Where("repo_id = ?", repoId).Get(&res)
	if err != nil {
		return nil, err
	} else if !has {
		return nil, nil
	}

	return &res, nil
}

// DeleteSonarSettings Удаление настроек Sonar репозитория
func DeleteSonarSettings(repoId int64) error {
	_, err := db.GetEngine(db.DefaultContext).Delete(&ScSonarSettings{RepoID: repoId})
	return err
}

// GetSonarSettingsByProjectKeyAndServerUrl получаем настройку для sonarQube по project_key и serverUrl for sonarQube
func GetSonarSettingsByProjectKeyAndServerUrl(projectKey, serverUrl string) (*ScSonarSettings, error) {
	var res ScSonarSettings
	has, err := db.GetEngine(db.DefaultContext).
		Where(builder.And(
			builder.Eq{"project_key": projectKey},
			builder.Eq{"url": serverUrl})).
		Get(&res)
	if err != nil {
		return nil, err
	} else if !has {
		return nil, nil
	}
	return &res, nil
}
