// Для генерации Swagger выполни команду ниже:
// swagger generate spec -w './routers/api/v2' -o './templates/swagger/v2_json.tmpl' --exclude-deps

// Package v2 SourceControl API.
//
// This documentation describes the SourceControl API V2.
//
//	Schemes: http, https
//	BasePath: {{AppSubUrl | JSEscape | Safe}}/api/v2
//	Version: {{AppVer | JSEscape | Safe}}
//	License: MIT http://opensource.org/licenses/MIT
//
//	Consumes:
//	- application/json
//	- text/plain
//
//	Produces:
//	- application/json
//	- text/html
//
//	Security:
//	- BasicAuth :
//	- Token :
//	- AccessToken :
//	- AuthorizationHeaderToken :
//	- SudoParam :
//	- SudoHeader :
//	- TOTPHeader :
//
//	SecurityDefinitions:
//	BasicAuth:
//	     type: basic
//	Token:
//	     type: apiKey
//	     name: token
//	     in: query
//	AccessToken:
//	     type: apiKey
//	     name: access_token
//	     in: query
//	AuthorizationHeaderToken:
//	     type: apiKey
//	     name: Authorization
//	     in: header
//	     description: API tokens must be prepended with "token" followed by a space.
//	SudoParam:
//	     type: apiKey
//	     name: sudo
//	     in: query
//	     description: Sudo API request as the user provided as the key. Admin privileges are required.
//	SudoHeader:
//	     type: apiKey
//	     name: Sudo
//	     in: header
//	     description: Sudo API request as the user provided as the key. Admin privileges are required.
//	TOTPHeader:
//	     type: apiKey
//	     name: X-GITEA-OTP
//	     in: header
//	     description: Must be used in combination with BasicAuth if two-factor authentication is enabled.
//
// swagger:meta
package v2

import (
	gocontext "context"
	"fmt"
	"net/http"

	"gitea.com/go-chi/binding"
	"github.com/go-chi/cors"

	auth_model "code.gitea.io/gitea/models/auth"
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/external_metric_counter/external_metric_counter_db"
	"code.gitea.io/gitea/models/internal_metric_counter/internal_metric_counter_db"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/repo_marks/marks"
	"code.gitea.io/gitea/models/repo_marks/repo_marks_db"
	"code.gitea.io/gitea/models/role_model"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/sbt/audit"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/routers/api/v2/admin"
	"code.gitea.io/gitea/routers/api/v2/external_counter"
	"code.gitea.io/gitea/routers/api/v2/internal_counter"
	"code.gitea.io/gitea/routers/api/v2/middleware"
	"code.gitea.io/gitea/routers/api/v2/models"
	"code.gitea.io/gitea/routers/api/v2/models/metrics"
	apirepo "code.gitea.io/gitea/routers/api/v2/models/repo"
	sshmodels "code.gitea.io/gitea/routers/api/v2/models/ssh"
	"code.gitea.io/gitea/routers/api/v2/project"
	"code.gitea.io/gitea/routers/api/v2/repo"
	"code.gitea.io/gitea/routers/api/v2/ssh"
	"code.gitea.io/gitea/routers/api/v2/tenant"
	"code.gitea.io/gitea/routers/api/v2/user"
	webhook "code.gitea.io/gitea/routers/api/v2/webhook"
	"code.gitea.io/gitea/routers/private/repo_mark"
	"code.gitea.io/gitea/services/auth"
	auth_service "code.gitea.io/gitea/services/auth"
	"code.gitea.io/gitea/services/auth/iamprivileger"
	"code.gitea.io/gitea/services/forms"
	privileges2 "code.gitea.io/gitea/services/privileges"
	webhook2 "code.gitea.io/gitea/services/webhook"
)

// bind binding an obj to a func(ctx *context.APIContext)
func bind[T any](_ T) any {
	return func(ctx *context.APIContext) {
		theObj := new(T) // create a new form obj for every request but not use obj directly
		errs := binding.Bind(ctx.Req, theObj)
		if len(errs) > 0 {
			ctx.Error(http.StatusBadRequest, "validationError", fmt.Sprintf("%s: %s", errs[0].FieldNames, errs[0].Error()))
			return
		}
		web.SetForm(ctx, theObj)
	}
}

