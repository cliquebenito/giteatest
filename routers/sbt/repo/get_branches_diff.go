package repo

import (
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/setting"
	apiError "code.gitea.io/gitea/routers/sbt/apierror"
	"code.gitea.io/gitea/routers/sbt/logger"
	"code.gitea.io/gitea/services/gitdiff"
	"net/http"
	"strings"
)

// GetRepoBranchesDiff получаем дифф одного файла по веткам репозитория и имени файла
// todo впилить функционал работы с форками (см. routers/web/repo/compare.go ParseCompareInfo)
func GetRepoBranchesDiff(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	file := ctx.FormString("file")

	if file == "" {
		validationErr := []apiError.ValidationError{{FieldName: "file", ErrorMessage: "Required"}}
		log.Debug("Error has occurred while validate request get pr diff file with error message: %s", validationErr)
		ctx.JSON(http.StatusBadRequest, apiError.RequestFieldValidationError("Validation error has occurred", validationErr))

		return
	}

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

	diff, err := gitdiff.GetDiffFile(ctx.Repo.GitRepo,
		&gitdiff.DiffOptions{
			BeforeCommitID:    beforeCommitID,
			AfterCommitID:     afterCommitID,
			MaxLines:          -1,
			MaxLineCharacters: setting.Git.MaxGitDiffLineCharacters,
			MaxFiles:          1,
		}, file)

	if err != nil {
		log.Error("Unknown error type has occurred: %v", err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		return
	}

	if diff == nil {
		log.Debug("File: %s was not found in repo: %s in diff of after commitId: %s and before commitId: %s", file, ctx.Repo.Repository.Name, afterCommitID, beforeCommitID)
		ctx.JSON(http.StatusBadRequest, apiError.FileNotFound(file))

		return
	}

	ctx.JSON(http.StatusOK, diff)
}
