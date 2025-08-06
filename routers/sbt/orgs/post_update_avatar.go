package orgs

import (
	"bytes"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/web"
	apiError "code.gitea.io/gitea/routers/sbt/apierror"
	"code.gitea.io/gitea/routers/sbt/logger"
	"code.gitea.io/gitea/routers/sbt/request"
	userService "code.gitea.io/gitea/services/user"
	"encoding/base64"
	"image"
	"net/http"
)

// UpdateAvatar метод обновления аватара организации
func UpdateAvatar(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	req := web.GetForm(ctx).(*request.OrganizationAvatar)

	content, err := base64.StdEncoding.DecodeString(req.Image)
	if err != nil {
		log.Debug("Can not decode string base64 while updating avatar for orgId: %d", ctx.Org.Organization.ID)
		ctx.JSON(http.StatusBadRequest, apiError.DecodeBase64Error())
		return
	}

	imgCfg, _, err := image.DecodeConfig(bytes.NewReader(content))
	if err != nil {
		log.Debug("Can not decode image's config while updating avatar for orgId: %d", ctx.Org.Organization.ID)
		ctx.JSON(http.StatusBadRequest, apiError.DecodeImageConfigError())
		return
	}

	if imgCfg.Width > setting.Avatar.MaxWidth || imgCfg.Height > setting.Avatar.MaxHeight {
		log.Debug("Not valid avatar size: %s*%s for orgId: %d", imgCfg.Height, imgCfg.Width, ctx.Org.Organization.ID)
		ctx.JSON(http.StatusBadRequest, apiError.NotValidImageSize(setting.Avatar.MaxHeight, setting.Avatar.MaxWidth))
		return
	}

	err = userService.UploadAvatar(ctx.Org.Organization.AsUser(), content)
	if err != nil {
		log.Error("Unknown error has occurred while updating avatar for orgId: %d. Error: %v", ctx.Org.Organization.ID, err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		return
	}

	ctx.Status(http.StatusOK)
}
