// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package auth

import (
	"net/http"
	"reflect"
	"strings"

	"code.gitea.io/gitea/models/auth"
	"code.gitea.io/gitea/models/organization"
	"code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/role_model"
	"code.gitea.io/gitea/models/tenant"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/sbt/audit"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/web/middleware"
)

// Auth is a middleware to authenticate a web user
func Auth(authMethod Method) func(*context.Context) {
	if !setting.IAM.Enabled {
		return basicAuth(authMethod)
	}

	return iamProxyAuth(authMethod)
}

func basicAuth(authMethod Method) func(ctx *context.Context) {
	return func(ctx *context.Context) {
		ar, err := authShared(ctx.Base, ctx.Session, authMethod)
		if err != nil {
			log.Error("Failed to verify user: %v", err)
			ctx.Error(http.StatusUnauthorized, "Verify")
			audit.CreateAndSendEvent(audit.UnauthorizedRequestEvent, audit.EmptyRequiredField, audit.EmptyRequiredField, audit.StatusFailure, audit.EmptyRequiredField, nil)
			return
		}
		ctx.Doer = ar.Doer
		ctx.IsSigned = ar.Doer != nil
		ctx.IsBasicAuth = ar.IsBasicAuth
		if ctx.Doer == nil {
			// ensure the session uid is deleted
			_ = ctx.Session.Delete("uid")
		}
	}
}

// APIAuth is a middleware to authenticate an api user
func APIAuth(authMethod Method) func(*context.APIContext) {
	return func(ctx *context.APIContext) {
		ar, err := authShared(ctx.Base, nil, authMethod)
		if err != nil {
			ctx.Error(http.StatusUnauthorized, "APIAuth", err)
			audit.CreateAndSendEvent(audit.UnauthorizedRequestEvent, audit.EmptyRequiredField, audit.EmptyRequiredField, audit.StatusFailure, audit.EmptyRequiredField, nil)
			return
		}
		ctx.Doer = ar.Doer
		ctx.IsSigned = ar.Doer != nil
		ctx.IsBasicAuth = ar.IsBasicAuth
	}
}

type authResult struct {
	Doer        *user_model.User
	IsBasicAuth bool
}

func authShared(ctx *context.Base, sessionStore SessionStore, authMethod Method) (ar authResult, err error) {
	ar.Doer, err = authMethod.Verify(ctx.Req, ctx.Resp, ctx, sessionStore)
	if err != nil {
		return ar, err
	}
	if ar.Doer != nil {
		if ctx.Locale.Language() != ar.Doer.Language {
			ctx.Locale = middleware.Locale(ctx.Resp, ctx.Req)
		}
		ar.IsBasicAuth = ctx.Data["AuthedMethod"].(string) == BasicMethodName

		ctx.Data["IsSigned"] = true
		ctx.Data[middleware.ContextDataKeySignedUser] = ar.Doer
		ctx.Data["SignedUserID"] = ar.Doer.ID
		ctx.Data["IsAdmin"] = ar.Doer.IsAdmin
	} else {
		ctx.Data["SignedUserID"] = int64(0)
	}
	return ar, nil
}

// VerifyOptions contains required or check options
type VerifyOptions struct {
	SignInRequired  bool
	SignOutRequired bool
	AdminRequired   bool
	DisableCSRF     bool
}

