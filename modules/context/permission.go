// Copyright 2018 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package context

import (
	"net/http"
	"strings"

	auth_model "code.gitea.io/gitea/models/auth"
	"code.gitea.io/gitea/models/organization"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/role_model"
	"code.gitea.io/gitea/models/tenant"
	"code.gitea.io/gitea/models/unit"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/trace"
)

// RequireRepoAdmin returns a middleware for requiring repository admin permission
func RequireRepoAdmin() func(ctx *Context) {
	return func(ctx *Context) {
		// если у нас включена ролевая модель SourceControl, то RequireRepoAdmin проверка пропускается
		if setting.SourceControl.TenantWithRoleModeEnabled {
			return
		}
		if !ctx.IsSigned || !ctx.Repo.IsAdmin() {
			ctx.NotFound(ctx.Req.URL.RequestURI(), nil)
			return
		}
	}
}

// RequireRepoWriter returns a middleware for requiring repository write to the specify unitType
func RequireRepoWriter(unitType unit.Type) func(ctx *Context) {
	return func(ctx *Context) {
		// если у нас включена ролевая модель SourceControl, то RequireRepoWriter проверка пропускается
		if setting.SourceControl.TenantWithRoleModeEnabled {
			return
		}
		if !ctx.Repo.CanWrite(unitType) {
			ctx.NotFound(ctx.Req.URL.RequestURI(), nil)
			return
		}
	}
}

// CanEnableEditor checks if the user is allowed to write to the branch of the repo
func CanEnableEditor() func(ctx *Context) {
	return func(ctx *Context) {
		// если у нас включена ролевая модель SourceControl, то CanEnableEditor проверка пропускается
		if setting.SourceControl.TenantWithRoleModeEnabled {
			return
		}
		if !ctx.Repo.CanWriteToBranch(ctx.Doer, ctx.Repo.BranchName) {
			ctx.NotFound("CanWriteToBranch denies permission", nil)
			return
		}
	}
}

// CanEnableEditor checks if the user came from one work
func CanEnableOneWork() func(ctx *Context) {
	return func(ctx *Context) {
		//  проверка, включен ли у нас OW
		if setting.OneWork.Enabled {
			headerValueXSSDMode := strings.ToLower(ctx.Req.Header.Get("X-SSD-MODE"))

			switch headerValueXSSDMode {
			case "works":
				ctx.Data["XSSDMode"] = "works"
			case "separate":
				ctx.Data["XSSDMode"] = "standalone"
				ctx.NotFound(ctx.Req.URL.RequestURI(), nil)
			default:
				ctx.Data["XSSDMode"] = "standalone"
				ctx.NotFound(ctx.Req.URL.RequestURI(), nil)
			}
			return
		}
	}
}

// RequireRepoWriterOr returns a middleware for requiring repository write to one of the unit permission
func RequireRepoWriterOr(unitTypes ...unit.Type) func(ctx *Context) {
	return func(ctx *Context) {
		// если у нас включена ролевая модель SourceControl, то RequireRepoWriterOr проверка пропускается
		if setting.SourceControl.TenantWithRoleModeEnabled {
			return
		}
		for _, unitType := range unitTypes {
			if ctx.Repo.CanWrite(unitType) {
				return
			}
		}
		ctx.NotFound(ctx.Req.URL.RequestURI(), nil)
	}
}

// RequireRepoReader returns a middleware for requiring repository read to the specify unitType
func RequireRepoReader(unitType unit.Type) func(ctx *Context) {
	return func(ctx *Context) {
		// если у нас включена ролевая модель SourceControl, то RequireRepoReader проверка пропускается
		if setting.SourceControl.TenantWithRoleModeEnabled {
			return
		}
		if !ctx.Repo.CanRead(unitType) {
			if log.IsTrace() {
				if ctx.IsSigned {
					log.Trace("Permission Denied: User %-v cannot read %-v in Repo %-v\n"+
						"User in Repo has Permissions: %-+v",
						ctx.Doer,
						unitType,
						ctx.Repo.Repository,
						ctx.Repo.Permission)
				} else {
					log.Trace("Permission Denied: Anonymous user cannot read %-v in Repo %-v\n"+
						"Anonymous user in Repo has Permissions: %-+v",
						unitType,
						ctx.Repo.Repository,
						ctx.Repo.Permission)
				}
			}
			ctx.NotFound(ctx.Req.URL.RequestURI(), nil)
			return
		}
	}
}

