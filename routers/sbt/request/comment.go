package request

import (
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/web/middleware"
	"gitea.com/go-chi/binding"
	"net/http"
	"strings"
)

// CreateComment создание комментария
type CreateComment struct {
	Content string
	Status  string `binding:"OmitEmpty;SbtIn(reopen,close)"`
	Files   []string
}

/*
Validate метод валидации полей запроса, вызываемый в методе bind
*/
func (f *CreateComment) Validate(req *http.Request, errs binding.Errors) binding.Errors {
	ctx := context.GetValidateContext(req)
	return middleware.Validate(errs, ctx.Data, f, ctx.Locale)
}

// UpdateComment обновление комментария
type UpdateComment struct {
	Content string
	Files   []string
}

/*
Validate метод валидации полей запроса, вызываемый в методе bind
*/
func (f *UpdateComment) Validate(req *http.Request, errs binding.Errors) binding.Errors {
	ctx := context.GetValidateContext(req)
	return middleware.Validate(errs, ctx.Data, f, ctx.Locale)
}

// Reaction реакция на коммент
type Reaction struct {
	Content string `binding:"Required"`
}

/*
Validate метод валидации полей запроса, вызываемый в методе bind
*/
func (f *Reaction) Validate(req *http.Request, errs binding.Errors) binding.Errors {
	ctx := context.GetValidateContext(req)
	return middleware.Validate(errs, ctx.Data, f, ctx.Locale)
}

// IssueLock причина ограничения возможности комментирования
type IssueLock struct {
	Reason string `binding:"Required"`
}

// HasValidReason проверяет, содержит ли запрос корректные варианты причины блокировки
// значения берутся из настроек приложения
func (lock IssueLock) HasValidReason() bool {
	if strings.TrimSpace(lock.Reason) == "" {
		return true
	}

	for _, v := range setting.Repository.Issue.LockReasons {
		if v == lock.Reason {
			return true
		}
	}

	return false
}

/*
Validate метод валидации полей запроса, вызываемый в методе bind
*/
func (lock *IssueLock) Validate(req *http.Request, errs binding.Errors) binding.Errors {
	ctx := context.GetValidateContext(req)
	return middleware.Validate(errs, ctx.Data, lock, ctx.Locale)
}
