package repo

import (
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/routers/sbt/logger"
	"net/http"
)

// GetRepoBranchesNamesList метод возвращает список имен веток репозитория
func GetRepoBranchesNamesList(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	if ctx.Repo.GitRepo == nil {
		log.Debug("GET /repos/%s/%s/branches_list complete", ctx.Repo.Repository.OwnerName, ctx.Repo.Repository.Name)
		ctx.JSON(http.StatusOK, make([]string, 0))
	}

	branches, err := ctx.Repo.GitRepo.GetBranchesNames()
	if err != nil {
		log.Error("Error has occurred while getting repo branches repoId: %d. Error message: %v", ctx.Repo.Repository.ID, err)
		ctx.JSON(http.StatusInternalServerError, "GetBranches err")
		return
	}

	ctx.JSON(http.StatusOK, branches)
}
