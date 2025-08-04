package user

import (
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/context"
	apiError "code.gitea.io/gitea/routers/sbt/apierror"
	"code.gitea.io/gitea/routers/sbt/logger"
	"code.gitea.io/gitea/services/convert"
	"net/http"
)

// GetAuthenticatedUser получить данные текущего (совершившего вход в систему) пользователя
func GetAuthenticatedUser(ctx *context.Context) {
	ctx.JSON(http.StatusOK, convert.ToUser(ctx, ctx.Doer, ctx.Doer))
}

// GetUserInfo получает данные пользователя согласно настойкам видимости
func GetUserInfo(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	if !user_model.IsUserVisibleToViewer(ctx, ctx.ContextUser, ctx.Doer) {
		// в случае наличия настроек приватности сообщение об ошибке фейковое, что бы избежать утечки данных
		log.Error("User with name: %s found, but it is invisible", ctx.ContextUser.LowerName)

		ctx.JSON(http.StatusNotFound, apiError.UserNotFoundByNameError(ctx.ContextUser.LowerName))
		return
	}

	ctx.JSON(http.StatusOK, convert.ToUser(ctx, ctx.ContextUser, ctx.Doer))
}
