// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

// Package private contains all internal routes. The package name "internal" isn't usable because Golang reserves it for disabling cross-package usage.
package private

import (
	goctx "context"
	"crypto/tls"
	"net/http"
	"strings"

	"code.gitea.io/gitea/models/code_hub_counter_task/code_hub_counter_task_db"
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/git/protected_branch/protected_branch_db"
	"code.gitea.io/gitea/models/pull/pullrequestidresolver"
	"code.gitea.io/gitea/models/role_model"
	"code.gitea.io/gitea/models/role_model/casbin_role_manager"
	"code.gitea.io/gitea/models/role_model/custom_casbin_role_manager"
	"code.gitea.io/gitea/models/unit_links/unit_links_db"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/gitnamesparser/branch"
	"code.gitea.io/gitea/modules/gitnamesparser/pullrequest"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/mtls"
	"code.gitea.io/gitea/modules/private"
	"code.gitea.io/gitea/modules/setting"
	setting_mtls "code.gitea.io/gitea/modules/setting/mtls"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/routers/private/code_hub_counter"
	"code.gitea.io/gitea/routers/private/hooks"
	"code.gitea.io/gitea/routers/private/pull_request_reader"
	"code.gitea.io/gitea/routers/private/task_tracker_client"
	"code.gitea.io/gitea/routers/private/unit_linker"
	"code.gitea.io/gitea/routers/web/user/accesser/org_accesser"
	"code.gitea.io/gitea/routers/web/user/accesser/repo_accesser"
	protected_brancher "code.gitea.io/gitea/services/protected_branch"

	"gitea.com/go-chi/binding"
	chi_middleware "github.com/go-chi/chi/v5/middleware"
)

// CheckInternalToken check internal token is set
func CheckInternalToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		tokens := req.Header.Get("Authorization")
		fields := strings.SplitN(tokens, " ", 2)
		if setting.InternalToken == "" {
			log.Warn(`The INTERNAL_TOKEN setting is missing from the configuration file: %q, internal API can't work.`, setting.CustomConf)
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}
		if len(fields) != 2 || fields[0] != "Bearer" || fields[1] != setting.InternalToken {
			log.Debug("Forbidden attempt to access internal url: Authorization header: %s", tokens)
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		} else {
			next.ServeHTTP(w, req)
		}
	})
}

// bind binding an obj to a handler
func bind[T any](_ T) any {
	return func(ctx *context.PrivateContext) {
		theObj := new(T) // create a new form obj for every request but not use obj directly
		binding.Bind(ctx.Req, theObj)
		web.SetForm(ctx, theObj)
	}
}

