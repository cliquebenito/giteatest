package user

import (
	userModel "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/web"
	apiError "code.gitea.io/gitea/routers/sbt/apierror"
	"code.gitea.io/gitea/routers/sbt/logger"
	"code.gitea.io/gitea/routers/sbt/request"
	authService "code.gitea.io/gitea/services/auth"
	"net/http"
)

/*
AuthUser метод авторизации пользователя.
В случае ошибки возвращается Unauthorized (401),
в случае успешного создания пользователя Ok (200)
*/
func AuthUser(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	req := web.GetForm(ctx).(*request.SignIn)

	u, err := authService.SbtSignInUser(req.Login, req.Password, ctx)
	if err != nil {
		log.Error("Failed authentication attempt for %s from %s. Error: %v", req.Login, ctx.RemoteAddr(), err)

		if userModel.IsErrUserNotExist(err) || userModel.IsErrEmailAddressNotExist(err) {
			ctx.JSON(http.StatusBadRequest, apiError.LoginOrPasswordNotValidError())
		} else if userModel.IsErrUserProhibitLogin(err) {
			ctx.JSON(http.StatusBadRequest, apiError.UserProhibitedLoginError())
		} else if userModel.IsErrUserInactive(err) {
			if setting.Service.RegisterEmailConfirm {
				ctx.JSON(http.StatusBadRequest, apiError.UserEmailNotConfirmedError())
			} else {
				ctx.JSON(http.StatusBadRequest, apiError.UserProhibitedLoginError())
			}
		} else if userModel.IsErrKeycloakWrongHttpRequest(err) || userModel.IsErrKeycloakWrongHttpStatus(err) {
			ctx.JSON(http.StatusBadRequest, apiError.KeycloakUserWasNotAuthenticate)
		} else {
			log.Error("Internal server error has occurred, unknown error type: %v", err)

			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		}
		return
	}

	handleSignIn(ctx, u, log)

	ctx.Status(http.StatusOK)
}
