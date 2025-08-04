package forms

import (
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/web/middleware"
	"gitea.com/go-chi/binding"
	"net/http"
)

// PreReceiveHookForm форма для создания или обновления pre-receive git хука
type PreReceiveHookForm struct {
	Path       string `binding:"Required"`
	Timeout    int64  `binding:"Required"`
	Parameters map[string]string
}

// Validate для формы PreReceiveHookForm
func (f *PreReceiveHookForm) Validate(req *http.Request, errs binding.Errors) binding.Errors {
	ctx := context.GetValidateContext(req)
	return middleware.Validate(errs, ctx.Data, f, ctx.Locale)
}
