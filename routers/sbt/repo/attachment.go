package repo

import (
	apiError "code.gitea.io/gitea/routers/sbt/apierror"
	"code.gitea.io/gitea/routers/sbt/logger"
	"code.gitea.io/gitea/routers/sbt/response"
	"net/http"

	accessModel "code.gitea.io/gitea/models/perm/access"
	repoModel "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/httpcache"

	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/storage"
	"code.gitea.io/gitea/modules/upload"
	"code.gitea.io/gitea/routers/common"
	"code.gitea.io/gitea/services/attachment"
	repoService "code.gitea.io/gitea/services/repository"
)

// UploadIssueAttachment загрузить файл
func UploadIssueAttachment(ctx *context.Context) {
	uploadAttachment(ctx, ctx.Repo.Repository.ID, setting.Attachment.AllowedTypes)
}

// uploadAttachment загрузить файл с учетом ограничений
func uploadAttachment(ctx *context.Context, repoID int64, allowedTypes string) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	if !setting.Attachment.Enabled {
		log.Debug("Attachments are not allowed in system settings")
		ctx.JSON(http.StatusForbidden, apiError.AttachmentsNotAllowed())
		return
	}

	file, header, err := ctx.Req.FormFile("file")
	if err != nil {
		log.Error("Not able to get form file for repoId: %d, error: %v", repoID, err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		return
	}
	defer file.Close()

	attach, err := attachment.UploadAttachment(file, allowedTypes, header.Size, &repoModel.Attachment{
		Name:       header.Filename,
		UploaderID: ctx.Doer.ID,
		RepoID:     repoID,
	})
	if err != nil {
		if upload.IsErrFileTypeForbidden(err) {
			log.Debug("Attachment type is not allowed")
			ctx.JSON(http.StatusBadRequest, apiError.FileTypeNotAllowed())
			return
		}
		log.Error("Not able to save attached file for repoId: %d, error: %v", repoID, err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		return
	}

	ctx.JSON(http.StatusOK, response.AttachmentUuid{UUID: attach.UUID})
}

// DeleteAttachment удалить приложенный файл
func DeleteAttachment(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	uuid := ctx.Params(":uuid")
	attach, err := repoModel.GetAttachmentByUUID(ctx, uuid)
	if err != nil {
		if repoModel.IsErrAttachmentNotExist(err) {
			log.Debug("Attachment file uuid: %d is not found", uuid)
			ctx.JSON(http.StatusBadRequest, apiError.AttachmentNotFound())
		} else {
			log.Error("Not able to get attachment file uuid: %s, error: %v", uuid, err)
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		}
		return
	}
	if !ctx.IsSigned || (ctx.Doer.ID != attach.UploaderID) {
		log.Debug("Not enough rights to delete attachment file uuid: %d is not found", uuid)
		ctx.JSON(http.StatusBadRequest, apiError.UserUnauthorized())
		return
	}
	err = repoModel.DeleteAttachment(attach, true)
	if err != nil {
		log.Error("Not able to delete attachment file uuid: %s, error: %v", uuid, err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		return
	}
	ctx.Status(http.StatusOK)
}

// ServeAttachment скачивает файл по его UUID
func ServeAttachment(ctx *context.Context, uuid string) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	attach, err := repoModel.GetAttachmentByUUID(ctx, uuid)
	if err != nil {
		if repoModel.IsErrAttachmentNotExist(err) {
			log.Debug("Attachment file uuid: %d is not found", uuid)
			ctx.JSON(http.StatusBadRequest, apiError.AttachmentNotFound())
		} else {
			log.Error("Not able to get attachment file uuidL %s, error: %v", uuid, err)
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		}
		return
	}

	repository, unitType, err := repoService.LinkedRepository(ctx, attach)
	if err != nil {
		log.Error("Not able to get attachment file's linked repository, file uuid: %s, error: %v", uuid, err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		return
	}

	if repository == nil { // Если не связан с репозиторием
		if !(ctx.IsSigned && attach.UploaderID == ctx.Doer.ID) { // Блокируем если не тот кто загрузил
			log.Debug("Attachment file uuid: %d is not allowed to download", uuid)
			ctx.JSON(http.StatusBadRequest, apiError.AttachmentNotFound())
			return
		}
	} else { // If we have the repository we check access
		context.CheckRepoScopedToken(ctx, repository)
		if ctx.Written() {
			return
		}

		perm, err := accessModel.GetUserRepoPermission(ctx, repository, ctx.Doer)
		if err != nil {
			log.Error("Not able to get attachment file's permissions file uuid: %s, error: %v", uuid, err)
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
			return
		}
		if !perm.CanRead(unitType) {
			log.Debug("Attachment file uuid: %d is not allowed to download", uuid)
			ctx.JSON(http.StatusBadRequest, apiError.AttachmentNotFound())
			return
		}
	}

	if err := attach.IncreaseDownloadCount(); err != nil {
		log.Error("Not able to increase attachment file's download count, file uuid: %s, error: %v", uuid, err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		return
	}

	if setting.Attachment.ServeDirect {
		// Редиректим в случае внешнего хранилища (S3, object storage)
		u, err := storage.Attachments.URL(attach.RelativePath(), attach.Name)
		if err != nil {
			log.Error("Not able to redirect download link to external storage, error: %v", err)
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
			return
		}
		if u != nil && err == nil {
			ctx.Redirect(u.String())
			return
		}
	}

	if httpcache.HandleGenericETagCache(ctx.Req, ctx.Resp, `"`+attach.UUID+`"`) {
		return
	}

	fr, err := storage.Attachments.Open(attach.RelativePath())
	if err != nil {
		log.Error("Not able to read attachment file uuid: %s, error: %v", uuid, err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		return
	}
	defer fr.Close()

	common.ServeContentByReadSeeker(ctx.Base, attach.Name, attach.CreatedUnix.AsTime(), fr)
}

// GetAttachment скачать вложенный файл
func GetAttachment(ctx *context.Context) {
	ServeAttachment(ctx, ctx.Params(":uuid"))
}
