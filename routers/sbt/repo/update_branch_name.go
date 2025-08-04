package repo

import (
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/web"
	apiError "code.gitea.io/gitea/routers/sbt/apierror"
	"code.gitea.io/gitea/routers/sbt/logger"
	"code.gitea.io/gitea/routers/sbt/request"
	"code.gitea.io/gitea/services/repository"
	"net/http"
)

// RenameBranch метод переименования ветки
func RenameBranch(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	req := web.GetForm(ctx).(*request.UpdateBranchName)

	err := repository.RenameBranch(ctx, ctx.Repo.Repository, ctx.Doer, ctx.Repo.GitRepo, req.OldName, req.NewName)
	if err != nil {
		log.Error("Error has occurred while updating branchName old name: %s to new name: %s in repoId: %d. Error: %v", req.OldName, req.NewName, ctx.Repo.Repository.ID, err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		return
	}

	//TODO msg переделать совместно с SC (при изменении метода RenameBranch)
	ctx.Status(http.StatusOK)
}
