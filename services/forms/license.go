package forms

import (
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/web/middleware"
	"gitea.com/go-chi/binding"
	"net/http"
)

// GetLicenseInfoForm form for getting some information about license
type GetLicenseInfoForm struct {
	RepositoryID int64  `binding:"Required"`
	Branch       string `binding:"Required"`
	PathFile     string `binding:"Required"`
}

// Validate validates the fields
func (f *GetLicenseInfoForm) Validate(req *http.Request, errs binding.Errors) binding.Errors {
	ctx := context.GetValidateContext(req)
	return middleware.Validate(errs, ctx.Data, f, ctx.Locale)
}
