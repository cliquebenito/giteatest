package user

import (
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/routers/sbt/response"
	"net/http"
)

/*
GetUserSettings - метод, возвращающий текущие настройки пользователя
*/
func GetUserSettings(ctx *context.Context) {
	ctx.JSON(http.StatusOK, convertUserToUserSettings(ctx.Doer))
}

func convertUserToUserSettings(user *user_model.User) response.UserSettings {
	return response.UserSettings{
		Name:                user.Name,
		FullName:            user.FullName,
		Website:             user.Website,
		Location:            user.Location,
		Description:         user.Description,
		KeepEmailPrivate:    user.KeepEmailPrivate,
		KeepActivityPrivate: user.KeepActivityPrivate,
		Visibility:          user.Visibility.String(),
	}
}