// Routes registers all internal APIs routes to web application.
// These APIs will be invoked by internal commands for example `gitea serv` and etc.
func Routes() *web.Route {
	r := web.NewRoute()
	r.Use(context.PrivateContexter())
	r.Use(CheckInternalToken)
	// Log the real ip address of the request from SSH is really helpful for diagnosing sometimes.
	// Since internal API will be sent only from Gitea sub commands and it's under control (checked by InternalToken), we can trust the headers.
	r.Use(chi_middleware.RealIP)

	ctx := goctx.Background()
	dbEngine := db.GetEngine(ctx)
	branchParser := branch.NewParser()
	pullRequestParser := pullrequest.NewParser()
	pullRequestIDResolver := pullrequestidresolver.NewResolver(dbEngine)
	unitLinkDB := unit_links_db.NewUnitLinkDB(dbEngine)
	pullRequestHeaderDB := pull_request_reader.NewReader(dbEngine)
	protectedBranchDB := protected_branch_db.NewProtectedBranchDB(dbEngine)

	mtlsConfig := &tls.Config{} // initialize tls config with default values
	if setting_mtls.CheckMTLSConfigSecManEnabled(task_tracker_client.TaskTrackerClientName) {
		mtlsCerts := setting_mtls.GetMTLSCertsFromSecMan(task_tracker_client.TaskTrackerClientName, setting.NewGetterForSecMan())
		mtlsConfig = mtls.GenerateTlsConfigForMTLS(task_tracker_client.TaskTrackerClientName, mtlsCerts.Cert, mtlsCerts.CertKey, mtlsCerts.CaCerts)
	}
	taskTrackerClient := task_tracker_client.New(
		setting.TaskTracker.APIBaseURL,
		setting.TaskTracker.APIToken,
		&http.Client{Transport: &http.Transport{TLSClientConfig: mtlsConfig}})

	unitLinker := unit_linker.NewUnitLinker(
		branchParser,
		pullRequestParser,
		unitLinkDB,
		pullRequestHeaderDB,
		taskTrackerClient,
		setting.TaskTracker.UnitsValidationEnabled,
	)

	protectedBranchGetter := protected_brancher.NewProtectedBranchGetter()
	protectedBranchChecker := protected_brancher.NewProtectedBranchChecker()
	protectedBranchMerger := protected_brancher.NewProtectedBranchMerger()
	protectedBranchUpdater := protected_brancher.NewProtectedBranchUpdater()
	protectedBranchManager := protected_brancher.NewProtectedBranchManager(protectedBranchGetter, protectedBranchChecker, protectedBranchMerger, protectedBranchUpdater, protectedBranchDB)

	casbinManager := casbin_role_manager.New()

	orgAccesser := org_accesser.New(casbinManager)
	casbinCustomManager := custom_casbin_role_manager.NewManager(role_model.GetSecurityEnforcer())
	requestAccessor := repo_accesser.NewRepoAccessor(casbinCustomManager)
	server := hooks.NewServer(unitLinker, pullRequestIDResolver, setting.TaskTracker.Enabled, requestAccessor, orgAccesser, protectedBranchManager)
	usagesDB := code_hub_counter_task_db.New(dbEngine)
	taskCreator := code_hub_counter.NewTaskCreator(usagesDB, setting.CodeHub.CodeHubMetricEnabled)
	commandServer := NewServer(taskCreator)

	r.Post("/ssh/authorized_keys", AuthorizedPublicKeyByContent)
	r.Post("/ssh/{id}/update/{repoid}", UpdatePublicKeyInRepo)
	r.Post("/ssh/log", bind(private.SSHLogOption{}), SSHLog)
	r.Post("/hook/pre-receive/{owner}/{repo}", RepoAssignment, bind(private.HookOptions{}), server.HookPreReceive)
	r.Get("/hook/git/pre-receive/{owner}/{repo}", RepoAssignment, GetGitHookPreReceive)
	r.Post("/hook/post-receive/{owner}/{repo}", context.OverrideContext, bind(private.HookOptions{}), server.HookPostReceive)
	r.Post("/hook/proc-receive/{owner}/{repo}", context.OverrideContext, RepoAssignment, bind(private.HookOptions{}), HookProcReceive)
	r.Post("/hook/set-default-branch/{owner}/{repo}/{branch}", RepoAssignment, SetDefaultBranch)
	r.Get("/serv/none/{keyid}", ServNoCommand)
	r.Get("/serv/command/{keyid}/{owner}/{repo}", commandServer.ServCommand)
	r.Post("/manager/shutdown", Shutdown)
	r.Post("/manager/restart", Restart)
	r.Post("/manager/reload-templates", ReloadTemplates)
	r.Post("/manager/flush-queues", bind(private.FlushOptions{}), FlushQueues)
	r.Post("/manager/pause-logging", PauseLogging)
	r.Post("/manager/resume-logging", ResumeLogging)
	r.Post("/manager/release-and-reopen-logging", ReleaseReopenLogging)
	r.Post("/manager/set-log-sql", SetLogSQL)
	r.Post("/manager/add-logger", bind(private.LoggerOptions{}), AddLogger)
	r.Post("/manager/remove-logger/{logger}/{writer}", RemoveLogger)
	r.Get("/manager/processes", Processes)
	r.Post("/mail/send", SendEmail)
	r.Post("/restore_repo", RestoreRepo)
	r.Post("/actions/generate_actions_runner_token", GenerateActionsRunnerToken)

	return r
}
