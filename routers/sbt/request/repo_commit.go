package request

import (
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/web/middleware"
	"gitea.com/go-chi/binding"
	"net/http"
	"time"
)

// Identity Данные автора и/или коммитира изменений
type Identity struct {
	// Имя
	Name string `json:"name" binding:"SbtMaxSize(100)"`
	// Почта
	Email string `json:"email" binding:"SbtMaxSize(254)"`
}

/*
Validate метод валидации полей запроса, вызываемый в методе bind
*/
func (f *Identity) Validate(req *http.Request, errs binding.Errors) binding.Errors {
	ctx := context.GetValidateContext(req)
	return middleware.Validate(errs, ctx.Data, f, ctx.Locale)
}

// CommitDateOptions Дата изменения автором и/или коммитером
type CommitDateOptions struct {
	Author    time.Time `json:"author"`
	Committer time.Time `json:"committer"`
}
