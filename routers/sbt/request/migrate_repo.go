package request

import (
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/web/middleware"
	"gitea.com/go-chi/binding"
	"net/http"
)

// MigrateRepo параметры запроса для миграции репозитория
type MigrateRepo struct {
	// Адрес исходного git репозитория
	CloneAddr string `json:"clone_addr" binding:"Required"`
	// Владелец репозитория после миграции
	RepoOwner string `json:"repo_owner"`
	// Имя нового репозитория
	RepoName string `json:"repo_name" binding:"Required;SbtAlphaDashDot;SbtMaxSize(100)"`

	// enum: git,github,gitea,gitlab
	Service      string `json:"service"`
	AuthUsername string `json:"auth_username"`
	AuthPassword string `json:"auth_password"`
	AuthToken    string `json:"auth_token"`

	Mirror         bool   `json:"mirror"`
	LFS            bool   `json:"lfs"`
	LFSEndpoint    string `json:"lfs_endpoint"`
	Private        bool   `json:"private"`
	Description    string `json:"description" binding:"SbtMaxSize(2048)"`
	Wiki           bool   `json:"wiki"`
	Milestones     bool   `json:"milestones"`
	Labels         bool   `json:"labels"`
	Issues         bool   `json:"issues"`
	PullRequests   bool   `json:"pull_requests"`
	Releases       bool   `json:"releases"`
	MirrorInterval string `json:"mirror_interval"`
}

/*
Validate метод валидации полей запроса, вызываемый в методе bind
*/
func (f *MigrateRepo) Validate(req *http.Request, errs binding.Errors) binding.Errors {
	ctx := context.GetValidateContext(req)
	return middleware.Validate(errs, ctx.Data, f, ctx.Locale)
}
