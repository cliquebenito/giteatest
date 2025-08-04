package repo

import (
	"code.gitea.io/gitea/models"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/web"
	apiError "code.gitea.io/gitea/routers/sbt/apierror"
	"code.gitea.io/gitea/routers/sbt/logger"
	"code.gitea.io/gitea/routers/sbt/request"
	files_service "code.gitea.io/gitea/services/repository/files"
	"fmt"
	"net/http"
	"time"
)

// DeleteFile удаление файла в репозитории, если указана ветка то в этой ветке, если нет то в ветки по умолчанию
func DeleteFile(ctx *context.Context) {
	deleteFileOptions := web.GetForm(ctx).(*request.DeleteFileOptions)

	if deleteFileOptions.BranchName == "" {
		deleteFileOptions.BranchName = ctx.Repo.Repository.DefaultBranch
	}

	opts := &files_service.DeleteRepoFileOptions{
		Message:   deleteFileOptions.Message,
		OldBranch: deleteFileOptions.BranchName,
		NewBranch: deleteFileOptions.NewBranchName,
		SHA:       deleteFileOptions.SHA,
		TreePath:  ctx.Params("*"),
		Committer: &files_service.IdentityOptions{
			Name:  deleteFileOptions.Committer.Name,
			Email: deleteFileOptions.Committer.Email,
		},
		Author: &files_service.IdentityOptions{
			Name:  deleteFileOptions.Author.Name,
			Email: deleteFileOptions.Author.Email,
		},
		Dates: &files_service.CommitDateOptions{
			Author:    deleteFileOptions.Dates.Author,
			Committer: deleteFileOptions.Dates.Committer,
		},
		Signoff: deleteFileOptions.Signoff,
	}
	if opts.Dates.Author.IsZero() {
		opts.Dates.Author = time.Now()
	}
	if opts.Dates.Committer.IsZero() {
		opts.Dates.Committer = time.Now()
	}

	if opts.Message == "" {
		opts.Message = ctx.Tr("repo.editor.delete", opts.TreePath)
	}

	if fileResponse, err := files_service.DeleteRepoFile(ctx, ctx.Repo.Repository, ctx.Doer, opts); err != nil {
		handleDeleteRepoFileError(ctx, err)
	} else {
		ctx.JSON(http.StatusOK, fileResponse.Commit)
	}
}

// handleDeleteRepoFileError обработка ошибок удаления файла
func handleDeleteRepoFileError(ctx *context.Context, err error) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	switch true {
	case git.IsErrBranchNotExist(err):
		log.Debug("Branch: %s not exist in repository: %s, error: %v", err.(git.ErrBranchNotExist).Name, ctx.Repo.Repository.Name, err)
		ctx.JSON(http.StatusBadRequest, apiError.BranchNotExist(err.(git.ErrBranchNotExist).Name))

	case models.IsErrBranchDoesNotExist(err):
		log.Debug("Branch: %s not exist in repository: %s, error: %v", err.(models.ErrBranchDoesNotExist).BranchName, ctx.Repo.Repository.Name, err)
		ctx.JSON(http.StatusBadRequest, apiError.BranchNotExist(err.(models.ErrBranchDoesNotExist).BranchName))

	case git.IsErrNotExist(err):
		log.Debug("Commit not exist, error: %v", err)
		ctx.JSON(http.StatusBadRequest, apiError.CommitNotExist(err.(git.ErrNotExist).RelPath))

	case models.IsErrBranchAlreadyExists(err):
		log.Debug("Branch with name: %s already exist, error: %v", err.(models.ErrBranchAlreadyExists).BranchName, err)
		ctx.JSON(http.StatusBadRequest, apiError.BranchAlreadyExist(err.(models.ErrBranchAlreadyExists).BranchName))

	case models.IsErrFilenameInvalid(err):
		log.Debug("Filename: %s is invalid, error: %v", err.(models.ErrFilenameInvalid).Path, err)
		ctx.JSON(http.StatusBadRequest, apiError.InvalidFilename(err.(models.ErrFilenameInvalid).Path))

	case models.IsErrSHADoesNotMatch(err):
		log.Debug("SHA of file: %s does not match, given SHA: %s current SHA: %s, error: %v",
			err.(models.ErrSHADoesNotMatch).Path,
			err.(models.ErrSHADoesNotMatch).GivenSHA,
			err.(models.ErrSHADoesNotMatch).CurrentSHA,
			err,
		)
		ctx.JSON(http.StatusBadRequest, apiError.FileSHANotMatch(err.(models.ErrSHADoesNotMatch).Path))

	case models.IsErrCommitIDDoesNotMatch(err):
		log.Debug("Commit id does not match, error: %v", err)
		ctx.JSON(http.StatusBadRequest, apiError.CommitIDDoesNotMatch(err.(models.ErrCommitIDDoesNotMatch).GivenCommitID))

	case models.IsErrSHAOrCommitIDNotProvided(err):
		log.Debug("SHA or commit ID must be proved, error: %v", err)
		ctx.JSON(http.StatusBadRequest, apiError.SHAOrCommitIDNotProvided())

	case models.IsErrUserCannotCommit(err):
		log.Debug("User: %s cannot commit to repo: %s, error: %v", ctx.Doer.Name, ctx.Repo.Repository.Name, err)
		ctx.JSON(http.StatusForbidden, apiError.UserInsufficientPermission(ctx.Doer.Name, fmt.Sprintf("commit to repo: %s", ctx.Repo.Repository.Name)))

	case models.IsErrRepoFileDoesNotExist(err):
		log.Debug("File with name: %s, not found in path: %s, error: %v", err.(models.ErrRepoFileDoesNotExist).Name, err.(models.ErrRepoFileDoesNotExist).Path, err)
		ctx.JSON(http.StatusBadRequest, apiError.FileNotFound(fmt.Sprintf("%s/%s", err.(models.ErrRepoFileDoesNotExist).Name, err.(models.ErrRepoFileDoesNotExist).Path)))

	default:
		log.Error("An error has occurred while try to delete file, error: %v", err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
	}
}
