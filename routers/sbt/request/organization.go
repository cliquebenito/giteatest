package request

import (
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/web/middleware"
	"gitea.com/go-chi/binding"
	"net/http"
)

// CreateOrganizationOptional структура запроса на создание организации.
// Поле name (название организации) обязательное, остальные поля опциональны
type CreateOrganizationOptional struct {
	Name                      string  `json:"name" binding:"Required;SbtMaxSize(50);SbtMinSize(2)"`
	Description               *string `json:"description" binding:"SbtMaxSize(255)"`
	FullName                  *string `json:"full_name" binding:"SbtMaxSize(100)"`
	RepoAdminChangeTeamAccess *bool   `json:"repo_admin_change_team_access"`
	Location                  *string `json:"location" binding:"SbtMaxSize(50)"`
	Visibility                *string `json:"visibility" binding:"SbtIn(public,limited,private)"`
	Website                   *string `json:"website" binding:"SbtUrl;SbtMaxSize(255)"`
}

// Validate validates the fields
func (f *CreateOrganizationOptional) Validate(req *http.Request, errs binding.Errors) binding.Errors {
	ctx := context.GetValidateContext(req)
	return middleware.Validate(errs, ctx.Data, f, ctx.Locale)
}

// UpdateOrgSettingsOptional структура запроса на обновление настроек организации
type UpdateOrgSettingsOptional struct {
	Name                      *string `json:"name" binding:"SbtMaxSize(50);SbtMinSize(2)"`
	Description               *string `json:"description" binding:"SbtMaxSize(255)"`
	FullName                  *string `json:"full_name" binding:"SbtMaxSize(100)"`
	RepoAdminChangeTeamAccess *bool   `json:"repo_admin_change_team_access"`
	Location                  *string `json:"location" binding:"SbtMaxSize(50)"`
	Visibility                *string `json:"visibility" binding:"SbtIn(public,limited,private)"`
	Website                   *string `json:"website" binding:"SbtUrl;SbtMaxSize(255)"`
	MaxRepoCreation           *int    `json:"max_repo_creation"`
}

// Validate validates the fields
func (f *UpdateOrgSettingsOptional) Validate(req *http.Request, errs binding.Errors) binding.Errors {
	ctx := context.GetValidateContext(req)
	return middleware.Validate(errs, ctx.Data, f, ctx.Locale)
}

// OrganizationAvatar структура запроса на смену аватара организации
type OrganizationAvatar struct {
	Image string `json:"image" binding:"Required"`
}

// Validate validates the fields
func (f *OrganizationAvatar) Validate(req *http.Request, errs binding.Errors) binding.Errors {
	ctx := context.GetValidateContext(req)
	return middleware.Validate(errs, ctx.Data, f, ctx.Locale)
}