// RequireRepoReaderOr returns a middleware for requiring repository write to one of the unit permission
func RequireRepoReaderOr(unitTypes ...unit.Type) func(ctx *Context) {
	return func(ctx *Context) {
		// если у нас включена ролевая модель SourceControl, то RequireRepoReaderOr проверка пропускается
		if setting.SourceControl.TenantWithRoleModeEnabled {
			return
		}
		for _, unitType := range unitTypes {
			if ctx.Repo.CanRead(unitType) {
				return
			}
		}
		if log.IsTrace() {
			var format string
			var args []interface{}
			if ctx.IsSigned {
				format = "Permission Denied: User %-v cannot read ["
				args = append(args, ctx.Doer)
			} else {
				format = "Permission Denied: Anonymous user cannot read ["
			}
			for _, unit := range unitTypes {
				format += "%-v, "
				args = append(args, unit)
			}

			format = format[:len(format)-2] + "] in Repo %-v\n" +
				"User in Repo has Permissions: %-+v"
			args = append(args, ctx.Repo.Repository, ctx.Repo.Permission)
			log.Trace(format, args...)
		}
		ctx.NotFound(ctx.Req.URL.RequestURI(), nil)
	}
}

// CheckRepoScopedToken check whether personal access token has repo scope
func CheckRepoScopedToken(ctx *Context, repo *repo_model.Repository) {
	if !ctx.IsBasicAuth || ctx.Data["IsApiToken"] != true {
		return
	}

	var err error
	scope, ok := ctx.Data["ApiTokenScope"].(auth_model.AccessTokenScope)
	if ok { // it's a personal access token but not oauth2 token
		var scopeMatched bool
		scopeMatched, err = scope.HasScope(auth_model.AccessTokenScopeRepo)
		if err != nil {
			ctx.ServerError("HasScope", err)
			return
		}
		if !scopeMatched && !repo.IsPrivate {
			scopeMatched, err = scope.HasScope(auth_model.AccessTokenScopePublicRepo)
			if err != nil {
				ctx.ServerError("HasScope", err)
				return
			}
		}
		if !scopeMatched {
			ctx.Error(http.StatusForbidden)
			return
		}
	}
}

// RequireRepoPermission returns a middleware for requiring repository permissions
func RequireRepoPermission(action role_model.Action) func(ctx *Context) {
	return func(ctx *Context) {
		logTracer := trace.NewLogTracer()
		message := logTracer.CreateTraceMessage(ctx)
		err := logTracer.Trace(message)
		if err != nil {
			log.Error("Error has occurred while creating trace message: %v", err)
		}
		defer func() {
			err = logTracer.TraceTime(message)
			if err != nil {
				log.Error("Error has occurred while creating trace time message: %v", err)
			}
		}()

		if ctx.IsSigned && ctx.Repo != nil && ctx.Repo.Repository != nil && setting.SourceControl.TenantWithRoleModeEnabled {
			tenantId, err := tenant.GetTenantByOrgIdOrDefault(ctx, ctx.Repo.Repository.OwnerID)
			if err != nil {
				ctx.NotFound(ctx.Req.URL.RequestURI(), nil)
				return
			}

			allowed, err := role_model.CheckUserPermissionToOrganization(ctx, ctx.Doer, tenantId, &organization.Organization{ID: ctx.Repo.Repository.OwnerID}, action)
			if err != nil || !allowed {
				ctx.NotFound(ctx.Req.URL.RequestURI(), nil)
			}
		}
	}
}

// RequireOrgPermission returns a middleware for requiring organization permissions
func RequireOrgPermission(action role_model.Action) func(ctx *Context) {
	return func(ctx *Context) {
		logTracer := trace.NewLogTracer()
		message := logTracer.CreateTraceMessage(ctx)
		err := logTracer.Trace(message)
		if err != nil {
			log.Error("Error has occurred while creating trace message: %v", err)
		}
		defer func() {
			err = logTracer.TraceTime(message)
			if err != nil {
				log.Error("Error has occurred while creating trace time message: %v", err)
			}
		}()

		if ctx.IsSigned && ctx.Org.Organization != nil && setting.SourceControl.TenantWithRoleModeEnabled {
			tenantId, err := tenant.GetTenantByOrgIdOrDefault(ctx, ctx.Org.Organization.ID)
			if err != nil {
				ctx.NotFound(ctx.Req.URL.RequestURI(), nil)
				return
			}

			allowed, err := role_model.CheckUserPermissionToOrganization(ctx, ctx.Doer, tenantId, ctx.Org.Organization, action)
			if err != nil || !allowed {
				ctx.NotFound(ctx.Req.URL.RequestURI(), nil)
			}
		}
	}
}

