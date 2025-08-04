package repo

import (
	"net/http"

	gitModel "code.gitea.io/gitea/models/git"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/git"
	apiError "code.gitea.io/gitea/routers/sbt/apierror"
	sbtConvert "code.gitea.io/gitea/routers/sbt/convert"
	"code.gitea.io/gitea/routers/sbt/logger"
)

/*
GetRepoBranch запрос на информацию о ветке репозитория по имени ветки
*/
func GetRepoBranch(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	branchName := ctx.Repo.BranchName

	branch, err := ctx.Repo.GitRepo.GetBranch(branchName)
	if err != nil {
		if git.IsErrBranchNotExist(err) {
			log.Debug("Branch: %s not exist in repository: %s, error: %v", branchName, ctx.Repo.Repository.Name, err)
			ctx.JSON(http.StatusBadRequest, apiError.BranchNotExist(branchName))
		} else {
			log.Error("Error has occurred while getting branch: %s repoId: %d. Error message: %v", branchName, ctx.Repo.Repository.ID, err)
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		}
		return
	}

	branchProtection, err := gitModel.GetMergeMatchProtectedBranchRule(ctx, ctx.Repo.Repository.ID, branchName)
	if err != nil {
		log.Error("Error has occurred while getting protected branch rule for repoId: %d. Error message: %v", ctx.Repo.Repository.ID, err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		return
	}

	branchRes, err := sbtConvert.ToBranch(ctx, ctx.Repo.Repository, branch, ctx.Repo.Commit, branchProtection, ctx.Doer, ctx.Repo.IsAdmin())
	if err != nil {

		log.Error("Error has occurred while converting git.Commit and git.Branch to an api.Branch for repoId: %d. Error message: %v", ctx.Repo.Repository.ID, err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		return
	}

	ctx.JSON(http.StatusOK, branchRes)
}
