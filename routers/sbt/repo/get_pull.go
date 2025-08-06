package repo

import (
	issuesModel "code.gitea.io/gitea/models/issues"
	"code.gitea.io/gitea/modules/context"
	apiError "code.gitea.io/gitea/routers/sbt/apierror"
	sbtConvert "code.gitea.io/gitea/routers/sbt/convert"
	"code.gitea.io/gitea/routers/sbt/logger"
	"net/http"
)

// GetPullRequest Возвращает данные о пулл-реквесте по его номеру-индексу (номер не нужно путать с id)
func GetPullRequest(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	index := ctx.ParamsInt64(":index")

	pr, err := issuesModel.GetPullRequestByIndex(ctx, ctx.Repo.Repository.ID, index)
	if err != nil {
		if issuesModel.IsErrPullRequestNotExist(err) {
			log.Debug("Pull request with index: %d in repo name: %s not found, error: %v", index, ctx.Repo.Repository.Name, err)
			ctx.JSON(http.StatusBadRequest, apiError.PullRequestNotFound(index))
		} else {
			log.Error("An error has occurred while getting pull-request with index: %d in repo name: %s, error: %v", index, ctx.Repo.Repository.Name, err)
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		}
		return
	}

	if err = pr.LoadBaseRepo(ctx); err != nil {
		log.Error("An error has occurred while try load base repo of pull-request with index: %d in repo name: %s, error: %v", index, ctx.Repo.Repository.Name, err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())

		return
	}

	if err = pr.LoadHeadRepo(ctx); err != nil {
		log.Error("An error has occurred while try load head repo of pull-request with index: %d in repo name: %s, error: %v", index, ctx.Repo.Repository.Name, err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())

		return
	}

	response, err := sbtConvert.ToPullRequest(ctx, pr, ctx.Doer, log)
	if err != nil {
		log.Error("An error has occurred while converting to response DTO pull-request with index: %d in repository: %s, error: %v", index, ctx.Repo.Repository.FullName(), err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())

		return
	}

	ctx.JSON(http.StatusOK, response)
}
