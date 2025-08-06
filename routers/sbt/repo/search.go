package repo

import (
	"code.gitea.io/gitea/models/db"
	accessModel "code.gitea.io/gitea/models/perm/access"
	repoModel "code.gitea.io/gitea/models/repo"
	ctx "code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/setting"
	api "code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/modules/util"
	apiError "code.gitea.io/gitea/routers/sbt/apierror"
	sbtConvert "code.gitea.io/gitea/routers/sbt/convert"
	"code.gitea.io/gitea/routers/sbt/logger"
	"code.gitea.io/gitea/routers/sbt/response"
	"code.gitea.io/gitea/services/convert"
	"net/http"
)

// SearchRepos поиск репозиториев по критериям (ключевая фраза,ЯП, сортировка, параметры пагинирования, релевантность)
// Deprecated: use Search instead
func SearchRepos(ctx *ctx.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	onlyShowRelevant := setting.UI.OnlyShowRelevantRepos
	if len(ctx.Req.Form["only_show_relevant"]) != 0 {
		onlyShowRelevant = ctx.FormBool("only_show_relevant")
	}

	pageSize := ctx.FormInt("limit")
	if pageSize == 0 {
		pageSize = setting.UI.ExplorePagingNum
	}
	pageOpts := db.ListOptions{
		Page:     ctx.FormInt("page"),
		PageSize: pageSize,
	}

	var ownerID int64
	if ctx.Doer != nil && !ctx.Doer.IsAdmin {
		ownerID = ctx.Doer.ID
	}

	var orderBy db.SearchOrderBy

	switch ctx.FormString("sort") {
	case "newest":
		orderBy = db.SearchOrderByNewest
	case "oldest":
		orderBy = db.SearchOrderByOldest
	case "leastupdate":
		orderBy = db.SearchOrderByLeastUpdated
	case "reversealphabetically":
		orderBy = db.SearchOrderByAlphabeticallyReverse
	case "alphabetically":
		orderBy = db.SearchOrderByAlphabetically
	case "reversesize":
		orderBy = db.SearchOrderBySizeReverse
	case "size":
		orderBy = db.SearchOrderBySize
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
	language := ctx.FormTrim("language")

	repos, count, err := repoModel.SearchRepository(ctx, &repoModel.SearchRepoOptions{
		ListOptions:        pageOpts,
		Actor:              ctx.Doer,
		OrderBy:            orderBy,
		Private:            ctx.Doer != nil,
		Keyword:            keyword,
		OwnerID:            ownerID,
		AllPublic:          true,
		AllLimited:         true,
		Language:           language,
		IncludeDescription: setting.UI.SearchRepoDescription,
		OnlyShowRelevant:   onlyShowRelevant,
	})

	if err != nil {
		log.Error("An error occurred while search repositories, err: %v", err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())

		return
	}

	apiRepos := make([]*response.Repository, 0, len(repos))
	for i := range repos {
		access, err := accessModel.AccessLevel(ctx, ctx.Doer, repos[i])
		if err != nil {
			log.Error("An error occurred while getting access level, err: %v", err)
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())

			return
		}

		apiRepos = append(apiRepos, sbtConvert.ToRepo(ctx, repos[i], access, log))
	}

	ctx.JSON(http.StatusOK,
		response.RepoListResults{
			Total: count,
			Data:  apiRepos,
		})
}

