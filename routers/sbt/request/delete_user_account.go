package request

import (
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/web/middleware"
	"gitea.com/go-chi/binding"
	"net/http"
)

/*
DeleteUserAccount структура запроса удаления аккаунта пользователя
*/
type DeleteUserAccount struct {
	Password string `json:"password" binding:"Required;SbtMaxSize(254);SbtMinSize(8)"`
}

/*
Validate метод валидации полей запроса, вызываемый в методе bind
*/
func (d *DeleteUserAccount) Validate(req *http.Request, errs binding.Errors) binding.Errors {
	ctx := context.GetValidateContext(req)
	return middleware.Validate(errs, ctx.Data, d, ctx.Locale)
}
