package user

import (
	"bytes"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/web"
	apiError "code.gitea.io/gitea/routers/sbt/apierror"
	"code.gitea.io/gitea/routers/sbt/logger"
	"code.gitea.io/gitea/routers/sbt/request"
	user_service "code.gitea.io/gitea/services/user"
	"encoding/base64"
	"image"
	"net/http"
)

// UpdateAvatar метод смены аватара пользователя с предварительной проверкой размера
func UpdateAvatar(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	form := web.GetForm(ctx).(*request.UserAvatar)

	content, err := base64.StdEncoding.DecodeString(form.Image)
	if err != nil {
		log.Debug("Can not decode string base64 while updating avatar for username: %s", ctx.Doer.Name)
		ctx.JSON(http.StatusBadRequest, apiError.DecodeBase64Error())
		return
	}

	imgCfg, _, err := image.DecodeConfig(bytes.NewReader(content))
	if err != nil {
		log.Debug("Can not decode image's config while updating avatar for username: %s", ctx.Doer.Name)
		ctx.JSON(http.StatusBadRequest, apiError.DecodeImageConfigError())
		return
	}

	if imgCfg.Width > setting.Avatar.MaxWidth || imgCfg.Height > setting.Avatar.MaxHeight {
		log.Debug("Not valid avatar size: %s*%s for username: %s", imgCfg.Height, imgCfg.Width, ctx.Doer.Name)
		ctx.JSON(http.StatusBadRequest, apiError.NotValidImageSize(setting.Avatar.MaxHeight, setting.Avatar.MaxWidth))
		return
	}

	err = user_service.UploadAvatar(ctx.Doer, content)
	if err != nil {
		log.Error("Unknown error has occurred while updating avatar for username: %s. Error: %v", ctx.Doer.Name, err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		return
	}

	ctx.Status(http.StatusOK)
}
