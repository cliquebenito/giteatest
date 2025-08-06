package repo

import (
	repoModel "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/modules/cache"
	ctx "code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/git"
	apiError "code.gitea.io/gitea/routers/sbt/apierror"
	sbtCache "code.gitea.io/gitea/routers/sbt/cache"
	"code.gitea.io/gitea/routers/sbt/logger"
	repoService "code.gitea.io/gitea/services/repository"
	"net/http"
	"strings"
)

/*
DeleteRepo метод удаления репозитория его владельцем
*/
func DeleteRepo(ctx *ctx.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	ownerName := ctx.Repo.Repository.OwnerName
	repoName := ctx.Repo.Repository.Name

	gitRepo, err := git.OpenRepository(ctx, ownerName, repoName, repoModel.RepoPath(ownerName, repoName))
	if err != nil {
		if strings.Contains(err.Error(), "repository does not exist") || strings.Contains(err.Error(), "no such file or directory") {
			log.Error("Repository %-v has a broken repository on the file system: %s with error: %v", ctx.Repo.Repository, ctx.Repo.Repository.RepoPath(), err)
		}
		log.Error("Try to delete invalid repo with repoPath: %s with error: %v", repoModel.RepoPath(ownerName, repoName), err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())

		return
	}

	if gitRepo != nil {
		gitRepo.Close()
	}

	if err := repoService.DeleteRepository(ctx, ctx.Doer, ctx.Repo.Repository, true); err != nil {
		log.Error("Can not to delete repository ownerName: %s repoName: %s with error: %v", ownerName, repoName, err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())

		return
	}

	ctx.Status(http.StatusOK)

	cache.RemoveItem(sbtCache.GenerateRepoListKey(ctx.Doer.Name) + "*")
}
