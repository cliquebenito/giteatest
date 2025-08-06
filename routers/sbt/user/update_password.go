package user

import (
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/sbt/auth/password"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/web"
	apiError "code.gitea.io/gitea/routers/sbt/apierror"
	"code.gitea.io/gitea/routers/sbt/logger"
	"code.gitea.io/gitea/routers/sbt/request"
	"fmt"
	"net/http"
)

// ChangePassword метод смены пароля от аккаунта в настройках пользователя
func ChangePassword(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	req := web.GetForm(ctx).(*request.ChangePassword)

	if len(req.NewPassword) < setting.MinPasswordLength {
		log.Debug("Short new password, required password length: %d", setting.MinPasswordLength)
		ctx.JSON(http.StatusBadRequest, apiError.WrongPasswordError(fmt.Sprintf("%d or more symbols", setting.MinPasswordLength)))

		return
	}

	if ok, errorMessage := password.IsComplexEnough(req.NewPassword); !ok {
		log.Debug("Wrong password in request, required %s", errorMessage)
		ctx.JSON(http.StatusBadRequest, apiError.WrongPasswordError(errorMessage))

		return
	}

	if ctx.Doer.IsPasswordSet() && !ctx.Doer.ValidatePassword(req.OldPassword) {
		log.Debug("While changing password for username: %s, old password was not validated", ctx.Doer.Name)
		ctx.JSON(http.StatusBadRequest, apiError.LoginOrPasswordNotValidError())

		return
	}

	if err := ctx.Doer.SetPassword(req.NewPassword); err != nil {
		log.Error("Internal server error has occurred while changing password, error message: %v", err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())

		return
	}

	if err := user_model.UpdateUserCols(ctx, ctx.Doer, "salt", "passwd_hash_algo", "passwd"); err != nil {
		log.Error("Internal server error has occurred while changing password in data base, error message: %v", err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())

		return
	}

	ctx.Status(http.StatusOK)
}