// Routes регистрирует все API роуты v2 версии
func Routes(ctx gocontext.Context) *web.Route {
	m := web.NewRoute()
	engine := db.GetEngine(ctx)
	repoMarksDb := repo_marks_db.NewRepoMarksDB(engine)
	repoKeyDb := repo_model.NewRepoKeyDB(engine)
	internalMetricDB := internal_metric_counter_db.New(engine)
	externalMetricDB := external_metric_counter_db.New(engine)
	editorRepoMarks := repo_mark.NewRepoMarksEditor(repoMarksDb, repoKeyDb)
	codeHubMark := marks.GetCodeHubMark(setting.CodeHub.CodeHubMarkLabelName)
	repoServer := repo.NewRepoServer(role_model.CheckUserPermissionToOrganization, repoKeyDb, editorRepoMarks, codeHubMark)
	tenantServer := tenant.NewTenantServer()
	internalMetricServer := internal_counter.New(internalMetricDB, repoKeyDb, setting.CodeHub.InternalMetricsNamesList, setting.CodeHub.CodeHubMetricEnabled)
	externalMetricServer := external_counter.New(externalMetricDB, repoKeyDb, setting.CodeHub.CodeHubMetricEnabled)
	enforcer := role_model.GetSecurityEnforcer()
	privilege, err := privileges2.NewPrivilege(engine, enforcer)
	if err != nil {
		log.Error("Error has occurred while creating privileges service: %v", err)
		return nil
	}
	server := admin.NewPrivilegesServer(privilege)

	service := webhook2.NewWebHookService()
	hookServer := webhook.NewServer(service, repoKeyDb)

	m.Use(securityHeaders())
	if setting.CORSConfig.Enabled {
		m.Use(cors.Handler(cors.Options{
			AllowedOrigins:   setting.CORSConfig.AllowDomain,
			AllowedMethods:   setting.CORSConfig.Methods,
			AllowCredentials: setting.CORSConfig.AllowCredentials,
			AllowedHeaders:   append([]string{"Authorization", "X-SourceControl-OTP"}, setting.CORSConfig.Headers...),
			MaxAge:           int(setting.CORSConfig.MaxAge.Seconds()),
		}))
	}
	m.Use(context.APIContexter())

	group := buildAuthGroup()
	if err := group.Init(ctx); err != nil {
		log.Error("Error has occurred while initializing '%s' auth method. Error: %v", group.Name(), err)
		return nil
	}

	// Получение пользователя из сессии, если залогинен
	m.Use(auth.APIAuth(group))

	m.Use(auth.VerifyAuthWithOptionsAPI(&auth.VerifyOptions{
		SignInRequired: setting.Service.RequireSignInView,
	}))

	mw := middleware.NewMiddleware(repoKeyDb)

	m.Group("", func() {
		m.Get("/swagger", func(ctx *context.APIContext) {
			ctx.Redirect(setting.AppSubURL + "/api/swagger_v2")
		})
		m.Group("/admin", func() {
			m.Group("/users", func() {
				m.Post("", reqToken(auth_model.AccessTokenScopeUser), bind(models.CreateUserRequest{}), user.CreateUserRequest)
				m.Get("", reqToken(auth_model.AccessTokenScopeReadUser), user.GetUserRequest)
				m.Group("/keys", func() {
					m.Post("", reqToken(auth_model.AccessTokenScopeAdminPublicKey), bind(sshmodels.CreateSSHKeyRequest{}), ssh.Create)
					m.Delete("", reqToken(auth_model.AccessTokenScopeAdminPublicKey), ssh.Delete)
				})
			})
			m.Group("/privileges", func() {
				m.Post("", reqToken(auth_model.AccessTokenScopeWritePrivileges), bind(forms.ApplyPrivilegeRequest{}), server.ApplyPrivileges)
				m.Get("", reqToken(auth_model.AccessTokenScopeReadPrivileges), bind(forms.GetPrivilegesRequest{}), server.GetPrivileges)
			})
		})
		m.Group("/projects", func() {
			m.Post("/create", reqToken(auth_model.AccessTokenScopeWriteProject), bind(forms.CreateProjectRequest{}), project.CreateProjectRequest)
			m.Get("", reqToken(auth_model.AccessTokenScopeReadProject), project.GetProjectRequest)
		})
		m.Group("/tenants", func() {
			m.Get("/", reqToken(auth_model.AccessTokenScopeReadTenant), tenantServer.GetTenantByKey)
			m.Post("/", reqToken(auth_model.AccessTokenScopeWriteTenant), bind(models.CreateTenantOptions{}), tenantServer.CreateTenant)
		})

		m.Group("/projects", func() {
			m.Group("/repos", func() {
				m.Get("", reqToken(auth_model.AccessTokenScopeReadOrg), repoServer.GetOrgRepo)
				m.Get("/metrics", reqToken(auth_model.AccessTokenScopeCodeHub), internalMetricServer.GetInternalMetricCounter)
				m.Get("/reuse_metric", reqToken(auth_model.AccessTokenScopeCodeHub), externalMetricServer.GetExternalMetricCounter)
				m.Post("/reuse_metric", reqToken(auth_model.AccessTokenScopeCodeHub), bind(metrics.SetExternalMetricRequest{}), externalMetricServer.SetExternalMetricCounter)
				m.Post("", reqToken(auth_model.AccessTokenScopeWriteOrg), bind(apirepo.CreateRepoOptions{}), repoServer.CreateTenantOrgRepo)
				m.Post("/marks/codehub", reqToken(auth_model.AccessTokenScopeCodeHub), bind(apirepo.SetMarkRequest{}), repoServer.SetMark)
				m.Delete("/marks/codehub", reqToken(auth_model.AccessTokenScopeCodeHub), bind(apirepo.DeleteMarkRequest{}), repoServer.DeleteMark)
				m.Delete("/reuse_metric", reqToken(auth_model.AccessTokenScopeCodeHub), bind(apirepo.DeleteMarkRequest{}), externalMetricServer.DeleteExternalMetricCounter)
			})
		})
		m.Group("/repos", func() {
			m.Group("/webhooks", func() {
				m.Post("", reqToken(auth_model.AccessTokenScopeWriteRepoHook), bind(models.CreateHookOption{}), hookServer.CreateHook)
				m.Get("", reqToken(auth_model.AccessTokenScopeReadRepoHook), hookServer.GetHook)
				m.Delete("", reqToken(auth_model.AccessTokenScopeWriteRepoHook), hookServer.DeleteHook)
			})
		}, assign(), reqWebhooksEnabled(), mw.KeysRequiredCheck(), context.RequireRepoPermissionApi(role_model.EDIT))
	})

	return m
}
func assign() func(ctx *context.APIContext) {
	return func(ctx *context.APIContext) {
		if !ctx.IsSigned {
			ctx.Error(http.StatusUnauthorized, "Assign", "Err: need authorization")
			return
		}
	}
}

