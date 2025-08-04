// Для генерации Swagger выполни команду ниже:
// swagger generate spec -w './routers/api/v3' -o './templates/swagger/v3_json.tmpl' --exclude-deps

// Package v3 SourceControl API.
//
// This documentation describes the SourceControl API V3.
//
//	Schemes: http, https
//	BasePath: {{AppSubUrl | JSEscape | Safe}}/api/v3
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

package v3

import (
	gocontext "context"
	"fmt"
	"net/http"
	"strings"

	"gitea.com/go-chi/binding"
	"github.com/go-chi/cors"
	"github.com/google/uuid"

	actions_model "code.gitea.io/gitea/models/actions"
	auth_model "code.gitea.io/gitea/models/auth"
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/default_reviewers/default_reviewers_db"
	"code.gitea.io/gitea/models/git/protected_branch/convert"
	"code.gitea.io/gitea/models/git/protected_branch/protected_branch_db"
	"code.gitea.io/gitea/models/perm"
	access_model "code.gitea.io/gitea/models/perm/access"
	"code.gitea.io/gitea/models/project"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/review_settings/review_settings_db"
	"code.gitea.io/gitea/models/role_model"
	repo2 "code.gitea.io/gitea/models/sonar/repo"
	"code.gitea.io/gitea/models/sonar/usecase"
	tenant2 "code.gitea.io/gitea/models/tenant"
	"code.gitea.io/gitea/models/unit"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/models/user/user_db"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/sbt/audit"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/web"
	protected_branch "code.gitea.io/gitea/routers/api/v3/branch_protection"
	"code.gitea.io/gitea/routers/api/v3/models"
	"code.gitea.io/gitea/routers/api/v3/review_settings"
	"code.gitea.io/gitea/routers/api/v3/sonar"
	"code.gitea.io/gitea/services/auth"
	"code.gitea.io/gitea/services/auth/iamprivileger"
	convert_v3 "code.gitea.io/gitea/services/convert/v3"
	protected_brancher "code.gitea.io/gitea/services/protected_branch"
	"code.gitea.io/gitea/services/user/user_manager"
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

// Routes регистрирует все API роуты v3 версии
func Routes(ctx gocontext.Context) *web.Route {
	m := web.NewRoute()
	engine := db.GetEngine(ctx)
	defaultReviewersDB := default_reviewers_db.New(engine)
	reviewSettingsDB := review_settings_db.New(engine)
	reviewSettingsServer := review_settings.NewServer(defaultReviewersDB, reviewSettingsDB)

	// -----------DI-----------

	repository := repo2.NewSonarSettings(engine)
	uc := usecase.NewUsecase(repository)
	api := sonar.NewSonarServer(uc)

	// -----------DI-----------

	// -----------DI user ----------------------
	userRepository := user_db.NewUserDB()
	userManager := user_manager.NewUserManager(userRepository)
	// -----------DI user ----------------------

	// -----------DI Protected Branch ----------

	protectedBranchRepository := protected_branch_db.NewProtectedBranchDB(engine)
	protectedBranchChecker := protected_brancher.NewProtectedBranchChecker()
	protectedBranchGetter := protected_brancher.NewProtectedBranchGetter()
	protectedBranchMerger := protected_brancher.NewProtectedBranchMerger()
	protectedBranchUpdater := protected_brancher.NewProtectedBranchUpdater()
	protectedBranchManager := protected_brancher.NewProtectedBranchManager(protectedBranchGetter, protectedBranchChecker, protectedBranchMerger, protectedBranchUpdater, protectedBranchRepository)
	branchProtectionConverter := convert_v3.NewBranchProtectionConverter()
	auditBranchProtectionConverter := convert.NewAuditConverter(userManager)
	protectedBranchAPI := protected_branch.NewBranchProtectionServer(protectedBranchManager, branchProtectionConverter, auditBranchProtectionConverter)

	// -----------DI Protected Branch ----------
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

	m.Group("", func() {
		m.Get("/swagger", func(ctx *context.APIContext) {
			ctx.Redirect(setting.AppSubURL + "/api/swagger_v3")
		})
	})

	// tenant - tenant id
	// project - название проекта
	// repo - название репозитория
	m.Group("repos/{tenant}/{project}/{repo}", func() {
		m.Group("/branch_protections", func() {
			m.Get("", protectedBranchAPI.GetBranchProtections)
			m.Post("", bind(models.BranchProtectionBody{}), protectedBranchAPI.CreateBranchProtection)
			m.Get("/{branch_name}", protectedBranchAPI.GetBranchProtection)
			m.Put("/{branch_name}", bind(models.BranchProtectionBody{}), protectedBranchAPI.UpdateBranchProtection)
			m.Delete("/{branch_name}", protectedBranchAPI.DeleteBranchProtection)
		})
		m.Group("/sonar", func() {
			m.Post("", bind(models.CreateOrUpdateSonarProjectRequest{}), api.CreateSonarSettings)
			m.Put("", bind(models.CreateOrUpdateSonarProjectRequest{}), api.UpdateSonarSettings)
			m.Get("", api.SonarSettings)
			m.Delete("", api.DeleteSonarSettings)
		})
	}, repoAssignment(), context.RequireRepoPermissionApi(role_model.EDIT), tenantAssigment())

	// review settings
	m.Group("/repos/{tenant}/{project}/{repo}/review_settings", func() {
		m.Get("", reviewSettingsServer.GetReviewSettingsHandler, context.RequireRepoPermissionApi(role_model.EDIT))
		m.Get("/{branch_name}", reviewSettingsServer.GetBranchReviewSettings, context.RequireRepoPermissionApi(role_model.EDIT))
		m.Post("", bind(models.ReviewSettingsRequest{}), reviewSettingsServer.CreateReviewSettings, context.RequireRepoPermissionApi(role_model.EDIT))
		m.Put("/{branch_name}", bind(models.ReviewSettingsRequest{}), reviewSettingsServer.UpdateReviewSettings, context.RequireRepoPermissionApi(role_model.EDIT))
		m.Delete("/{branch_name}", reviewSettingsServer.DeleteReviewSettings, context.RequireRepoPermissionApi(role_model.EDIT))
	}, repoAssignment(), tenantAssigment())

	return m
}

func repoAssignment() func(ctx *context.APIContext) {
	return func(ctx *context.APIContext) {
		projectName := ctx.Params("project")
		repoName := ctx.Params("repo")

		var (
			owner *user_model.User
			err   error
		)
		if !ctx.IsSigned {
			log.Warn("Err: need authorization")
			ctx.Error(http.StatusUnauthorized, "LookupRepoRedirect", "Err: need authorization")
			return
		}

		// Check if the user is the same as the repository owner.
		if ctx.Doer.LowerName == strings.ToLower(projectName) {
			owner = ctx.Doer
		} else {
			// TODO: в будущем нужно разделить сущности юзера и проекта. Данная логика вызывает путанницу
			// Получение проекта
			owner, err = user_model.GetUserByName(ctx, projectName)
			if err != nil {
				if user_model.IsErrUserNotExist(err) {
					log.Warn("Org with name - %s, not exist", projectName)
					ctx.Error(http.StatusNotFound, "GetUserByName", project.ErrProjectWithNameNotExist{Name: projectName})
				} else {
					log.Error("Err: get user by name: %v", err)
					ctx.Error(http.StatusInternalServerError, "GetUserByName", err)
				}
				return
			}
		}
		ctx.Repo.Owner = owner
		ctx.ContextUser = owner

		// Get repository.
		repo, err := repo_model.GetRepositoryByName(owner.ID, repoName)
		if err != nil {
			if repo_model.IsErrRepoNotExist(err) {
				log.Warn("Repo with name - %s, not exist", repoName)
				ctx.Error(http.StatusNotFound, "LookupRepoRedirect", repo_model.ErrRepositoryNotExist{Name: repoName})
				return
			} else {
				log.Error("Err: get repository by name: %v", err)
				ctx.Error(http.StatusNotFound, "GetRepositoryByName", err)
				return
			}
		}

		repo.Owner = owner
		ctx.Repo.Repository = repo

		if ctx.Doer != nil && ctx.Doer.ID == user_model.ActionsUserID {
			taskID := ctx.Data["ActionsTaskID"].(int64)
			task, err := actions_model.GetTaskByID(ctx, taskID)
			if err != nil {
				ctx.Error(http.StatusInternalServerError, "actions_model.GetTaskByID", err)
				return
			}
			if task.RepoID != repo.ID {
				ctx.NotFound()
				return
			}

			if task.IsForkPullRequest {
				ctx.Repo.Permission.AccessMode = perm.AccessModeRead
			} else {
				ctx.Repo.Permission.AccessMode = perm.AccessModeWrite
			}

			if err := ctx.Repo.Repository.LoadUnits(ctx); err != nil {
				log.Error("Err: load units: %v", err)
				ctx.Error(http.StatusInternalServerError, "LoadUnits", err)
				return
			}
			ctx.Repo.Permission.Units = ctx.Repo.Repository.Units
			ctx.Repo.Permission.UnitsMode = make(map[unit.Type]perm.AccessMode)
			for _, u := range ctx.Repo.Repository.Units {
				ctx.Repo.Permission.UnitsMode[u.Type] = ctx.Repo.Permission.AccessMode
			}
		} else {
			ctx.Repo.Permission, err = access_model.GetUserRepoPermission(ctx, repo, ctx.Doer)
			if err != nil {
				log.Error("Err: get user repo permission: %v", err)
				ctx.Error(http.StatusInternalServerError, "GetUserRepoPermission", err)
				return
			}
		}

		if !ctx.Repo.HasAccess() {
			ctx.NotFound()
			return
		}
	}
}

func tenantAssigment() func(ctx *context.APIContext) {
	return func(ctx *context.APIContext) {
		tenantID := ctx.Params("tenant")
		if tenantID == "" {
			log.Warn("Err: tenant required")
			ctx.Error(http.StatusBadRequest, "Get tenant", "Err: tenant required")
			return
		}
		tenantUUID, err := uuid.Parse(tenantID)
		if err != nil {
			log.Warn("Err: invalid UUID format")
			ctx.Error(http.StatusBadRequest, "Err: invalid UUID", fmt.Errorf("Err: invalid UUID format"))
			return
		}
		targetTenant, err := tenant2.GetTenantByID(ctx, tenantUUID.String())
		if err != nil {
			if tenant2.IsErrorTenantNotExists(err) {
				log.Warn("Err: tenant with uuid - %s, not exist", tenantID)
				ctx.Error(http.StatusNotFound, "Get tenant", err)
				return
			}
			log.Error("Err: tenant with uuid - %s: %v", tenantID, err)
			ctx.Error(http.StatusBadRequest, "Get tenant", err)
			return
		}
		scTenantOrg, err := tenant2.GetTenantOrganizations(ctx, targetTenant.ID)
		if err != nil {
			log.Error("Err: get tenant organizations: %v", err)
			ctx.Error(http.StatusBadRequest, "Get tenant", err)
			return
		}
		var counter int
		for _, v := range scTenantOrg {
			if (v.TenantID == targetTenant.ID) && (v.OrganizationID == ctx.Repo.Owner.ID) {
				counter++
				ctx.Tenant = v
			}
		}
		if counter == 0 {
			log.Warn("Err: project not correspond to tenant")
			ctx.Error(http.StatusNotFound, "Get tenant", "Err: project not correspond to tenant")
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
		group.Add(auth.NewIAMProxy(privileger, engine))
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
			log.Warn("token does not have required scope - %s", requiredScope)
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
