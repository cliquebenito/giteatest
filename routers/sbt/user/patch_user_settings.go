package user

import (
	"code.gitea.io/gitea/models/db"
	userModel "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/cache"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/modules/web"
	apiError "code.gitea.io/gitea/routers/sbt/apierror"
	sbtCache "code.gitea.io/gitea/routers/sbt/cache"
	"code.gitea.io/gitea/routers/sbt/logger"
	"code.gitea.io/gitea/routers/sbt/request"
	userService "code.gitea.io/gitea/services/user"
	"net/http"
	"strings"
)

/*
UpdateUserSettings метод обновления настроек пользователя
*/
func UpdateUserSettings(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	userSettings := web.GetForm(ctx).(*request.UserSettingsOptional)

	log.Debug("PATCH /user/settings request from username: %s with request body: %s", ctx.Doer.Name, userSettings.ToString())

	if userSettings.Name != nil && ctx.Doer.Name != *userSettings.Name {
		if err := handleUsernameChange(ctx, ctx.Doer, *userSettings.Name, log); err != nil {

			return
		}
		log.Debug("Username was changed from %s to %s", ctx.Doer.Name, userSettings.Name)

		ctx.Doer.Name = *userSettings.Name
		ctx.Doer.LowerName = strings.ToLower(*userSettings.Name)
	}
	if userSettings.FullName != nil {
		ctx.Doer.FullName = *userSettings.FullName
	}
	if userSettings.Description != nil {
		ctx.Doer.Description = *userSettings.Description
	}
	if userSettings.Website != nil {
		ctx.Doer.Website = *userSettings.Website
	}
	if userSettings.Location != nil {
		ctx.Doer.Location = *userSettings.Location
	}
	if userSettings.Visibility != nil {
		v, _ := structs.VisibilityModes[*userSettings.Visibility]
		ctx.Doer.Visibility = v
	}
	if userSettings.KeepEmailPrivate != nil {
		ctx.Doer.KeepEmailPrivate = *userSettings.KeepEmailPrivate
	}
	if userSettings.KeepActivityPrivate != nil {
		ctx.Doer.KeepActivityPrivate = *userSettings.KeepActivityPrivate
	}

	if err := userModel.UpdateUser(ctx, ctx.Doer, false); err != nil {
		log.Error("Unknown error type has occurred: %v", err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())

		return
	}

	ctx.JSON(http.StatusOK, convertUserToUserSettings(ctx.Doer))

	cache.RemoveItem(sbtCache.GenerateUserKey(ctx.Doer.Name) + "*")
}

/*
handleUsernameChange - метод обновления имени пользователя
*/
func handleUsernameChange(ctx *context.Context, user *userModel.User, newName string, log logger.Logger) error {
	if err := userService.RenameUser(ctx, user, newName); err != nil {
		switch {
		case userModel.IsErrUserAlreadyExist(err):
			log.Debug("User with userName: %s can't change username because username: %s already exist", user.Name, newName)
			ctx.JSON(http.StatusBadRequest, apiError.UserNameAlreadyExistError(newName))

		case userModel.IsErrEmailAlreadyUsed(err):
			log.Debug("User with userName: %s can't change username because email: %s already exist", user.Name, user.Email)
			ctx.JSON(http.StatusBadRequest, apiError.EmailAlreadyExistError(user.Email))

		case db.IsErrNameReserved(err):
			log.Debug("User with userName: %s can't change username because username: %s is reserved", user.Name, newName)
			ctx.JSON(http.StatusBadRequest, apiError.UserNameReservedError(newName))

		case db.IsErrNamePatternNotAllowed(err):
			log.Debug("User with userName: %s can't change username because username: %s pattern is not allowed", user.Name, newName)
			ctx.JSON(http.StatusBadRequest, apiError.UserNamePatternNotAllowedError(newName))

		case db.IsErrNameCharsNotAllowed(err):
			log.Debug("User with userName: %s can't change username because username: %s has not allowed characters", user.Name, newName)
			ctx.JSON(http.StatusBadRequest, apiError.UserNameHasNotAllowedCharsError(newName))

		default:
			log.Error("While user with username: %s changing username unknown error type has occurred: %v", err)
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		}
		return err
	}

	return nil
}
