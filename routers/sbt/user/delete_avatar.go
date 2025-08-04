package user

import (
	"code.gitea.io/gitea/modules/context"
	apiError "code.gitea.io/gitea/routers/sbt/apierror"
	"code.gitea.io/gitea/routers/sbt/logger"
	user_service "code.gitea.io/gitea/services/user"
	"net/http"
)

// DeleteAvatar метод удаления аватара. После удаления аватар генерируется граватаром
func DeleteAvatar(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	err := user_service.DeleteAvatar(ctx.Doer)
	if err != nil {
		log.Error("Unknown error has occurred while deleting avatar for username: %s. Error: %v", ctx.Doer.Name, err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		return
	}

	ctx.Status(http.StatusOK)
}
