package request

import (
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/web/middleware"
	"gitea.com/go-chi/binding"
	"net/http"
)

// CreateRepo DTO создания репозитория
type CreateRepo struct {
	// Идентификатор организации
	OrgId int64 `json:"org_id"`
	// Имя репозитория
	Name string `json:"name" binding:"Required;SbtAlphaDashDot;SbtMaxSize(100)"`
	// Описание
	Description string `json:"description" binding:"SbtMaxSize(2048)"`
	// Приватный?
	Private bool `json:"private"`
	// Список меток
	IssueLabels string `json:"issue_labels"`
	// Инициализировать при создании?
	AutoInit bool `json:"auto_init"`
	// Шаблон?
	Template bool `json:"template"`
	// Gitignores
	Gitignores string `json:"gitignores"`
	// Лицензия
	License string `json:"license"`
	// Readme создаваемый с репой
	Readme string `json:"readme"`
	// Имя ветки по умолчанию
	DefaultBranch string `json:"default_branch" binding:"SbtMaxSize(100)"`
	// enum: default,collaborator,committer,collaboratorcommitter
	TrustModel string `json:"trust_model"`
}

/*
Validate метод валидации полей запроса, вызываемый в методе bind
*/
func (f *CreateRepo) Validate(req *http.Request, errs binding.Errors) binding.Errors {
	ctx := context.GetValidateContext(req)
	return middleware.Validate(errs, ctx.Data, f, ctx.Locale)
}

// Repo DTO репозитория
type Repo struct {
	// Идентификатор репозитория
	ID int64 `json:"id"`
}

// ForkRepo DTO форка репозитория
type ForkRepo struct {
	// Название организации, если форкаем в организацию
	Organization string `json:"organization"`
	// name of the forked repository
	Name string `json:"name" binding:"Required;SbtAlphaDashDot;SbtMaxSize(100)"`
	// Описание
	Description string `json:"description" binding:"SbtMaxSize(2048)"`
}

/*
Validate метод валидации полей запроса, вызываемый в методе bind
*/
func (f *ForkRepo) Validate(req *http.Request, errs binding.Errors) binding.Errors {
	ctx := context.GetValidateContext(req)
	return middleware.Validate(errs, ctx.Data, f, ctx.Locale)
}
