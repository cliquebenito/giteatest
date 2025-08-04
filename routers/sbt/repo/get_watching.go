package repo

import (
	"code.gitea.io/gitea/models/db"
	accessModel "code.gitea.io/gitea/models/perm/access"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/util"
	apiError "code.gitea.io/gitea/routers/sbt/apierror"
	sbtConvert "code.gitea.io/gitea/routers/sbt/convert"
	"code.gitea.io/gitea/routers/sbt/logger"
	"code.gitea.io/gitea/routers/sbt/response"
	"net/http"
)

// GetCurrentUserWatchingRepoList метод возвращает список репозиториев, за которым следит текущий пользователь,
// с возможностью поиска по списку репозиториев, сортировки этого списка, пагинации и поиска только по имени репозитория
func GetCurrentUserWatchingRepoList(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	pageSize := ctx.FormInt("limit")
	if pageSize == 0 {
		pageSize = setting.UI.ExplorePagingNum
	}
	pageOpts := db.ListOptions{
		Page:     ctx.FormInt("page"),
		PageSize: pageSize,
	}

	var orderBy db.SearchOrderBy

	switch ctx.FormString("sort") {
	case "newest":
		orderBy = db.SearchOrderByNewest
	case "oldest":
		orderBy = db.SearchOrderByOldest
	case "recentupdate":
		orderBy = db.SearchOrderByRecentUpdated
	case "leastupdate":
		orderBy = db.SearchOrderByLeastUpdated
	case "reversealphabetically":
		orderBy = db.SearchOrderByAlphabeticallyReverse
	case "alphabetically":
		orderBy = db.SearchOrderByAlphabetically
	case "moststars":
		orderBy = db.SearchOrderByStarsReverse
	case "feweststars":
		orderBy = db.SearchOrderByStars
	case "mostforks":
		orderBy = db.SearchOrderByForksReverse
	case "fewestforks":
		orderBy = db.SearchOrderByForks
	default:
		orderBy = db.SearchOrderByRecentUpdated
	}

	keyword := ctx.FormTrim("q")

	repos, count, err := repo_model.SearchRepository(ctx, &repo_model.SearchRepoOptions{
		ListOptions:        pageOpts,
		Actor:              ctx.Doer,
		Keyword:            keyword,
		OrderBy:            orderBy,
		Private:            ctx.IsSigned,
		WatchedByID:        ctx.Doer.ID,
		Collaborate:        util.OptionalBoolFalse,
		TopicOnly:          ctx.FormBool("topic"),
		IncludeDescription: setting.UI.SearchRepoDescription,
	})

	if err != nil {
		log.Error("An error has occurred while searching watching repositories for current userId: %d, err: %v", ctx.Doer.ID, err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())

		return
	}

	resRepos := make([]*response.Repository, 0, len(repos))
	for i := range repos {
		access, err := accessModel.AccessLevel(ctx, ctx.Doer, repos[i])
		if err != nil {
			log.Error("An error occurred while getting access level, err: %v", err)
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())

			return
		}

		resRepos = append(resRepos, sbtConvert.ToRepo(ctx, repos[i], access, log))
	}

	ctx.JSON(http.StatusOK, response.RepoListResults{
		Total: count,
		Data:  resRepos,
	})
}