// reqWebhooksEnabled requires webhooks to be enabled
func reqWebhooksEnabled() func(ctx *context.APIContext) {
	return func(ctx *context.APIContext) {
		if setting.DisableWebhooks {
			ctx.Error(http.StatusForbidden, "", "webhooks disabled by administrator")
			return
		}
	}
}

func securityHeaders() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
			// CORB: https://www.chromium.org/Home/chromium-security/corb-for-developers
			// http://stackoverflow.com/a/3146618/244009
			resp.Header().Set("x-content-type-options", "nosniff")
			next.ServeHTTP(resp, req)
		})
	}
}
func buildAuthGroup() *auth.Group {
	group := auth.NewGroup(
		&auth.OAuth2{},
		&auth.HTTPSign{},
		&auth.Basic{}, // FIXME: this should be removed once we don't allow basic auth in API
	)

	enforcer := role_model.GetSecurityEnforcer()
	engine := db.GetEngine(gocontext.Background())

	privileger, err := iamprivileger.New(enforcer, engine)
	if err != nil {
		log.Fatal("create privileger: %s", err)
	}

	if setting.IAM.Enabled {
		group.Add(auth_service.NewIAMProxy(privileger, engine))
	}

	return group
}

// Contexter middleware already checks token for user sign in process.
func reqToken(requiredScope auth_model.AccessTokenScope) func(ctx *context.APIContext) {
	return func(ctx *context.APIContext) {
		// If actions token is present
		if true == ctx.Data["IsActionsToken"] {
			return
		}

		// If OAuth2 token is present
		if _, ok := ctx.Data["ApiTokenScope"]; ctx.Data["IsApiToken"] == true && ok {
			// no scope required
			if requiredScope == "" {
				return
			}

			// check scope
			scope := ctx.Data["ApiTokenScope"].(auth_model.AccessTokenScope)
			allow, err := scope.HasScope(requiredScope)
			if err != nil {
				ctx.Error(http.StatusForbidden, "reqToken", "parsing token failed: "+err.Error())
				return
			}
			if allow {
				return
			}

			// if requires 'repo' scope, but only has 'public_repo' scope, allow it only if the repo is public
			if requiredScope == auth_model.AccessTokenScopeRepo {
				if allowPublicRepo, err := scope.HasScope(auth_model.AccessTokenScopePublicRepo); err == nil && allowPublicRepo {
					if ctx.Repo.Repository != nil && !ctx.Repo.Repository.IsPrivate {
						return
					}
				}
			}

			ctx.Error(http.StatusForbidden, "reqToken", "token does not have required scope: "+requiredScope)
			return
		}
		if ctx.IsBasicAuth {
			ctx.CheckForOTP()
			return
		}
		if ctx.IsSigned {
			return
		}
		ctx.Error(http.StatusUnauthorized, "reqToken", "Err: need authorization")
		auditParams := map[string]string{
			"request_url": ctx.Req.URL.RequestURI(),
		}
		audit.CreateAndSendEvent(audit.UnauthorizedRequestEvent, audit.EmptyRequiredField, audit.EmptyRequiredField, audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
	}
}
