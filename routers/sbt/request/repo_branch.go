package request

import (
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/web/middleware"
	"gitea.com/go-chi/binding"
	"net/http"
)

/*
CreateRepoBranch запрос на создание ветки в репозитории
new_branch_name - имя новой ветки
old_ref_name - имя старой ветки, тега или коммита от которого создастся новая ветка
*/
type CreateRepoBranch struct {
	BranchName string `json:"new_branch_name" binding:"Required;SbtGitRefName;SbtMaxSize(100)"`
	OldRefName string `json:"old_ref_name" binding:"SbtGitRefName;SbtMaxSize(100)"`
}

/*
Validate метод валидации полей запроса, вызываемый в методе bind
*/
func (br *CreateRepoBranch) Validate(req *http.Request, errs binding.Errors) binding.Errors {
	ctx := context.GetValidateContext(req)
	return middleware.Validate(errs, ctx.Data, br, ctx.Locale)
}

// UpdateBranchName структура запроса на изменение имени ветки
type UpdateBranchName struct {
	OldName string `json:"oldBranchName" binding:"Required;SbtGitRefName;SbtMaxSize(100)"`
	NewName string `json:"newBranchName" binding:"Required;SbtGitRefName;SbtMaxSize(100)"`
}

/*
Validate метод валидации полей запроса, вызываемый в методе bind
*/
func (br *UpdateBranchName) Validate(req *http.Request, errs binding.Errors) binding.Errors {
	ctx := context.GetValidateContext(req)
	return middleware.Validate(errs, ctx.Data, br, ctx.Locale)
}
