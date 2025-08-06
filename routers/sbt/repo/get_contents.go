package repo

import (
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/git"
	apiError "code.gitea.io/gitea/routers/sbt/apierror"
	"code.gitea.io/gitea/routers/sbt/logger"
	files_service "code.gitea.io/gitea/services/repository/files"
	"net/http"
)

// GetContents Получение метаданных о файле (если запрошен файл) или списка содержимого если запрошена директория
func GetContents(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	treePath := ctx.Params("*")
	ref := ctx.FormTrim("ref")

	if fileList, err := files_service.GetContentsOrListSbt(ctx, ctx.Repo.Repository, treePath, ref); err != nil {
		if git.IsErrNotExist(err) {
			log.Debug("File or dir with name: %s and ref: %s contents not found", treePath, ref)
			ctx.JSON(http.StatusBadRequest, apiError.FileNotFound(treePath))

			return
		}
		log.Error("Error has occurred while getting contents of file or dir with name: %s, ref: %s, error: %v", treePath, ref, err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
	} else {

		ctx.JSON(http.StatusOK, fileList)
	}
}

// GetRootContentsList Получение содержимого корня репозитория
func GetRootContentsList(ctx *context.Context) {
	GetContents(ctx)
}
