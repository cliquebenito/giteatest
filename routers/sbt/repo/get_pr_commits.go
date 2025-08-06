package repo

import (
	issuesModel "code.gitea.io/gitea/models/issues"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/git"
	apiError "code.gitea.io/gitea/routers/sbt/apierror"
	sbtConvert "code.gitea.io/gitea/routers/sbt/convert"
	"code.gitea.io/gitea/routers/sbt/logger"
	"code.gitea.io/gitea/routers/sbt/response"
	"code.gitea.io/gitea/routers/sbt/utils"
	"net/http"
)

// GetPullRequestCommits получаем список коммитов по номеру пулл реквеста
func GetPullRequestCommits(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	prIndex := ctx.ParamsInt64(":index")
	listOptions := utils.GetListOptions(ctx)

	pr, err := issuesModel.GetPullRequestByIndex(ctx, ctx.Repo.Repository.ID, prIndex)
	if err != nil {
		if issuesModel.IsErrPullRequestNotExist(err) {
			log.Debug("Pull request with index: %d not found in repo %s", prIndex, ctx.Repo.Repository.FullName())
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

	commits := prInfo.Commits

	totalNumberOfCommits := len(commits)

	start, end := listOptions.GetStartEnd()
	if end > totalNumberOfCommits {
		end = totalNumberOfCommits
	}

	responseCommits := make([]*response.Commit, 0)

	for i := start; i < end; i++ {
		commit, err := sbtConvert.ToResponseCommit(baseGitRepo, commits[i], sbtConvert.ToCommitOptions{})
		if err != nil {
			log.Error("Error has occurred while converting git.Commit to response. Commit: %s in repo: %s. Error: %v", commits[i].ID, ctx.Repo.Repository.FullName(), err)
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
			return
		}
		responseCommits = append(responseCommits, commit)
	}

	ctx.JSON(http.StatusOK, response.CommitListResult{
		Total:          int64(totalNumberOfCommits),
		Data:           responseCommits,
		BeforeCommitId: prInfo.BaseCommitID,
		AfterCommitId:  prInfo.HeadCommitID,
	})
}
