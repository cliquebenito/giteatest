package repo

import (
	"code.gitea.io/gitea/models"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/git"
	api "code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/modules/web"
	apiError "code.gitea.io/gitea/routers/sbt/apierror"
	"code.gitea.io/gitea/routers/sbt/logger"
	"code.gitea.io/gitea/routers/sbt/request"
	files_service "code.gitea.io/gitea/services/repository/files"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"
)

// CreateFile создание файла в репозитории. Основа взята отсюда routers/api/v1/repo/file.go#CreateFile()
func CreateFile(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	fileOptions := web.GetForm(ctx).(*request.CreateFileOptions)

	if fileOptions.BranchName == "" {
		fileOptions.BranchName = ctx.Repo.Repository.DefaultBranch
	}

	opts := &files_service.UpdateRepoFileOptions{
		Content:   *fileOptions.Content,
		IsNewFile: true,
		Message:   fileOptions.Message,
		TreePath:  ctx.Params("*"),
		OldBranch: fileOptions.BranchName,
		NewBranch: fileOptions.NewBranchName,
		Committer: &files_service.IdentityOptions{
			Name:  fileOptions.Committer.Name,
			Email: fileOptions.Committer.Email,
		},
		Author: &files_service.IdentityOptions{
			Name:  fileOptions.Author.Name,
			Email: fileOptions.Author.Email,
		},
		Dates: &files_service.CommitDateOptions{
			Author:    fileOptions.Dates.Author,
			Committer: fileOptions.Dates.Committer,
		},
		Signoff: fileOptions.Signoff,
	}
	if opts.Dates.Author.IsZero() {
		opts.Dates.Author = time.Now()
	}
	if opts.Dates.Committer.IsZero() {
		opts.Dates.Committer = time.Now()
	}

	if opts.Message == "" {
		opts.Message = ctx.Tr("repo.editor.add", opts.TreePath)
	}

	if fileResponse, err := createOrUpdateFile(ctx, opts); err != nil {
		handleCreateOrUpdateFileError(ctx, err, log)
	} else {
		ctx.JSON(http.StatusOK, fileResponse)
	}
}

// создание или обновление файла
func createOrUpdateFile(ctx *context.Context, opts *files_service.UpdateRepoFileOptions) (*api.FileResponse, error) {
	content, err := base64.StdEncoding.DecodeString(opts.Content)
	if err != nil {
		return nil, err
	}
	opts.Content = string(content)

	return files_service.CreateOrUpdateRepoFile(ctx, ctx.Repo.Repository, ctx.Doer, opts)
}

// IsBase64CorruptInputError проверка на тип ошибки возникающей при декодировании содержимого создавемого файла из base64
func IsBase64CorruptInputError(err error) bool {
	_, ok := err.(base64.CorruptInputError)
	return ok
}

// handleCreateOrUpdateFileError обработка возможных ошибок при создании файла
func handleCreateOrUpdateFileError(ctx *context.Context, err error, log logger.Logger) {
	switch true {
	case IsBase64CorruptInputError(err):
		log.Debug("Error is occurred while decode content of file, error: %s", err.(base64.CorruptInputError).Error())
		ctx.JSON(http.StatusBadRequest, apiError.CorruptedFileContent())

	case models.IsErrUserCannotCommit(err):
		log.Debug("User: %s cannot commit to repo: %s, error: %v", ctx.Doer.Name, ctx.Repo.Repository.Name, err)
		ctx.JSON(http.StatusForbidden, apiError.UserInsufficientPermission(ctx.Doer.Name, fmt.Sprintf("commit to repo: %s", ctx.Repo.Repository.Name)))

	case models.IsErrFilePathProtected(err):
		log.Debug("User: %s cannot commit to protected file: %s, error: %v", ctx.Doer.Name, err.(models.ErrFilePathProtected).Path, err)
		ctx.JSON(http.StatusForbidden, apiError.UserInsufficientPermission(ctx.Doer.Name, "commit to protected file"))

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

	case models.IsErrFilePathInvalid(err):
		log.Debug("File path: %s is invalid, error: %v", err.(models.ErrFilePathInvalid).Path, err)
		ctx.JSON(http.StatusBadRequest, apiError.InvalidFilePath(err.(models.ErrFilePathInvalid).Path))

	case models.IsErrRepoFileAlreadyExists(err):
		log.Debug("File already exist in path: %s, error: %v", err.(models.ErrRepoFileAlreadyExists).Path, err)
		ctx.JSON(http.StatusBadRequest, apiError.FileAlreadyExist(err.(models.ErrRepoFileAlreadyExists).Path))

	case models.IsErrBranchDoesNotExist(err):
		log.Debug("Branch: %s not exist in repository: %s, error: %v", err.(models.ErrBranchDoesNotExist).BranchName, ctx.Repo.Repository.Name, err)
		ctx.JSON(http.StatusBadRequest, apiError.BranchNotExist(err.(models.ErrBranchDoesNotExist).BranchName))

	case git.IsErrBranchNotExist(err):
		log.Debug("Branch: %s not exist in repository: %s, error: %v", err.(git.ErrBranchNotExist).Name, ctx.Repo.Repository.Name, err)
		ctx.JSON(http.StatusBadRequest, apiError.BranchNotExist(err.(git.ErrBranchNotExist).Name))

	case git.IsErrNotExist(err):
		log.Debug("Git file: %s not exist in repository: %s, error: %v", err.(git.ErrNotExist).RelPath, ctx.Repo.Repository.Name, err)
		ctx.JSON(http.StatusBadRequest, apiError.FileNotFound(err.(git.ErrNotExist).RelPath))

	default:
		log.Error("Unknown error type has occurred, error: %v", err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
	}
}
