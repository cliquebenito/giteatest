package repo

import (
	issuesModel "code.gitea.io/gitea/models/issues"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/setting"
	apiError "code.gitea.io/gitea/routers/sbt/apierror"
	"code.gitea.io/gitea/routers/sbt/logger"
	"code.gitea.io/gitea/services/gitdiff"
	"net/http"
)

// GetPullRequestDiff получаем дифф для файла по номеру пулл реквеста и имени файла
func GetPullRequestDiff(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	file := ctx.FormString("file")

	prIndex := ctx.ParamsInt64(":index")

	if file == "" {
		validationErr := []apiError.ValidationError{{FieldName: "file", ErrorMessage: "Required"}}
		log.Debug("Error has occurred while validate request get pr diff file with error message: %s", validationErr)
		ctx.JSON(http.StatusBadRequest, apiError.RequestFieldValidationError("Validation error has occurred", validationErr))

		return
	}

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

	diff, err := gitdiff.GetDiffFile(baseGitRepo,
		&gitdiff.DiffOptions{
			BeforeCommitID:    startCommitID,
			AfterCommitID:     endCommitID,
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
		log.Debug("File: %s was not found in repo: %s in diff of pull request with index: %d", file, ctx.Repo.Repository.Name, prIndex)
		ctx.JSON(http.StatusBadRequest, apiError.FileNotFound(file))

		return
	}

	ctx.JSON(http.StatusOK, diff)
}
