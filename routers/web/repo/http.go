// Copyright 2014 The Gogs Authors. All rights reserved.
// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"compress/gzip"
	gocontext "context"
	"fmt"
	"net/http"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"code.gitea.io/gitea/modules/trace"
	"gitlab.com/gitlab-org/gitaly/v16/proto/go/gitalypb"

	"github.com/go-chi/cors"

	"code.gitea.io/gitea/integration/gitaly"
	actions_model "code.gitea.io/gitea/models/actions"
	auth_model "code.gitea.io/gitea/models/auth"
	"code.gitea.io/gitea/models/code_hub_counter_task"
	"code.gitea.io/gitea/models/code_hub_counter_task/code_hub_counter_task_db"
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/organization"
	"code.gitea.io/gitea/models/perm"
	access_model "code.gitea.io/gitea/models/perm/access"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/role_model"
	"code.gitea.io/gitea/models/tenant"
	"code.gitea.io/gitea/models/unit"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/git/utils"
	"code.gitea.io/gitea/modules/log"
	repo_module "code.gitea.io/gitea/modules/repository"
	"code.gitea.io/gitea/modules/sbt/audit"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/modules/util"
	"code.gitea.io/gitea/routers/private/code_hub_counter"
	repo_service "code.gitea.io/gitea/services/repository"
)

func HTTPGitEnabledHandler(ctx *context.Context) {
	if setting.Repository.DisableHTTPGit {
		ctx.Resp.WriteHeader(http.StatusForbidden)
		_, _ = ctx.Resp.Write([]byte("Interacting with repositories by HTTP protocol is not allowed"))
	}
}

func CorsHandler() func(next http.Handler) http.Handler {
	if setting.Repository.AccessControlAllowOrigin != "" {
		return cors.Handler(cors.Options{
			AllowedOrigins: []string{setting.Repository.AccessControlAllowOrigin},
			AllowedHeaders: []string{"Content-Type", "Authorization", "User-Agent"},
		})
	}
	return func(next http.Handler) http.Handler {
		return next
	}
}

