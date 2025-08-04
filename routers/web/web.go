// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package web

import (
	gocontext "context"
	"crypto/tls"
	"net/http"

	"code.gitea.io/gitea/models/default_reviewers/default_reviewers_db"
	"code.gitea.io/gitea/models/external_metric_counter/external_metric_counter_db"
	"code.gitea.io/gitea/models/git/protected_branch/protected_branch_db"
	"code.gitea.io/gitea/models/review_settings/review_settings_db"
	"code.gitea.io/gitea/routers/api/v3/models"

	"code.gitea.io/gitea/models/internal_metric_counter/internal_metric_counter_db"
	"code.gitea.io/gitea/models/organization/custom"

	"code.gitea.io/gitea/models/code_hub_counter_task/code_hub_counter_task_db"
	auth_service "code.gitea.io/gitea/services/auth"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/git/protected_branch/convert"
	"code.gitea.io/gitea/models/perm"
	"code.gitea.io/gitea/models/pull_request_sender/task_tracker_db"
	"code.gitea.io/gitea/models/repo_marks"
	"code.gitea.io/gitea/models/repo_marks/marks"
	"code.gitea.io/gitea/models/repo_marks/repo_marks_db"
	"code.gitea.io/gitea/models/role_model"
	"code.gitea.io/gitea/models/role_model/casbin_role_manager"
	"code.gitea.io/gitea/models/role_model/custom_casbin_role_manager"
	"code.gitea.io/gitea/models/unit"
	"code.gitea.io/gitea/models/unit_links/unit_links_db"
	"code.gitea.io/gitea/models/user/user_db"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/gitnamesparser/branch"
	"code.gitea.io/gitea/modules/gitnamesparser/pullrequest"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/metrics"
	"code.gitea.io/gitea/modules/mtls"
	"code.gitea.io/gitea/modules/public"
	_ "code.gitea.io/gitea/modules/session" // to registers all internal adapters
	"code.gitea.io/gitea/modules/setting"
	setting_mtls "code.gitea.io/gitea/modules/setting/mtls"
	"code.gitea.io/gitea/modules/storage"
	"code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/modules/templates"
	"code.gitea.io/gitea/modules/validation"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/modules/web/routing"
	"code.gitea.io/gitea/routers/common"
	"code.gitea.io/gitea/routers/private/code_hub_counter"
	"code.gitea.io/gitea/routers/private/pull_request_reader"
	"code.gitea.io/gitea/routers/private/pull_request_task_creator"
	"code.gitea.io/gitea/routers/private/task_tracker_client"
	"code.gitea.io/gitea/routers/private/unit_linker"
	"code.gitea.io/gitea/routers/web/admin"
	"code.gitea.io/gitea/routers/web/auth"
	"code.gitea.io/gitea/routers/web/devtest"
	"code.gitea.io/gitea/routers/web/events"
	"code.gitea.io/gitea/routers/web/explore"
	"code.gitea.io/gitea/routers/web/feed"
	"code.gitea.io/gitea/routers/web/healthcheck"
	"code.gitea.io/gitea/routers/web/misc"
	"code.gitea.io/gitea/routers/web/org"
	org_setting "code.gitea.io/gitea/routers/web/org/setting"
	"code.gitea.io/gitea/routers/web/repo"
	"code.gitea.io/gitea/routers/web/repo/actions"
	"code.gitea.io/gitea/routers/web/repo/issues"
	"code.gitea.io/gitea/routers/web/repo/protected_branch"
	form_converter "code.gitea.io/gitea/routers/web/repo/protected_branch/convert"
	"code.gitea.io/gitea/routers/web/repo/pulls"
	repo_setting "code.gitea.io/gitea/routers/web/repo/setting"
	"code.gitea.io/gitea/routers/web/unit_links"
	"code.gitea.io/gitea/routers/web/user"
	"code.gitea.io/gitea/routers/web/user/accesser/org_accesser"
	"code.gitea.io/gitea/routers/web/user/accesser/repo_accesser"
	"code.gitea.io/gitea/routers/web/user/accesser/user_accesser"
	"code.gitea.io/gitea/routers/web/user/repo_server"
	user_setting "code.gitea.io/gitea/routers/web/user/setting"
	"code.gitea.io/gitea/routers/web/user/setting/security"
	"code.gitea.io/gitea/routers/web/user/team_server"
	"code.gitea.io/gitea/routers/web/user/user_or_organization"
	"code.gitea.io/gitea/services/auth/iamprivileger"
	context_service "code.gitea.io/gitea/services/context"
	"code.gitea.io/gitea/services/custom_creator"
	"code.gitea.io/gitea/services/forms"
	"code.gitea.io/gitea/services/lfs"
	protected_brancher "code.gitea.io/gitea/services/protected_branch"
	"code.gitea.io/gitea/services/user/user_manager"

	"gitea.com/go-chi/captcha"
	"github.com/NYTimes/gziphandler"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	// GzipMinSize represents min size to compress for the body size of response
	GzipMinSize = 1400
)

// CorsHandler return a http handler who set CORS options if enabled by config
func CorsHandler() func(next http.Handler) http.Handler {
	if setting.CORSConfig.Enabled {
		return cors.Handler(cors.Options{
			// Scheme:           setting.CORSConfig.Scheme, // FIXME: the cors middleware needs scheme option
			AllowedOrigins: setting.CORSConfig.AllowDomain,
			// setting.CORSConfig.AllowSubdomain // FIXME: the cors middleware needs allowSubdomain option
			AllowedMethods:   setting.CORSConfig.Methods,
			AllowCredentials: setting.CORSConfig.AllowCredentials,
			AllowedHeaders:   setting.CORSConfig.Headers,
			MaxAge:           int(setting.CORSConfig.MaxAge.Seconds()),
		})
	}

	return func(next http.Handler) http.Handler {
		return next
	}
}