// VerifyAuthWithOptions checks authentication according to options
func VerifyAuthWithOptions(options *VerifyOptions) func(ctx *context.Context) {
	return func(ctx *context.Context) {
		// Check prohibit login users.
		if ctx.IsSigned {
			if !ctx.Doer.IsActive && setting.Service.RegisterEmailConfirm {
				ctx.Data["Title"] = ctx.Tr("auth.active_your_account")
				ctx.HTML(http.StatusOK, "user/auth/activate")
				return
			}
			if !ctx.Doer.IsActive || ctx.Doer.ProhibitLogin {
				log.Info("Failed authentication attempt for %s from %s", ctx.Doer.Name, ctx.RemoteAddr())
				ctx.Data["Title"] = ctx.Tr("auth.prohibit_login")
				ctx.HTML(http.StatusOK, "user/auth/prohibit_login")
				return
			}

			if ctx.Doer.MustChangePassword {
				if ctx.Req.URL.Path != "/user/settings/change_password" {
					if strings.HasPrefix(ctx.Req.UserAgent(), "git") {
						ctx.Error(http.StatusUnauthorized, ctx.Tr("auth.must_change_password"))
						auditParams := map[string]string{
							"request_url": ctx.Req.URL.RequestURI(),
						}
						audit.CreateAndSendEvent(audit.UnauthorizedRequestEvent, audit.EmptyRequiredField, audit.EmptyRequiredField, audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
						return
					}
					ctx.Data["Title"] = ctx.Tr("auth.must_change_password")
					ctx.Data["ChangePasscodeLink"] = setting.AppSubURL + "/user/change_password"
					if ctx.Req.URL.Path != "/user/events" {
						middleware.SetRedirectToCookie(ctx.Resp, setting.AppSubURL+ctx.Req.URL.RequestURI())
					}
					ctx.Redirect(setting.AppSubURL + "/user/settings/change_password")
					return
				}
			} else if ctx.Req.URL.Path == "/user/settings/change_password" {
				// make sure that the form cannot be accessed by users who don't need this
				ctx.Redirect(setting.AppSubURL + "/")
				return
			}
		}

		// Redirect to dashboard if user tries to visit any non-login page.
		if options.SignOutRequired && ctx.IsSigned && ctx.Req.URL.RequestURI() != "/" {
			ctx.Redirect(setting.AppSubURL + "/")
			return
		}

		if !options.SignOutRequired && !options.DisableCSRF && ctx.Req.Method == "POST" {
			ctx.Csrf.Validate(ctx)
			if ctx.Written() {
				return
			}
		}

		if options.SignInRequired {
			if !ctx.IsSigned {
				if ctx.Req.URL.Path != "/user/events" {
					middleware.SetRedirectToCookie(ctx.Resp, setting.AppSubURL+ctx.Req.URL.RequestURI())
				}
				ctx.Redirect(setting.AppSubURL + "/user/login")
				return
			} else if !ctx.Doer.IsActive && setting.Service.RegisterEmailConfirm {
				ctx.Data["Title"] = ctx.Tr("auth.active_your_account")
				ctx.HTML(http.StatusOK, "user/auth/activate")
				return
			}
		}

		// Redirect to log in page if auto-signin info is provided and has not signed in.
		if !options.SignOutRequired && !ctx.IsSigned &&
			len(ctx.GetSiteCookie(setting.CookieUserName)) > 0 {
			if ctx.Req.URL.Path != "/user/events" {
				middleware.SetRedirectToCookie(ctx.Resp, setting.AppSubURL+ctx.Req.URL.RequestURI())
			}
			ctx.Redirect(setting.AppSubURL + "/user/login")
			return
		}

		if options.AdminRequired {
			if !ctx.Doer.IsAdmin {
				ctx.NotFound("", nil)
				return
			}
			ctx.Data["PageIsAdmin"] = true
		}
	}
}

// CheckPermissionUserMultiTenant метод необходимый для фильтрации запросов по мультитенантности.
func CheckPermissionUserMultiTenant(options *VerifyOptions) func(ctx *context.Context) {
	return func(ctx *context.Context) {
		// если в запросе нет объекта пользователя, дальше не идем.
		if ctx.Doer == nil {
			return
		}

		// если CS выключен или мильтитенантность выключена или пользователь - админ - нет необходимости использовать.
		if !setting.SourceControl.Enabled || !setting.SourceControl.MultiTenantEnabled || ctx.Doer.IsAdmin {
			return
		}

		// Находим тенанта, для пользователя делающего запрос.
		doerTenantIDs, err := role_model.GetUserTenantIDsOrDefaultTenantID(ctx.Doer)
		if err != nil {
			return
		}
		// Поскольку у пользователя может быть несколько тенантов.
		targetTenantIDs := make(map[string]struct{})
		// Структурируем.
		for _, doerTenantID := range doerTenantIDs {
			targetTenantIDs[doerTenantID] = struct{}{}
		}

		//context.RepoAssignment(ctx)
		//context_service.UserAssignmentWeb()(ctx)

		if !matchTenant(map[string][]string{
			"ContextUser": foundTenant(ctx, ctx.ContextUser),
			"Repo":        foundTenant(ctx, ctx.Repo.Repository),
			"Org":         foundTenant(ctx, ctx.Org.Organization),
		}, targetTenantIDs) {
			// если не нашли совпадения, падаем с ошибкой.
			log.Debug("matchTenant NotFound.")
			ctx.NotFound("", nil)
			return
		}
	}
}

// foundTenant идентификация объекта запроса. возвращаем массив tenantIDs
func foundTenant(ctx *context.Context, obj interface{}) (result []string) {
	if reflect.ValueOf(obj).IsNil() {
		log.Debug("foundTenant obj is nil.")
		return
	}
	switch obj.(type) {
	case *user_model.User:
		var tenantIds []string
		var err error
		// нам точно приходит пользователь, отправивший запрос. (уже есть)
		tenantIds, err = role_model.GetUserTenantIDsOrDefaultTenantID(obj.(*user_model.User))
		if err != nil {
			log.Debug("GetUserTenantIds ended with err: %s.", err.Error())
			return
		}
		result = append(result, tenantIds...)
	case *repo.Repository:
		// находим тенанта для репозиториев.
		if tenantId, err := role_model.GetRepoTenantId(obj.(*repo.Repository)); err != nil {
			log.Debug("GetRepoTenantId ended with err: %s.", err.Error())
			return
		} else {
			result = append(result, tenantId)
		}
	case *organization.Organization:
		// находим тенанта для организаций.
		if tenantId, err := tenant.GetTenantByOrgIdOrDefault(ctx, obj.(*organization.Organization).ID); err != nil {
			log.Debug("GetTenantByOrgIdOrDefault ended with err: %s.", err.Error())
			return
		} else {
			result = append(result, tenantId)
		}
	}
	return
}

// находим тенанты для всех возможных структур
func matchTenant(tenantObjectIDs map[string][]string, targetTenantIDs map[string]struct{}) (ok bool) {
	for _, permissionObject := range tenantObjectIDs {
		// ищем хотя бы 1 пересечение по тенантам
		for _, objectTenantIDs := range permissionObject {
			if _, ok = targetTenantIDs[objectTenantIDs]; ok {
				return
			}
		}
	}
	return
}

// VerifyAuthWithOptionsAPI checks authentication according to options
func VerifyAuthWithOptionsAPI(options *VerifyOptions) func(ctx *context.APIContext) {
	const changePasswordUrl = "user/settings/change_password"
	return func(ctx *context.APIContext) {
		// Check prohibit login users.
		if ctx.IsSigned {
			if !ctx.Doer.IsActive && setting.Service.RegisterEmailConfirm {
				ctx.Data["Title"] = ctx.Tr("auth.active_your_account")
				ctx.JSON(http.StatusForbidden, map[string]string{
					"message": "This account is not activated.",
				})
				return
			}
			if !ctx.Doer.IsActive || ctx.Doer.ProhibitLogin {
				log.Info("Failed authentication attempt for %s from %s", ctx.Doer.Name, ctx.RemoteAddr())
				ctx.Data["Title"] = ctx.Tr("auth.prohibit_login")
				ctx.JSON(http.StatusForbidden, map[string]string{
					"message": "This account is prohibited from signing in, please contact your site administrator.",
				})
				return
			}

			if ctx.Doer.MustChangePassword {
				ctx.JSON(http.StatusForbidden, map[string]string{
					"message": "You must change your password. Change it at: " + setting.AppURL + changePasswordUrl,
				})
				return
			}
		}

		// Redirect to dashboard if user tries to visit any non-login page.
		if options.SignOutRequired && ctx.IsSigned && ctx.Req.URL.RequestURI() != "/" {
			ctx.Redirect(setting.AppSubURL + "/")
			return
		}

		if options.SignInRequired {
			if !ctx.IsSigned {
				// Restrict API calls with error message.
				ctx.JSON(http.StatusForbidden, map[string]string{
					"message": "Only signed in user is allowed to call APIs.",
				})
				return
			} else if !ctx.Doer.IsActive && setting.Service.RegisterEmailConfirm {
				ctx.Data["Title"] = ctx.Tr("auth.active_your_account")
				ctx.JSON(http.StatusForbidden, map[string]string{
					"message": "This account is not activated.",
				})
				return
			}
			if ctx.IsSigned && ctx.IsBasicAuth {
				if skip, ok := ctx.Data["SkipLocalTwoFA"]; ok && skip.(bool) {
					return // Skip 2FA
				}
				twofa, err := auth.GetTwoFactorByUID(ctx.Doer.ID)
				if err != nil {
					if auth.IsErrTwoFactorNotEnrolled(err) {
						return // No 2FA enrollment for this user
					}
					ctx.InternalServerError(err)
					return
				}
				otpHeader := ctx.Req.Header.Get("X-SourceControl-OTP")
				ok, err := twofa.ValidateTOTP(otpHeader)
				if err != nil {
					ctx.InternalServerError(err)
					return
				}
				if !ok {
					ctx.JSON(http.StatusForbidden, map[string]string{
						"message": "Only signed in user is allowed to call APIs.",
					})
					return
				}
			}
		}

		if options.AdminRequired {
			if !ctx.Doer.IsAdmin {
				ctx.JSON(http.StatusForbidden, map[string]string{
					"message": "You have no permission to request for this.",
				})
				return
			}
		}
	}
}