// httpBase implementation git smart HTTP protocol
func httpBase(ctx *context.Context) (h *serviceHandler) {
	logTracer := trace.NewLogTracer()
	message := logTracer.CreateTraceMessage(ctx)
	errTrace := logTracer.Trace(message)
	if errTrace != nil {
		log.Error("Error has occurred while creating trace message: %v", errTrace)
	}
	defer func() {
		errTrace = logTracer.TraceTime(message)
		if errTrace != nil {
			log.Error("Error has occurred while creating trace time message: %v", errTrace)
		}
	}()

	username := ctx.Params(":username")
	reponame := strings.TrimSuffix(ctx.Params(":reponame"), ".git")

	if ctx.FormString("go-get") == "1" {
		context.EarlyResponseForGoGetMeta(ctx)
		return
	}

	auditParams := map[string]string{
		"repository": reponame,
		"owner":      username,
	}

	auditParamsForUnauthorized := map[string]string{
		"request_url": ctx.Req.URL.RequestURI(),
	}

	var isPull, receivePack bool
	service := ctx.FormString("service")
	if service == "git-receive-pack" ||
		strings.HasSuffix(ctx.Req.URL.Path, "git-receive-pack") {
		isPull = false
		receivePack = true
	} else if service == "git-upload-pack" ||
		strings.HasSuffix(ctx.Req.URL.Path, "git-upload-pack") {
		isPull = true

	} else if service == "git-upload-archive" ||
		strings.HasSuffix(ctx.Req.URL.Path, "git-upload-archive") {
		isPull = true
	} else {
		isPull = ctx.Req.Method == "GET"
	}

	var accessMode perm.AccessMode
	var event audit.Event
	var action role_model.Action
	if isPull {
		accessMode = perm.AccessModeRead
		event = audit.RepositoryPullOrCloneEvent
		action = role_model.READ
	} else {
		accessMode = perm.AccessModeWrite
		event = audit.ChangesPushEvent
		action = role_model.WRITE
	}

	isWiki := false
	unitType := unit.TypeCode
	var wikiRepoName string
	if strings.HasSuffix(reponame, ".wiki") {
		isWiki = true
		unitType = unit.TypeWiki
		wikiRepoName = reponame
		reponame = reponame[:len(reponame)-5]
	}

	owner := ctx.ContextUser
	if !owner.IsOrganization() && !owner.IsActive {
		ctx.PlainText(http.StatusForbidden, "Repository cannot be accessed. You cannot push or open Issues/pull-requests.")
		auditParams["error"] = "Repository cannot be accessed"
		audit.CreateAndSendEvent(event, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}
	repoExist := true
	repo, err := repo_model.GetRepositoryByName(owner.ID, reponame)
	if err != nil {
		if repo_model.IsErrRepoNotExist(err) {
			if redirectRepoID, err := repo_model.LookupRedirect(owner.ID, reponame); err == nil {
				context.RedirectToRepo(ctx.Base, redirectRepoID)
				auditParams["error"] = "Cannot find repository"
				audit.CreateAndSendEvent(event, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
				return
			}
			repoExist = false
		} else {
			ctx.ServerError("GetRepositoryByName", err)
			auditParams["error"] = "Error has occurred while getting repository by owner and name"
			audit.CreateAndSendEvent(event, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}
	}
	auditParams["repository_id"] = strconv.FormatInt(repo.ID, 10)

	// если у нас включена ролевая модель SourceControl, то запускается проверка привилегий на чтение или запись в репозиторий
	if setting.SourceControl.TenantWithRoleModeEnabled {
		if ctx.IsSigned && owner.IsOrganization() {
			tenantId, err := tenant.GetTenantByOrgIdOrDefault(ctx, owner.ID)
			if err != nil {
				log.Error("Error has occurred while getting tenant by organization: %v", err)
				ctx.PlainText(http.StatusNotFound, "The repository you are trying to reach either does not exist or you are not authorized to view it.")
				return
			}

			allowed, err := role_model.CheckUserPermissionToOrganization(ctx, ctx.Doer, tenantId, &organization.Organization{ID: owner.ID}, action)
			if err != nil {
				log.Error("Error has occurred while checking user's permissions: %v", err)
				ctx.PlainText(http.StatusNotFound, "The repository you are trying to reach either does not exist or you are not authorized to view it.")
				return
			}
			if !allowed {

				// todo: здесь нужно испрользовать динамический экшен ведь может быть запись в репозиторий
				allow, err := role_model.CheckUserPermissionToTeam(ctx, ctx.Doer, tenantId, &organization.Organization{ID: owner.ID}, &repo_model.Repository{ID: repo.ID}, role_model.ViewBranch.String())
				if err != nil || !allow {
					log.Error("Error has occurred while checking user's permissions: %v", err)
					ctx.PlainText(http.StatusNotFound, "The repository you are trying to reach either does not exist or you are not authorized to view it.")
					return
				}
			}
		} else {
			ctx.Resp.Header().Set("WWW-Authenticate", "Basic realm=\".\"")
			ctx.Error(http.StatusUnauthorized)
			audit.CreateAndSendEvent(audit.UnauthorizedRequestEvent, audit.EmptyRequiredField, audit.EmptyRequiredField, audit.StatusFailure, ctx.Req.RemoteAddr, auditParamsForUnauthorized)
			return
		}
	}

	// если у нас включена ролевая модель SourceControl, то запускается проверка привилегий на чтение приватного репозитория
	if setting.SourceControl.TenantWithRoleModeEnabled {
		if repo.IsPrivate && owner.IsOrganization() {
			tenantId, err := tenant.GetTenantByOrgIdOrDefault(ctx, owner.ID)
			if err != nil {
				ctx.PlainText(http.StatusNotFound, "The repository you are trying to reach either does not exist or you are not authorized to view it.")
				return
			}

			allowed, err := role_model.CheckUserPermissionToOrganization(ctx, ctx.Doer, tenantId, &organization.Organization{ID: owner.ID}, role_model.READ_PRIVATE)
			if err != nil {
				log.Error("Error has occurred while checking user's permissions: %v", err)
				ctx.PlainText(http.StatusNotFound, "The repository you are trying to reach either does not exist or you are not authorized to view it.")
				return
			}

			if !allowed {

				allow, err := role_model.CheckUserPermissionToTeam(ctx, ctx.Doer, tenantId, &organization.Organization{ID: owner.ID}, &repo_model.Repository{ID: repo.ID}, role_model.ViewBranch.String())
				if err != nil || !allow {
					ctx.PlainText(http.StatusForbidden, "User does not have the required permissions to view this private repository.")
					return
				}
			}
		}
	}

	// Don't allow pushing if the repo is archived
	if repoExist && repo.IsArchived && !isPull {
		ctx.PlainText(http.StatusForbidden, "This repo is archived. You can view files and clone it, but cannot push or open Issues/pull-requests.")
		auditParams["error"] = "Repo is archived"
		audit.CreateAndSendEvent(event, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}

	// Only public pull don't need auth.
	isPublicPull := repoExist && !repo.IsPrivate && isPull
	var (
		askAuth = !isPublicPull || setting.Service.RequireSignInView
		environ []string
	)

	// don't allow anonymous pulls if organization is not public
	if isPublicPull {
		if err := repo.LoadOwner(ctx); err != nil {
			ctx.ServerError("LoadOwner", err)
			auditParams["error"] = "Error has occurred while loading owner"
			audit.CreateAndSendEvent(event, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}

		askAuth = askAuth || (repo.Owner.Visibility != structs.VisibleTypePublic) || setting.SourceControl.TenantWithRoleModeEnabled
	}

	// check access
	if askAuth {
		// rely on the results of Contexter
		if !ctx.IsSigned {
			// TODO: support digit auth - which would be Authorization header with digit
			ctx.Resp.Header().Set("WWW-Authenticate", "Basic realm=\".\"")
			ctx.Error(http.StatusUnauthorized)
			audit.CreateAndSendEvent(audit.UnauthorizedRequestEvent, audit.EmptyRequiredField, audit.EmptyRequiredField, audit.StatusFailure, ctx.Req.RemoteAddr, auditParamsForUnauthorized)
			return
		}

		context.CheckRepoScopedToken(ctx, repo)
		if ctx.Written() {
			auditParams["error"] = "Error has occurred while checking repository scoped token"
			audit.CreateAndSendEvent(event, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}

		if ctx.IsBasicAuth && ctx.Data["IsApiToken"] != true && ctx.Data["IsActionsToken"] != true {
			_, err = auth_model.GetTwoFactorByUID(ctx.Doer.ID)
			if err == nil {
				// TODO: This response should be changed to "invalid credentials" for security reasons once the expectation behind it (creating an app token to authenticate) is properly documented
				ctx.PlainText(http.StatusUnauthorized, "Users with two-factor authentication enabled cannot perform HTTP/HTTPS operations via plain username and password. Please create and use a personal access token on the user settings page")
				audit.CreateAndSendEvent(audit.UnauthorizedRequestEvent, audit.EmptyRequiredField, audit.EmptyRequiredField, audit.StatusFailure, ctx.Req.RemoteAddr, auditParamsForUnauthorized)
				return
			} else if !auth_model.IsErrTwoFactorNotEnrolled(err) {
				ctx.ServerError("IsErrTwoFactorNotEnrolled", err)
				auditParams["error"] = "Error has occurred while getting two factor by user id"
				audit.CreateAndSendEvent(event, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
				return
			}
		}

		if !ctx.Doer.IsActive || ctx.Doer.ProhibitLogin {
			ctx.PlainText(http.StatusForbidden, "Your account is disabled.")
			auditParams["error"] = "Account is disabled"
			audit.CreateAndSendEvent(event, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}

		environ = []string{
			repo_module.EnvRepoUsername + "=" + username,
			repo_module.EnvRepoName + "=" + reponame,
			repo_module.EnvPusherName + "=" + ctx.Doer.Name,
			repo_module.EnvPusherID + fmt.Sprintf("=%d", ctx.Doer.ID),
			repo_module.EnvAppURL + "=" + setting.AppURL,
		}

		if repoExist {
			// Because of special ref "refs/for" .. , need delay write permission check
			if git.SupportProcReceive {
				accessMode = perm.AccessModeRead
			}

			if !setting.SourceControl.TenantWithRoleModeEnabled {
				if ctx.Data["IsActionsToken"] == true {
					taskID := ctx.Data["ActionsTaskID"].(int64)
					task, err := actions_model.GetTaskByID(ctx, taskID)
					if err != nil {
						ctx.ServerError("GetTaskByID", err)
						auditParams["error"] = "Error has occurred while getting task by id"
						audit.CreateAndSendEvent(event, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
						return
					}
					if task.RepoID != repo.ID {
						ctx.PlainText(http.StatusForbidden, "User permission denied")
						auditParams["error"] = "User permission denied"
						audit.CreateAndSendEvent(event, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
						return
					}

					if task.IsForkPullRequest {
						if accessMode > perm.AccessModeRead {
							ctx.PlainText(http.StatusForbidden, "User permission denied")
							auditParams["error"] = "User permission denied"
							audit.CreateAndSendEvent(event, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
							return
						}
						environ = append(environ, fmt.Sprintf("%s=%d", repo_module.EnvActionPerm, perm.AccessModeRead))
					} else {
						if accessMode > perm.AccessModeWrite {
							ctx.PlainText(http.StatusForbidden, "User permission denied")
							auditParams["error"] = "User permission denied"
							audit.CreateAndSendEvent(event, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
							return
						}
						environ = append(environ, fmt.Sprintf("%s=%d", repo_module.EnvActionPerm, perm.AccessModeWrite))
					}
				} else {
					p, err := access_model.GetUserRepoPermission(ctx, repo, ctx.Doer)
					if err != nil {
						ctx.ServerError("GetUserRepoPermission", err)
						auditParams["error"] = "Error has occurred while getting user repository permission"
						audit.CreateAndSendEvent(event, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
						return
					}

					if !p.CanAccess(accessMode, unitType) {
						ctx.PlainText(http.StatusNotFound, "Repository not found")
						auditParams["error"] = "User permission denied"
						audit.CreateAndSendEvent(event, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
						return
					}
				}
			}

			if !isPull && repo.IsMirror {
				ctx.PlainText(http.StatusForbidden, "mirror repository is read-only")
				auditParams["error"] = "Mirror repository is read-only"
				audit.CreateAndSendEvent(event, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
				return
			}
		}

		if !ctx.Doer.KeepEmailPrivate {
			environ = append(environ, repo_module.EnvPusherEmail+"="+ctx.Doer.Email)
		}

		if isWiki {
			environ = append(environ, repo_module.EnvRepoIsWiki+"=true")
		} else {
			environ = append(environ, repo_module.EnvRepoIsWiki+"=false")
		}
	}

	if !repoExist {
		if !receivePack {
			ctx.PlainText(http.StatusNotFound, "Repository not found")
			auditParams["error"] = "Cannot find repository"
			audit.CreateAndSendEvent(event, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}

		if isWiki { // you cannot send wiki operation before create the repository
			ctx.PlainText(http.StatusNotFound, "Repository not found")
			auditParams["error"] = "Cannot find repository"
			audit.CreateAndSendEvent(event, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}

		if owner.IsOrganization() && !setting.Repository.EnablePushCreateOrg {
			ctx.PlainText(http.StatusForbidden, "Push to create is not enabled for organizations.")
			auditParams["error"] = "Push to create is not enabled for organizations"
			audit.CreateAndSendEvent(event, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}
		if !owner.IsOrganization() && !setting.Repository.EnablePushCreateUser {
			ctx.PlainText(http.StatusForbidden, "Push to create is not enabled for users.")
			auditParams["error"] = "Push to create is not enabled for users"
			audit.CreateAndSendEvent(event, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}

		// Return dummy payload if GET receive-pack
		if ctx.Req.Method == http.MethodGet {
			dummyInfoRefs(ctx)
			auditParams["error"] = "Incorrect request"
			audit.CreateAndSendEvent(event, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}

		repo, err = repo_service.PushCreateRepo(ctx, ctx.Doer, owner, reponame)
		if err != nil {
			log.Error("pushCreateRepo: %v", err)
			ctx.Status(http.StatusNotFound)
			auditParams["error"] = "Error has occurred while pushing create repository"
			audit.CreateAndSendEvent(event, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}
	}

	if isWiki {
		// Ensure the wiki is enabled before we allow access to it
		if _, err := repo.GetUnit(ctx, unit.TypeWiki); err != nil {
			if repo_model.IsErrUnitTypeNotExist(err) {
				ctx.PlainText(http.StatusForbidden, "repository wiki is disabled")
				auditParams["error"] = "Repository wiki is disabled"
				audit.CreateAndSendEvent(event, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
				return
			}
			log.Error("Failed to get the wiki unit in %-v Error: %v", repo, err)
			ctx.ServerError("GetUnit(UnitTypeWiki) for "+repo.FullName(), err)
			auditParams["error"] = "Error has occurred while getting wiki unit"
			audit.CreateAndSendEvent(event, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}
	}

	environ = append(environ, repo_module.EnvRepoID+fmt.Sprintf("=%d", repo.ID))

	w := ctx.Resp
	r := ctx.Req
	cfg := &serviceConfig{
		UploadPack:  true,
		ReceivePack: true,
		Env:         environ,
	}

	r.URL.Path = strings.ToLower(r.URL.Path) // blue: In case some repo name has upper case name

	dir := repo_model.RepoPath(username, reponame)
	if isWiki {
		dir = repo_model.RepoPath(username, wikiRepoName)
	}

	return &serviceHandler{cfg, w, r, dir, cfg.Env}
}

var (
	infoRefsCache []byte
	infoRefsOnce  sync.Once
)

func dummyInfoRefs(ctx *context.Context) {
	infoRefsOnce.Do(func() {
		tmpDir, err := os.MkdirTemp(os.TempDir(), "gitea-info-refs-cache")
		if err != nil {
			log.Error("Failed to create temp dir for git-receive-pack cache: %v", err)
			return
		}

		defer func() {
			if err := util.RemoveAll(tmpDir); err != nil {
				log.Error("RemoveAll: %v", err)
			}
		}()

		if err := git.InitRepository(ctx, tmpDir, true); err != nil {
			log.Error("Failed to init bare repo for git-receive-pack cache: %v", err)
			return
		}

		refs, _, err := git.NewCommand(ctx, "receive-pack", "--stateless-rpc", "--advertise-refs", ".").RunStdBytes(&git.RunOpts{Dir: tmpDir})
		if err != nil {
			log.Error(fmt.Sprintf("%v - %s", err, string(refs)))
		}

		log.Debug("populating infoRefsCache: \n%s", string(refs))
		infoRefsCache = refs
	})

	ctx.RespHeader().Set("Expires", "Fri, 01 Jan 1980 00:00:00 GMT")
	ctx.RespHeader().Set("Pragma", "no-cache")
	ctx.RespHeader().Set("Cache-Control", "no-cache, max-age=0, must-revalidate")
	ctx.RespHeader().Set("Content-Type", "application/x-git-receive-pack-advertisement")
	_, _ = ctx.Write(packetWrite("# service=git-receive-pack\n"))
	_, _ = ctx.Write([]byte("0000"))
	_, _ = ctx.Write(infoRefsCache)
}

type serviceConfig struct {
	UploadPack  bool
	ReceivePack bool
	Env         []string
}

type serviceHandler struct {
	cfg     *serviceConfig
	w       http.ResponseWriter
	r       *http.Request
	dir     string
	environ []string
}

func (h *serviceHandler) setHeaderNoCache() {
	h.w.Header().Set("Expires", "Fri, 01 Jan 1980 00:00:00 GMT")
	h.w.Header().Set("Pragma", "no-cache")
	h.w.Header().Set("Cache-Control", "no-cache, max-age=0, must-revalidate")
}

func (h *serviceHandler) setHeaderCacheForever() {
	now := time.Now().Unix()
	expires := now + 31536000
	h.w.Header().Set("Date", fmt.Sprintf("%d", now))
	h.w.Header().Set("Expires", fmt.Sprintf("%d", expires))
	h.w.Header().Set("Cache-Control", "public, max-age=31536000")
}

func containsParentDirectorySeparator(v string) bool {
	if !strings.Contains(v, "..") {
		return false
	}
	for _, ent := range strings.FieldsFunc(v, isSlashRune) {
		if ent == ".." {
			return true
		}
	}
	return false
}

func isSlashRune(r rune) bool { return r == '/' || r == '\\' }

func (h *serviceHandler) sendFile(contentType, file string) {
	if containsParentDirectorySeparator(file) {
		log.Error("request file path contains invalid path: %v", file)
		h.w.WriteHeader(http.StatusBadRequest)
		return
	}
	reqFile := path.Join(h.dir, file)

	fi, err := os.Stat(reqFile)
	if os.IsNotExist(err) {
		h.w.WriteHeader(http.StatusNotFound)
		return
	}

	h.w.Header().Set("Content-Type", contentType)
	h.w.Header().Set("Content-Length", fmt.Sprintf("%d", fi.Size()))
	h.w.Header().Set("Last-Modified", fi.ModTime().Format(http.TimeFormat))
	http.ServeFile(h.w, h.r, reqFile)
}

// one or more key=value pairs separated by colons
var safeGitProtocolHeader = regexp.MustCompile(`^[0-9a-zA-Z]+=[0-9a-zA-Z]+(:[0-9a-zA-Z]+=[0-9a-zA-Z]+)*$`)

func prepareGitCmdWithAllowedService(service string, h *serviceHandler) (*git.Command, error) {
	if service == "receive-pack" && h.cfg.ReceivePack {
		return git.NewCommand(h.r.Context(), "receive-pack"), nil
	}
	if service == "upload-pack" && h.cfg.UploadPack {
		return git.NewCommand(h.r.Context(), "upload-pack"), nil
	}

	return nil, fmt.Errorf("service %q is not allowed", service)
}

func serviceRPC(h *serviceHandler, service string) {
	var event audit.Event
	if service == "upload-pack" {
		event = audit.RepositoryPullOrCloneEvent
	} else if service == "receive-pack" {
		event = audit.ChangesPushEvent
	}

	var auditUserId, auditUserName string
	auditParams := make(map[string]string)

	for _, env := range h.environ {
		if strings.Contains(env, repo_module.EnvRepoName) {
			auditParams["repository"] = strings.ReplaceAll(env, repo_module.EnvRepoName+"=", "")
		}
		if strings.Contains(env, repo_module.EnvPusherID) {
			auditUserId = strings.ReplaceAll(env, repo_module.EnvPusherID+"=", "")
		}
		if strings.Contains(env, repo_module.EnvPusherName) {
			auditUserName = strings.ReplaceAll(env, repo_module.EnvPusherName+"=", "")
		}
	}

	if auditUserId == "" {
		auditUserId = audit.EmptyRequiredField
	}
	if auditUserName == "" {
		auditUserName = audit.EmptyRequiredField
	}
	if auditParams["repository"] == "" {
		split := strings.Split(h.dir, "/")
		auditParams["repository"] = strings.Split(split[len(split)-1], ".")[0]
	}

	defer func() {
		if err := h.r.Body.Close(); err != nil {
			log.Error("serviceRPC: Close: %v", err)
			auditParams["error"] = fmt.Sprintf("Error has occurred while trying %s", strings.ToLower(event.String()))
			audit.CreateAndSendEvent(event, auditUserName, auditUserId, audit.StatusFailure, h.r.RemoteAddr, auditParams)
		}
	}()

	expectedContentType := fmt.Sprintf("application/x-git-%s-request", service)
	if h.r.Header.Get("Content-Type") != expectedContentType {
		log.Error("Content-Type (%q) doesn't match expected: %q", h.r.Header.Get("Content-Type"), expectedContentType)
		h.w.WriteHeader(http.StatusUnauthorized)
		auditParamsForUnauthorized := map[string]string{
			"request_url": h.r.URL.RequestURI(),
		}
		audit.CreateAndSendEvent(audit.UnauthorizedRequestEvent, auditUserName, auditUserId, audit.StatusFailure, h.r.RemoteAddr, auditParamsForUnauthorized)
		return
	}

	h.w.Header().Set("Content-Type", fmt.Sprintf("application/x-git-%s-result", service))

	reqBody := h.r.Body

	var err error
	// Handle GZIP.
	if h.r.Header.Get("Content-Encoding") == "gzip" {
		reqBody, err = gzip.NewReader(reqBody)
		if err != nil {
			log.Error("Fail to create gzip reader: %v", err)
			h.w.WriteHeader(http.StatusInternalServerError)
			auditParams["error"] = fmt.Sprintf("Fail to create gzip reader %s", strings.ToLower(event.String()))
			audit.CreateAndSendEvent(event, auditUserName, auditUserId, audit.StatusFailure, h.r.RemoteAddr, auditParams)
			return
		}
	}

	// set this for allow pre-receive and post-receive execute
	h.environ = append(h.environ, "SSH_ORIGINAL_COMMAND="+service)

	protocol := h.r.Header.Get("Git-Protocol")

	if protocol != "" && safeGitProtocolHeader.MatchString(protocol) {
		h.environ = append(h.environ, "GIT_PROTOCOL="+protocol)
	}

	ctx, sc, err := gitaly.NewSmartHTTPClient(gocontext.Background())
	if err != nil {
		return
	}

	var repo, owner, userId, user string
	for _, env := range h.environ {
		if strings.Contains(env, repo_module.EnvRepoName) {
			repo = strings.ReplaceAll(env, repo_module.EnvRepoName+"=", "")

		}
		if strings.Contains(env, repo_module.EnvRepoUsername) {
			owner = strings.ReplaceAll(env, repo_module.EnvRepoUsername+"=", "")
		}
		if strings.Contains(env, repo_module.EnvPusherID) {
			userId = strings.ReplaceAll(env, repo_module.EnvPusherID+"=", "")
		}
		if strings.Contains(env, repo_module.EnvPusherName) {
			user = strings.ReplaceAll(env, repo_module.EnvPusherName+"=", "")
		}
	}

	if repo == "" {
		split := strings.Split(h.dir, "/")
		repo = strings.Split(split[len(split)-1], ".")[0]
	}

	gitalyRepo := &gitalypb.Repository{
		GlRepository:  repo,
		GlProjectPath: owner,
		RelativePath:  h.dir,
		StorageName:   setting.Gitaly.MainServerName,
	}

	if service == "receive-pack" && h.cfg.ReceivePack {
		if err := sc.ReceivePack(ctx, gitalyRepo, userId, user, repo, nil, reqBody, h.w, protocol); err != nil {
			log.Error("Error has occurred while receiving pack: %v", err)
			auditParams["error"] = "Error has occurred while receiving pack"
			audit.CreateAndSendEvent(event, auditUserName, auditUserId, audit.StatusFailure, h.r.RemoteAddr, auditParams)
			return
		}

		gitRepo, err := git.OpenRepository(ctx, owner, repo, h.dir)
		if err != nil {
			auditParams["error"] = "Error has occurred while opening repository"
			audit.CreateAndSendEvent(event, auditUserName, auditUserId, audit.StatusFailure, h.r.RemoteAddr, auditParams)
			log.Error("Error has occurred while opening repository: %v", err)
			return
		}
		defer gitRepo.Close()

		needSettingDefaultBranch, err := gitRepo.HasOnlyOneBranch()
		if err != nil {
			auditParams["error"] = "Error has occurred while checking the number of branches on server"
			audit.CreateAndSendEvent(event, auditUserName, auditUserId, audit.StatusFailure, h.r.RemoteAddr, auditParams)
			log.Error("Error has occurred while checking the number of branches on server: %v", err)
			return
		}

		if needSettingDefaultBranch {
			if err = utils.SetServerDefaultBranch(ctx, gitRepo); err != nil {
				auditParams["error"] = "Error has occurred while setting default branch on server"
				audit.CreateAndSendEvent(event, auditUserName, auditUserId, audit.StatusFailure, h.r.RemoteAddr, auditParams)
				log.Error("Error has occurred while setting default branch on server: %v", err)
				return
			}
		}
	}
	if service == "upload-pack" && h.cfg.UploadPack {
		_, err := sc.UploadPack(ctx, gitalyRepo, reqBody, h.w, nil, protocol)
		if err != nil {
			return
		}
	}

	if event == audit.RepositoryPullOrCloneEvent {
		audit.CreateAndSendEvent(event, auditUserName, auditUserId, audit.StatusSuccess, h.r.RemoteAddr, auditParams)
	}
}

// ServiceUploadPack implements Git Smart HTTP protocol
func ServiceUploadPack(ctx *context.Context) {
	h := httpBase(ctx)
	if h != nil {
		serviceRPC(h, "upload-pack")
	}
}

// ServiceReceivePack implements Git Smart HTTP protocol
func ServiceReceivePack(ctx *context.Context) {
	h := httpBase(ctx)
	if h != nil {
		serviceRPC(h, "receive-pack")
	}
}

func getServiceType(r *http.Request) string {
	serviceType := r.FormValue("service")
	if !strings.HasPrefix(serviceType, "git-") {
		return ""
	}
	return strings.TrimPrefix(serviceType, "git-")
}

func updateServerInfo(ctx gocontext.Context, dir string) []byte {
	out, _, err := git.NewCommand(ctx, "update-server-info").RunStdBytes(&git.RunOpts{Dir: dir})
	if err != nil {
		log.Error(fmt.Sprintf("%v - %s", err, string(out)))
	}
	return out
}

func packetWrite(str string) []byte {
	s := strconv.FormatInt(int64(len(str)+4), 16)
	if len(s)%4 != 0 {
		s = strings.Repeat("0", 4-len(s)%4) + s
	}
	return []byte(s + str)
}

// GetInfoRefs implements Git dumb HTTP
func GetInfoRefs(ctx *context.Context) {
	h := httpBase(ctx)
	if h == nil {
		return
	}
	h.setHeaderNoCache()
	service := getServiceType(h.r)

	protocol := h.r.Header.Get("Git-Protocol")
	if protocol != "" && safeGitProtocolHeader.MatchString(protocol) {
		h.environ = append(h.environ, "GIT_PROTOCOL="+protocol)
	}

	ctx1, sc, err := gitaly.NewSmartHTTPClient(ctx)
	if err != nil {
		return
	}

	var repo, owner string
	for _, env := range h.environ {
		if strings.Contains(env, repo_module.EnvRepoName) {
			repo = strings.ReplaceAll(env, repo_module.EnvRepoName+"=", "")

		}
		if strings.Contains(env, repo_module.EnvRepoUsername) {
			owner = strings.ReplaceAll(env, repo_module.EnvRepoUsername+"=", "")
		}
	}

	if repo == "" {
		split := strings.Split(h.dir, "/")
		repo = strings.Split(split[len(split)-1], ".")[0]
	}

	if ctx.Doer != nil {
		dbEngine := db.GetEngine(ctx)
		taskDB := code_hub_counter_task_db.New(dbEngine)
		taskCreator := code_hub_counter.NewTaskCreator(taskDB, setting.CodeHub.CodeHubMetricEnabled)
		if err = taskCreator.CreateByRepoNameOwner(ctx, repo, owner, ctx.Doer.ID, code_hub_counter_task.CloneRepositoryAction); err != nil {
			log.Error("error has occurred while while inserting repository task: %v", err)
		}
	}

	gitalyRepo := &gitalypb.Repository{
		GlRepository:  repo,
		GlProjectPath: owner,
		RelativePath:  h.dir,
		StorageName:   setting.Gitaly.MainServerName,
	}
	req := &gitalypb.InfoRefsRequest{
		Repository:  gitalyRepo,
		GitProtocol: protocol,
	}

	if service == "receive-pack" && h.cfg.ReceivePack {
		resp, err := sc.InfoRefsReceivePack(ctx1, req)
		if err != nil {
			return
		}

		h.w.Header().Set("Content-Type", fmt.Sprintf("application/x-git-%s-advertisement", service))

		h.w.WriteHeader(http.StatusOK)

		canRead := true
		for canRead {
			recvM, err := resp.Recv()
			if err != nil {
				return
			}
			if recvM == nil {
				canRead = false
			} else {
				_, _ = h.w.Write(recvM.Data)
			}
		}
	}
	if service == "upload-pack" && h.cfg.UploadPack {
		resp, err := sc.InfoRefsUploadPack(ctx1, req)
		if err != nil {
			return
		}
		h.w.Header().Set("Content-Type", fmt.Sprintf("application/x-git-%s-advertisement", service))
		h.w.WriteHeader(http.StatusOK)
		canRead := true
		for canRead {
			recvM, err := resp.Recv()
			if err != nil {
				return
			}
			if recvM == nil {
				canRead = false
			} else {
				_, _ = h.w.Write(recvM.Data)
			}
		}
	}

}

// GetTextFile implements Git dumb HTTP
func GetTextFile(p string) func(*context.Context) {
	return func(ctx *context.Context) {
		h := httpBase(ctx)
		if h != nil {
			h.setHeaderNoCache()
			file := ctx.Params("file")
			if file != "" {
				h.sendFile("text/plain", "objects/info/"+file)
			} else {
				h.sendFile("text/plain", p)
			}
		}
	}
}

// GetInfoPacks implements Git dumb HTTP
func GetInfoPacks(ctx *context.Context) {
	h := httpBase(ctx)
	if h != nil {
		h.setHeaderCacheForever()
		h.sendFile("text/plain; charset=utf-8", "objects/info/packs")
	}
}

// GetLooseObject implements Git dumb HTTP
func GetLooseObject(ctx *context.Context) {
	h := httpBase(ctx)
	if h != nil {
		h.setHeaderCacheForever()
		h.sendFile("application/x-git-loose-object", fmt.Sprintf("objects/%s/%s",
			ctx.Params("head"), ctx.Params("hash")))
	}
}

// GetPackFile implements Git dumb HTTP
func GetPackFile(ctx *context.Context) {
	h := httpBase(ctx)
	if h != nil {
		h.setHeaderCacheForever()
		h.sendFile("application/x-git-packed-objects", "objects/pack/pack-"+ctx.Params("file")+".pack")
	}
}

// GetIdxFile implements Git dumb HTTP
func GetIdxFile(ctx *context.Context) {
	h := httpBase(ctx)
	if h != nil {
		h.setHeaderCacheForever()
		h.sendFile("application/x-git-packed-objects-toc", "objects/pack/pack-"+ctx.Params("file")+".idx")
	}
}
