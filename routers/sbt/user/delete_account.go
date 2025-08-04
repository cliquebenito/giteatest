package user

import (
	"code.gitea.io/gitea/models"
	userModel "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/cache"
	ctx "code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/web"
	apiError "code.gitea.io/gitea/routers/sbt/apierror"
	sbtCache "code.gitea.io/gitea/routers/sbt/cache"
	"code.gitea.io/gitea/routers/sbt/logger"
	"code.gitea.io/gitea/routers/sbt/request"
	"code.gitea.io/gitea/services/auth"
	"code.gitea.io/gitea/services/user"
	"net/http"
)

/*
DeleteAccountUser метод удаления аккаунта пользователя
*/
func DeleteAccountUser(ctx *ctx.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	req := web.GetForm(ctx).(*request.DeleteUserAccount)

	if _, _, err := auth.UserSignIn(ctx.Doer.Name, req.Password); err != nil {
		if userModel.IsErrUserNotExist(err) || userModel.IsErrEmailAddressNotExist(err) {
			log.Debug("While deleting user account failed authentication attempt user with username: %s", ctx.Doer.Name)
			ctx.JSON(http.StatusBadRequest, apiError.LoginOrPasswordNotValidError())

		} else {
			log.Error("Internal server error has occurred while user authentication before deletion account, error message: %v", err)
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		}
		return
	}

	if err := user.DeleteUser(ctx, ctx.Doer, false); err != nil {
		switch {
		case models.IsErrUserOwnRepos(err):
			log.Debug("While deleting user account username: %s error has occurred: user owns of repositories", ctx.Doer.Name)
			ctx.JSON(http.StatusBadRequest, apiError.UserHasReposError())

		case models.IsErrUserHasOrgs(err):
			log.Debug("While deleting user account username: %s error has occurred: user is member of organisations", ctx.Doer.Name)
			ctx.JSON(http.StatusBadRequest, apiError.UserHasOrgsError())

		case models.IsErrUserOwnPackages(err):
			log.Debug("While deleting user account username: %s error has occurred: user owns of packages", ctx.Doer.Name)
			ctx.JSON(http.StatusBadRequest, apiError.UserHasPackagesError())

		default:
			log.Error("Internal server error has occurred while deleting username: %s account, error message: %v", ctx.Doer.Name, err)
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())

		}
	} else {

		ctx.Status(http.StatusOK)

		cache.RemoveItem(sbtCache.GenerateUserKey(ctx.Doer.Name) + "*")
	}
}
