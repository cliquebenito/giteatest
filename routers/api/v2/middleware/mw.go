package middleware

import (
	"fmt"
	"net/http"
	"strconv"

	repo_model "code.gitea.io/gitea/models/repo"
	tenant2 "code.gitea.io/gitea/models/tenant"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
	project2 "code.gitea.io/gitea/services/project"
)

// Middleware структура с полем для получения доступа к бд sc_repo_key
type Middleware struct {
	repoKeyDB repo_model.RepoKeyDB
}

func NewMiddleware(repoKey repo_model.RepoKeyDB) *Middleware {
	return &Middleware{
		repoKey,
	}
}

// KeysRequiredCheck - мидлварь для валидации и заполнения объектов: репозитория и проекта
func (m Middleware) KeysRequiredCheck() func(ctx *context.APIContext) {
	return func(ctx *context.APIContext) {
		repokey := ctx.FormString("repo_key")
		projectKey := ctx.FormString("project_key")
		tenantKey := ctx.FormString("tenant_key")
		if errs := validate(repokey, projectKey, tenantKey); len(errs) > 0 {
			ctx.JSON(http.StatusBadRequest, map[string]any{
				"errors": errs,
				"url":    setting.API.SwaggerURL,
			})
			return
		}

		scRepoKey, err := m.repoKeyDB.GetRepoByKey(ctx, repokey)
		if err != nil {
			if repo_model.IsErrorRepoKeyDoesntExists(err) {
				log.Debug("Error has occurred while getting repository by key %s: %v", repokey, err)
				ctx.Error(http.StatusNotFound, "", fmt.Sprintf("Err: repository not found, repo_key: %s", repokey))
			} else {
				log.Error("Error has occurred while getting repository by key %s: %v", repokey, err)
				ctx.Error(http.StatusInternalServerError, "", "Failed to get repository")
			}
			return
		}

		repoID, err := strconv.ParseInt(scRepoKey.RepoID, 10, 64)
		if err != nil {
			log.Error("Error has occurred while parsing repository ID %s: %v", scRepoKey.RepoID, err)
			ctx.Error(http.StatusInternalServerError, "", "Invalid repository ID")
			return
		}

		repository, err := repo_model.GetRepositoryByID(ctx, repoID)
		if err != nil {
			if repo_model.IsErrRepoNotExist(err) {
				log.Debug("Error has occurred while getting repository by ID %d: %v", repoID, err)
				ctx.Error(http.StatusNotFound, "", fmt.Sprintf("Err: repository not found, repo_key: %s", repokey))
			} else {
				log.Error("Error has occurred while getting repository by ID %d: %v", repoID, err)
				ctx.Error(http.StatusInternalServerError, "", "Failed to get repository")
			}
			return
		}
		ctx.Repo.Repository = repository

		var owner *user_model.User

		owner, err = project2.GetProjectByKeys(ctx, tenantKey, projectKey)
		if err != nil {
			if tenant2.IsTenantOrganizationsNotExists(err) {
				ctx.Error(http.StatusNotFound, "Get Project", err)
				log.Error("Error has occurred while getting project: %v,project not exists", err)
				return
			}
			log.Error("Error has occurred while getting project: %v", err)
			ctx.JSON(http.StatusInternalServerError, err)
			return
		}

		ctx.Repo.Owner = owner
		ctx.ContextUser = owner

	}
}

func validate(repokey, projectkey, tenantkey string) []string {
	var errs []string
	if repokey == "" {
		errs = append(errs, "Err:repo_key is required")
	}
	if projectkey == "" {
		errs = append(errs, "Err:project_key is required")
	}
	if tenantkey == "" {
		errs = append(errs, "Err:tenant_key is required")
	}
	return errs
}
