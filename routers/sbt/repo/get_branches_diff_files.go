package repo

import (
	"code.gitea.io/gitea/modules/context"
	apiError "code.gitea.io/gitea/routers/sbt/apierror"
	"code.gitea.io/gitea/routers/sbt/logger"
	"code.gitea.io/gitea/services/gitdiff"
	"net/http"
	"strings"
)

// GetRepoBranchesDiffFileList метод возвращает список файлов со статусами в сравнении веток
func GetRepoBranchesDiffFileList(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

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
		log.Debug("Branch not found by name: %s", baseBranch)
		ctx.JSON(http.StatusBadRequest, apiError.BranchNotExist(baseBranch))
		return
	}

	if !ctx.Repo.GitRepo.IsBranchExist(headBranch) {
		log.Debug("Branch not found by name: %s", headBranch)
		ctx.JSON(http.StatusBadRequest, apiError.BranchNotExist(headBranch))
		return
	}

	compareInfo, err := ctx.Repo.GitRepo.GetCompareInfo(baseRepo.RepoPath(), baseBranch, headBranch, false, false)
	if err != nil {
		log.Error("Unknown error type has occurred: %v", err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		return
	}

	beforeCommitID := compareInfo.MergeBase
	afterCommitID := compareInfo.HeadCommitID

	diff, err := gitdiff.GetDiffFilesWithStat(ctx.Repo.GitRepo,
		&gitdiff.DiffOptions{
			BeforeCommitID: beforeCommitID,
			AfterCommitID:  afterCommitID,
		})

	if err != nil {
		log.Error("Unknown error type has occurred: %v", err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		return
	}

	ctx.JSON(http.StatusOK, diff)
}
