package forms

import (
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/web/middleware"
	"gitea.com/go-chi/binding"
	"net/http"
)

// AddSonarSettings ДТО настроек интеграции с Sonar
type AddSonarSettings struct {
	URL        string `binding:"Required;ValidUrl"`
	Token      string `binding:"Required"`
	ProjectKey string `binding:"Required"`
}

/*
Validate метод валидации полей запроса, вызываемый в методе bind
*/
func (f *AddSonarSettings) Validate(req *http.Request, errs binding.Errors) binding.Errors {
	ctx := context.GetValidateContext(req)
	return middleware.Validate(errs, ctx.Data, f, ctx.Locale)
}

// GetMetricsSonarQube форма для получения метрик
type GetMetricsSonarQube struct {
	RepositoryID int64  `binding:"Required"`
	Branch       string `binding:"Required"`
}

/*
Validate метод валидации полей запроса, вызываемый в методе bind
*/
func (f *GetMetricsSonarQube) Validate(req *http.Request, errs binding.Errors) binding.Errors {
	ctx := context.GetValidateContext(req)
	return middleware.Validate(errs, ctx.Data, f, ctx.Locale)
}

// GetStatusPullRequest форма для получения status для pull request из sonarqube
type GetStatusPullRequest struct {
	RepositoryID  int64  `binding:"Required"`
	Base          string `binding:"Required"`
	Branch        string `binding:"Required"`
	PullRequestID int    `binding:"Required"`
}

/*
Validate метод валидации полей запроса, вызываемый в методе bind
*/
func (f *GetStatusPullRequest) Validate(req *http.Request, errs binding.Errors) binding.Errors {
	ctx := context.GetValidateContext(req)
	return middleware.Validate(errs, ctx.Data, f, ctx.Locale)
}
