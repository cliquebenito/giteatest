package repo

import (
	"code.gitea.io/gitea/models/db"
	repoModel "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/modules/web"
	apiError "code.gitea.io/gitea/routers/sbt/apierror"
	"code.gitea.io/gitea/routers/sbt/logger"
	"code.gitea.io/gitea/routers/sbt/request"
	repoService "code.gitea.io/gitea/services/repository"
	"net/http"
	"strings"
)

// UpdateRepoSettings метод обновления основных настроек репозитория
func UpdateRepoSettings(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	req := web.GetForm(ctx).(*request.RepoBaseSettingsOptional)
	repo := ctx.Repo.Repository

	if req.RepoName != nil && repo.LowerName != strings.ToLower(*req.RepoName) {
		updateRepoName(ctx, *req.RepoName)

		if ctx.Written() {
			return
		}

		repo.Name = *req.RepoName
		repo.LowerName = strings.ToLower(*req.RepoName)
	}

	if req.Description != nil {
		repo.Description = *req.Description
	}
	if req.Website != nil {
		repo.Website = *req.Website
	}
	if req.Template != nil {
		repo.IsTemplate = *req.Template
	}

	var visibilityChanged bool
	if req.Private != nil && *req.Private != repo.IsPrivate {
		// Видимость у форк репозитория должна быть такая же, как у базового репозитория
		if repo.IsFork {
			*req.Private = repo.BaseRepo.IsPrivate || repo.BaseRepo.Owner.Visibility == structs.VisibleTypePrivate
		}

		visibilityChanged = repo.IsPrivate != *req.Private
		//если включен режим ForcePrivate (Все новые репозитории создаются приватными),
		//пользователь может сделать репозиторий из публичного приватным, а из приватного публичным может сделать только админ
		if visibilityChanged && setting.Repository.ForcePrivate && !*req.Private && !ctx.Doer.IsAdmin {
			log.Debug("Can not update repository's visibility to public for repoId: %d because repository FORCE_PRIVATE is true and current userId: %d is not admin", repo.ID, ctx.Doer.ID)
			ctx.JSON(http.StatusBadRequest, apiError.UserInsufficientPermission(ctx.Doer.Name, "make repository public"))

			return
		}

		repo.IsPrivate = *req.Private
	}

	if err := repoService.UpdateRepository(ctx, repo, visibilityChanged); err != nil {
		log.Error("An error has occurred while updating settings for repoId: %d, error: %v", ctx.Repo.Repository.ID, err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())

		return
	}

	ctx.Status(http.StatusOK)
}

// updateRepoName метод переименования репозитория
func updateRepoName(ctx *context.Context, newRepoName string) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	repo := ctx.Repo.Repository

	if ctx.Repo.GitRepo != nil {
		ctx.Repo.GitRepo.Close()
		ctx.Repo.GitRepo = nil
	}

	if err := repoService.ChangeRepositoryName(ctx, ctx.Doer, repo, newRepoName); err != nil {
		switch {
		case repoModel.IsErrRepoAlreadyExist(err):
			log.Debug("Can not change name for repoId: %d because repo with repoName: %s already exist", repo.ID, newRepoName)
			ctx.JSON(http.StatusBadRequest, apiError.RepoAlreadyExists())

		case db.IsErrNameReserved(err):
			log.Debug("Can not change name for repoId: %d because new repoName: %s is reserved name", repo.ID, newRepoName)
			ctx.JSON(http.StatusBadRequest, apiError.RepoWrongName())

		case repoModel.IsErrRepoFilesAlreadyExist(err):
			log.Debug("Can not change name for repoId: %d because files already exist for this repository.", repo.ID)
			ctx.JSON(http.StatusBadRequest, apiError.RepoNotEmpty())

		case db.IsErrNamePatternNotAllowed(err):
			log.Debug("Can not change name for repoId: %d because new repoName: %s pattern is not allowed.", repo.ID, newRepoName)
			ctx.JSON(http.StatusBadRequest, apiError.RepoWrongName())

		default:
			log.Error("An error has occurred while updating repoName: %s for repoId: %d, error: %v", newRepoName, ctx.Repo.Repository.ID, err)
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		}
	}
}
