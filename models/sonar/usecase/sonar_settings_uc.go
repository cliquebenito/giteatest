package usecase

import (
	"context"

	"code.gitea.io/gitea/models/sonar/domain"
	"code.gitea.io/gitea/models/sonar/repo"
)

type SonarSettingsUsecaser interface {
	CreateSonarSettings(ctx context.Context, settings domain.CreateOrUpdateSonarProjectRequest) error
	UpdateSonarSettings(ctx context.Context, settings domain.CreateOrUpdateSonarProjectRequest) error
	DeleteSonarSettings(ctx context.Context, repoID int64) error
	SonarSettings(ctx context.Context, repoID int64) (*domain.SonarSettingsResponse, error)
}

type Usecase struct {
	repo repo.SonarSettingsProvider
}

func NewUsecase(repo repo.SonarSettingsProvider) *Usecase {
	return &Usecase{repo: repo}
}

func (u *Usecase) CreateSonarSettings(ctx context.Context, settings domain.CreateOrUpdateSonarProjectRequest) error {
	return u.repo.InsertSonarSettings(ctx, settings)
}
func (u *Usecase) UpdateSonarSettings(ctx context.Context, settings domain.CreateOrUpdateSonarProjectRequest) error {
	return u.repo.UpdateSonarSettings(ctx, settings)
}
func (u *Usecase) SonarSettings(ctx context.Context, repoID int64) (*domain.SonarSettingsResponse, error) {
	return u.repo.SonarSettings(ctx, repoID)
}

func (u *Usecase) DeleteSonarSettings(ctx context.Context, repoID int64) error {
	return u.repo.DeleteSonarSettings(ctx, repoID)
}
