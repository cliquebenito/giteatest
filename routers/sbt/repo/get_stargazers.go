package repo

import (
	"code.gitea.io/gitea/models/db"
	repoModel "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/modules/context"
	apiError "code.gitea.io/gitea/routers/sbt/apierror"
	sbtConvert "code.gitea.io/gitea/routers/sbt/convert"
	"code.gitea.io/gitea/routers/sbt/logger"
	"code.gitea.io/gitea/routers/sbt/response"
	"code.gitea.io/gitea/services/convert"
	"net/http"
)

// GetStargazersList метод поиска списка людей, которые пометили репозиторий звездой (добавили в избранное)
func GetStargazersList(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	page := ctx.FormInt("page")
	if page <= 1 {
		page = 1
	}
	pageSize := convert.ToCorrectPageSize(ctx.FormInt("limit"))

	opts := db.ListOptions{
		PageSize: pageSize,
		Page:     page,
	}

	userList, err := repoModel.GetStargazers(ctx.Repo.Repository, opts)

	if err != nil {
		log.Error("An error has occurred while getting users who starred repoId: %d, err: %v", ctx.Repo.Repository.ID, err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())

		return
	}

	resUsers := make([]*response.User, 0, len(userList))

	for i := range userList {
		resUsers = append(resUsers, sbtConvert.ToUser(ctx, userList[i], ctx.Doer))
	}

	ctx.JSON(http.StatusOK, response.UserListResults{
		Total: int64(ctx.Repo.Repository.NumStars),
		Data:  resUsers,
	})
}
