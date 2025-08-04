package repo

import (
	"code.gitea.io/gitea/models"
	"code.gitea.io/gitea/models/organization"
	repoModel "code.gitea.io/gitea/models/repo"
	userModel "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/modules/web"
	apiError "code.gitea.io/gitea/routers/sbt/apierror"
	"code.gitea.io/gitea/routers/sbt/logger"
	"code.gitea.io/gitea/routers/sbt/request"
	repoService "code.gitea.io/gitea/services/repository"
	"net/http"
)

// Transfer передача прав на репозиторий
func Transfer(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	req := web.GetForm(ctx).(*request.TransferRepoOptional)

	newOwner, err := userModel.GetUserByName(ctx, req.NewOwnerName)
	if err != nil {
		if userModel.IsErrUserNotExist(err) {
			log.Debug("User with username: %s was not found", req.NewOwnerName)
			ctx.JSON(http.StatusBadRequest, apiError.UserNotFoundByNameError(req.NewOwnerName))
		} else {
			log.Error("An error has occurred while getting user with username: %s for transfer repository, error: %v", req.NewOwnerName, err)
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		}
		return
	}

	if newOwner.Type == userModel.UserTypeOrganization {
		if !ctx.Doer.IsAdmin && newOwner.Visibility == structs.VisibleTypePrivate && !organization.OrgFromUser(newOwner).HasMemberWithUserID(ctx.Doer.ID) {
			// в случае наличия настроек приватности сообщение об ошибке фейковое, что бы избежать утечки данных
			log.Error("User with username: %s found, but it is invisible", ctx.ContextUser.LowerName)
			ctx.JSON(http.StatusNotFound, apiError.UserNotFoundByNameError(ctx.ContextUser.LowerName))
			return
		}
	}

	// Close the GitRepo if open
	if ctx.Repo.GitRepo != nil {
		ctx.Repo.GitRepo.Close()
		ctx.Repo.GitRepo = nil
	}

	if err := repoService.StartRepositoryTransfer(ctx, ctx.Doer, newOwner, ctx.Repo.Repository, nil); err != nil {
		if repoModel.IsErrRepoAlreadyExist(err) {
			log.Debug("User with userId: %d already has repository with name: %s", newOwner.ID, ctx.Repo.Repository.Name)
			ctx.JSON(http.StatusBadRequest, apiError.RepoAlreadyExists())

		} else if models.IsErrRepoTransferInProgress(err) {
			log.Debug("Repository with repoId already in transfer progress", ctx.Repo.Repository.ID)
			ctx.JSON(http.StatusBadRequest, apiError.RepoTransferInProgress())

		} else {
			log.Error("An error has occurred while transfer repositoryId: %d to user with userId: %d, error: %v", ctx.Repo.Repository.ID, newOwner.ID, err)
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		}

		return
	}

	ctx.Status(http.StatusOK)
}