// The OAuth2 plugin is expected to be executed first, as it must ignore the user id stored
// in the session (if there is a user id stored in session other plugins might return the user
// object for that id).
//
// The Session plugin is expected to be executed second, in order to skip authentication
// for users that have already signed in.
func buildAuthGroup() *auth_service.Group {
	group := auth_service.NewGroup(
		&auth_service.OAuth2{}, // FIXME: this should be removed and only applied in download and oauth related routers
		&auth_service.Basic{},  // FIXME: this should be removed and only applied in download and git/lfs routers
		&auth_service.Session{},
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

	if setting.Service.EnableReverseProxyAuth {
		group.Add(&auth_service.ReverseProxy{})
	}

	specialAdd(group)

	return group
}

func ctxDataSet(args ...any) func(ctx *context.Context) {
	return func(ctx *context.Context) {
		for i := 0; i < len(args); i += 2 {
			ctx.Data[args[i].(string)] = args[i+1]
		}
	}
}

// Routes returns all web routes
func Routes(ctx gocontext.Context) *web.Route {
	routes := web.NewRoute()

	routes.Head("/", misc.DummyOK) // for health check - doesn't need to be passed through gzip handler
	routes.RouteMethods("/webhooks/sonarqube", "GET, POST", WebhookSonarQube)
	routes.RouteMethods("/assets/*", "GET, HEAD", CorsHandler(), public.AssetsHandlerFunc("/assets/"))
	routes.RouteMethods("/avatars/*", "GET, HEAD", storageHandler(setting.Avatar.Storage, "avatars", storage.Avatars))
	routes.RouteMethods("/repo-avatars/*", "GET, HEAD", storageHandler(setting.RepoAvatar.Storage, "repo-avatars", storage.RepoAvatars))
	routes.RouteMethods("/apple-touch-icon.png", "GET, HEAD", misc.StaticRedirect("/assets/img/apple-touch-icon.png"))
	routes.RouteMethods("/apple-touch-icon-precomposed.png", "GET, HEAD", misc.StaticRedirect("/assets/img/apple-touch-icon.png"))
	routes.RouteMethods("/favicon.ico", "GET, HEAD", misc.StaticRedirect("/assets/img/favicon.png"))

	_ = templates.HTMLRenderer()

	var mid []any

	if setting.EnableGzip {
		h, err := gziphandler.GzipHandlerWithOpts(gziphandler.MinSize(GzipMinSize))
		if err != nil {
			log.Fatal("GzipHandlerWithOpts failed: %v", err)
		}
		mid = append(mid, h)
	}

	if setting.Service.EnableCaptcha {
		// The captcha http.Handler should only fire on /captcha/* so we can just mount this on that url
		routes.RouteMethods("/captcha/*", "GET,HEAD", append(mid, captcha.Captchaer(context.GetImageCaptcha()))...)
	}

	if setting.HasRobotsTxt {
		routes.Get("/robots.txt", append(mid, misc.RobotsTxt)...)
	}

	if setting.Metrics.Enabled {
		prometheus.MustRegister(metrics.NewCollector())
		routes.Get("/metrics", append(mid, Metrics)...)
	}

	routes.Get("/ssh_info", misc.SSHInfo)
	routes.Get("/api/healthz", healthcheck.Check)

	mid = append(mid, common.Sessioner(), context.Contexter())

	group := buildAuthGroup()
	if err := group.Init(ctx); err != nil {
		log.Error("Could not initialize '%s' auth method, error: %s", group.Name(), err)
	}

	// Get user from session if logged in.
	mid = append(mid, auth_service.Auth(group))

	// GetHead allows a HEAD request redirect to GET if HEAD method is not defined for that route
	mid = append(mid, middleware.GetHead)

	if setting.API.EnableSwagger {
		// Note: The route is here but no in API routes because it renders a web page
		routes.Get("/api/swagger_v2", append(mid, misc.SwaggerV2)...)
		routes.Get("/api/swagger_v3", append(mid, misc.SwaggerV3)...)
		routes.Get("/api/swagger", append(mid, misc.Swagger)...) // Render V1 by default
	}

	// TODO: These really seem like things that could be folded into Contexter or as helper functions
	mid = append(mid, user.GetNotificationCount)
	mid = append(mid, repo.GetActiveStopwatch)
	mid = append(mid, goGet)

	others := web.NewRoute()
	others.Use(mid...)
	if setting.UI.EnableClassic {
		registerRoutes(others)
	} else {
		log.Warn("Classic routes are disabled")
	}
	registerGitRoutes(others)
	routes.Mount("", others)
	return routes
}

// registerRoutes register routes
func registerRoutes(m *web.Route) {
	ctx := gocontext.Background()
	dbEngine := db.GetEngine(ctx)
	reqSignIn := auth_service.VerifyAuthWithOptions(&auth_service.VerifyOptions{SignInRequired: true})
	reqSignOut := auth_service.VerifyAuthWithOptions(&auth_service.VerifyOptions{SignOutRequired: true})
	// TODO: rename them to "optSignIn", which means that the "sign-in" could be optional, depends on the VerifyOptions (RequireSignInView)
	ignSignIn := auth_service.VerifyAuthWithOptions(&auth_service.VerifyOptions{SignInRequired: setting.Service.RequireSignInView})
	ignExploreSignIn := auth_service.VerifyAuthWithOptions(&auth_service.VerifyOptions{SignInRequired: setting.Service.RequireSignInView || setting.Service.Explore.RequireSigninView})
	ignSignInAndCsrf := auth_service.VerifyAuthWithOptions(&auth_service.VerifyOptions{DisableCSRF: true})
	validation.AddBindingRules()
	reqOneWork := context.CanEnableOneWork()
	codeHubCounterDB := internal_metric_counter_db.New(dbEngine)
	externalCounterDB := external_metric_counter_db.New(dbEngine)
	repoMarksDB := repo_marks_db.NewRepoMarksDB(dbEngine)
	codeHubMark := marks.GetCodeHubMark(setting.CodeHub.CodeHubMarkLabelName)
	exploreRepoServer := explore.New(codeHubCounterDB, externalCounterDB, repoMarksDB, []repo_marks.RepoMark{codeHubMark}, setting.CodeHub.CodeHubMetricEnabled, setting.CodeHub.CodeHubMarkEnabled)
	adminRepoServer := admin.New(exploreRepoServer)

	linkAccountEnabled := func(ctx *context.Context) {
		if !setting.Service.EnableOpenIDSignIn && !setting.Service.EnableOpenIDSignUp && !setting.OAuth2.Enable {
			ctx.Error(http.StatusForbidden)
			return
		}
	}

	openIDSignInEnabled := func(ctx *context.Context) {
		if !setting.Service.EnableOpenIDSignIn {
			ctx.Error(http.StatusForbidden)
			return
		}
	}

	openIDSignUpEnabled := func(ctx *context.Context) {
		if !setting.Service.EnableOpenIDSignUp {
			ctx.Error(http.StatusForbidden)
			return
		}
	}

	reqMilestonesDashboardPageEnabled := func(ctx *context.Context) {
		if !setting.Service.ShowMilestonesDashboardPage {
			ctx.Error(http.StatusForbidden)
			return
		}
	}

	// webhooksEnabled requires webhooks to be enabled by admin.
	webhooksEnabled := func(ctx *context.Context) {
		if setting.DisableWebhooks {
			ctx.Error(http.StatusForbidden)
			return
		}
	}

	federationEnabled := func(ctx *context.Context) {
		if !setting.Federation.Enabled {
			ctx.Error(http.StatusNotFound)
			return
		}
	}

	dlSourceEnabled := func(ctx *context.Context) {
		if setting.Repository.DisableDownloadSourceArchives {
			ctx.Error(http.StatusNotFound)
			return
		}
	}

	sitemapEnabled := func(ctx *context.Context) {
		if !setting.Other.EnableSitemap {
			ctx.Error(http.StatusNotFound)
			return
		}
	}

	packagesEnabled := func(ctx *context.Context) {
		if !setting.Packages.Enabled {
			ctx.Error(http.StatusForbidden)
			return
		}
	}

	feedEnabled := func(ctx *context.Context) {
		if !setting.Other.EnableFeed {
			ctx.Error(http.StatusNotFound)
			return
		}
	}

	privilegeManagementEnabled := func() func(ctx *context.Context) {
		return func(ctx *context.Context) {
			if !setting.SourceControl.InternalPrivilegeManagement && setting.SourceControl.TenantWithRoleModeEnabled {
				ctx.NotFound("", nil)
			}
		}
	}

	projectCreateEnabled := func() func(ctx *context.Context) {
		return func(ctx *context.Context) {
			if !setting.SourceControl.InternalProjectCreate && setting.SourceControl.TenantWithRoleModeEnabled || !ctx.Doer.IsActive {
				ctx.NotFound("", nil)
			}
		}
	}

	reqUnitAccess := func(unitType unit.Type, accessMode perm.AccessMode) func(ctx *context.Context) {
		return func(ctx *context.Context) {
			// если у нас включена ролевая модель SourceControl, то reqUnitAccess проверка пропускается
			if setting.SourceControl.TenantWithRoleModeEnabled {
				return
			}
			if unitType.UnitGlobalDisabled() {
				ctx.NotFound(unitType.String(), nil)
				return
			}

			if ctx.ContextUser == nil {
				ctx.NotFound(unitType.String(), nil)
				return
			}

			if ctx.ContextUser.IsOrganization() {
				if ctx.Org.Organization.UnitPermission(ctx, ctx.Doer, unitType) < accessMode {
					ctx.NotFound(unitType.String(), nil)
					return
				}
			}
		}
	}

	disableFunctionalByMultiTenantsDisabled := func() func(ctx *context.Context) {
		return func(ctx *context.Context) {
			if !setting.SourceControl.MultiTenantEnabled {
				ctx.NotFound("", nil)
			}
		}
	}

	disableFunctionalByExternalPreReceiveHookDisabled := func() func(ctx *context.Context) {
		return func(ctx *context.Context) {
			if !setting.SourceControl.ExternalPreReceiveHookEnabled {
				ctx.NotFound("", nil)
			}
		}
	}

	addWebhookAddRoutes := func() {
		m.Get("/{type}/new", repo.WebhooksNew)
		m.Post("/sourcecontrol/new", web.Bind(forms.NewWebhookForm{}), repo.SCHooksNewPost)
		m.Post("/gogs/new", web.Bind(forms.NewGogshookForm{}), repo.GogsHooksNewPost)
		m.Post("/slack/new", web.Bind(forms.NewSlackHookForm{}), repo.SlackHooksNewPost)
		m.Post("/discord/new", web.Bind(forms.NewDiscordHookForm{}), repo.DiscordHooksNewPost)
		m.Post("/dingtalk/new", web.Bind(forms.NewDingtalkHookForm{}), repo.DingtalkHooksNewPost)
		m.Post("/telegram/new", web.Bind(forms.NewTelegramHookForm{}), repo.TelegramHooksNewPost)
		m.Post("/matrix/new", web.Bind(forms.NewMatrixHookForm{}), repo.MatrixHooksNewPost)
		m.Post("/msteams/new", web.Bind(forms.NewMSTeamsHookForm{}), repo.MSTeamsHooksNewPost)
		m.Post("/feishu/new", web.Bind(forms.NewFeishuHookForm{}), repo.FeishuHooksNewPost)
		m.Post("/wechatwork/new", web.Bind(forms.NewWechatWorkHookForm{}), repo.WechatworkHooksNewPost)
		m.Post("/packagist/new", web.Bind(forms.NewPackagistHookForm{}), repo.PackagistHooksNewPost)
	}

	addWebhookEditRoutes := func() {
		m.Post("/sourcecontrol/{id}", web.Bind(forms.NewWebhookForm{}), repo.SCHooksEditPost)
		m.Post("/gogs/{id}", web.Bind(forms.NewGogshookForm{}), repo.GogsHooksEditPost)
		m.Post("/slack/{id}", web.Bind(forms.NewSlackHookForm{}), repo.SlackHooksEditPost)
		m.Post("/discord/{id}", web.Bind(forms.NewDiscordHookForm{}), repo.DiscordHooksEditPost)
		m.Post("/dingtalk/{id}", web.Bind(forms.NewDingtalkHookForm{}), repo.DingtalkHooksEditPost)
		m.Post("/telegram/{id}", web.Bind(forms.NewTelegramHookForm{}), repo.TelegramHooksEditPost)
		m.Post("/matrix/{id}", web.Bind(forms.NewMatrixHookForm{}), repo.MatrixHooksEditPost)
		m.Post("/msteams/{id}", web.Bind(forms.NewMSTeamsHookForm{}), repo.MSTeamsHooksEditPost)
		m.Post("/feishu/{id}", web.Bind(forms.NewFeishuHookForm{}), repo.FeishuHooksEditPost)
		m.Post("/wechatwork/{id}", web.Bind(forms.NewWechatWorkHookForm{}), repo.WechatworkHooksEditPost)
		m.Post("/packagist/{id}", web.Bind(forms.NewPackagistHookForm{}), repo.PackagistHooksEditPost)
	}

	addSettingsSecretsRoutes := func() {
		m.Group("/secrets", func() {
			m.Get("", repo_setting.Secrets)
			m.Post("", web.Bind(forms.AddSecretForm{}), repo_setting.SecretsPost)
			m.Post("/delete", repo_setting.SecretsDelete)
		})
	}

	addSettingsRunnersRoutes := func() {
		m.Group("/runners", func() {
			m.Get("", repo_setting.Runners)
			m.Combo("/{runnerid}").Get(repo_setting.RunnersEdit).
				Post(web.Bind(forms.EditRunnerForm{}), repo_setting.RunnersEditPost)
			m.Post("/{runnerid}/delete", repo_setting.RunnerDeletePost)
			m.Get("/reset_registration_token", repo_setting.ResetRunnerRegistrationToken)
		})
	}
	casbinManager := casbin_role_manager.New()

	userDB := user_db.NewUserDB()
	userManager := user_manager.NewUserManager(userDB)

	protectedBranchDB := protected_branch_db.NewProtectedBranchDB(dbEngine)

	casbinCustomManager := custom_casbin_role_manager.NewManager(role_model.GetSecurityEnforcer())
	orgAccesser := org_accesser.New(casbinManager)
	userAccesser := user_accesser.New()
	repoAccessor := repo_accesser.NewRepoAccessor(casbinCustomManager)
	customCreator := custom_creator.NewCustomCreator(casbinCustomManager)
	protectedBranchGetter := protected_brancher.NewProtectedBranchGetter()
	protectedBranchChecker := protected_brancher.NewProtectedBranchChecker()
	protectedBranchMerger := protected_brancher.NewProtectedBranchMerger()
	protecteBranchUpdater := protected_brancher.NewProtectedBranchUpdater()
	protectedBranchManager := protected_brancher.NewProtectedBranchManager(protectedBranchGetter, protectedBranchChecker, protectedBranchMerger, protecteBranchUpdater, protectedBranchDB)

	auditConverter := convert.NewAuditConverter(userManager)
	formConverter := form_converter.NewFormConverter()
	protectedBranchServer := protected_branch.NewServer(protectedBranchManager, auditConverter, formConverter)

	userOrOrgServer := user_or_organization.NewServer(
		orgAccesser,
		userAccesser,
		setting.SourceControl.TenantWithRoleModeEnabled,
		repoAccessor,
	)
	repoServer := repo_server.NewRepoServer(orgAccesser, userAccesser, repoAccessor, protectedBranchManager)
	teamServer := team_server.NewTeamServer(orgAccesser, userAccesser, repoAccessor, customCreator)
	customDB := custom.NewCustomDB(dbEngine)

	branchesServer := repo.New(casbinManager, customDB, teamServer)
	taskDB := code_hub_counter_task_db.New(dbEngine)
	taskCreator := code_hub_counter.NewTaskCreator(taskDB, setting.CodeHub.CodeHubMetricEnabled)
	processedMarks := []repo_marks.RepoMark{codeHubMark}
	repoServerCodeHub := repo.NewRepoServer(taskCreator, repoMarksDB, processedMarks, setting.CodeHub.CodeHubMarkEnabled)

	privileges := org.NewServerForPrivileges(casbinManager, customDB, teamServer)
	// FIXME: not all routes need go through same middleware.
	// Especially some AJAX requests, we can reduce middleware number to improve performance.
	// Routers.
	// for health check
	m.Get("/", userOrOrgServer.Home)
	m.Get("/sitemap.xml", sitemapEnabled, ignExploreSignIn, HomeSitemap)
	m.Group("/.well-known", func() {
		m.Get("/openid-configuration", auth.OIDCWellKnown)
		m.Group("", func() {
			m.Get("/nodeinfo", NodeInfoLinks)
			m.Get("/webfinger", WebfingerQuery)
		}, federationEnabled)
		m.Get("/change-password", func(ctx *context.Context) {
			ctx.Redirect(setting.AppSubURL + "/user/settings/account")
		})
	})

	m.Group("/explore", func() {
		m.Get("", func(ctx *context.Context) {
			ctx.Redirect(setting.AppSubURL + "/explore/repos")
		})
		m.Get("/repos", exploreRepoServer.Repos)
		m.Get("/repos/sitemap-{idx}.xml", sitemapEnabled, exploreRepoServer.Repos)
		m.Get("/users", explore.Users)
		m.Get("/users/sitemap-{idx}.xml", sitemapEnabled, explore.Users)
		m.Get("/organizations", explore.Organizations)
		m.Get("/code", reqUnitAccess(unit.TypeCode, perm.AccessModeRead), context.RequireRepoReadPermission(), explore.Code)
		m.Get("/topics/search", explore.TopicSearch)
	}, ignExploreSignIn)
	m.Group("/issues", func() {
		m.Get("", repoServer.Issues)
		m.Get("/search", repo.SearchIssues)
	}, reqSignIn)

	m.Get("/pulls", reqSignIn, repoServer.Pulls)
	m.Get("/milestones", reqOneWork, reqSignIn, reqMilestonesDashboardPageEnabled, user.Milestones)

	// ***** START: User *****
	m.Group("/user", func() {
		m.Get("/login", auth.SignIn)
		m.Post("/login", web.Bind(forms.SignInForm{}), auth.SignInPost)
		m.Group("", func() {
			m.Combo("/login/openid").
				Get(auth.SignInOpenID).
				Post(web.Bind(forms.SignInOpenIDForm{}), auth.SignInOpenIDPost)
		}, openIDSignInEnabled)
		m.Group("/openid", func() {
			m.Combo("/connect").
				Get(auth.ConnectOpenID).
				Post(web.Bind(forms.ConnectOpenIDForm{}), auth.ConnectOpenIDPost)
			m.Group("/register", func() {
				m.Combo("").
					Get(auth.RegisterOpenID, openIDSignUpEnabled).
					Post(web.Bind(forms.SignUpOpenIDForm{}), auth.RegisterOpenIDPost)
			}, openIDSignUpEnabled)
		}, openIDSignInEnabled)
		m.Get("/sign_up", auth.SignUp)
		m.Post("/sign_up", web.Bind(forms.RegisterForm{}), auth.SignUpPost)
		m.Get("/link_account", linkAccountEnabled, auth.LinkAccount)
		m.Post("/link_account_signin", linkAccountEnabled, web.Bind(forms.SignInForm{}), auth.LinkAccountPostSignIn)
		m.Post("/link_account_signup", linkAccountEnabled, web.Bind(forms.RegisterForm{}), auth.LinkAccountPostRegister)
		m.Group("/two_factor", func() {
			m.Get("", auth.TwoFactor)
			m.Post("", web.Bind(forms.TwoFactorAuthForm{}), auth.TwoFactorPost)
			m.Get("/scratch", auth.TwoFactorScratch)
			m.Post("/scratch", web.Bind(forms.TwoFactorScratchAuthForm{}), auth.TwoFactorScratchPost)
		})
		m.Group("/webauthn", func() {
			m.Get("", auth.WebAuthn)
			m.Get("/assertion", auth.WebAuthnLoginAssertion)
			m.Post("/assertion", auth.WebAuthnLoginAssertionPost)
		})
	}, reqSignOut)

	m.Any("/user/events", routing.MarkLongPolling, events.Events)

	m.Group("/login/oauth", func() {
		m.Get("/authorize", web.Bind(forms.AuthorizationForm{}), auth.AuthorizeOAuth)
		m.Post("/grant", web.Bind(forms.GrantApplicationForm{}), auth.GrantApplicationOAuth)
		// TODO manage redirection
		m.Post("/authorize", web.Bind(forms.AuthorizationForm{}), auth.AuthorizeOAuth)
	}, ignSignInAndCsrf, reqSignIn)
	m.Get("/login/oauth/userinfo", ignSignInAndCsrf, auth.InfoOAuth)
	m.Post("/login/oauth/access_token", CorsHandler(), web.Bind(forms.AccessTokenForm{}), ignSignInAndCsrf, auth.AccessTokenOAuth)
	m.Get("/login/oauth/keys", ignSignInAndCsrf, auth.OIDCKeys)
	m.Post("/login/oauth/introspect", CorsHandler(), web.Bind(forms.IntrospectTokenForm{}), ignSignInAndCsrf, auth.IntrospectOAuth)

	m.Group("/user/settings", func() {
		m.Get("", user_setting.Profile)
		m.Post("", reqOneWork, web.Bind(forms.UpdateProfileForm{}), user_setting.ProfilePost)
		m.Get("/change_password", auth.MustChangePassword)
		m.Post("/change_password", web.Bind(forms.MustChangePasswordForm{}), auth.MustChangePasswordPost)
		m.Post("/avatar", reqOneWork, web.Bind(forms.AvatarForm{}), user_setting.AvatarPost)
		m.Post("/avatar/delete", reqOneWork, user_setting.DeleteAvatar)
		m.Group("/account", func() {
			m.Combo("").Get(user_setting.Account).Post(web.Bind(forms.ChangePasswordForm{}), user_setting.AccountPost)
			m.Post("/email", web.Bind(forms.AddEmailForm{}), user_setting.EmailPost)
			m.Post("/email/delete", user_setting.DeleteEmail)
			m.Post("/delete", reqOneWork, user_setting.DeleteAccount)
		})
		m.Group("/appearance", func() {
			m.Get("", user_setting.Appearance)
			m.Post("/language", web.Bind(forms.UpdateLanguageForm{}), user_setting.UpdateUserLang)
			m.Post("/hidden_comments", user_setting.UpdateUserHiddenComments)
			m.Post("/theme", web.Bind(forms.UpdateThemeForm{}), user_setting.UpdateUIThemePost)
		})
		m.Group("/security", func() {
			m.Get("", reqOneWork, security.Security)
			m.Group("/two_factor", func() {
				m.Post("/regenerate_scratch", security.RegenerateScratchTwoFactor)
				m.Post("/disable", security.DisableTwoFactor)
				m.Get("/enroll", security.EnrollTwoFactor)
				m.Post("/enroll", web.Bind(forms.TwoFactorAuthForm{}), security.EnrollTwoFactorPost)
			})
			m.Group("/webauthn", func() {
				m.Post("/request_register", web.Bind(forms.WebauthnRegistrationForm{}), security.WebAuthnRegister)
				m.Post("/register", security.WebauthnRegisterPost)
				m.Post("/delete", web.Bind(forms.WebauthnDeleteForm{}), security.WebauthnDelete)
			})
			m.Group("/openid", func() {
				m.Post("", web.Bind(forms.AddOpenIDForm{}), security.OpenIDPost)
				m.Post("/delete", security.DeleteOpenID)
				m.Post("/toggle_visibility", security.ToggleOpenIDVisibility)
			}, openIDSignInEnabled)
			m.Post("/account_link", linkAccountEnabled, security.DeleteAccountLink)
		})
		m.Group("/applications/oauth2", func() {
			m.Get("/{id}", user_setting.OAuth2ApplicationShow)
			m.Post("/{id}", web.Bind(forms.EditOAuth2ApplicationForm{}), user_setting.OAuthApplicationsEdit)
			m.Post("/{id}/regenerate_secret", user_setting.OAuthApplicationsRegenerateSecret)
			m.Post("", web.Bind(forms.EditOAuth2ApplicationForm{}), user_setting.OAuthApplicationsPost)
			m.Post("/{id}/delete", user_setting.DeleteOAuth2Application)
			m.Post("/{id}/revoke/{grantId}", user_setting.RevokeOAuth2Grant)
		})
		m.Combo("/applications").Get(user_setting.Applications).
			Post(web.Bind(forms.NewAccessTokenForm{}), user_setting.ApplicationsPost)
		m.Post("/applications/delete", user_setting.DeleteApplication)
		m.Combo("/keys").Get(user_setting.Keys).
			Post(web.Bind(forms.AddKeyForm{}), user_setting.KeysPost)
		m.Post("/keys/delete", user_setting.DeleteKey)
		m.Group("/packages", func() {
			m.Get("", user_setting.Packages)
			m.Group("/rules", func() {
				m.Group("/add", func() {
					m.Get("", user_setting.PackagesRuleAdd)
					m.Post("", web.Bind(forms.PackageCleanupRuleForm{}), user_setting.PackagesRuleAddPost)
				})
				m.Group("/{id}", func() {
					m.Get("", user_setting.PackagesRuleEdit)
					m.Post("", web.Bind(forms.PackageCleanupRuleForm{}), user_setting.PackagesRuleEditPost)
					m.Get("/preview", user_setting.PackagesRulePreview)
				})
			})
			m.Group("/cargo", func() {
				m.Post("/initialize", user_setting.InitializeCargoIndex)
				m.Post("/rebuild", user_setting.RebuildCargoIndex)
			})
			m.Post("/chef/regenerate_keypair", user_setting.RegenerateChefKeyPair)
		}, packagesEnabled)

		m.Group("/actions", func() {
			m.Get("", user_setting.RedirectToDefaultSetting)
			addSettingsSecretsRoutes()
		}, actions.MustEnableActions)

		m.Get("/organization", user_setting.Organization)
		m.Get("/repos", repoServer.Repos)
		m.Post("/repos/unadopted", user_setting.AdoptOrDeleteRepository)

		m.Group("/hooks", func() {
			m.Get("", user_setting.Webhooks)
			m.Post("/delete", user_setting.DeleteWebhook)
			addWebhookAddRoutes()
			m.Group("/{id}", func() {
				m.Get("", repo.WebHooksEdit)
				m.Post("/replay/{uuid}", repo.ReplayWebhook)
			})
			addWebhookEditRoutes()
		}, webhooksEnabled)
	}, reqSignIn, ctxDataSet("PageIsUserSettings", true, "AllThemes", setting.UI.Themes, "EnablePackages", setting.Packages.Enabled))

	m.Group("/user", func() {
		m.Get("/activate", auth.Activate)
		m.Post("/activate", auth.ActivatePost)
		m.Any("/activate_email", auth.ActivateEmail)
		m.Get("/avatar/{username}/{size}", user.AvatarByUserName)
		m.Get("/recover_account", auth.ResetPasswd)
		m.Post("/recover_account", auth.ResetPasswdPost)
		m.Get("/forgot_password", auth.ForgotPasswd)
		m.Post("/forgot_password", auth.ForgotPasswdPost)
		m.Post("/logout", auth.SignOut)
		m.Get("/task/{task}", reqSignIn, user.TaskStatus)
		m.Get("/stopwatches", reqSignIn, user.GetStopwatches)
		m.Get("/search", ignExploreSignIn, user.Search)
		m.Group("/oauth2", func() {
			m.Get("/{provider}", auth.SignInOAuth)
			m.Get("/{provider}/callback", auth.SignInOAuthCallback)
		})
	})
	// ***** END: User *****

	m.Get("/avatar/{hash}", user.AvatarByEmailHash)

	adminReq := auth_service.VerifyAuthWithOptions(&auth_service.VerifyOptions{SignInRequired: true, AdminRequired: true})

	// ***** START: Admin *****
	m.Group("/admin", func() {
		m.Get("", adminReq, admin.Dashboard)
		m.Post("", adminReq, web.Bind(forms.AdminDashboardForm{}), admin.DashboardPost)

		m.Group("/config", func() {
			m.Get("", admin.Config)
			m.Post("", admin.ChangeConfig)
			m.Post("/test_mail", admin.SendTestMail)
		})

		m.Group("/monitor", func() {
			m.Get("/cron", admin.CronTasks)
			m.Get("/stacktrace", admin.Stacktrace)
			m.Post("/stacktrace/cancel/{pid}", admin.StacktraceCancel)
			m.Get("/queue", admin.Queues)
			m.Group("/queue/{qid}", func() {
				m.Get("", admin.QueueManage)
				m.Post("/set", admin.QueueSet)
				m.Post("/remove-all-items", admin.QueueRemoveAllItems)
			})
			m.Get("/diagnosis", admin.MonitorDiagnosis)
		})

		m.Group("/users", func() {
			m.Get("", admin.Users)
			m.Combo("/new").Get(admin.NewUser).Post(web.Bind(forms.AdminCreateUserForm{}), admin.NewUserPost)
			m.Combo("/{userid}").Get(admin.EditUser).Post(web.Bind(forms.AdminEditUserForm{}), admin.EditUserPost)
			m.Post("/{userid}/delete", admin.DeleteUser)
			m.Post("/{userid}/avatar", web.Bind(forms.AvatarForm{}), admin.AvatarPost)
			m.Post("/{userid}/avatar/delete", admin.DeleteAvatar)
		})

		m.Group("/emails", func() {
			m.Get("", admin.Emails)
			m.Post("/activate", admin.ActivateEmail)
		}, reqOneWork)

		m.Group("/orgs", func() {
			m.Get("", admin.Organizations)
		}, reqOneWork)

		m.Group("/repos", func() {
			m.Get("", adminRepoServer.Repos)
			m.Combo("/unadopted").Get(admin.UnadoptedRepos).Post(admin.AdoptOrDeleteRepository)
			m.Post("/delete", admin.DeleteRepo)
		})

		m.Group("/packages", func() {
			m.Get("", admin.Packages)
			m.Post("/delete", admin.DeletePackageVersion)
		}, packagesEnabled)

		m.Group("/hooks", func() {
			m.Get("", admin.DefaultOrSystemWebhooks)
			m.Post("/delete", admin.DeleteDefaultOrSystemWebhook)
			m.Group("/{id}", func() {
				m.Get("", repo.WebHooksEdit)
				m.Post("/replay/{uuid}", repo.ReplayWebhook)
			})
			addWebhookEditRoutes()
		}, webhooksEnabled)

		m.Group("/{configType:default-hooks|system-hooks}", func() {
			addWebhookAddRoutes()
		})

		m.Group("/auths", func() {
			m.Get("", admin.Authentications)
			m.Combo("/new").Get(admin.NewAuthSource).Post(web.Bind(forms.AuthenticationForm{}), admin.NewAuthSourcePost)
			m.Combo("/{authid}").Get(admin.EditAuthSource).
				Post(web.Bind(forms.AuthenticationForm{}), admin.EditAuthSourcePost)
			m.Post("/{authid}/delete", admin.DeleteAuthSource)
		}, reqOneWork)

		m.Group("/notices", func() {
			m.Get("", admin.Notices)
			m.Post("/delete", admin.DeleteNotices)
			m.Post("/empty", admin.EmptyNotices)
		})

		m.Group("/applications", func() {
			m.Get("", admin.Applications)
			m.Post("/oauth2", web.Bind(forms.EditOAuth2ApplicationForm{}), admin.ApplicationsPost)
			m.Group("/oauth2/{id}", func() {
				m.Combo("").Get(admin.EditApplication).Post(web.Bind(forms.EditOAuth2ApplicationForm{}), admin.EditApplicationPost)
				m.Post("/regenerate_secret", admin.ApplicationsRegenerateSecret)
				m.Post("/delete", admin.DeleteApplication)
			})
		}, func(ctx *context.Context) {
			if !setting.OAuth2.Enable {
				ctx.Error(http.StatusForbidden)
				return
			}
		})

		m.Group("/actions", func() {
			m.Get("", admin.RedirectToDefaultSetting)
			addSettingsRunnersRoutes()
		})
		m.Group("/tenants", func() {
			m.Get("", admin.Tenants)
			m.Get("/list", admin.TenantsList)
			m.Post("/new", admin.NewTenant)
			m.Group("/{tenantid}", func() {
				m.Get("", admin.Tenant)
				m.Post("/delete", admin.DeleteTenant)
				m.Post("/edit", admin.EditTenant)
				m.Group("/projects", func() {
					m.Get("", admin.RelationTenantOrganizations)
					m.Post("/add", admin.AddTenantOrganization)
					m.Post("/{projectid}/delete", admin.DeleteTenantOrganization)
				})
			})
		}, reqOneWork, disableFunctionalByMultiTenantsDisabled())
		m.Group("/git_hooks/", func() {
			m.Combo("pre-receive").
				Get(admin.PreReceiveHook).
				Post(web.Bind(forms.PreReceiveHookForm{}), admin.PreReceiveHookPost).
				Delete(admin.PreReceiveHookDelete)
		}, disableFunctionalByExternalPreReceiveHookDisabled())
	}, adminReq, ctxDataSet("EnableOAuth2", setting.OAuth2.Enable, "EnablePackages", setting.Packages.Enabled))
	// ***** END: Admin *****

	//TODO: dependency initialization must be moved to upper level https://sberworks.ru/jira/browse/VCS-1300

	branchParser := branch.NewParser()
	pullRequestParser := pullrequest.NewParser()
	unitLinkDB := unit_links_db.NewUnitLinkDB(dbEngine)
	taskTrackerDb := task_tracker_db.NewTaskTrackerDB(dbEngine)
	pullRequestHeaderDB := pull_request_reader.NewReader(dbEngine)

	mtlsConfig := &tls.Config{} // initialize tls config with default values
	if setting_mtls.CheckMTLSConfigSecManEnabled(task_tracker_client.TaskTrackerClientName) {
		mtlsCerts := setting_mtls.GetMTLSCertsFromSecMan(task_tracker_client.TaskTrackerClientName, setting.NewGetterForSecMan())
		mtlsConfig = mtls.GenerateTlsConfigForMTLS(task_tracker_client.TaskTrackerClientName, mtlsCerts.Cert, mtlsCerts.CertKey, mtlsCerts.CaCerts)
	}
	taskTrackerClient := task_tracker_client.New(
		setting.TaskTracker.APIBaseURL,
		setting.TaskTracker.APIToken,
		&http.Client{Transport: &http.Transport{TLSClientConfig: mtlsConfig}},
	)

	pullRequestTaskCreator := pull_request_task_creator.NewPullRequestTaskCreator(taskTrackerDb)
	unitLinker := unit_linker.NewUnitLinker(
		branchParser,
		pullRequestParser,
		unitLinkDB,
		pullRequestHeaderDB,
		taskTrackerClient,
		setting.TaskTracker.UnitsValidationEnabled,
	)

	pullRequestsServer := pulls.NewServer(unitLinker, pullRequestTaskCreator, setting.TaskTracker.Enabled)
	issuesServer := issues.NewServer(unitLinker, pullRequestTaskCreator, setting.TaskTracker.Enabled)
	unitLinksServer := unit_links.NewServer(unitLinkDB, taskTrackerClient, setting.TaskTracker.UnitBaseURL, setting.TaskTracker.IAMUnitBaseURL)

	defaultReviewersDB := default_reviewers_db.New(dbEngine)
	reviewSettingsDB := review_settings_db.New(dbEngine)
	reviewSettingsServer := pulls.NewReviewSettingsServer(orgAccesser, defaultReviewersDB, reviewSettingsDB)

	m.Group("", func() {
		m.Get("/{username}", userOrOrgServer.GetUserOrOrganizationSubRoute)
		m.Get("/attachments/{uuid}", repo.GetAttachment)
	}, ignSignIn)

	// пользователь
	m.Post("/{username}", reqSignIn, context_service.UserAssignmentWeb(), user.Action)

	reqRepoAdmin := context.RequireRepoAdmin()
	reqRepoCodeWriter := context.RequireRepoWriter(unit.TypeCode)
	canEnableEditor := context.CanEnableEditor()
	reqRepoCodeReader := context.RequireRepoReader(unit.TypeCode)
	reqRepoReleaseWriter := context.RequireRepoWriter(unit.TypeReleases)
	reqRepoReleaseReader := context.RequireRepoReader(unit.TypeReleases)
	reqRepoWikiWriter := context.RequireRepoWriter(unit.TypeWiki)
	reqRepoIssueReader := context.RequireRepoReader(unit.TypeIssues)
	reqRepoPullsReader := context.RequireRepoReader(unit.TypePullRequests)
	reqRepoIssuesOrPullsWriter := context.RequireRepoWriterOr(unit.TypeIssues, unit.TypePullRequests)
	reqRepoIssuesOrPullsReader := context.RequireRepoReaderOr(unit.TypeIssues, unit.TypePullRequests)
	reqRepoProjectsReader := context.RequireRepoReader(unit.TypeProjects)
	reqRepoProjectsWriter := context.RequireRepoWriter(unit.TypeProjects)
	reqRepoActionsReader := context.RequireRepoReader(unit.TypeActions)
	reqRepoActionsWriter := context.RequireRepoWriter(unit.TypeActions)

	reqOrgOwn := context.RequireOrgPermission(role_model.OWN)
	reqOrgEdit := context.RequireOrgPermission(role_model.EDIT_PROJECT)
	reqRepoRead := context.RequireRepoReadPermission()
	reqRepoEdit := context.RequireRepoPermission(role_model.EDIT)
	reqRepoWrite := context.RequireRepoPermission(role_model.WRITE)

	conflictCustomEditAndCreatePR := context.RequireCustomPermission(role_model.EDIT, role_model.CreatePR)
	conflictCustomWriteAndCreatePR := context.RequireCustomPermission(role_model.WRITE, role_model.CreatePR)
	conflictCustomWriteAndChangeBranch := context.RequireCustomPermission(role_model.WRITE, role_model.ChangeBranch)
	conflictCustomWriteAndApprovePR := context.RequireCustomPermission(role_model.WRITE, role_model.ApprovePR)
	conflictCustomWriteAndMergePR := context.RequireCustomPermission(role_model.WRITE, role_model.MergePR)

	reqPackageAccess := func(accessMode perm.AccessMode) func(ctx *context.Context) {
		return func(ctx *context.Context) {
			// если у нас включена ролевая модель SourceControl, то reqPackageAccess проверка пропускается
			if setting.SourceControl.TenantWithRoleModeEnabled {
				return
			}
			if ctx.Package.AccessMode < accessMode && !ctx.IsUserSiteAdmin() {
				ctx.NotFound("", nil)
			}
		}
	}

	disableFunctionalByRoleModel := func() func(ctx *context.Context) {
		return func(ctx *context.Context) {
			if setting.SourceControl.TenantWithRoleModeEnabled {
				ctx.NotFound("", nil)
			}
		}
	}

	m.Group("/unit_links", func() {
		m.Post("", web.Bind(structs.GetUnitLinksWithDescriptionRequest{}), unitLinksServer.GetUnitLinksWithDescription)
	})

	disableFunctionalMigration := func() func(ctx *context.Context) {
		return func(ctx *context.Context) {
			ctx.NotFound("", nil)
		}
	}

	// ***** START: Organization *****
	m.Group("/org", func() {
		m.Group("/{org}", func() {
			m.Get("/members", org.Members)
		}, context.OrgAssignment())
	}, ignSignIn, disableFunctionalByRoleModel())

	m.Group("/org", func() {
		m.Group("", func() {
			m.Get("/create", org.Create)
			m.Post("/create", web.Bind(forms.CreateOrgForm{}), org.CreatePost)
		}, projectCreateEnabled(), reqOneWork)

		m.Group("/invite/{token}", func() {
			m.Get("", org.TeamInvite)
			m.Post("", org.TeamInvitePost)
		})

		m.Group("/{org}", func() {
			m.Get("/all/repositories", org.GetRepositoriesFromOrg)
			m.Get("/all/users", userOrOrgServer.GetUsersByOrgName)
			m.Get("/dashboard", userOrOrgServer.Dashboard)
			m.Get("/dashboard/{team}", userOrOrgServer.Dashboard)
			m.Get("/issues", repoServer.Issues)
			m.Get("/issues/{team}", repoServer.Issues)
			m.Get("/pulls", repoServer.Pulls)
			m.Get("/pulls/{team}", repoServer.Pulls)
			m.Get("/milestones", reqMilestonesDashboardPageEnabled, user.Milestones)
			m.Get("/milestones/{team}", reqMilestonesDashboardPageEnabled, user.Milestones)
			m.Post("/members/action/{action}", org.MembersAction)
			m.Get("/teams", reqOrgOwn, org.Teams)
		}, context.OrgAssignment(true, false, true))

		m.Group("/{org}", func() {
			m.Get("/teams/{team}", org.TeamMembers)
			m.Get("/teams/{team}/repositories", org.TeamRepositories)
			m.Post("/teams/{team}/action/{action}", teamServer.TeamsAction)
			m.Post("/teams/{team}/action/repo/{action}", web.Bind(forms.AddReposForTeam{}), org.TeamsRepoAction)
		}, context.OrgAssignment(true, false, true), reqOrgOwn)

		m.Group("/{org}", func() {
			m.Group("/teams", func() {
				//ONEWORKMIDDLEWARE
				m.Get("/new", org.NewTeam)
				m.Post("/new", web.Bind(forms.CreateTeamForm{}), teamServer.NewTeamPost)
				m.Get("/-/search", org.SearchTeam)
				m.Get("/{team}/edit", org.EditTeam)
				m.Post("/{team}/edit", web.Bind(forms.CreateTeamForm{}), teamServer.EditTeamPost)
				m.Post("/{team}/delete", teamServer.DeleteTeam)
			}, reqOrgOwn)

			m.Group("/privileges", func() {
				m.Post("/check", web.Bind(forms.CheckPrivilegesForm{}), org.CheckPrivileges)
				m.Post("/user", web.Bind(forms.UserPrivilegesForm{}), org.GetPrivilegesForUser)
			})
			m.Group("/privileges", func() {
				m.Get("", org.Privileges)
				m.Combo("/grant").
					Get(org.GrantPrivileges).
					Post(web.Bind(forms.GrantPrivilegesForm{}), org.GrantPrivilegesPost)
				m.Post("/revoke", web.Bind(forms.GrantPrivilegesForm{}), privileges.RevokePrivilegesPost)
			}, privilegeManagementEnabled(), reqOrgOwn)
			m.Group("/settings", func() {
				m.Combo("").
					Get(org.Settings).
					Post(web.Bind(forms.UpdateOrgSettingForm{}), reqOneWork, org.SettingsPost)
				m.Post("/avatar", web.Bind(forms.AvatarForm{}), org.SettingsAvatar)
				m.Post("/avatar/delete", org.SettingsDeleteAvatar)
				m.Group("/applications", func() {
					m.Get("", org.Applications)
					m.Post("/oauth2", web.Bind(forms.EditOAuth2ApplicationForm{}), org.OAuthApplicationsPost)
					m.Group("/oauth2/{id}", func() {
						m.Combo("").Get(org.OAuth2ApplicationShow).Post(web.Bind(forms.EditOAuth2ApplicationForm{}), org.OAuth2ApplicationEdit)
						m.Post("/regenerate_secret", org.OAuthApplicationsRegenerateSecret)
						m.Post("/delete", org.DeleteOAuth2Application)
					})
				}, func(ctx *context.Context) {
					if !setting.OAuth2.Enable {
						ctx.Error(http.StatusForbidden)
						return
					}
				})

				m.Group("/hooks", func() {
					m.Get("", org.Webhooks)
					m.Post("/delete", org.DeleteWebhook)
					addWebhookAddRoutes()
					m.Group("/{id}", func() {
						m.Get("", repo.WebHooksEdit)
						m.Post("/replay/{uuid}", repo.ReplayWebhook)
					})
					addWebhookEditRoutes()
				}, webhooksEnabled)

				m.Group("/labels", func() {
					m.Get("", org.RetrieveLabels, org.Labels)
					m.Post("/new", web.Bind(forms.CreateLabelForm{}), org.NewLabel)
					m.Post("/edit", web.Bind(forms.CreateLabelForm{}), org.UpdateLabel)
					m.Post("/delete", org.DeleteLabel)
					m.Post("/initialize", web.Bind(forms.InitializeLabelsForm{}), org.InitializeLabels)
				})

				m.Group("/actions", func() {
					m.Get("", org_setting.RedirectToDefaultSetting)
					addSettingsRunnersRoutes()
					addSettingsSecretsRoutes()
				}, actions.MustEnableActions)

				m.RouteMethods("/delete", "GET,POST", reqOneWork, org.SettingsDelete)

				m.Group("/packages", func() {
					m.Get("", org.Packages)
					m.Group("/rules", func() {
						m.Group("/add", func() {
							m.Get("", org.PackagesRuleAdd)
							m.Post("", web.Bind(forms.PackageCleanupRuleForm{}), org.PackagesRuleAddPost)
						})
						m.Group("/{id}", func() {
							m.Get("", org.PackagesRuleEdit)
							m.Post("", web.Bind(forms.PackageCleanupRuleForm{}), org.PackagesRuleEditPost)
							m.Get("/preview", org.PackagesRulePreview)
						})
					})
					m.Group("/cargo", func() {
						m.Post("/initialize", org.InitializeCargoIndex)
						m.Post("/rebuild", org.RebuildCargoIndex)
					})
				}, packagesEnabled)
			}, ctxDataSet("EnableOAuth2", setting.OAuth2.Enable, "EnablePackages", setting.Packages.Enabled, "PageIsOrgSettings", true), reqOrgEdit)
		}, context.OrgAssignment(true, true))
	}, reqSignIn)
	// ***** END: Organization *****

	// ***** START: Repository *****
	m.Group("/repo", func() {
		m.Get("/create", repo.Create)
		m.Post("/create", web.Bind(forms.CreateRepoForm{}), branchesServer.CreatePost)

		m.Group("/migrate", func() {
			m.Get("", repo.Migrate)
			m.Post("", web.Bind(forms.MigrateRepoForm{}), repo.MigratePost)
		}, disableFunctionalMigration())

		m.Group("/fork", func() {
			m.Combo("/{repoid}").Get(repo.Fork).
				Post(web.Bind(forms.CreateRepoForm{}), repo.ForkPost)
		}, context.RepoIDAssignment(), context.UnitTypes(), reqRepoCodeReader, reqRepoRead)
		m.Get("/search", repo.SearchRepo)
	}, reqSignIn)

	m.Group("/{username}/-", func() {
		if setting.Packages.Enabled {
			m.Group("/packages", func() {
				m.Get("", user.ListPackages)
				m.Group("/{type}/{name}", func() {
					m.Get("", user.RedirectToLastVersion)
					m.Get("/versions", user.ListPackageVersions)
					m.Group("/{version}", func() {
						m.Get("", user.ViewPackageVersion)
						m.Get("/files/{fileid}", user.DownloadPackageFile)
						m.Group("/settings", func() {
							m.Get("", user.PackageSettings)
							m.Post("", web.Bind(forms.PackageSettingForm{}), user.PackageSettingsPost)
						}, reqPackageAccess(perm.AccessModeWrite), reqRepoWrite)
					})
				})
			}, context.PackageAssignment(), reqPackageAccess(perm.AccessModeRead), reqRepoRead)
		}

		m.Group("/projects", func() {
			m.Group("", func() {
				m.Get("", org.Projects)
				m.Get("/{id}", org.ViewProject)
			}, reqUnitAccess(unit.TypeProjects, perm.AccessModeRead), context.RequireRepoReadPermission())
			m.Group("", func() { //nolint:dupl
				m.Get("/new", org.NewProject)
				m.Post("/new", web.Bind(forms.CreateProjectForm{}), org.NewProjectPost)
				m.Group("/{id}", func() {
					m.Post("", web.Bind(forms.EditProjectBoardForm{}), org.AddBoardToProjectPost)
					m.Post("/delete", org.DeleteProject)

					m.Get("/edit", org.EditProject)
					m.Post("/edit", web.Bind(forms.CreateProjectForm{}), org.EditProjectPost)
					m.Post("/{action:open|close}", org.ChangeProjectStatus)

					m.Group("/{boardID}", func() {
						m.Put("", web.Bind(forms.EditProjectBoardForm{}), org.EditProjectBoard)
						m.Delete("", org.DeleteProjectBoard)
						m.Post("/default", org.SetDefaultProjectBoard)
						m.Post("/unsetdefault", org.UnsetDefaultProjectBoard)

						m.Post("/move", org.MoveIssues)
					})
				})
			}, reqSignIn, reqUnitAccess(unit.TypeProjects, perm.AccessModeWrite), context.RequireRepoPermission(role_model.WRITE), func(ctx *context.Context) {
				if ctx.ContextUser.IsIndividual() && ctx.ContextUser.ID != ctx.Doer.ID {
					ctx.NotFound("NewProject", nil)
					return
				}
			})
		}, repo.MustEnableProjects)

		m.Group("", func() {
			m.Get("/code", user.CodeSearch)
		}, reqUnitAccess(unit.TypeCode, perm.AccessModeRead), reqRepoRead)
	}, ignSignIn, context_service.UserAssignmentWeb(), context.OrgAssignment()) // for "/{username}/-" (packages, projects, code)

	// ***** Release Attachment Download without Signin
	m.Get("/{username}/{reponame}/releases/download/{vTag}/{fileName}", ignSignIn, context.RepoAssignment, repo.MustBeNotEmpty, repo.RedirectDownload)
	m.Group("/{username}/{reponame}", func() {
		m.Group("/sonarqube", func() {
			m.Post("/metrics", web.Bind(forms.GetMetricsSonarQube{}), repo_setting.GetMetricsFromSonarQube)
			m.Post("/metrics/pull", web.Bind(forms.GetStatusPullRequest{}), GetMetricsForPagePullRequest)
		})
		m.Group("/settings", func() {
			m.Group("", func() {
				m.Combo("").Get(repo.Settings).
					Post(web.Bind(forms.RepoSettingForm{}), repo.SettingsPost)
			}, repo.SettingsCtxData)
			m.Post("/avatar", web.Bind(forms.AvatarForm{}), repo.SettingsAvatar)
			m.Post("/avatar/delete", repo.SettingsDeleteAvatar)

			m.Group("/collaboration", func() {
				m.Combo("").Get(repo.Collaboration).Post(repo.CollaborationPost)
				m.Post("/access_mode", repo.ChangeCollaborationAccessMode)
				m.Post("/delete", repo.DeleteCollaboration)
				m.Group("/team", func() {
					m.Post("", repo.AddTeamPost)
					m.Post("/delete", repo.DeleteTeam)
				})
			}, disableFunctionalByRoleModel())

			m.Group("/branches", func() {
				m.Post("/", protectedBranchServer.SetDefaultBranchPost)
			}, repo.MustBeNotEmpty)

			m.Group("/review_settings", func() {
				m.Post("/create", web.Bind(models.ReviewSettingsRequest{}), reviewSettingsServer.CreateReviewSettings)
				m.Put("/{branch_name}", web.Bind(models.ReviewSettingsRequest{}), reviewSettingsServer.UpdateReviewSettings)
				m.Delete("/{branch_name}", reviewSettingsServer.DeleteReviewSettings)

				m.Get("", reviewSettingsServer.ReviewSettings)
				m.Get("/edit", reviewSettingsServer.EditReviewSetting)
				m.Get("/create", reviewSettingsServer.CreateReviewSetting)
			}, context.OrgAssignment())

			m.Group("/branches", func() {
				m.Get("/", protectedBranchServer.ProtectedBranchRules)
				m.Combo("/edit").Get(protectedBranchServer.SettingsProtectedBranch).
					Post(web.Bind(forms.ProtectBranchForm{}), context.RepoMustNotBeArchived(), protectedBranchServer.SettingsProtectedBranchPost)
				m.Post("/{id}/delete", protectedBranchServer.DeleteProtectedBranchRulePost)
			})

			m.Group("/tags", func() {
				m.Get("", repo.Tags)
				m.Post("", web.Bind(forms.ProtectTagForm{}), context.RepoMustNotBeArchived(), repo.NewProtectedTagPost)
				m.Post("/delete", context.RepoMustNotBeArchived(), repo.DeleteProtectedTagPost)
				m.Get("/{id}", repo.EditProtectedTag)
				m.Post("/{id}", web.Bind(forms.ProtectTagForm{}), context.RepoMustNotBeArchived(), repo.EditProtectedTagPost)
			})

			m.Group("/hooks/git", func() {
				m.Get("", repo.GitHooks)
				m.Combo("/{name}").Get(repo.GitHooksEdit).
					Post(repo.GitHooksEditPost)
			}, context.GitHookService())

			m.Group("/hooks", func() {
				m.Get("", repo.Webhooks)
				m.Post("/delete", repo.DeleteWebhook)
				addWebhookAddRoutes()
				m.Group("/{id}", func() {
					m.Get("", repo.WebHooksEdit)
					m.Post("/test", repo.TestWebhook)
					m.Post("/replay/{uuid}", repo.ReplayWebhook)
				})
				addWebhookEditRoutes()
			}, webhooksEnabled)

			m.Group("/keys", func() {
				m.Combo("").Get(repo.DeployKeys).
					Post(web.Bind(forms.AddKeyForm{}), repo.DeployKeysPost)
				m.Post("/delete", repo.DeleteDeployKey)
			})

			m.Group("/lfs", func() {
				m.Get("/", repo.LFSFiles)
				m.Get("/show/{oid}", repo.LFSFileGet)
				m.Post("/delete/{oid}", repo.LFSDelete)
				m.Get("/pointers", repo.LFSPointerFiles)
				m.Post("/pointers/associate", repo.LFSAutoAssociate)
				m.Get("/find", repo.LFSFileFind)
				m.Group("/locks", func() {
					m.Get("/", repo.LFSLocks)
					m.Post("/", repo.LFSLockFile)
					m.Post("/{lid}/unlock", repo.LFSUnlock)
				})
			})
			m.Group("/actions", func() {
				m.Get("", repo_setting.RedirectToDefaultSetting)
				addSettingsRunnersRoutes()
				addSettingsSecretsRoutes()
			}, actions.MustEnableActions)
			m.Group("/sonar", func() {
				m.Get("", repo_setting.GetSonarSettings)
				m.Post("", context.RepoMustNotBeArchived(), web.Bind(forms.AddSonarSettings{}), repo_setting.PostSonarSettings)
				m.Delete("", context.RepoMustNotBeArchived(), repo_setting.DeleteSonarSettings)
			})
			m.Post("/migrate/cancel", repo.MigrateCancelPost) // this handler must be under "settings", otherwise this incomplete repo can't be accessed
		}, ctxDataSet("PageIsRepoSettings", true, "LFSStartServer", setting.LFS.StartServer))
	}, reqSignIn, context.RepoAssignment, context.UnitTypes(), reqRepoAdmin, reqRepoEdit, context.RepoRef())
	m.Group("/{username}/{reponame}", func() {
		m.Group("/sonarqube", func() {
			m.Post("/metrics", web.Bind(forms.GetMetricsSonarQube{}), repo_setting.GetMetricsFromSonarQube)
			m.Post("/metrics/pull", web.Bind(forms.GetStatusPullRequest{}), GetMetricsForPagePullRequest)
		})
	}, reqSignIn, context.RepoAssignment, reqRepoRead)
	//m.Post("/{username}/{reponame}/settings/rename_branch", web.Bind(forms.RenameBranchForm{}), reqSignIn, context.RepoAssignment, conflictCustomWriteAndChangeBranch, repo.RenameBranchPost)

	m.Post("/{username}/{reponame}/action/{action}", reqSignIn, context.RepoAssignment, context.UnitTypes(), repo.Action)

	// Grouping for those endpoints not requiring authentication (but should respect ignSignIn)
	m.Group("/{username}/{reponame}", func() {
		m.Group("/milestone", func() {
			m.Get("/{id}", repo.MilestoneIssuesAndPulls)
		}, reqRepoIssuesOrPullsReader, reqRepoRead, context.RepoRef())
		m.Get("/find/*", repo.FindFiles)
		m.Group("/tree-list", func() {
			m.Get("/branch/*", context.RepoRefByType(context.RepoRefBranch), repo.TreeList)
			m.Get("/tag/*", context.RepoRefByType(context.RepoRefTag), repo.TreeList)
			m.Get("/commit/*", context.RepoRefByType(context.RepoRefCommit), repo.TreeList)
		})
		m.Get("/compare", repo.MustBeNotEmpty, reqRepoCodeReader, repo.SetEditorconfigIfExists, ignSignIn, repo.SetDiffViewStyle, repo.SetWhitespaceBehavior, repoServer.CompareDiff)
		m.Combo("/compare/*", repo.MustBeNotEmpty, reqRepoCodeReader, reqRepoRead, repo.SetEditorconfigIfExists).
			Get(repo.SetDiffViewStyle, repo.SetWhitespaceBehavior, repoServer.CompareDiff).
			Post(reqSignIn, context.RepoMustNotBeArchived(), reqRepoPullsReader, conflictCustomWriteAndCreatePR, repo.MustAllowPulls, web.Bind(forms.CreateIssueForm{}), repo.SetWhitespaceBehavior, pullRequestsServer.CompareAndPullRequestPost)
		m.Group("/{type:issues|pulls}", func() {
			m.Group("/{index}", func() {
				m.Get("/info", repo.GetIssueInfo)
			})
		})
	}, ignSignIn, context.RepoAssignment, context.UnitTypes()) // for "/{username}/{reponame}" which doesn't require authentication

	// Grouping for those endpoints that do require authentication
	m.Group("/{username}/{reponame}", func() {
		m.Group("/issues", func() {
			m.Group("/new", func() {
				m.Combo("").Get(context.RepoRef(), reqRepoWrite, repo.NewIssue).
					Post(web.Bind(forms.CreateIssueForm{}), reqRepoWrite, repo.NewIssuePost)
				m.Get("/choose", context.RepoRef(), reqRepoWrite, repo.NewIssueChooseTemplate)
			})
			m.Get("/search", repo.ListIssues)
		}, context.RepoMustNotBeArchived(), reqRepoIssueReader, reqRepoRead)
		// FIXME: should use different URLs but mostly same logic for comments of issue and pull request.
		// So they can apply their own enable/disable logic on routers.
		m.Group("/{type:issues|pulls}", func() {
			m.Group("/{index}", func() {
				m.Post("/title", conflictCustomWriteAndCreatePR, pullRequestsServer.UpdateIssueTitle)
				m.Post("/content", conflictCustomWriteAndCreatePR, repo.UpdateIssueContent)
				m.Post("/deadline", web.Bind(structs.EditDeadlineOption{}), repo.UpdateIssueDeadline)
				m.Post("/watch", repo.IssueWatch)
				m.Post("/ref", conflictCustomWriteAndCreatePR, repo.UpdateIssueRef)
				m.Post("/viewed-files", repo.UpdateViewedFiles)
				m.Group("/dependency", func() {
					m.Post("/add", conflictCustomWriteAndCreatePR, repo.AddDependency)
					m.Post("/delete", conflictCustomWriteAndCreatePR, repo.RemoveDependency)
				})
				m.Combo("/comments").Post(repo.MustAllowUserComment, web.Bind(forms.CreateCommentForm{}), conflictCustomWriteAndApprovePR, issuesServer.NewComment)
				m.Group("/times", func() {
					m.Post("/add", web.Bind(forms.AddTimeManuallyForm{}), repo.AddTimeManually)
					m.Post("/{timeid}/delete", conflictCustomWriteAndCreatePR, repo.DeleteTime)
					m.Group("/stopwatch", func() {
						m.Post("/toggle", repo.IssueStopwatch)
						m.Post("/cancel", repo.CancelStopwatch)
					})
				})
				m.Post("/reactions/{action}", web.Bind(forms.ReactionForm{}), repo.ChangeIssueReaction)
				m.Post("/lock", reqRepoIssuesOrPullsWriter, conflictCustomWriteAndCreatePR, web.Bind(forms.IssueLockForm{}), repo.LockIssue)
				m.Post("/unlock", reqRepoIssuesOrPullsWriter, conflictCustomWriteAndCreatePR, repo.UnlockIssue)
				m.Post("/delete", reqRepoAdmin, conflictCustomEditAndCreatePR, issuesServer.DeleteIssue)
			}, context.RepoMustNotBeArchived())
			m.Group("/{index}", func() {
				m.Get("/attachments", repo.GetIssueAttachments)
				m.Get("/attachments/{uuid}", repo.GetAttachment)
			})
			m.Group("/{index}", func() {
				m.Post("/content-history/soft-delete", repo.SoftDeleteContentHistory)
			})

			m.Post("/labels", reqRepoIssuesOrPullsWriter, conflictCustomWriteAndCreatePR, repo.UpdateIssueLabel)
			m.Post("/milestone", reqRepoIssuesOrPullsWriter, conflictCustomWriteAndCreatePR, repo.UpdateIssueMilestone)
			m.Post("/projects", reqOneWork, reqRepoIssuesOrPullsWriter, reqRepoProjectsReader, conflictCustomWriteAndCreatePR, reqRepoRead, repo.UpdateIssueProject)
			m.Post("/assignee", reqRepoIssuesOrPullsWriter, conflictCustomWriteAndCreatePR, repo.UpdateIssueAssignee)
			m.Post("/request_review", reqRepoIssuesOrPullsReader, reqRepoRead, repo.UpdatePullReviewRequest)
			m.Post("/dismiss_review", reqRepoAdmin, reqRepoEdit, web.Bind(forms.DismissReviewForm{}), repo.DismissReview)
			m.Post("/status", reqRepoIssuesOrPullsWriter, conflictCustomWriteAndCreatePR, issuesServer.UpdateIssueStatus)
			m.Post("/resolve_conversation", reqRepoIssuesOrPullsReader, conflictCustomWriteAndCreatePR, repo.UpdateResolveConversation)
			m.Post("/attachments", repo.UploadIssueAttachment)
			m.Post("/attachments/remove", conflictCustomWriteAndCreatePR, repo.DeleteAttachment)
		}, context.RepoMustNotBeArchived())
		m.Group("/comments/{id}", func() {
			m.Post("", repo.UpdateCommentContent)
			m.Post("/delete", repo.DeleteComment)
			m.Post("/reactions/{action}", web.Bind(forms.ReactionForm{}), repo.ChangeCommentReaction)
		}, conflictCustomWriteAndApprovePR, context.RepoMustNotBeArchived(), context.RepoAssignment)
		m.Group("/comments/{id}", func() {
			m.Get("/attachments", repo.GetCommentAttachments)
		})
		m.Post("/markup", web.Bind(structs.MarkupOption{}), misc.Markup)
		m.Group("/labels", func() {
			m.Post("/new", web.Bind(forms.CreateLabelForm{}), repo.NewLabel)
			m.Post("/edit", web.Bind(forms.CreateLabelForm{}), repo.UpdateLabel)
			m.Post("/delete", repo.DeleteLabel)
			m.Post("/initialize", web.Bind(forms.InitializeLabelsForm{}), repo.InitializeLabels)
		}, context.RepoMustNotBeArchived(), reqRepoIssuesOrPullsWriter, conflictCustomWriteAndCreatePR, context.RepoRef())
		m.Group("/milestones", func() {
			m.Combo("/new").Get(repo.NewMilestone).
				Post(web.Bind(forms.CreateMilestoneForm{}), repo.NewMilestonePost)
			m.Get("/{id}/edit", repo.EditMilestone)
			m.Post("/{id}/edit", web.Bind(forms.CreateMilestoneForm{}), repo.EditMilestonePost)
			m.Post("/{id}/{action}", repo.ChangeMilestoneStatus)
			m.Post("/delete", repo.DeleteMilestone)
		}, context.RepoMustNotBeArchived(), reqRepoIssuesOrPullsWriter, conflictCustomWriteAndCreatePR, context.RepoRef())
		m.Group("/pull", func() {
			m.Post("/{index}/target_branch", repo.UpdatePullRequestTarget)
		}, context.RepoMustNotBeArchived())

		m.Group("/_edit/*", func() {
			m.Combo("").Get(repoServer.EditFile).
				Post(web.Bind(forms.EditRepoFileForm{}), repoServer.EditFilePost)
		}, context.RepoRef(), repo.MustBeEditable, conflictCustomWriteAndChangeBranch, canEnableEditor, reqRepoRead, context.RepoMustNotBeArchived())

		m.Group("", func() {
			m.Group("", func() {
				m.Combo("/_new/*").Get(repoServer.NewFile).
					Post(web.Bind(forms.EditRepoFileForm{}), repoServer.NewFilePost)
				m.Post("/_preview/*", web.Bind(forms.EditPreviewDiffForm{}), repo.DiffPreviewPost)
				m.Combo("/_delete/*").Get(repo.DeleteFile).
					Post(web.Bind(forms.DeleteRepoFileForm{}), repo.DeleteFilePost)
				m.Combo("/_upload/*", repo.MustBeAbleToUpload).
					Get(repo.UploadFile).
					Post(web.Bind(forms.UploadRepoFileForm{}), repo.UploadFilePost)
				m.Combo("/_diffpatch/*").Get(repo.NewDiffPatch).
					Post(web.Bind(forms.EditRepoFileForm{}), repo.NewDiffPatchPost)
				m.Combo("/_cherrypick/{sha:([a-f0-9]{7,40})}/*").Get(repo.CherryPick).
					Post(web.Bind(forms.CherryPickForm{}), repo.CherryPickPost)
			}, repo.MustBeEditable)
			m.Group("", func() {
				m.Post("/upload-file", repo.UploadFileToServer)
				m.Post("/upload-remove", web.Bind(forms.RemoveUploadFileForm{}), repo.RemoveUploadFileFromServer)
			}, repo.MustBeEditable, repo.MustBeAbleToUpload)
		}, conflictCustomWriteAndChangeBranch, context.RepoRef(), canEnableEditor, context.RepoMustNotBeArchived())

		m.Group("/branches", func() {
			m.Group("/_new", func() {
				m.Post("/tag/*", context.RepoRefByType(context.RepoRefTag), repo.CreateBranch)
				m.Post("/commit/*", context.RepoRefByType(context.RepoRefCommit), repo.CreateBranch)
			}, web.Bind(forms.NewBranchForm{}))
			m.Post("/restore", repo.RestoreBranchPost)
		}, context.RepoMustNotBeArchived(), reqRepoCodeWriter, conflictCustomWriteAndChangeBranch, reqRepoRead, repo.MustBeNotEmpty)
		m.Post("/branches/_new/branch/*", conflictCustomWriteAndChangeBranch, web.Bind(forms.NewBranchForm{}), context.RepoRefByType(context.RepoRefBranch),
			repo.CreateBranch, context.RepoMustNotBeArchived(), reqRepoCodeWriter, reqRepoRead, repo.MustBeNotEmpty)
		m.Post("/branches/delete", conflictCustomWriteAndChangeBranch, repo.DeleteBranchPost, context.RepoMustNotBeArchived(), reqRepoCodeWriter, reqRepoRead, repo.MustBeNotEmpty)
		m.Post("/settings/rename_branch", web.Bind(forms.RenameBranchForm{}), context.RepoMustNotBeArchived(), conflictCustomWriteAndChangeBranch, repo.RenameBranchPost)
	}, reqSignIn, context.RepoAssignment, context.UnitTypes())

	// Tags
	m.Group("/{username}/{reponame}", func() {
		m.Group("/tags", func() {
			m.Get("", repo.TagsList)
			m.Get(".rss", feedEnabled, repo.TagsListFeedRSS)
			m.Get(".atom", feedEnabled, repo.TagsListFeedAtom)
		}, ctxDataSet("EnableFeed", setting.Other.EnableFeed),
			repo.MustBeNotEmpty, reqRepoCodeReader, reqRepoRead, context.RepoRefByType(context.RepoRefTag, true))
		m.Post("/tags/delete", repo.DeleteTag, reqSignIn,
			repo.MustBeNotEmpty, context.RepoMustNotBeArchived(), reqRepoCodeWriter, conflictCustomWriteAndChangeBranch, context.RepoRef())
	}, ignSignIn, context.RepoAssignment, context.UnitTypes())

	// Releases
	m.Group("/{username}/{reponame}", func() {
		m.Group("/releases", func() {
			m.Get("/", repo.Releases)
			m.Get("/tag/*", repo.SingleRelease)
			m.Get("/latest", repo.LatestRelease)
			m.Get(".rss", feedEnabled, repo.ReleasesFeedRSS)
			m.Get(".atom", feedEnabled, repo.ReleasesFeedAtom)
		}, ctxDataSet("EnableFeed", setting.Other.EnableFeed),
			repo.MustBeNotEmpty, reqRepoReleaseReader, reqRepoRead, context.RepoRefByType(context.RepoRefTag, true))
		m.Get("/releases/attachments/{uuid}", repo.GetAttachment, repo.MustBeNotEmpty, reqRepoReleaseReader, reqRepoRead)
		m.Group("/releases", func() {
			m.Get("/new", repo.NewRelease)
			m.Post("/new", web.Bind(forms.NewReleaseForm{}), repo.NewReleasePost)
			m.Post("/delete", repo.DeleteRelease)
			m.Post("/attachments", repo.UploadReleaseAttachment)
			m.Post("/attachments/remove", repo.DeleteAttachment)
		}, reqSignIn, repo.MustBeNotEmpty, context.RepoMustNotBeArchived(), reqRepoReleaseWriter, conflictCustomWriteAndChangeBranch, context.RepoRef())
		m.Group("/releases", func() {
			m.Get("/edit/*", repo.EditRelease)
			m.Post("/edit/*", web.Bind(forms.EditReleaseForm{}), repo.EditReleasePost)
		}, reqSignIn, repo.MustBeNotEmpty, context.RepoMustNotBeArchived(), reqRepoReleaseWriter, conflictCustomWriteAndChangeBranch, repo.CommitInfoCache)
	}, reqSignIn, context.RepoAssignment, context.UnitTypes(), reqRepoReleaseReader, reqRepoRead)

	// to maintain compatibility with old attachments
	m.Group("/{username}/{reponame}", func() {
		m.Get("/attachments/{uuid}", repo.GetAttachment)
	}, ignSignIn, context.RepoAssignment, context.UnitTypes())

	m.Group("/{username}/{reponame}", func() {
		m.Post("/topics", repo.TopicsPost)
	}, context.RepoAssignment, context.RepoMustNotBeArchived(), reqRepoAdmin, reqRepoEdit)

	m.Group("/{username}/{reponame}", func() {
		m.Group("", func() {
			m.Group("/{type:issues|pulls}", func() {
				m.Get("", repoServer.IssuesOrPull)
				m.Get("/posters", repo.IssuePosters)
			})
			m.Get("/{type:issues|pulls}/{index}", repoServer.ViewIssue)
			m.Group("/{type:issues|pulls}/{index}/content-history", func() {
				m.Get("/overview", repo.GetContentHistoryOverview)
				m.Get("/list", repo.GetContentHistoryList)
				m.Get("/detail", repo.GetContentHistoryDetail)
			})
			m.Get("/labels", reqRepoIssuesOrPullsReader, reqRepoRead, repo.RetrieveLabels, repo.Labels)
			m.Get("/milestones", reqRepoIssuesOrPullsReader, reqRepoRead, repo.Milestones)
		}, context.RepoRef())

		if setting.Packages.Enabled {
			m.Get("/packages", repo.Packages)
		}

		m.Group("/projects", func() {
			m.Get("", reqOneWork, repoServer.Projects)
			m.Get("/{id}", repo.ViewProject)
			m.Group("", func() { //nolint:dupl
				m.Get("/new", repo.NewProject)
				m.Post("/new", web.Bind(forms.CreateProjectForm{}), repo.NewProjectPost)
				m.Group("/{id}", func() {
					m.Post("", web.Bind(forms.EditProjectBoardForm{}), repo.AddBoardToProjectPost)
					m.Post("/delete", repo.DeleteProject)
					m.Get("/edit", repo.EditProject)
					m.Post("/edit", web.Bind(forms.CreateProjectForm{}), repo.EditProjectPost)
					m.Post("/{action:open|close}", repo.ChangeProjectStatus)

					m.Group("/{boardID}", func() {
						m.Put("", web.Bind(forms.EditProjectBoardForm{}), repo.EditProjectBoard)
						m.Delete("", repo.DeleteProjectBoard)
						m.Post("/default", repo.SetDefaultProjectBoard)
						m.Post("/unsetdefault", repo.UnSetDefaultProjectBoard)

						m.Post("/move", repo.MoveIssues)
					})
				})
			}, reqRepoProjectsWriter, reqRepoWrite, context.RepoMustNotBeArchived())
		}, reqRepoProjectsReader, reqRepoRead, repo.MustEnableProjects)

		m.Group("/actions", func() {
			m.Get("", actions.List)

			m.Group("/runs/{run}", func() {
				m.Combo("").
					Get(actions.View).
					Post(web.Bind(actions.ViewRequest{}), actions.ViewPost)
				m.Group("/jobs/{job}", func() {
					m.Combo("").
						Get(actions.View).
						Post(web.Bind(actions.ViewRequest{}), actions.ViewPost)
					m.Post("/rerun", reqRepoActionsWriter, reqRepoWrite, actions.RerunOne)
				})
				m.Post("/cancel", reqRepoActionsWriter, reqRepoWrite, actions.Cancel)
				m.Post("/approve", reqRepoActionsWriter, reqRepoWrite, actions.Approve)
				m.Post("/artifacts", actions.ArtifactsView)
				m.Get("/artifacts/{id}", actions.ArtifactsDownloadView)
				m.Post("/rerun", reqRepoActionsWriter, reqRepoWrite, actions.RerunAll)
			})
		}, reqRepoActionsReader, reqRepoRead, actions.MustEnableActions)

		m.Group("/wiki", func() {
			m.Combo("/").
				Get(repo.Wiki).
				Post(context.RepoMustNotBeArchived(), reqSignIn, reqRepoWikiWriter, reqRepoWrite, web.Bind(forms.NewWikiForm{}), repo.WikiPost)
			m.Combo("/*").
				Get(repo.Wiki).
				Post(context.RepoMustNotBeArchived(), reqSignIn, reqRepoWikiWriter, reqRepoWrite, web.Bind(forms.NewWikiForm{}), repo.WikiPost)
			m.Get("/commit/{sha:[a-f0-9]{7,40}}", repo.SetEditorconfigIfExists, repo.SetDiffViewStyle, repo.SetWhitespaceBehavior, repoServer.Diff)
			m.Get("/commit/{sha:[a-f0-9]{7,40}}.{ext:patch|diff}", repo.RawDiff)
		}, repo.MustEnableWiki, func(ctx *context.Context) {
			ctx.Data["PageIsWiki"] = true
			ctx.Data["CloneButtonOriginLink"] = ctx.Repo.Repository.WikiCloneLink()
		})

		m.Group("/wiki", func() {
			m.Get("/raw/*", repo.WikiRaw)
		}, repo.MustEnableWiki)

		m.Group("/activity", func() {
			m.Get("", repo.Activity)
			m.Get("/{period}", repo.Activity)
		}, context.RepoRef(), repo.MustBeNotEmpty, context.RequireRepoReaderOr(unit.TypePullRequests, unit.TypeIssues, unit.TypeReleases), reqRepoRead)

		m.Group("/activity_author_data", func() {
			m.Get("", repo.ActivityAuthors)
			m.Get("/{period}", repo.ActivityAuthors)
		}, context.RepoRef(), repo.MustBeNotEmpty, context.RequireRepoReaderOr(unit.TypeCode), reqRepoRead)

		m.Group("/archive", func() {
			m.Get("/*", repoServerCodeHub.Download)
			m.Post("/*", repo.InitiateDownload)
		}, repo.MustBeNotEmpty, dlSourceEnabled, reqRepoCodeReader, reqRepoRead)

		m.Group("/branches", func() {
			m.Get("", branchesServer.Branches)
		}, repo.MustBeNotEmpty, context.RepoRef(), reqRepoCodeReader, reqRepoRead)

		m.Group("/blob_excerpt", func() {
			m.Get("/{sha}", repo.SetEditorconfigIfExists, repo.SetDiffViewStyle, repo.ExcerptBlob)
		}, func(ctx *context.Context) (cancel gocontext.CancelFunc) {
			if ctx.FormBool("wiki") {
				ctx.Data["PageIsWiki"] = true
				repo.MustEnableWiki(ctx)
				return
			}

			reqRepoCodeReader(ctx)
			reqRepoRead(ctx)
			if ctx.Written() {
				return
			}
			cancel = context.RepoRef()(ctx)
			if ctx.Written() {
				return
			}

			repo.MustBeNotEmpty(ctx)
			return cancel
		})

		m.Group("/pulls/{index}", func() {
			m.Get(".diff", repo.DownloadPullDiff)
			m.Get(".patch", repo.DownloadPullPatch)
			m.Get("/commits", context.RepoRef(), repoServer.ViewPullCommits)
			m.Get("/state", context.RepoMustNotBeArchived(), repo.GetPullRequestStatusState)
			m.Post("/unit_links", web.Bind(structs.GetUnitLinksWithDescriptionRequest{}), unitLinksServer.GetUnitLinksWithDescription)
		}, repo.MustAllowPulls, reqRepoRead)

		m.Group("/pulls/{index}", func() {
			m.Post("/rebuild", context.RepoMustNotBeArchived(), conflictCustomWriteAndMergePR, repo.RebuildPullRequest)
			m.Post("/merge", context.RepoMustNotBeArchived(), conflictCustomWriteAndMergePR, web.Bind(forms.MergePullRequestForm{}), pullRequestsServer.MergePullRequest)
			m.Post("/cancel_auto_merge", context.RepoMustNotBeArchived(), conflictCustomWriteAndMergePR, repo.CancelAutoMergePullRequest)
			m.Post("/update", conflictCustomWriteAndCreatePR, repo.UpdatePullRequest)
			m.Post("/set_allow_maintainer_edit", conflictCustomWriteAndCreatePR, web.Bind(forms.UpdateAllowEditsForm{}), repo.SetAllowEdits)
			m.Post("/cleanup", context.RepoMustNotBeArchived(), context.RepoRef(), conflictCustomWriteAndCreatePR, repo.CleanUpPullRequest)
		}, repo.MustAllowPulls, reqRepoRead)
		m.Group("/pulls/{index}/files", func() {
			m.Get("", context.RepoRef(), repo.SetEditorconfigIfExists, repo.SetDiffViewStyle, repo.SetWhitespaceBehavior, repoServer.ViewPullFiles)
			m.Group("/reviews", func() {
				m.Get("/new_comment", repo.RenderNewCodeCommentForm)
				m.Post("/comments", web.Bind(forms.CodeCommentForm{}), repo.CreateCodeComment)
				m.Post("/submit", web.Bind(forms.SubmitReviewForm{}), repoServer.SubmitReview)
			}, conflictCustomWriteAndApprovePR, context.RepoMustNotBeArchived())
		}, repo.MustAllowPulls, reqRepoRead)
		m.Group("/media", func() {
			m.Get("/branch/*", context.RepoRefByType(context.RepoRefBranch), repoServerCodeHub.SingleDownloadOrLFS)
			m.Get("/tag/*", context.RepoRefByType(context.RepoRefTag), repoServerCodeHub.SingleDownloadOrLFS)
			m.Get("/commit/*", context.RepoRefByType(context.RepoRefCommit), repoServerCodeHub.SingleDownloadOrLFS)
			m.Get("/blob/{sha}", context.RepoRefByType(context.RepoRefBlob), repo.DownloadByIDOrLFS)
			// "/*" route is deprecated, and kept for backward compatibility
			m.Get("/*", context.RepoRefByType(context.RepoRefLegacy), repoServerCodeHub.SingleDownloadOrLFS)
		}, repo.MustBeNotEmpty, reqRepoCodeReader, reqRepoRead)

		m.Group("/raw", func() {
			m.Get("/branch/*", context.RepoRefByType(context.RepoRefBranch), repoServerCodeHub.SingleDownload)
			m.Get("/tag/*", context.RepoRefByType(context.RepoRefTag), repoServerCodeHub.SingleDownload)
			m.Get("/commit/*", context.RepoRefByType(context.RepoRefCommit), repoServerCodeHub.SingleDownload)
			m.Get("/blob/{sha}", context.RepoRefByType(context.RepoRefBlob), repo.DownloadByID)
			// "/*" route is deprecated, and kept for backward compatibility
			m.Get("/*", context.RepoRefByType(context.RepoRefLegacy), repoServerCodeHub.SingleDownload)
		}, repo.MustBeNotEmpty, reqRepoCodeReader, reqRepoRead)

		m.Group("/render", func() {
			m.Get("/branch/*", context.RepoRefByType(context.RepoRefBranch), repo.RenderFile)
			m.Get("/tag/*", context.RepoRefByType(context.RepoRefTag), repo.RenderFile)
			m.Get("/commit/*", context.RepoRefByType(context.RepoRefCommit), repo.RenderFile)
			m.Get("/blob/{sha}", context.RepoRefByType(context.RepoRefBlob), repo.RenderFile)
		}, repo.MustBeNotEmpty, reqRepoCodeReader, reqRepoRead)

		m.Group("/commits", func() {
			m.Get("/branch/*", context.RepoRefByType(context.RepoRefBranch), repoServer.RefCommits)
			m.Get("/tag/*", context.RepoRefByType(context.RepoRefTag), repoServer.RefCommits)
			m.Get("/commit/*", context.RepoRefByType(context.RepoRefCommit), repoServer.RefCommits)
			// "/*" route is deprecated, and kept for backward compatibility
			m.Get("/*", context.RepoRefByType(context.RepoRefLegacy), repoServer.RefCommits)
		}, repo.MustBeNotEmpty, reqRepoCodeReader, reqRepoRead)

		m.Group("/blame", func() {
			m.Get("/branch/*", context.RepoRefByType(context.RepoRefBranch), repo.RefBlame)
			m.Get("/tag/*", context.RepoRefByType(context.RepoRefTag), repo.RefBlame)
			m.Get("/commit/*", context.RepoRefByType(context.RepoRefCommit), repo.RefBlame)
		}, repo.MustBeNotEmpty, reqRepoCodeReader, reqRepoRead)

		m.Group("", func() {
			m.Get("/graph", repo.Graph)
			m.Get("/commit/{sha:([a-f0-9]{7,40})$}", repo.SetEditorconfigIfExists, repo.SetDiffViewStyle, repo.SetWhitespaceBehavior, repoServer.Diff)
			m.Get("/cherry-pick/{sha:([a-f0-9]{7,40})$}", repo.SetEditorconfigIfExists, repo.CherryPick)
		}, repo.MustBeNotEmpty, context.RepoRef(), reqRepoCodeReader, reqRepoRead)

		m.Get("/rss/branch/*", context.RepoRefByType(context.RepoRefBranch), feedEnabled, feed.RenderBranchFeed)
		m.Get("/atom/branch/*", context.RepoRefByType(context.RepoRefBranch), feedEnabled, feed.RenderBranchFeed)

		m.Group("/src", func() {
			m.Get("/branch/*", context.RepoRefByType(context.RepoRefBranch), repoServerCodeHub.Home)
			m.Get("/tag/*", context.RepoRefByType(context.RepoRefTag), repoServerCodeHub.Home)
			m.Get("/commit/*", context.RepoRefByType(context.RepoRefCommit), repoServerCodeHub.Home)
			// "/*" route is deprecated, and kept for backward compatibility
			m.Get("/*", context.RepoRefByType(context.RepoRefLegacy), repoServerCodeHub.Home)
		}, reqRepoRead, repo.SetEditorconfigIfExists)
		m.Group("", func() {
			m.Get("/forks", repo.Forks)
		}, context.RepoRef(), reqRepoCodeReader, reqRepoRead)
		m.Get("/commit/{sha:([a-f0-9]{7,40})}.{ext:patch|diff}", repo.MustBeNotEmpty, reqRepoCodeReader, reqRepoRead, repo.RawDiff)
	}, ignSignIn, context.RepoAssignment, context.UnitTypes())

	m.Post("/{username}/{reponame}/lastcommit/*", ignSignInAndCsrf, context.RepoAssignment, context.UnitTypes(), context.RepoRefByType(context.RepoRefCommit), reqRepoCodeReader, reqRepoRead, repo.LastCommit)

	m.Group("/{username}/{reponame}", func() {
		m.Get("/stars", repo.Stars)
		m.Get("/watchers", repo.Watchers)
		m.Get("/search", reqRepoCodeReader, reqRepoRead, repo.Search)
	}, ignSignIn, context.RepoAssignment, context.RepoRef(), context.UnitTypes())

	m.Group("/{username}", func() {
		m.Group("/{reponame}", func() {
			m.Get("", repo.SetEditorconfigIfExists, repoServerCodeHub.Home)
		}, ignSignIn, context.RepoAssignment, reqRepoRead, context.RepoRef(), context.UnitTypes())
		m.Post("/{reponame}/license", web.Bind(forms.GetLicenseInfoForm{}), repo.GetLicenseInfo)
	})
	// ***** END: Repository *****

	m.Group("/notifications", func() {
		m.Get("", user.Notifications)
		m.Get("/subscriptions", user.NotificationSubscriptions)
		m.Get("/watching", user.NotificationWatching)
		m.Post("/status", user.NotificationStatusPost)
		m.Post("/purge", user.NotificationPurgePost)
		m.Get("/new", user.NewAvailable)
	}, reqSignIn)

	if setting.API.EnableSwagger {
		m.Get("/swagger.v1.json", SwaggerV1Json)
		m.Get("/swagger.v2.json", SwaggerV2Json)
		m.Get("/swagger.v3.json", SwaggerV3Json)
	}

	if !setting.IsProd {
		m.Any("/devtest", devtest.List)
		m.Any("/devtest/{sub}", devtest.Tmpl)
	}

	m.NotFound(func(w http.ResponseWriter, req *http.Request) {
		ctx := context.GetWebContext(req)
		ctx.NotFound("", nil)
	})
}

func registerGitRoutes(m *web.Route) {
	ignSignInAndCsrf := auth_service.VerifyAuthWithOptions(&auth_service.VerifyOptions{DisableCSRF: true})

	lfsServerEnabled := func(ctx *context.Context) {
		if !setting.LFS.StartServer {
			ctx.Error(http.StatusNotFound)
			return
		}
	}

	m.Group("/{username}", func() {

		m.Group("/{reponame}", func() {
			m.Group("/info/lfs", func() {
				m.Post("/objects/batch", lfs.CheckAcceptMediaType, lfs.BatchHandler)
				m.Put("/objects/{oid}/{size}", lfs.UploadHandler)
				m.Get("/objects/{oid}/{filename}", lfs.DownloadHandler)
				m.Get("/objects/{oid}", lfs.DownloadHandler)
				m.Post("/verify", lfs.CheckAcceptMediaType, lfs.VerifyHandler)
				m.Group("/locks", func() {
					m.Get("/", lfs.GetListLockHandler)
					m.Post("/", lfs.PostLockHandler)
					m.Post("/verify", lfs.VerifyLockHandler)
					m.Post("/{lid}/unlock", lfs.UnLockHandler)
				}, lfs.CheckAcceptMediaType)
				m.Any("/*", func(ctx *context.Context) {
					ctx.NotFound("", nil)
				})
			}, ignSignInAndCsrf, lfsServerEnabled)

			m.Group("", func() {
				m.PostOptions("/git-upload-pack", repo.ServiceUploadPack)
				m.PostOptions("/git-receive-pack", repo.ServiceReceivePack)
				m.GetOptions("/info/refs", repo.GetInfoRefs)
				m.GetOptions("/HEAD", repo.GetTextFile("HEAD"))
				m.GetOptions("/objects/info/alternates", repo.GetTextFile("objects/info/alternates"))
				m.GetOptions("/objects/info/http-alternates", repo.GetTextFile("objects/info/http-alternates"))
				m.GetOptions("/objects/info/packs", repo.GetInfoPacks)
				m.GetOptions("/objects/info/{file:[^/]*}", repo.GetTextFile(""))
				m.GetOptions("/objects/{head:[0-9a-f]{2}}/{hash:[0-9a-f]{38}}", repo.GetLooseObject)
				m.GetOptions("/objects/pack/pack-{file:[0-9a-f]{40}}.pack", repo.GetPackFile)
				m.GetOptions("/objects/pack/pack-{file:[0-9a-f]{40}}.idx", repo.GetIdxFile)
			}, ignSignInAndCsrf, repo.HTTPGitEnabledHandler, repo.CorsHandler(), context_service.UserAssignmentWeb())
		})
	})
}
