package forms

import (
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/web/middleware"
	"gitea.com/go-chi/binding"
	"net/http"
)

// CreateTenantForm форма для создания tenant
type CreateTenantForm struct {
	Name     string `binding:"Required;MaxSize(50)"`
	IsActive bool
}

// Validate для формы CreateTenantForm
func (f *CreateTenantForm) Validate(req *http.Request, errs binding.Errors) binding.Errors {
	ctx := context.GetValidateContext(req)
	return middleware.Validate(errs, ctx.Data, f, ctx.Locale)
}

// EditTenantForm форма для обновления tenant
type EditTenantForm struct {
	Name     string `binding:"Required;MaxSize(50)"`
	IsActive bool
}

// Validate для формы EditTenantForm
func (f *EditTenantForm) Validate(req *http.Request, errs binding.Errors) binding.Errors {
	ctx := context.GetValidateContext(req)
	return middleware.Validate(errs, ctx.Data, f, ctx.Locale)
}

// CreateTenantOrganizationForm форма для создания связи tenant c project
type CreateTenantOrganizationForm struct {
	TenantID       string `binding:"Required"`
	OrganizationID int64  `binding:"Required"`
}

// Validate для формы CreateTenantProjectForm
func (f *CreateTenantOrganizationForm) Validate(req *http.Request, errs binding.Errors) binding.Errors {
	ctx := context.GetValidateContext(req)
	return middleware.Validate(errs, ctx.Data, f, ctx.Locale)
}

// CreateTenantApiForm форма для создания tenant
type CreateTenantApiForm struct {
	Name   string `json:"name" binding:"Required;MaxSize(50)"`
	OrgKey string `json:"org_key" binding:"Required;MaxSize(50)"`
}

// Validate для формы CreateTenantApiForm
func (f *CreateTenantApiForm) Validate(req *http.Request, errs binding.Errors) binding.Errors {
	ctx := context.GetValidateContext(req)
	return middleware.Validate(errs, ctx.Data, f, ctx.Locale)
}

// EditTenantApiForm форма для редактирования tenant
type EditTenantApiForm struct {
	Name string `json:"name" binding:"Required;MaxSize(50)"`
}

// Validate для формы EditTenantApiForm
func (f *EditTenantApiForm) Validate(req *http.Request, errs binding.Errors) binding.Errors {
	ctx := context.GetValidateContext(req)
	return middleware.Validate(errs, ctx.Data, f, ctx.Locale)
}
