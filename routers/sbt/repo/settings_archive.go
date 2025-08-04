package repo

import (
	repoModel "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/modules/context"
	apiError "code.gitea.io/gitea/routers/sbt/apierror"
	"code.gitea.io/gitea/routers/sbt/logger"
	"net/http"
)

// Archive метод архивирования репозитория
func Archive(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	if ctx.Repo.Repository.IsMirror {
		log.Debug("Can not archive repoId: %d because repository is mirror", ctx.Repo.Repository.ID)
		ctx.JSON(http.StatusBadRequest, apiError.RepoIsMirror())
		return
	}

	if err := repoModel.SetArchiveRepoState(ctx.Repo.Repository, true); err != nil {
		log.Error("An error has occurred while trying to archive repository with repoId: %d, err: %v", ctx.Repo.Repository.ID, err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		return
	}

	ctx.Status(http.StatusOK)
}

// Unarchive метод разархивирования репозитория
func Unarchive(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	if err := repoModel.SetArchiveRepoState(ctx.Repo.Repository, false); err != nil {
		log.Error("An error has occurred while trying to unarchive repository with repoId: %d, err: %v", ctx.Repo.Repository.ID, err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		return
	}

	ctx.Status(http.StatusOK)
}
