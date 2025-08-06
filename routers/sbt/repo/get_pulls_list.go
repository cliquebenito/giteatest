package repo

import (
	"code.gitea.io/gitea/models/db"
	issuesModel "code.gitea.io/gitea/models/issues"
	"code.gitea.io/gitea/modules/context"
	apiError "code.gitea.io/gitea/routers/sbt/apierror"
	sbtConvert "code.gitea.io/gitea/routers/sbt/convert"
	"code.gitea.io/gitea/routers/sbt/logger"
	"code.gitea.io/gitea/routers/sbt/response"
	"code.gitea.io/gitea/services/convert"
	"net/http"
)

/*
ListPullRequests
Возвращает список пулл-реквестов с учетом настроек видимости (ПРы приватного репозитория доступны только владельцу),
а так же в запросе могут быть использованы необязательные параметры для фильтрации и сортировки:

  - state - статус ПРа, возможные варианты: closed, open, all
  - sort - сортировка, возможные варианты: oldest, recentupdate, leastupdate, mostcomment, leastcomment, priority
  - milestone - этап, ИД специальной метки ПРа
  - labels - ИД метки ПРа, возможно использовать список меток в формате ?labels=foo&labels=bar&labels=etc

Параметры пагинации
  - page - номер страницы
  - limit - размер страницы
*/
func ListPullRequests(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	page := ctx.FormInt("page")
	if page <= 1 {
		page = 1
	}
	pageSize := convert.ToCorrectPageSize(ctx.FormInt("limit"))

	listOptions := db.ListOptions{
		PageSize: pageSize,
		Page:     page,
	}

	prs, maxResults, err := issuesModel.PullRequests(ctx.Repo.Repository.ID, &issuesModel.PullRequestsOptions{
		ListOptions: listOptions,
		State:       ctx.FormTrim("state"),
		SortType:    ctx.FormTrim("sort"),
		Labels:      ctx.FormStrings("labels"),
		MilestoneID: ctx.FormInt64("milestone"),
	})

	if err != nil {
		log.Error("An error has occurred while try get pull request, error: %v", err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())

		return
	}

	responsePrs := make([]*response.PullRequest, len(prs))
	for i := range prs {
		if err = prs[i].LoadIssue(ctx); err != nil {
			log.Error("An error has occurred while try load issue of pull request with id: %s, error: %v", prs[i].ID, err)
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())

			return
		}
		if err = prs[i].LoadAttributes(ctx); err != nil {
			log.Error("An error has occurred while try load attributes of pull request with id: %s, error: %v", prs[i].ID, err)
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())

			return
		}
		if err = prs[i].LoadBaseRepo(ctx); err != nil {
			log.Error("An error has occurred while try load base repo of pull request with id: %s, error: %v", prs[i].ID, err)
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())

			return
		}
		if err = prs[i].LoadHeadRepo(ctx); err != nil {
			log.Error("An error has occurred while try load head repo of pull request with id: %s, error: %v", prs[i].ID, err)
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())

			return
		}

		dto, err := sbtConvert.ToPullRequest(ctx, prs[i], ctx.Doer, log)
		if err != nil {
			log.Error("An error has occurred while converting to response DTO pull-request with index: %d in repository: %s, error: %v", prs[i].ID, ctx.Repo.Repository.FullName(), err)
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())

			return
		}

		responsePrs[i] = dto
	}

	ctx.JSON(http.StatusOK, response.PullsList{
		Total: int(maxResults),
		Data:  &responsePrs,
	})
}
