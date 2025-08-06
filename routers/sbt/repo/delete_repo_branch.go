package repo

import (
	"errors"
	"net/http"

	"code.gitea.io/gitea/models/git/protected_branch"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/git"
	apiError "code.gitea.io/gitea/routers/sbt/apierror"
	"code.gitea.io/gitea/routers/sbt/logger"
	repo_service "code.gitea.io/gitea/services/repository"
)

/*
DeleteRepoBranch метод удаления ветки репозитория по ее имени
*/
func DeleteRepoBranch(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	branchName := ctx.Repo.BranchName

	if err := repo_service.DeleteBranch(ctx, ctx.Doer, ctx.Repo.Repository, ctx.Repo.GitRepo, branchName); err != nil {
		switch {

		case git.IsErrNotExist(err):
			log.Debug("Branch: %s not exist in repository: %s", branchName, ctx.Repo.Repository.Name)
			ctx.JSON(http.StatusBadRequest, apiError.BranchNotExist(branchName))

		case errors.Is(err, repo_service.ErrBranchIsDefault):
			log.Debug("Can not delete branch: %s in repository: %s because branch is default", branchName, ctx.Repo.Repository.Name)
			ctx.JSON(http.StatusBadRequest, apiError.BranchIsDefault(branchName))

		case protected_branch.IsBranchIsProtectedError(err):
			log.Debug("Can not delete branch: %s in repository: %s because branch is protected", branchName, ctx.Repo.Repository.Name)
			ctx.JSON(http.StatusBadRequest, apiError.BranchIsProtected(branchName))

		default:
			log.Error("An error has occurred while try to delete branch: %s in repository: %s, error: %v", branchName, ctx.Repo.Repository.Name, err)
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		}
		return
	}

	ctx.Status(http.StatusOK)
}
