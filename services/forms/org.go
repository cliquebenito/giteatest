// Copyright 2014 The Gogs Authors. All rights reserved.
// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forms

import (
	"net/http"

	"code.gitea.io/gitea/models/role_model"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/modules/web/middleware"

	"gitea.com/go-chi/binding"
)

// ________                            .__                __  .__
// \_____  \_______  _________    ____ |__|____________ _/  |_|__| ____   ____
//  /   |   \_  __ \/ ___\__  \  /    \|  \___   /\__  \\   __\  |/  _ \ /    \
// /    |    \  | \/ /_/  > __ \|   |  \  |/    /  / __ \|  | |  (  <_> )   |  \
// \_______  /__|  \___  (____  /___|  /__/_____ \(____  /__| |__|\____/|___|  /
//         \/     /_____/     \/     \/         \/     \/                    \/

// CreateOrgForm form for creating organization
type CreateOrgForm struct {
	OrgName                   string `binding:"Required;Username;MaxSize(40)" locale:"org.org_name_holder"`
	Visibility                structs.VisibleType
	RepoAdminChangeTeamAccess bool
}

// Validate validates the fields
func (f *CreateOrgForm) Validate(req *http.Request, errs binding.Errors) binding.Errors {
	ctx := context.GetValidateContext(req)
	return middleware.Validate(errs, ctx.Data, f, ctx.Locale)
}

// UpdateOrgSettingForm form for updating organization settings
type UpdateOrgSettingForm struct {
	Name                      string `binding:"Required;Username;MaxSize(40)" locale:"org.org_name_holder"`
	FullName                  string `binding:"MaxSize(100)"`
	Description               string `binding:"MaxSize(255)"`
	Website                   string `binding:"ValidUrl;MaxSize(255)"`
	Location                  string `binding:"MaxSize(50)"`
	Visibility                structs.VisibleType
	MaxRepoCreation           int
	RepoAdminChangeTeamAccess bool
}

// Validate validates the fields
func (f *UpdateOrgSettingForm) Validate(req *http.Request, errs binding.Errors) binding.Errors {
	ctx := context.GetValidateContext(req)
	return middleware.Validate(errs, ctx.Data, f, ctx.Locale)
}

// ___________
// \__    ___/___ _____    _____
//   |    |_/ __ \\__  \  /     \
//   |    |\  ___/ / __ \|  Y Y  \
//   |____| \___  >____  /__|_|  /
//              \/     \/      \/

// CustomPrivileges form for adding custom privileges
type CustomPrivileges struct {
	AllRepositories bool                         `json:"all_repositories"`
	RepoID          int64                        `json:"repo_id"`
	Privileges      []role_model.CustomPrivilege `json:"privileges"`
}

// CreateTeamForm form for creating team
type CreateTeamForm struct {
	TeamName         string `json:"team_name" binding:"Required;AlphaDashDot;MaxSize(30)"`
	Description      string `binding:"MaxSize(255)"`
	Permission       string
	RepoAccess       string
	CanCreateOrgRepo bool
	CustomPrivileges []CustomPrivileges `json:"custom_privileges"`
	UserIDs          []int64            `json:"user_ids"`
}

// Validate validates the fields
func (f *CustomPrivileges) Validate(req *http.Request, errs binding.Errors) binding.Errors {
	ctx := context.GetValidateContext(req)
	return middleware.Validate(errs, ctx.Data, f, ctx.Locale)
}

// Validate validates the fields
func (f *CreateTeamForm) Validate(req *http.Request, errs binding.Errors) binding.Errors {
	ctx := context.GetValidateContext(req)
	return middleware.Validate(errs, ctx.Data, f, ctx.Locale)
}

// GrantPrivilegesForm form for grant privileges
type GrantPrivilegesForm struct {
	UserId   int64  `json:"user_id" binding:"required"`
	TenantId string `json:"tenant_id" binding:"required"`
	OrgId    int64  `json:"org_id" binding:"required"`
	Role     string `json:"role" binding:"required"`
}

// Validate validates the fields
func (f *GrantPrivilegesForm) Validate(req *http.Request, errs binding.Errors) binding.Errors {
	ctx := context.GetValidateContext(req)
	return middleware.Validate(errs, ctx.Data, f, ctx.Locale)
}

// CheckPrivilegesForm form for check privileges
type CheckPrivilegesForm struct {
	UserId   int64  `json:"user_id" binding:"required"`
	TenantId string `json:"tenant_id" binding:"required"`
	OrgId    int64  `json:"org_id" binding:"required"`
	Action   string `json:"action" binding:"required"`
}

// Validate validates the fields
func (f *CheckPrivilegesForm) Validate(req *http.Request, errs binding.Errors) binding.Errors {
	ctx := context.GetValidateContext(req)
	return middleware.Validate(errs, ctx.Data, f, ctx.Locale)
}

// UserPrivilegesForm form for get privileges
type UserPrivilegesForm struct {
	UserId   int64  `json:"user_id" binding:"required"`
	TenantId string `json:"tenant_id" binding:"required"`
	OrgId    int64  `json:"org_id" binding:"required"`
}

// Validate validates the fields
func (f *UserPrivilegesForm) Validate(req *http.Request, errs binding.Errors) binding.Errors {
	ctx := context.GetValidateContext(req)
	return middleware.Validate(errs, ctx.Data, f, ctx.Locale)
}

// CreateProjectApiForm form for create project from public api
type CreateProjectApiForm struct {
	Name        string              `json:"name" binding:"required;MaxSize(50)"`
	OrgKey      string              `json:"org_key" binding:"required;MaxSize(50)"`
	ProjectKey  string              `json:"project_key" binding:"required;MaxSize(50)"`
	Description string              `json:"description"`
	Visibility  structs.VisibleType `json:"visibility" binding:"required"`
}

// Validate validates the fields
func (f *CreateProjectApiForm) Validate(req *http.Request, errs binding.Errors) binding.Errors {
	ctx := context.GetValidateContext(req)
	return middleware.Validate(errs, ctx.Data, f, ctx.Locale)
}

// ModifyProjectApiForm form for modify project from public api
type ModifyProjectApiForm struct {
	Name        string              `json:"name" binding:"MaxSize(50)"`
	ProjectKey  string              `json:"project_key" binding:"required;MaxSize(50)"`
	Description string              `json:"description"`
	Visibility  structs.VisibleType `json:"visibility"`
}

// Validate validates the fields
func (f *ModifyProjectApiForm) Validate(req *http.Request, errs binding.Errors) binding.Errors {
	ctx := context.GetValidateContext(req)
	return middleware.Validate(errs, ctx.Data, f, ctx.Locale)
}

// DeleteProjectApiForm form for delete project from public api
type DeleteProjectApiForm struct {
	OrgKey     string `json:"org_key" binding:"required;MaxSize(50)"`
	ProjectKey string `json:"project_key" binding:"required;MaxSize(50)"`
}

// Validate validates the fields
func (f *DeleteProjectApiForm) Validate(req *http.Request, errs binding.Errors) binding.Errors {
	ctx := context.GetValidateContext(req)
	return middleware.Validate(errs, ctx.Data, f, ctx.Locale)
}

// AddReposForTeam добавляем репозитории к команде
type AddReposForTeam struct {
	RepoIDs []int64 `json:"repo_ids" binding:"required"`
}

// Validate validates the fields
func (f *AddReposForTeam) Validate(req *http.Request, errs binding.Errors) binding.Errors {
	ctx := context.GetValidateContext(req)
	return middleware.Validate(errs, ctx.Data, f, ctx.Locale)
}
