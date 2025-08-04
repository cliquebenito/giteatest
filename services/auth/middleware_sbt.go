package auth

import (
	"code.gitea.io/gitea/models/unit"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/sbt/audit"
	"code.gitea.io/gitea/modules/setting"
	apiError "code.gitea.io/gitea/routers/sbt/apierror"
	"net/http"
)

// VerifyAuthWithOptionsSbt проверяет аутентификацию в зависимости от переданных опций
func VerifyAuthWithOptionsSbt(options *VerifyOptions) func(ctx *context.Context) {
	return func(ctx *context.Context) {
		// Разрешены ли операции.
		if ctx.IsSigned {
			if !ctx.Doer.IsActive && setting.Service.RegisterEmailConfirm {
				log.Debug("Not activated user: %s login request was rejected", ctx.Doer.Name)
				ctx.JSON(http.StatusBadRequest, apiError.UserNotActivatedError())

				return
			}

			if !ctx.Doer.IsActive || ctx.Doer.ProhibitLogin {
				log.Debug("Inactive or prohibited user: %s 's request was rejected", ctx.Doer.Name)
				ctx.JSON(http.StatusBadRequest, apiError.UserProhibitedLoginError())

				return
			}

			if ctx.Doer.MustChangePassword {
				if ctx.Req.URL.Path != "/user/password" {
					log.Debug("User: %s must change password", ctx.Doer.Name)
					ctx.JSON(http.StatusBadRequest, apiError.UserMustChangePasswordError())

					return
				}
			}
		}

		if options.SignInRequired {
			if !ctx.IsSigned {
				log.Debug("Unauthorized request to path %s from address %s was received", ctx.Req.URL.Path, ctx.RemoteAddr())
				ctx.JSON(http.StatusUnauthorized, apiError.UserUnauthorized())
				auditParams := map[string]string{
					"request_url": ctx.Req.URL.RequestURI(),
				}
				audit.CreateAndSendEvent(audit.UnauthorizedRequestEvent, audit.EmptyRequiredField, audit.EmptyRequiredField, audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
				return
			} else if !ctx.Doer.IsActive && setting.Service.RegisterEmailConfirm {
				log.Debug("Not activated user: %s login request was rejected", ctx.Doer.Name)
				ctx.JSON(http.StatusBadRequest, apiError.UserNotActivatedError())

				return
			}
		}

		if options.AdminRequired {
			if !ctx.Doer.IsAdmin {
				log.Debug("User: %s has not admin privileges", ctx.Doer.Name)
				ctx.JSON(http.StatusBadRequest, apiError.UserNotAdminError())

				return
			}
		}
	}
}

// RequireRepoAdmin Проверяет пользователя на права администратора репозитория
func RequireRepoAdmin() func(ctx *context.Context) {
	return func(ctx *context.Context) {

		if !ctx.Repo.IsAdmin() {
			log.Debug("User: %s is not admin of repo: %s", ctx.Doer.Name, ctx.Repo.Repository.Name)
			ctx.JSON(http.StatusBadRequest, apiError.UserIsNotRepoAdmin())
			return
		}
	}
}

// RequireRepoEditor Проверяет полномочия пользователя на редактирование файла в ветке
func RequireRepoEditor() func(ctx *context.Context) {
	return func(ctx *context.Context) {
		if !ctx.Repo.CanWriteToBranch(ctx.Doer, ctx.Repo.BranchName) {
			log.Debug("User with userId: %d have not permission to edit repoId: %d", ctx.Doer.ID, ctx.Repo.Repository.ID)
			ctx.JSON(http.StatusBadRequest, apiError.UserInsufficientPermission(ctx.Doer.Name, "edit repo"))
			return
		}
	}
}

// RequireRepoWriter проверяет полномочия пользователя на запись по типу Юнита
func RequireRepoWriter(unitType unit.Type) func(ctx *context.Context) {
	return func(ctx *context.Context) {
		if !ctx.Repo.CanWrite(unitType) {
			log.Debug("User with userId: %d have not permission to write in repo with repoId: %d", ctx.Doer.ID, ctx.Repo.Repository.ID)
			ctx.JSON(http.StatusBadRequest, apiError.UserInsufficientPermission(ctx.Doer.Name, "write repo"))
			return
		}
	}
}
