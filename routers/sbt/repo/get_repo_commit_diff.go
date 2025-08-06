package repo

import (
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/setting"
	apiError "code.gitea.io/gitea/routers/sbt/apierror"
	"code.gitea.io/gitea/routers/sbt/logger"
	"code.gitea.io/gitea/services/gitdiff"
	"net/http"
)

/*
GetRepoCommitDiff метод возвращает diff определенного коммита

MaxLineCharacters: setting.Git.MaxGitDiffLineCharacters - Поле настраиваемое в app.ini, по умолчанию 5000
MaxLines: -1  -Безлимитное количество строк
MaxFiles:  1  -Максимальное число файлов

file - путь до файла - обязательное поле
*/
func GetRepoCommitDiff(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	sha := ctx.Params(":sha")
	file := ctx.FormString("file")

	if file == "" {
		validationErr := []apiError.ValidationError{{FieldName: "file", ErrorMessage: "Required"}}
		log.Debug("Error has occurred while validate request get repo commit diff with error message: %s", validationErr)
		ctx.JSON(http.StatusBadRequest, apiError.RequestFieldValidationError("Validation error has occurred", validationErr))

		return
	}

	if !git.IsValidRefPattern(sha) {
		log.Debug("Wrong git reference name sha: %s", sha)
		ctx.JSON(http.StatusBadRequest, apiError.ValidationError{
			FieldName:    "sha",
			ErrorMessage: "Wrong git reference name"},
		)

		return
	}

	diff, err := gitdiff.GetDiffFile(ctx.Repo.GitRepo, &gitdiff.DiffOptions{
		AfterCommitID:     sha,
		MaxLines:          -1,
		MaxLineCharacters: setting.Git.MaxGitDiffLineCharacters,
		MaxFiles:          1,
	}, file)

	if err != nil {
		if git.IsErrNotExist(err) {
			log.Debug("No such SHA: %s in repo with repoId: %d. Error message: %v", sha, ctx.Repo.Repository.ID, err)
			ctx.JSON(http.StatusBadRequest, apiError.GitReferenceNotExist(sha))

		} else {
			log.Error("Error has occurred while getting diff commitId: %s in repoId: %d. Error: %v", sha, ctx.Repo.Repository.ID, err)
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		}

		return
	}

	if diff == nil {
		log.Debug("File: %s was not found in repo: %s in diff of commit SHA: %s", file, ctx.Repo.Repository.Name, sha)
		ctx.JSON(http.StatusBadRequest, apiError.FileNotFound(file))

		return
	}

	ctx.JSON(http.StatusOK, diff)
}
