package repo

import (
	"code.gitea.io/gitea/modules/cache"
	ctx "code.gitea.io/gitea/modules/context"
	apiError "code.gitea.io/gitea/routers/sbt/apierror"
	sbtCache "code.gitea.io/gitea/routers/sbt/cache"
	sbtConvert "code.gitea.io/gitea/routers/sbt/convert"
	"code.gitea.io/gitea/routers/sbt/logger"
	"net/http"
)

// GetRepo Получение данных репозитория пользователя (репозиторий получается по имени и владельцу в context_service.RepoAssigmentSbt())
func GetRepo(ctx *ctx.Context) {

	log := logger.Logger{}
	log.SetTraceId(ctx)

	value, err := cache.GetItem(
		sbtCache.GenerateRepoKey(ctx.Repo.Repository.OwnerName, ctx.Repo.Repository.Name),
		func() (cache.Item, error) {
			return getRepo(ctx, log)
		},
	)
	if cache.IsCanNotBeCached(err) {
		log.Error("Caching error: %v", err)
	}

	ctx.JSON(value.Status, value.Body)
}

func getRepo(ctx *ctx.Context, log logger.Logger) (cache.Item, error) {
	if err := ctx.Repo.Repository.LoadAttributes(ctx); err != nil {
		log.Error("Error has occurred while try load repo attributes, error: %v", err)

		return cache.Item{
				Status: http.StatusInternalServerError,
				Body:   apiError.InternalServerError(),
			},
			nil
	}

	return cache.Item{
			Status: http.StatusOK,
			Body:   sbtConvert.ToRepo(ctx, ctx.Repo.Repository, ctx.Repo.AccessMode, log),
		},
		nil
}
