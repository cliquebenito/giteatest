package repo

import (
	"code.gitea.io/gitea/models"
	repoModel "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/modules/context"
	apiError "code.gitea.io/gitea/routers/sbt/apierror"
	"code.gitea.io/gitea/routers/sbt/logger"
	repoService "code.gitea.io/gitea/services/repository"
	"errors"
	"net/http"
)

// UpdateAction метод обновления действия относительно репозиториев:
//   - Следить/не следить
//   - Добавить в избранное / убрать из избранного
//   - Принять права на репозиторий / отказаться от прав на репозиторий
func UpdateAction(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	var err error
	switch ctx.Params(":action") {
	case "watch":
		err = repoModel.WatchRepo(ctx, ctx.Doer.ID, ctx.Repo.Repository.ID, true)
	case "unwatch":
		err = repoModel.WatchRepo(ctx, ctx.Doer.ID, ctx.Repo.Repository.ID, false)
	case "star":
		err = repoModel.StarRepo(ctx.Doer.ID, ctx.Repo.Repository.ID, true)
	case "unstar":
		err = repoModel.StarRepo(ctx.Doer.ID, ctx.Repo.Repository.ID, false)
	case "acceptTransfer":
		err = acceptOrRejectRepoTransfer(ctx, true)
	case "rejectTransfer":
		err = acceptOrRejectRepoTransfer(ctx, false)
	default:
		log.Debug("Unknown type of action: %s", ctx.Params(":action"))
		ctx.JSON(http.StatusBadRequest, apiError.RepoUnknownActionType(ctx.Params(":action")))
		return
	}

	if err != nil {
		log.Error("Unknown error type has occurred while updating repoId: %d actions, error: %v", ctx.Repo.Repository, err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		return
	}

	ctx.Status(http.StatusOK)
}

// acceptOrRejectRepoTransfer метод принятия или отказа от прав на репозиторий
func acceptOrRejectRepoTransfer(ctx *context.Context, accept bool) error {
	repoTransfer, err := models.GetPendingRepositoryTransfer(ctx, ctx.Repo.Repository)
	if err != nil {
		return err
	}

	if err := repoTransfer.LoadAttributes(ctx); err != nil {
		return err
	}

	if !repoTransfer.CanUserAcceptTransfer(ctx.Doer) {
		return errors.New("user does not have enough permissions")
	}

	if accept {
		if ctx.Repo.GitRepo != nil {
			ctx.Repo.GitRepo.Close()
			ctx.Repo.GitRepo = nil
		}

		if err := repoService.TransferOwnership(ctx, repoTransfer.Doer, repoTransfer.Recipient, ctx.Repo.Repository, repoTransfer.Teams); err != nil {
			return err
		}
	} else {
		if err := models.CancelRepositoryTransfer(ctx.Repo.Repository); err != nil {
			return err
		}
	}

	return nil
}
