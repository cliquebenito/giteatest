package repo

import (
	"context"
	"fmt"
	"time"

	"xorm.io/builder"
	"xorm.io/xorm"

	"code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/sonar"
	"code.gitea.io/gitea/models/sonar/domain"
	"code.gitea.io/gitea/modules/timeutil"
)

type SonarSettingsProvider interface {
	InsertSonarSettings(ctx context.Context, settings domain.CreateOrUpdateSonarProjectRequest) error
	UpdateSonarSettings(ctx context.Context, settings domain.CreateOrUpdateSonarProjectRequest) error
	DeleteSonarSettings(ctx context.Context, id int64) error
	SonarSettings(ctx context.Context, id int64) (*domain.SonarSettingsResponse, error)
}

type dbEngine interface {
	Where(interface{}, ...interface{}) *xorm.Session
	OrderBy(order interface{}, args ...interface{}) *xorm.Session
	Delete(beans ...interface{}) (int64, error)
	SQL(interface{}, ...interface{}) *xorm.Session
	Insert(...interface{}) (int64, error)
}

type SonarSettings struct {
	engine dbEngine
}

func NewSonarSettings(engine dbEngine) *SonarSettings {
	return &SonarSettings{engine: engine}
}

// InsertSonarSettings добавление настроек Sonar для репозитория
func (s SonarSettings) InsertSonarSettings(ctx context.Context, settings domain.CreateOrUpdateSonarProjectRequest) error {
	var res repo.ScSonarSettings

	has, err := s.engine.Where(builder.Eq{"repo_id": settings.RepoId}).Get(&res)
	if err != nil {
		return fmt.Errorf("get sonar settings: %w", err)
	}
	if !has {
		_, err = s.engine.Insert(repo.ScSonarSettings{
			URL:        settings.SonarServerURL,
			RepoID:     settings.RepoId,
			Token:      settings.SonarToken,
			ProjectKey: settings.SonarProjectKey,
		})
		if err != nil {
			return fmt.Errorf("insert sonar settings: %w", err)
		}
	} else {
		return sonar.ErrSonarSettingsAlreadyExists{SonarProjectKey: settings.SonarProjectKey}
	}
	return nil
}

func (s SonarSettings) UpdateSonarSettings(ctx context.Context, settings domain.CreateOrUpdateSonarProjectRequest) error {
	var res repo.ScSonarSettings
	has, err := s.engine.Where(builder.Eq{"repo_id": settings.RepoId}).Get(&res)
	if err != nil {
		return fmt.Errorf("get sonar settings: %w", err)
	}

	newSettings := repo.ScSonarSettings{
		URL:        settings.SonarServerURL,
		RepoID:     settings.RepoId,
		Token:      settings.SonarToken,
		ProjectKey: settings.SonarProjectKey,
		Updated:    timeutil.TimeStamp(time.Now().Unix()),
	}

	if !has {
		return sonar.ErrSonarSettingsNotFound{SonarProjectKey: settings.SonarProjectKey}
	} else {
		_, err = s.engine.Where(builder.Eq{"repo_id": settings.RepoId}).Update(newSettings)
		if err != nil {
			return fmt.Errorf("update sonar settings: %w", err)
		}
	}

	return nil
}
func (s SonarSettings) SonarSettings(ctx context.Context, id int64) (*domain.SonarSettingsResponse, error) {
	var res repo.ScSonarSettings
	has, err := s.engine.Where(builder.Eq{"repo_id": id}).Get(&res)
	if err != nil {
		return nil, fmt.Errorf("get sonar settings: %w", err)
	}
	if !has {
		return nil, sonar.ErrSonarSettingsNotFound{SonarProjectKey: ""}
	}
	return &domain.SonarSettingsResponse{
		SonarServerURL:  res.URL,
		SonarProjectKey: res.ProjectKey,
		SonarToken:      res.Token,
	}, nil
}
func (s SonarSettings) DeleteSonarSettings(ctx context.Context, id int64) error {
	var existing repo.ScSonarSettings
	has, err := s.engine.
		Where(builder.Eq{"repo_id": id}).
		Get(&existing)
	if err != nil {
		return fmt.Errorf("check sonar settings existence: %w", err)
	}
	if !has {
		return sonar.ErrSonarSettingsNotExist{}
	}
	_, err = s.engine.Delete(&existing)
	if err != nil {
		return fmt.Errorf("delete sonar settings: %w", err)
	}
	return nil
}
