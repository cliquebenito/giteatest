package repo

import (
	"code.gitea.io/gitea/modules/context"
	apiError "code.gitea.io/gitea/routers/sbt/apierror"
	sbtConvert "code.gitea.io/gitea/routers/sbt/convert"
	"code.gitea.io/gitea/routers/sbt/logger"
	"code.gitea.io/gitea/routers/sbt/response"
	"code.gitea.io/gitea/routers/sbt/utils"
	"net/http"
	"strings"
)

// GetRepoBranchesCommitDiff получаем список коммитов при сравнении веток репозитория
func GetRepoBranchesCommitDiff(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	listOptions := utils.GetListOptions(ctx)

	baseRepo := ctx.Repo.Repository

	var (
		infoPath   string
		headBranch string
		baseBranch string
	)

	//разбиваем параметр {baseBranch}...{headOwner}/{headRepoName}:{headBranch} на составляющие
	infoPath = ctx.Params("*")
	var infos []string
	if infoPath == "" {
		infos = []string{baseRepo.DefaultBranch, baseRepo.DefaultBranch}
	} else {
		infos = strings.SplitN(infoPath, "...", 2)
	}

	baseBranch = infos[0]

	headInfos := strings.Split(infos[1], ":")
	if len(headInfos) == 1 {
		headBranch = headInfos[0]
	} else {
		ctx.Status(http.StatusNotImplemented)
		return
	}

	if !ctx.Repo.GitRepo.IsBranchExist(baseBranch) {
		log.Debug("Branch not found by name: %s in repo: %s", baseBranch, ctx.Repo.Repository.FullName())
		ctx.JSON(http.StatusBadRequest, apiError.BranchNotExist(baseBranch))
		return
	}

	if !ctx.Repo.GitRepo.IsBranchExist(headBranch) {
		log.Debug("Branch not found by name: %s in repo: %s", headBranch, ctx.Repo.Repository.FullName())
		ctx.JSON(http.StatusBadRequest, apiError.BranchNotExist(headBranch))
		return
	}

	compareInfo, err := ctx.Repo.GitRepo.GetCompareInfo(baseRepo.RepoPath(), baseBranch, headBranch, false, false)
	if err != nil {
		log.Error("Unknown error type has occurred: %v", err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		return
	}

	commits := compareInfo.Commits

	totalNumberOfCommits := len(commits)

	start, end := listOptions.GetStartEnd()
	if end > totalNumberOfCommits {
		end = totalNumberOfCommits
	}

	apiCommits := make([]*response.Commit, 0)

	for i := start; i < end; i++ {
		commit, err := sbtConvert.ToResponseCommit(ctx.Repo.GitRepo, commits[i], sbtConvert.ToCommitOptions{})
		if err != nil {
			log.Error("Error has occurred while converting git.Commit to response. Commit: %s in repo: %s. Error: %v", commits[i].ID, ctx.Repo.Repository.FullName(), err)
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
			return
		}
		apiCommits = append(apiCommits, commit)
	}

	ctx.JSON(http.StatusOK, response.CommitListResult{
		Total:          int64(totalNumberOfCommits),
		Data:           apiCommits,
		BeforeCommitId: compareInfo.BaseCommitID,
		AfterCommitId:  compareInfo.HeadCommitID,
	})
}
