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

// GetWatchersList метод возвращает список пользователей, которые следят за репозиторием
func GetWatchersList(ctx *context.Context) {
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

	userList, err := repoModel.GetRepoWatchers(ctx.Repo.Repository.ID, opts)

	if err != nil {
		log.Error("An error has occurred while getting users watching repoId: %d, err: %v", ctx.Repo.Repository.ID, err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())

		return
	}

	resUsers := make([]*response.User, 0, len(userList))

	for i := range userList {
		resUsers = append(resUsers, sbtConvert.ToUser(ctx, userList[i], ctx.Doer))
	}

	ctx.JSON(http.StatusOK, response.UserListResults{
		Total: int64(ctx.Repo.Repository.NumWatches),
		Data:  resUsers,
	})
}