// RequireRepoReadPermission returns a middleware for requiring repository read permission
func RequireRepoReadPermission() func(ctx *Context) {
	return func(ctx *Context) {
		logTracer := trace.NewLogTracer()
		message := logTracer.CreateTraceMessage(ctx)
		err := logTracer.Trace(message)
		if err != nil {
			log.Error("Error has occurred while creating trace message: %v", err)
		}
		defer func() {
			err = logTracer.TraceTime(message)
			if err != nil {
				log.Error("Error has occurred while creating trace time message: %v", err)
			}
		}()

		allowPermissions(ctx, role_model.ViewBranch)
	}
}

// allowPermissions проверка доступа пользователя к конкретным действиям по кастомным привилегиям
func allowPermissions(ctx *Context, customPrivilege role_model.CustomPrivilege) {
	if ctx.IsSigned && setting.SourceControl.TenantWithRoleModeEnabled {
		action := role_model.READ
		if ctx.Repo.Repository.IsPrivate {
			action = role_model.READ_PRIVATE
		}
		logTracer := trace.NewLogTracer()
		message := logTracer.CreateTraceMessage(ctx)
		err := logTracer.Trace(message)
		if err != nil {
			log.Error("Error has occurred while creating trace message: %v", err)
		}
		defer func() {
			err = logTracer.TraceTime(message)
			if err != nil {
				log.Error("Error has occurred while creating trace time message: %v", err)
			}
		}()

		tenantId, err := tenant.GetTenantByOrgIdOrDefault(ctx, ctx.Repo.Repository.OwnerID)
		if err != nil {
			log.Error("Error has occurred while getting tenant for organization")
			ctx.NotFound(ctx.Req.URL.RequestURI(), nil)
			return
		}

		allow, err := role_model.CheckUserPermissionToOrganization(ctx, ctx.Doer, tenantId, &organization.Organization{ID: ctx.Repo.Repository.OwnerID}, action)
		if !allow || err != nil {
			allowed, err := role_model.CheckUserPermissionToTeam(ctx, ctx.Doer, tenantId, &organization.Organization{ID: ctx.Repo.Repository.OwnerID},
				&repo_model.Repository{ID: ctx.Repo.Repository.ID}, customPrivilege.String(),
			)
			if err != nil || !allowed {
				log.Error("Error has occurred while checking user permission to organization or custom privileges")
				ctx.NotFound(ctx.Req.URL.RequestURI(), nil)
				return
			}
		}
	}
}

func RequireCustomPermission(action role_model.Action, customPrivilege role_model.CustomPrivilege) func(ctx *Context) {
	return func(ctx *Context) {
		if ctx.IsSigned && ctx.Org.Organization != nil || ctx.Repo.Repository != nil && setting.SourceControl.TenantWithRoleModeEnabled {
			var (
				tenantID string
				err      error
			)
			logTracer := trace.NewLogTracer()
			message := logTracer.CreateTraceMessage(ctx)
			err = logTracer.Trace(message)
			if err != nil {
				log.Error("Error has occurred while creating trace message: %v", err)
			}
			defer func() {
				err = logTracer.TraceTime(message)
				if err != nil {
					log.Error("Error has occurred while creating trace time message: %v", err)
				}
			}()

			if ctx.Org.Organization != nil {
				tenantID, err = tenant.GetTenantByOrgIdOrDefault(ctx, ctx.Org.Organization.ID)
				if err != nil {
					log.Error("Error has occurred while getting tenant")
					ctx.NotFound(ctx.Req.URL.RequestURI(), nil)
					return
				}
			} else if ctx.Repo.Repository != nil {
				tenantID, err = tenant.GetTenantByOrgIdOrDefault(ctx, ctx.Repo.Repository.Owner.ID)
				if err != nil {
					log.Error("Error has occurred while getting tenant")
					ctx.NotFound(ctx.Req.URL.RequestURI(), nil)
					return
				}
			}

			allow, err := role_model.CheckUserPermissionToOrganization(ctx, ctx.Doer, tenantID, &organization.Organization{ID: ctx.Repo.Repository.OwnerID}, action)
			if err != nil {
				ctx.NotFound(ctx.Req.URL.RequestURI(), nil)
				return
			}
			if !allow {
				allowed, err := role_model.CheckUserPermissionToTeam(ctx, ctx.Doer, tenantID, &organization.Organization{ID: ctx.Repo.Repository.OwnerID},
					&repo_model.Repository{ID: ctx.Repo.Repository.ID}, customPrivilege.String())
				if err != nil {
					log.Error("Error has occurred while checking user permission to organization or custom privileges")
					ctx.NotFound(ctx.Req.URL.RequestURI(), nil)
					return
				}
				if !allowed {
					log.Warn("Access denied: user does not have the required role or privilege")
					ctx.NotFound(ctx.Req.URL.RequestURI(), nil)
					return
				}
			}
		}
	}
}
