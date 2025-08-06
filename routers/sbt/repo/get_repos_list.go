package repo

import (
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/perm"
	accessModel "code.gitea.io/gitea/models/perm/access"
	repoModel "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/modules/cache"
	"code.gitea.io/gitea/modules/context"
	apiError "code.gitea.io/gitea/routers/sbt/apierror"
	sbtCache "code.gitea.io/gitea/routers/sbt/cache"
	sbtConvert "code.gitea.io/gitea/routers/sbt/convert"
	"code.gitea.io/gitea/routers/sbt/logger"
	"code.gitea.io/gitea/routers/sbt/response"
	"code.gitea.io/gitea/services/convert"
	"net/http"
	"strconv"
)

// ListUserRepos Получение списка репозиториев пользователя по имени пользователя.
//
//	Если пользователь не авторизован, то в списке будут только публичные репозитории указанного в пути пользователя
//	Если пользователь авторизован и в пути запроса указан этот авторизованный пользователь, то в списке будут и публичные и приватные репозитории
//	Если пользователь авторизован и в пути запроса указан другой пользователь, то в списке будут только публичные репозитории указанного пользователя
func ListUserRepos(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	opts := db.ListOptions{
		Page:     ctx.FormInt("page"),
		PageSize: convert.ToCorrectPageSize(ctx.FormInt("limit")),
	}
	if !ctx.IsSigned {

		value, err := cache.GetItem(
			sbtCache.GenerateRepoListKey(ctx.ContextUser.Name)+"?page="+strconv.Itoa(opts.Page)+"&limit="+strconv.Itoa(opts.PageSize),
			func() (cache.Item, error) {
				return listUserRepos(ctx, opts, log)
			},
		)
		if cache.IsCanNotBeCached(err) {
			log.Error("Caching error: %v", err)
		}

		ctx.JSON(value.Status, value.Body)
	} else {
		//todo попробовать впилить кеш для залогиненных пользователей с учетом их прав
		value, _ := listUserRepos(ctx, opts, log)

		ctx.JSON(value.Status, value.Body)
	}
}

func listUserRepos(ctx *context.Context, opts db.ListOptions, log logger.Logger) (cache.Item, error) {
	private := ctx.IsSigned
	u := ctx.ContextUser

	repos, count, err := repoModel.GetUserRepositories(&repoModel.SearchRepoOptions{
		Actor:       u,
		Private:     private,
		ListOptions: opts,
		OrderBy:     "id ASC",
	})
	if err != nil {
		log.Error("An error occurred while getting user: %s repositories, err: %v", u.Name, err)

		return cache.Item{
				Status: http.StatusInternalServerError,
				Body:   apiError.InternalServerError(),
			},
			nil
	}

	if err := repos.LoadAttributes(ctx); err != nil {
		log.Error("An error occurred while loading attributes for repositories, err: %v", err)

		return cache.Item{
				Status: http.StatusInternalServerError,
				Body:   apiError.InternalServerError(),
			},
			nil
	}

	apiRepos := make([]*response.Repository, 0, count)
	for i := range repos {
		access, err := accessModel.AccessLevel(ctx, ctx.Doer, repos[i])
		if err != nil {
			log.Error("An error occurred while getting access level, err: %v", err)

			return cache.Item{
					Status: http.StatusInternalServerError,
					Body:   apiError.InternalServerError(),
				},
				nil
		}
		if ctx.IsSigned && ctx.Doer.IsAdmin || access >= perm.AccessModeRead {
			apiRepos = append(apiRepos, sbtConvert.ToRepo(ctx, repos[i], access, log))
		}
	}

	return cache.Item{
		Status: http.StatusOK,
		Body: response.RepoListResults{
			Total: count,
			Data:  apiRepos,
		},
	}, nil
}
