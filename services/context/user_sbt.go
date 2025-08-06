// Sbertech

package context

import (
	apiError "code.gitea.io/gitea/routers/sbt/apierror"
	"code.gitea.io/gitea/routers/sbt/logger"
	"net/http"
	"strings"

	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/context"
)

// UserAssignmentSbt добавляет в контекст запрашиваемого пользователя
func UserAssignmentSbt() func(ctx *context.Context) {
	return func(ctx *context.Context) {
		log := logger.Logger{}
		log.SetTraceId(ctx)

		ctx.ContextUser = userAssignmentSbt(ctx.Base, ctx.Doer, log)
	}
}

// userAssignmentSbt возвращает пользователя по имени из переменной пути
func userAssignmentSbt(ctx *context.Base, doer *user_model.User, log logger.Logger) (contextUser *user_model.User) {
	username := ctx.Params(":username")

	if doer != nil && doer.LowerName == strings.ToLower(username) {
		contextUser = doer
	} else {
		var err error
		contextUser, err = user_model.GetUserByName(ctx, username)
		if err != nil {
			log.Debug("User with name: %s is not found", username)

			ctx.JSON(http.StatusNotFound, apiError.UserNotFoundByNameError(username))

			return
		}
	}
	return contextUser
}

// RequireRepoOwner проверяет что пользователь является владельцем репозитория
func RequireRepoOwner() func(ctx *context.Context) {
	return func(ctx *context.Context) {
		log := logger.Logger{}
		log.SetTraceId(ctx)

		repoName := ctx.Params(":reponame")
		ownerName := ctx.Params(":username")

		// пользователь не является владельцем репозитория
		if strings.ToLower(ownerName) != ctx.Doer.LowerName {
			log.Debug("User: %s is not owner of repo: %s", ctx.Doer.Name, repoName)

			ctx.JSON(http.StatusForbidden, apiError.UserIsNotOwner())
			return
		}
	}
}
