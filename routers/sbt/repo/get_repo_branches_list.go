package repo

import (
	"net/http"

	"code.gitea.io/gitea/models/db"
	gitModel "code.gitea.io/gitea/models/git"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/git"
	apiError "code.gitea.io/gitea/routers/sbt/apierror"
	sbtConvert "code.gitea.io/gitea/routers/sbt/convert"
	"code.gitea.io/gitea/routers/sbt/logger"
	"code.gitea.io/gitea/routers/sbt/response"
	"code.gitea.io/gitea/services/convert"
)

// GetRepoBranchesList метод возвращает список веток репозитория
func GetRepoBranchesList(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	var totalNumOfBranches int
	var responseBranches []*response.Branch

	page := ctx.FormInt("page")
	if page <= 1 {
		page = 1
	}
	pageSize := convert.ToCorrectPageSize(ctx.FormInt("limit"))

	listOptions := db.ListOptions{
		PageSize: pageSize,
		Page:     page,
	}

	if !ctx.Repo.Repository.IsEmpty && ctx.Repo.GitRepo != nil {
		rules, err := gitModel.FindRepoProtectedBranchRules(ctx, ctx.Repo.Repository.ID)
		if err != nil {

			log.Error("Error has occurred while getting repo protected branch rules for repoId: %d. Error message: %v", ctx.Repo.Repository.ID, err)
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
			return
		}

		skip, _ := listOptions.GetStartEnd()
		branches, total, err := ctx.Repo.GitRepo.GetBranches(skip, listOptions.PageSize)
		if err != nil {

			log.Error("Error has occurred while getting repo branches for repoId: %d. Error message: %v", ctx.Repo.Repository.ID, err)
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
			return
		}

		responseBranches = make([]*response.Branch, 0, len(branches))
		for i := range branches {
			c, err := branches[i].GetCommit()
			if err != nil {
				// Skip if this branch doesn't exist anymore.
				if git.IsErrNotExist(err) {
					total--
					continue
				}

				log.Error("Error has occurred while getting repo branches commits for repoId: %d. Error message: %v", ctx.Repo.Repository.ID, err)
				ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
				return
			}

			branchProtection := gitModel.GetFirstMatched(rules, branches[i].Name)
			branch, err := sbtConvert.ToBranch(ctx, ctx.Repo.Repository, branches[i], c, branchProtection, ctx.Doer, ctx.Repo.IsAdmin())
			if err != nil {

				log.Error("Error has occurred while converting git.Commit and git.Branch to an api.Branch for repoId: %d. Error message: %v", ctx.Repo.Repository.ID, err)
				ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
				return
			}
			responseBranches = append(responseBranches, branch)
		}

		totalNumOfBranches = total
	}

	ctx.JSON(http.StatusOK, response.BranchesListResult{
		Total: totalNumOfBranches,
		Data:  responseBranches,
	})
}
