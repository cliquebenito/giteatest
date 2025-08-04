package repo

import (
	issuesModel "code.gitea.io/gitea/models/issues"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/git"
	apiError "code.gitea.io/gitea/routers/sbt/apierror"
	"code.gitea.io/gitea/routers/sbt/logger"
	"code.gitea.io/gitea/services/gitdiff"
	"net/http"
)

// GetPullRequestDiffFileList метод возвращает список файлов со статусом изменений произошедших в Pull request-e
func GetPullRequestDiffFileList(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	prIndex := ctx.ParamsInt64(":index")

	pr, err := issuesModel.GetPullRequestByIndex(ctx, ctx.Repo.Repository.ID, prIndex)
	if err != nil {
		if issuesModel.IsErrPullRequestNotExist(err) {
			log.Debug("Pull request with index: %d not found in repo", prIndex)
			ctx.JSON(http.StatusBadRequest, apiError.PullRequestNotFound(prIndex))
		} else {
			log.Error("Unknown error type has occurred: %v", err)
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		}
		return
	}

	if err := pr.LoadBaseRepo(ctx); err != nil {
		log.Error("Unknown error type has occurred: %v", err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		return
	}

	if err := pr.LoadHeadRepo(ctx); err != nil {
		log.Error("Unknown error type has occurred: %v", err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		return
	}

	baseGitRepo := ctx.Repo.GitRepo

	var prInfo *git.CompareInfo
	if pr.HasMerged {
		prInfo, err = baseGitRepo.GetCompareInfo(pr.BaseRepo.RepoPath(), pr.MergeBase, pr.GetGitRefName(), true, false)
	} else {
		prInfo, err = baseGitRepo.GetCompareInfo(pr.BaseRepo.RepoPath(), pr.BaseBranch, pr.GetGitRefName(), true, false)
	}
	if err != nil {
		log.Error("Unknown error type has occurred: %v", err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		return
	}

	endCommitID, err := baseGitRepo.GetRefCommitID(pr.GetGitRefName())
	if err != nil {
		log.Error("Unknown error type has occurred: %v", err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		return
	}

	startCommitID := prInfo.MergeBase

	diff, err := gitdiff.GetDiffFilesWithStat(baseGitRepo,
		&gitdiff.DiffOptions{
			BeforeCommitID: startCommitID,
			AfterCommitID:  endCommitID,
		})

	if err != nil {
		log.Error("Unknown error type has occurred: %v", err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		return
	}

	ctx.JSON(http.StatusOK, diff)
}
