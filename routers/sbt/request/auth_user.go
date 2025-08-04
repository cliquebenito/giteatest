package request

import (
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/web/middleware"
	"gitea.com/go-chi/binding"
	"net/http"
)

// SignIn DTO авторизации пользователя по почте/паролю
// todo использовать универсальный Login вместо Email в OpenAPI
type SignIn struct {
	Login    string `json:"login" binding:"Required"`
	Password string `json:"password" binding:"Required"`
}

/*
Validate метод валидации полей запроса, вызываемый в методе bind
*/
func (f *SignIn) Validate(req *http.Request, errs binding.Errors) binding.Errors {
	ctx := context.GetValidateContext(req)
	return middleware.Validate(errs, ctx.Data, f, ctx.Locale)
}
