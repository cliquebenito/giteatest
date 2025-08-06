package user

import (
	"net/http"
	"strconv"

	"code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/sbt/audit"
	"code.gitea.io/gitea/modules/setting"
	apiError "code.gitea.io/gitea/routers/sbt/apierror"
	"code.gitea.io/gitea/routers/sbt/logger"
)

/*
LogoutUser метод выхода пользователя из своей учетной записи, возвращает статус OK(200)
*/
func LogoutUser(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)
	var doerName, doerEmail, doerID, remoteAddress string

	if ctx.Doer != nil {
		doerEmail = ctx.Doer.Email
		doerName = ctx.Doer.Name
		doerID = strconv.FormatInt(ctx.Doer.ID, 10)
	} else {
		doerName = audit.EmptyRequiredField
		doerID = audit.EmptyRequiredField
		doerEmail = audit.EmptyRequiredField
	}

	if ctx.Req != nil {
		remoteAddress = ctx.Req.RemoteAddr
	} else {
		remoteAddress = audit.EmptyRequiredField
	}

	auditParams := map[string]string{
		"email": doerEmail,
	}

	if setting.SbtKeycloakForm.Enabled && ctx.Doer != nil {
		if token := ctx.Session.Get(user.RefTokenKey); token != nil {
			if err := user.KeycloakLogoutSession(token.(string), ctx.Doer.LowerName); err != nil {
				log.Debug("Keycloak session was not logout. Error: %v", err)
				ctx.JSON(http.StatusBadRequest, apiError.KeycloakUserWasNotLogout)
				auditParams["error"] = "Error occurred while logging out of the KeyCloak session"
				audit.CreateAndSendEvent(audit.UserLogoutEvent, doerName, doerID, audit.StatusFailure, remoteAddress, auditParams)

				return
			}
			log.Debug("Keycloak session was closed for username: %s", ctx.Doer.LowerName)
		}
	}
	_ = ctx.Session.Flush()
	_ = ctx.Session.Destroy(ctx.Resp, ctx.Req)
	ctx.DeleteSiteCookie(setting.SessionConfig.CookieName)
	ctx.Csrf.DeleteCookie(ctx)
	audit.CreateAndSendEvent(audit.UserLogoutEvent, doerName, doerID, audit.StatusSuccess, remoteAddress, auditParams)

	ctx.Status(http.StatusOK)
}