// Search поиск репозитория
func Search(ctx *ctx.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	opts := &repoModel.SearchRepoOptions{
		ListOptions: db.ListOptions{
			Page:     ctx.FormInt("page"),
			PageSize: convert.ToCorrectPageSize(ctx.FormInt("limit")),
		},
		Actor:              ctx.Doer,
		AllPublic:          ctx.FormBool("allPublic"),
		AllLimited:         ctx.FormBool("allLimited"),
		Keyword:            ctx.FormTrim("q"),
		Language:           ctx.FormTrim("language"),
		TopicOnly:          ctx.FormBool("topic"),
		IncludeDescription: ctx.FormBool("includeDesc"),
		OwnerID:            ctx.FormInt64("uid"),
		TeamID:             ctx.FormInt64("team_id"),
		Collaborate:        util.OptionalBoolNone,
		Private:            ctx.IsSigned && (ctx.FormString("private") == "" || ctx.FormBool("private")),
		Template:           util.OptionalBoolNone,
		StarredByID:        ctx.FormInt64("starredBy"),
		WatchedByID:        ctx.FormInt64("watchedBy"),
	}

	onlyShowRelevant := setting.UI.OnlyShowRelevantRepos
	if len(ctx.Req.Form["only_show_relevant"]) != 0 {
		onlyShowRelevant = ctx.FormBool("only_show_relevant")
	}
	opts.OnlyShowRelevant = onlyShowRelevant

	if ctx.FormString("template") != "" {
		opts.Template = util.OptionalBoolOf(ctx.FormBool("template"))
	}

	if ctx.FormBool("exclusive") {
		opts.Collaborate = util.OptionalBoolFalse
	}

	mode := ctx.FormString("mode")
	switch mode {
	case "source":
		opts.Fork = util.OptionalBoolFalse
		opts.Mirror = util.OptionalBoolFalse
	case "fork":
		opts.Fork = util.OptionalBoolTrue
	case "mirror":
		opts.Mirror = util.OptionalBoolTrue
	case "collaborative":
		opts.Mirror = util.OptionalBoolFalse
		opts.Collaborate = util.OptionalBoolTrue
	case "":
	default:
		log.Debug("Unknown type of search mode: %s", mode)
		ctx.JSON(http.StatusBadRequest, apiError.RepoUnknownSearchMode(mode))
		return
	}

	if ctx.FormString("archived") != "" {
		opts.Archived = util.OptionalBoolOf(ctx.FormBool("archived"))
	}

	if ctx.FormString("is_private") != "" {
		opts.IsPrivate = util.OptionalBoolOf(ctx.FormBool("is_private"))
	}

	switch ctx.FormString("sort") {
	case "newest":
		opts.OrderBy = db.SearchOrderByNewest
	case "oldest":
		opts.OrderBy = db.SearchOrderByOldest
	case "leastupdate":
		opts.OrderBy = db.SearchOrderByLeastUpdated
	case "reversealphabetically":
		opts.OrderBy = db.SearchOrderByAlphabeticallyReverse
	case "alphabetically":
		opts.OrderBy = db.SearchOrderByAlphabetically
	case "reversesize":
		opts.OrderBy = db.SearchOrderBySizeReverse
	case "size":
		opts.OrderBy = db.SearchOrderBySize
	case "moststars":
		opts.OrderBy = db.SearchOrderByStarsReverse
	case "feweststars":
		opts.OrderBy = db.SearchOrderByStars
	case "mostforks":
		opts.OrderBy = db.SearchOrderByForksReverse
	case "fewestforks":
		opts.OrderBy = db.SearchOrderByForks
	default:
		opts.OrderBy = db.SearchOrderByRecentUpdated
	}

	repos, count, err := repoModel.SearchRepository(ctx, opts)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, api.SearchError{
			OK:    false,
			Error: err.Error(),
		})
		return
	}

	// Для улучшения производительности, если нужно только количество
	if ctx.FormBool("count_only") {
		ctx.JSON(http.StatusOK,
			response.RepoListResults{
				Total: count,
			})
		return
	}

	if err != nil {
		log.Error("An error occurred while search repositories, err: %v", err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())

		return
	}

	apiRepos := make([]*response.Repository, 0, len(repos))
	for i := range repos {
		access, err := accessModel.AccessLevel(ctx, ctx.Doer, repos[i])
		if err != nil {
			log.Error("An error occurred while getting access level, err: %v", err)
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())

			return
		}

		apiRepos = append(apiRepos, sbtConvert.ToRepo(ctx, repos[i], access, log))
	}

	ctx.JSON(http.StatusOK,
		response.RepoListResults{
			Total: count,
			Data:  apiRepos,
		})
}
