package orgs

import (
	"code.gitea.io/gitea/modules/context"
	apiError "code.gitea.io/gitea/routers/sbt/apierror"
	"code.gitea.io/gitea/routers/sbt/logger"
	userService "code.gitea.io/gitea/services/user"
	"net/http"
)

// DeleteAvatar метод удаления аватара организации
func DeleteAvatar(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	err := userService.DeleteAvatar(ctx.Org.Organization.AsUser())
	if err != nil {
		log.Error("Unknown error has occurred while deleting avatar for orgId: %s. Error: %v", ctx.Org.Organization.ID, err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		return
	}

	ctx.Status(http.StatusOK)
}
