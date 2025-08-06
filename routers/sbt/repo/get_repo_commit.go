package repo

import (
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/git"
	apiError "code.gitea.io/gitea/routers/sbt/apierror"
	sbtConvert "code.gitea.io/gitea/routers/sbt/convert"
	"code.gitea.io/gitea/routers/sbt/logger"
	"net/http"
)

/*
GetRepoCommit - метод возвращает коммит по SHA (коммит или ветка репозитория)

Bool- переменные, по дефолту они равны false
stat - статистика коммита
files - список файлов коммита
*/
func GetRepoCommit(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	sha := ctx.Params(":sha")

	if !git.IsValidRefPattern(sha) {
		log.Debug("Wrong git reference name sha: %s", sha)
		ctx.JSON(http.StatusBadRequest, apiError.ValidationError{FieldName: sha, ErrorMessage: "Wrong git reference name"})

		return
	}

	commitOpts := sbtConvert.ToCommitOptions{
		Stat:  ctx.FormString("stat") != "" && ctx.FormBool("stat"),
		Files: ctx.FormString("files") != "" && ctx.FormBool("files"),
	}

	commit, err := ctx.Repo.GitRepo.GetCommit(sha)
	if err != nil {
		if git.IsErrNotExist(err) {
			log.Debug("No such SHA: %s in repo with repoId: %d. Error message: %v", sha, ctx.Repo.Repository.ID, err)
			ctx.JSON(http.StatusBadRequest, apiError.GitReferenceNotExist(sha))
		} else {
			log.Error("Error has occurred while getting branch commit by SHA: %s in repoId: %d. Error: %v", sha, ctx.Repo.Repository.ID, err)
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		}
		return
	}

	json, err := sbtConvert.ToResponseCommit(ctx.Repo.GitRepo, commit, commitOpts)
	if err != nil {
		log.Error("Error has occurred while converting git.Commit to response.Commit in repoId: %d. Error: %v", ctx.Repo.Repository.ID, err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())

		return
	}

	branchName, err := commit.GetBranchName()
	if err != nil {
		log.Error("Error has occurred while getting branch name from commitId: %s in repoId: %d. Error: %v", commit.ID, ctx.Repo.Repository.ID, err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())

		return
	}
	json.BranchName = &branchName

	ctx.JSON(http.StatusOK, json)
}
