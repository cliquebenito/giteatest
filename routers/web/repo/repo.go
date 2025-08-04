// Copyright 2014 The Gogs Authors. All rights reserved.
// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	gocontext "context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"code.gitea.io/gitea/models"
	"code.gitea.io/gitea/models/code_hub_counter_task"
	"code.gitea.io/gitea/models/db"
	git_model "code.gitea.io/gitea/models/git"
	"code.gitea.io/gitea/models/organization"
	access_model "code.gitea.io/gitea/models/perm/access"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/repo_marks"
	"code.gitea.io/gitea/models/role_model"
	"code.gitea.io/gitea/models/tenant"
	"code.gitea.io/gitea/models/unit"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/base"
	"code.gitea.io/gitea/modules/cache"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/indexer/stats"
	"code.gitea.io/gitea/modules/log"
	repo_module "code.gitea.io/gitea/modules/repository"
	"code.gitea.io/gitea/modules/sbt/audit"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/storage"
	api "code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/modules/trace"
	"code.gitea.io/gitea/modules/util"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/routers/utils"
	"code.gitea.io/gitea/services/convert"
	"code.gitea.io/gitea/services/forms"
	"code.gitea.io/gitea/services/org"
	repo_service "code.gitea.io/gitea/services/repository"
	archiver_service "code.gitea.io/gitea/services/repository/archiver"
)

const (
	TplCreate       base.TplName = "repo/create"
	TplAlertDetails base.TplName = "base/alert_details"
)

type RepoServer struct {
	taskCreator
	repoMarksDB
	processedMarks []repo_marks.RepoMark
	marksEnabled   bool
}

type repoMarksDB interface {
	GetRepoMarks(ctx gocontext.Context, repoID int64) ([]repo_marks.RepoMarks, error)
}

type taskCreator interface {
	Create(ctx gocontext.Context, repoID int64, userID int64, action code_hub_counter_task.CodeHubAction) error
}

func NewRepoServer(creator taskCreator, marksDB repoMarksDB, processedMarks []repo_marks.RepoMark, marksEnabled bool) RepoServer {
	return RepoServer{taskCreator: creator, repoMarksDB: marksDB, processedMarks: processedMarks, marksEnabled: marksEnabled}
}

// MustBeNotEmpty render when a repo is a empty git dir
func MustBeNotEmpty(ctx *context.Context) {
	if ctx.Repo.Repository.IsEmpty {
		ctx.NotFound("MustBeNotEmpty", nil)
	}
}

// MustBeEditable check that repo can be edited
func MustBeEditable(ctx *context.Context) {
	if !ctx.Repo.Repository.CanEnableEditor() || ctx.Repo.IsViewCommit {
		ctx.NotFound("", nil)
		return
	}
}

// MustBeAbleToUpload check that repo can be uploaded to
func MustBeAbleToUpload(ctx *context.Context) {
	if !setting.Repository.Upload.Enabled {
		ctx.NotFound("", nil)
	}
}

func CommitInfoCache(ctx *context.Context) {
	var err error
	ctx.Repo.Commit, err = ctx.Repo.GitRepo.GetBranchCommit(ctx.Repo.Repository.DefaultBranch)
	if err != nil {
		ctx.ServerError("GetBranchCommit", err)
		return
	}
	ctx.Repo.CommitsCount, err = ctx.Repo.GetCommitsCount()
	if err != nil {
		ctx.ServerError("GetCommitsCount", err)
		return
	}
	ctx.Data["CommitsCount"] = ctx.Repo.CommitsCount
	ctx.Repo.GitRepo.LastCommitCache = git.NewLastCommitCache(ctx.Repo.CommitsCount, ctx.Repo.Repository.FullName(), ctx.Repo.GitRepo, cache.GetCache())
}

func checkContextUser(ctx *context.Context, uid int64) *user_model.User {
	orgs, err := organization.GetOrgsCanCreateRepoByUserID(ctx.Doer.ID)
	if err != nil {
		ctx.ServerError("GetOrgsCanCreateRepoByUserID", err)
		return nil
	}

	if !ctx.Doer.IsAdmin {
		orgsAvailable := []*organization.Organization{}
		for i := 0; i < len(orgs); i++ {
			if orgs[i].CanCreateRepo() {
				orgsAvailable = append(orgsAvailable, orgs[i])
			}
		}
		ctx.Data["Orgs"] = orgsAvailable
	} else {
		ctx.Data["Orgs"] = orgs
	}

	// Not equal means current user is an organization.
	if uid == ctx.Doer.ID || uid == 0 {
		// При включенной ролевой модели пользователь не может форкать репозиторий себе
		if setting.SourceControl.TenantWithRoleModeEnabled {
			ctx.Error(http.StatusForbidden)
			return nil
		}
		return ctx.Doer
	}

	org, err := user_model.GetUserByID(ctx, uid)
	if user_model.IsErrUserNotExist(err) {
		return ctx.Doer
	}

	if err != nil {
		ctx.ServerError("GetUserByID", fmt.Errorf("[%d]: %w", uid, err))
		return nil
	}

	// Check ownership of organization.
	if !org.IsOrganization() {
		ctx.Error(http.StatusForbidden)
		return nil
	}
	if !ctx.Doer.IsAdmin {
		canCreate, err := organization.OrgFromUser(org).CanCreateOrgRepo(ctx.Doer.ID)
		if err != nil {
			ctx.ServerError("CanCreateOrgRepo", err)
			return nil
		} else if !canCreate {
			ctx.Error(http.StatusForbidden)
			return nil
		}
	} else {
		ctx.Data["Orgs"] = orgs
	}
	return org
}

func getRepoPrivate(ctx *context.Context) bool {
	switch strings.ToLower(setting.Repository.DefaultPrivate) {
	case setting.RepoCreatingLastUserVisibility:
		return ctx.Doer.LastRepoVisibility
	case setting.RepoCreatingPrivate:
		return true
	case setting.RepoCreatingPublic:
		return false
	default:
		return ctx.Doer.LastRepoVisibility
	}
}

// Create render creating repository page
func Create(ctx *context.Context) {
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

	ctx.Data["Title"] = ctx.Tr("new_repo")

	// Give default value for template to render.
	ctx.Data["Gitignores"] = repo_module.Gitignores
	ctx.Data["LabelTemplateFiles"] = repo_module.LabelTemplateFiles
	ctx.Data["Licenses"] = repo_module.Licenses
	ctx.Data["Readmes"] = repo_module.Readmes
	ctx.Data["readme"] = "Default"
	ctx.Data["private"] = getRepoPrivate(ctx)
	ctx.Data["IsForcedPrivate"] = setting.Repository.ForcePrivate
	ctx.Data["default_branch"] = setting.Repository.DefaultBranch

	ctxUser := checkContextUser(ctx, ctx.FormInt64("org"))
	if ctx.Written() {
		return
	}
	ctx.Data["ContextUser"] = ctxUser

	// если у нас включена ролевая модель SourceControl, то запускается проверка привилегий на создание репозитория
	if setting.SourceControl.TenantWithRoleModeEnabled {
		tenantId, err := tenant.GetTenantByOrgIdOrDefault(ctx, ctxUser.ID)
		if err != nil {
			ctx.NotFound(ctx.Req.URL.RequestURI(), nil)
			return
		}

		allowed, err := role_model.CheckUserPermissionToOrganization(ctx, ctx.Doer, tenantId, &organization.Organization{ID: ctxUser.ID}, role_model.CREATE)
		if err != nil || !allowed {
			ctx.NotFound(ctx.Req.URL.RequestURI(), nil)
			return
		}
	}

	ctx.Data["repo_template_name"] = ctx.Tr("repo.template_select")
	templateID := ctx.FormInt64("template_id")
	if templateID > 0 {
		templateRepo, err := repo_model.GetRepositoryByID(ctx, templateID)
		if err == nil && access_model.CheckRepoUnitUser(ctx, templateRepo, ctxUser, unit.TypeCode) {
			ctx.Data["repo_template"] = templateID
			ctx.Data["repo_template_name"] = templateRepo.Name
		}
	}

	ctx.Data["CanCreateRepo"] = ctx.Doer.CanCreateRepo()
	ctx.Data["MaxCreationLimit"] = ctx.Doer.MaxCreationLimit()

	ctx.HTML(http.StatusOK, TplCreate)
}

func handleCreateError(ctx *context.Context, owner *user_model.User, err error, name string, tpl base.TplName, form interface{}) {
	auditParams := map[string]string{
		"repository": (*form.(**forms.CreateRepoForm)).RepoName,
		"owner":      owner.Name,
	}
	switch {
	case repo_model.IsErrReachLimitOfRepo(err):
		maxCreationLimit := owner.MaxCreationLimit()
		msg := ctx.TrN(maxCreationLimit, "repo.form.reach_limit_of_creation_1", "repo.form.reach_limit_of_creation_n", maxCreationLimit)
		ctx.RenderWithErr(msg, tpl, form)
		auditParams["error"] = "Reached limit of repositories"
	case repo_model.IsErrRepoAlreadyExist(err):
		ctx.Data["Err_RepoName"] = true
		ctx.RenderWithErr(ctx.Tr("form.repo_name_been_taken"), tpl, form)
		auditParams["error"] = "Repository name been taken"
	case repo_model.IsErrRepoFilesAlreadyExist(err):
		ctx.Data["Err_RepoName"] = true
		auditParams["error"] = "Files already exist for this repository"
		switch {
		case ctx.IsUserSiteAdmin() || (setting.Repository.AllowAdoptionOfUnadoptedRepositories && setting.Repository.AllowDeleteOfUnadoptedRepositories):
			ctx.RenderWithErr(ctx.Tr("form.repository_files_already_exist.adopt_or_delete"), tpl, form)
		case setting.Repository.AllowAdoptionOfUnadoptedRepositories:
			ctx.RenderWithErr(ctx.Tr("form.repository_files_already_exist.adopt"), tpl, form)
		case setting.Repository.AllowDeleteOfUnadoptedRepositories:
			ctx.RenderWithErr(ctx.Tr("form.repository_files_already_exist.delete"), tpl, form)
		default:
			ctx.RenderWithErr(ctx.Tr("form.repository_files_already_exist"), tpl, form)
		}
	case db.IsErrNameReserved(err):
		ctx.Data["Err_RepoName"] = true
		ctx.RenderWithErr(ctx.Tr("repo.form.name_reserved", err.(db.ErrNameReserved).Name), tpl, form)
		auditParams["error"] = "Name reserved"
	case db.IsErrNamePatternNotAllowed(err):
		ctx.Data["Err_RepoName"] = true
		ctx.RenderWithErr(ctx.Tr("repo.form.name_pattern_not_allowed", err.(db.ErrNamePatternNotAllowed).Pattern), tpl, form)
		auditParams["error"] = "Name pattern not allowed"
	case repo_model.IsErrCreateUserRepo(err):
		ctx.RenderWithErr(ctx.Tr("repo.create_user_repo_not_allowed"), tpl, form)
		auditParams["error"] = "Creating a repository outside the project is prohibited"
	default:
		ctx.ServerError(name, err)
		auditParams["error"] = "Error has occurred while creating repository"
	}
	audit.CreateAndSendEvent(audit.RepositoryCreateEvent, owner.Name, strconv.FormatInt(owner.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
}

// CreatePost response for creating repository
// todo create receiver
func (s *Server) CreatePost(ctx *context.Context) {
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

	form := web.GetForm(ctx).(*forms.CreateRepoForm)
	ctx.Data["Title"] = ctx.Tr("new_repo")
	ctx.Data["Gitignores"] = repo_module.Gitignores
	ctx.Data["LabelTemplateFiles"] = repo_module.LabelTemplateFiles
	ctx.Data["Licenses"] = repo_module.Licenses
	ctx.Data["Readmes"] = repo_module.Readmes

	ctx.Data["CanCreateRepo"] = ctx.Doer.CanCreateRepo()
	ctx.Data["MaxCreationLimit"] = ctx.Doer.MaxCreationLimit()
	ctxUser := checkContextUser(ctx, form.UID)
	auditParams := map[string]string{
		"repository": form.RepoName,
		"owner":      ctxUser.Name,
	}
	if ctx.Written() {
		auditParams["error"] = "Failed to check context user"
		audit.CreateAndSendEvent(audit.RepositoryCreateEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}
	ctx.Data["ContextUser"] = ctxUser

	if ctx.HasError() {
		ctx.HTML(http.StatusOK, TplCreate)
		auditParams["error"] = "Error occurs in form validation"
		audit.CreateAndSendEvent(audit.RepositoryCreateEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}
	// если у нас включена ролевая модель SourceControl, то запускается проверка привилегий на создание репозитория
	if setting.SourceControl.TenantWithRoleModeEnabled {
		var err error
		tenantId, err := tenant.GetTenantByOrgIdOrDefault(ctx, ctxUser.ID)
		if err != nil {
			ctx.NotFound(ctx.Req.URL.RequestURI(), nil)
			return
		}

		allowed, err := role_model.CheckUserPermissionToOrganization(ctx, ctx.Doer, tenantId, &organization.Organization{ID: ctxUser.ID}, role_model.CREATE)
		if err != nil || !allowed {
			ctx.NotFound(ctx.Req.URL.RequestURI(), nil)
			return
		}
	}

	var repo *repo_model.Repository
	if form.RepoTemplate > 0 {
		opts := repo_module.GenerateRepoOptions{
			Name:        form.RepoName,
			Description: form.Description,
			Private:     form.Private,
			GitContent:  form.GitContent,
			Topics:      form.Topics,
			GitHooks:    form.GitHooks,
			Webhooks:    form.Webhooks,
			Avatar:      form.Avatar,
			IssueLabels: form.Labels,
		}

		if !opts.IsValid() {
			ctx.RenderWithErr(ctx.Tr("repo.template.one_item"), TplCreate, form)
			auditParams["error"] = "Options isn't valid"
			audit.CreateAndSendEvent(audit.RepositoryCreateEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}

		templateRepo := getRepository(ctx, form.RepoTemplate)
		if ctx.Written() {
			auditParams["error"] = "Failed to get template repository"
			audit.CreateAndSendEvent(audit.RepositoryCreateEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}

		if !templateRepo.IsTemplate {
			ctx.RenderWithErr(ctx.Tr("repo.template.invalid"), TplCreate, form)
			auditParams["error"] = "Template isn't valid"
			audit.CreateAndSendEvent(audit.RepositoryCreateEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}
		// todo: create repo
		repo, err = repo_service.GenerateRepository(ctx, ctx.Doer, ctxUser, templateRepo, opts)
		if err == nil {
			auditParams["repository_id"] = strconv.FormatInt(repo.ID, 10)
			audit.CreateAndSendEvent(audit.RepositoryCreateEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)

			log.Trace("Repository generated [%d]: %s/%s", repo.ID, ctxUser.Name, repo.Name)
			ctx.Redirect(repo.Link())
			return
		}
	} else {
		// todo: create repo
		repo, err = repo_service.CreateRepository(ctx, ctx.Doer, ctxUser, repo_module.CreateRepoOptions{
			Name:          form.RepoName,
			Description:   form.Description,
			Gitignores:    form.Gitignores,
			IssueLabels:   form.IssueLabels,
			License:       form.License,
			Readme:        form.Readme,
			IsPrivate:     form.Private || setting.Repository.ForcePrivate,
			DefaultBranch: form.DefaultBranch,
			AutoInit:      form.AutoInit,
			IsTemplate:    form.Template,
			TrustModel:    repo_model.ToTrustModel(form.TrustModel),
		})
		if err == nil {
			s.SyncPrivileges(ctx, ctxUser, repo)
			auditParams["repository_id"] = strconv.FormatInt(repo.ID, 10)
			audit.CreateAndSendEvent(audit.RepositoryCreateEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)

			log.Trace("Repository created [%d]: %s/%s", repo.ID, ctxUser.Name, repo.Name)
			if form.License != "" {
				statsDbIndexer := stats.DBIndexer{}
				errDbIndexer := statsDbIndexer.Index(repo.ID)
				if errDbIndexer != nil {
					log.Error(fmt.Sprintf("Failed while analyzing licenses for repository_id %v: %v", repo.ID, errDbIndexer))
				}
			}
			ctx.Redirect(repo.Link())
			return
		}
	}

	handleCreateError(ctx, ctxUser, err, "CreatePost", TplCreate, &form)
}
func (s *Server) SyncPrivileges(ctx *context.Context, ctxUser *user_model.User, repo *repo_model.Repository) {
	auditParams := map[string]string{
		"repository_name": repo.Name,
		"project_name":    ctxUser.Name,
	}
	teams, err := organization.GetTeamsByOrgID(ctx, ctxUser.ID)
	if err != nil {
		auditParams["error"] = "Error has occurred while getting teams"
		audit.CreateAndSendEvent(audit.UpdateCustomPrivilegesEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		log.Error("Error has occurred while getting team. Error: %v", err)
		return
	}
	if len(teams) == 0 {
		log.Warn("teams is not found")
		return
	}

	for _, team := range teams {
		teamPrivileges, err := s.casbinCustomRepo.GetCustomPrivilegesByTeam(team.Name)
		if err != nil {
			auditParams["error"] = "Error has occurred while getting custom privileges"
			audit.CreateAndSendEvent(audit.UpdateCustomPrivilegesEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			log.Error("Error has occurred while getting custom privileges. Error: %v", err)
			return
		}
		for _, privilege := range teamPrivileges {
			if !privilege.AllRepositories {
				continue
			}

			if err := org.TeamAddRepository(team, repo); err != nil {
				auditParams["error"] = "Error has occurred while adding team in repository"
				audit.CreateAndSendEvent(audit.UpdateCustomPrivilegesEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
				log.Error("Error has occurred while adding team in repository. Error: %v", err)
				return
			}

			privileges, err := parseCustomPrivileges(privilege.CustomPrivileges)
			if err != nil {
				auditParams["error"] = "Error has occurred while adding parse custom privileges"
				audit.CreateAndSendEvent(audit.UpdateCustomPrivilegesEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
				log.Error("Error has occurred while parse custom privileges. Error: %v", err)
				return
			}

			custommm := forms.CustomPrivileges{
				AllRepositories: privilege.AllRepositories,
				RepoID:          repo.ID,
				Privileges:      privileges,
			}

			if err := s.creator.UpdateCustomPrivilegeToTeamUser(ctx, ctxUser.ID, team.Name, []forms.CustomPrivileges{custommm}); err != nil {
				auditParams["error"] = "Error has occurred while updating custom privileges"
				audit.CreateAndSendEvent(audit.UpdateCustomPrivilegesEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
				log.Error("Error has occurred while updating custom privileges Error: %v", err)
				return
			}
		}
	}
	audit.CreateAndSendEvent(audit.UpdateCustomPrivilegesEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)
	return
}

func parseCustomPrivileges(privilegesStr string) ([]role_model.CustomPrivilege, error) {
	var privileges []role_model.CustomPrivilege

	ids := strings.Split(privilegesStr, ",")
	for _, idStr := range ids {
		idStr = strings.TrimSpace(idStr)
		id, err := strconv.Atoi(idStr)
		if err != nil {
			return nil, fmt.Errorf("invalid privilege id '%s': %w", idStr, err)
		}
		privileges = append(privileges, role_model.CustomPrivilege(id))
	}
	return privileges, nil
}

// Action response for actions to a repository
func Action(ctx *context.Context) {
	var err error
	switch ctx.Params(":action") {
	case "watch":
		err = repo_model.WatchRepo(ctx, ctx.Doer.ID, ctx.Repo.Repository.ID, true)
	case "unwatch":
		err = repo_model.WatchRepo(ctx, ctx.Doer.ID, ctx.Repo.Repository.ID, false)
	case "star":
		err = repo_model.StarRepo(ctx.Doer.ID, ctx.Repo.Repository.ID, true)
	case "unstar":
		err = repo_model.StarRepo(ctx.Doer.ID, ctx.Repo.Repository.ID, false)
	case "accept_transfer":
		err = acceptOrRejectRepoTransfer(ctx, true)
	case "reject_transfer":
		err = acceptOrRejectRepoTransfer(ctx, false)
	case "desc": // FIXME: this is not used
		if !ctx.Repo.IsOwner() {
			ctx.Error(http.StatusNotFound)
			return
		}

		ctx.Repo.Repository.Description = ctx.FormString("desc")
		ctx.Repo.Repository.Website = ctx.FormString("site")
		err = repo_service.UpdateRepository(ctx, ctx.Repo.Repository, false)
	}

	if err != nil {
		ctx.ServerError(fmt.Sprintf("Action (%s)", ctx.Params(":action")), err)
		return
	}

	ctx.RedirectToFirst(ctx.FormString("redirect_to"), ctx.Repo.RepoLink)
}

func acceptOrRejectRepoTransfer(ctx *context.Context, accept bool) error {
	repoTransfer, err := models.GetPendingRepositoryTransfer(ctx, ctx.Repo.Repository)
	if err != nil {
		return err
	}

	if err := repoTransfer.LoadAttributes(ctx); err != nil {
		return err
	}

	if !repoTransfer.CanUserAcceptTransfer(ctx.Doer) {
		return errors.New("user does not have enough permissions")
	}

	if accept {
		if ctx.Repo.GitRepo != nil {
			ctx.Repo.GitRepo.Close()
			ctx.Repo.GitRepo = nil
		}

		if err := repo_service.TransferOwnership(ctx, repoTransfer.Doer, repoTransfer.Recipient, ctx.Repo.Repository, repoTransfer.Teams); err != nil {
			return err
		}
		ctx.Flash.Success(ctx.Tr("repo.settings.transfer.success"))
	} else {
		if err := models.CancelRepositoryTransfer(ctx.Repo.Repository); err != nil {
			return err
		}
		ctx.Flash.Success(ctx.Tr("repo.settings.transfer.rejected"))
	}

	ctx.Redirect(ctx.Repo.Repository.Link())
	return nil
}

// RedirectDownload return a file based on the following infos:
func RedirectDownload(ctx *context.Context) {
	var (
		vTag     = ctx.Params("vTag")
		fileName = ctx.Params("fileName")
	)
	tagNames := []string{vTag}
	curRepo := ctx.Repo.Repository
	releases, err := repo_model.GetReleasesByRepoIDAndNames(ctx, curRepo.ID, tagNames)
	if err != nil {
		if repo_model.IsErrAttachmentNotExist(err) {
			ctx.Error(http.StatusNotFound)
			return
		}
		ctx.ServerError("RedirectDownload", err)
		return
	}
	if len(releases) == 1 {
		release := releases[0]
		att, err := repo_model.GetAttachmentByReleaseIDFileName(ctx, release.ID, fileName)
		if err != nil {
			ctx.Error(http.StatusNotFound)
			return
		}
		if att != nil {
			ServeAttachment(ctx, att.UUID)
			return
		}
	}
	ctx.Error(http.StatusNotFound)
}

// Download an archive of a repository
func (r RepoServer) Download(ctx *context.Context) {
	uri := ctx.Params("*")
	aReq, err := archiver_service.NewRequest(ctx.Repo.Repository.ID, ctx.Repo.GitRepo, uri)
	if err != nil {
		if errors.Is(err, archiver_service.ErrUnknownArchiveFormat{}) {
			ctx.Error(http.StatusBadRequest, err.Error())
		} else if errors.Is(err, archiver_service.RepoRefNotFoundError{}) {
			ctx.Error(http.StatusNotFound, err.Error())
		} else {
			ctx.ServerError("archiver_service.NewRequest", err)
		}
		return
	}

	archiver, err := aReq.Await(ctx)
	if err != nil {
		ctx.ServerError("archiver.Await", err)
		return
	}

	r.download(ctx, aReq.GetArchiveName(), archiver)
}

func (r RepoServer) download(ctx *context.Context, archiveName string, archiver *repo_model.RepoArchiver) {
	downloadName := ctx.Repo.Repository.Name + "-" + archiveName

	rPath := archiver.RelativePath()
	if setting.RepoArchive.ServeDirect {
		// If we have a signed url (S3, object storage), redirect to this directly.
		u, err := storage.RepoArchives.URL(rPath, downloadName)
		if u != nil && err == nil {
			ctx.Redirect(u.String())
			return
		}
	}

	// If we have matched and access to release or issue
	fr, err := storage.RepoArchives.Open(rPath)
	if err != nil {
		ctx.ServerError("Open", err)
		return
	}
	defer fr.Close()

	if ctx.Doer != nil {
		if err = r.taskCreator.Create(*ctx, ctx.Repo.Repository.ID, ctx.Doer.ID, code_hub_counter_task.CloneRepositoryAction); err != nil {
			log.Error("error has occurred while while inserting repository task: %v", err)
		}
	}

	ctx.ServeContent(fr, &context.ServeHeaderOptions{
		Filename:     downloadName,
		LastModified: archiver.CreatedUnix.AsLocalTime(),
	})
}

// InitiateDownload will enqueue an archival request, as needed.  It may submit
// a request that's already in-progress, but the archiver service will just
// kind of drop it on the floor if this is the case.
func InitiateDownload(ctx *context.Context) {
	uri := ctx.Params("*")
	aReq, err := archiver_service.NewRequest(ctx.Repo.Repository.ID, ctx.Repo.GitRepo, uri)
	if err != nil {
		ctx.ServerError("archiver_service.NewRequest", err)
		return
	}
	if aReq == nil {
		ctx.Error(http.StatusNotFound)
		return
	}

	archiver, err := repo_model.GetRepoArchiver(ctx, aReq.RepoID, aReq.Type, aReq.CommitID)
	if err != nil {
		ctx.ServerError("archiver_service.StartArchive", err)
		return
	}
	if archiver == nil || archiver.Status != repo_model.ArchiverReady {
		if err := archiver_service.StartArchive(aReq); err != nil {
			ctx.ServerError("archiver_service.StartArchive", err)
			return
		}
	}

	var completed bool
	if archiver != nil && archiver.Status == repo_model.ArchiverReady {
		completed = true
	}

	ctx.JSON(http.StatusOK, map[string]interface{}{
		"complete": completed,
	})
}

// SearchRepo repositories via options
func SearchRepo(ctx *context.Context) {
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

	a := ctx.Req.Form.Get("uid")
	userID, errStrConv := strconv.Atoi(a)
	if errStrConv != nil {
		ctx.Error(http.StatusNotFound, fmt.Sprintf("strconv.Atoi failed: %v", errStrConv))
		return
	}
	opts := &repo_model.SearchRepoOptions{
		ListOptions: db.ListOptions{
			Page:     ctx.FormInt("page"),
			PageSize: convert.ToCorrectPageSize(ctx.FormInt("limit")),
		},
		Actor:              ctx.Doer,
		Keyword:            ctx.FormTrim("q"),
		OwnerID:            ctx.FormInt64("uid"),
		PriorityOwnerID:    ctx.FormInt64("priority_owner_id"),
		TeamID:             ctx.FormInt64("team_id"),
		TopicOnly:          ctx.FormBool("topic"),
		Collaborate:        util.OptionalBoolNone,
		Private:            ctx.IsSigned && (ctx.FormString("private") == "" || ctx.FormBool("private")),
		Template:           util.OptionalBoolNone,
		StarredByID:        ctx.FormInt64("starredBy"),
		IncludeDescription: ctx.FormBool("includeDesc"),
	}
	user, errGetOrg := organization.GetOrganizationByIDs(ctx, []int64{opts.OwnerID})
	if errGetOrg != nil {
		ctx.Error(http.StatusNotFound, fmt.Sprintf("organization.GetOrganizationByIDs failed: %v", errGetOrg))
		return
	}
	if setting.SourceControl.TenantWithRoleModeEnabled {
		privileges, err := utils.GetTenantsPrivilegesByUserID(ctx, int64(userID))
		if err != nil {
			ctx.Error(http.StatusNotFound, fmt.Sprintf("SearchRepo role_model.GetPrivilegesByUserId failed: %v", err))
			return
		}

		var ownerIDs []int64
		if user[0].Type == user_model.UserTypeIndividual {
			privilegeOrganizations := utils.ConvertPrivilegesTenantFromOrganizationsOrUsers(privileges, user_model.UserTypeOrganization)
			ownerIDs = append(ownerIDs, utils.ConvertMapUserOrOrganizationsInSlice(privilegeOrganizations)...)
		} else {
			ownerIDs = []int64{opts.OwnerID}
		}
		opts.OwnerIDs = ownerIDs
	}

	if ctx.FormString("template") != "" {
		opts.Template = util.OptionalBoolOf(ctx.FormBool("template"))
	}

	if ctx.FormBool("exclusive") {
		opts.Collaborate = util.OptionalBoolFalse
	}

	mode := ctx.FormString("mode")
	switch mode {
	case "source":
		opts.Fork = util.OptionalBoolFalse
		opts.Mirror = util.OptionalBoolFalse
	case "fork":
		opts.Fork = util.OptionalBoolTrue
	case "mirror":
		opts.Mirror = util.OptionalBoolTrue
	case "collaborative":
		opts.Mirror = util.OptionalBoolFalse
		opts.Collaborate = util.OptionalBoolTrue
	case "":
	default:
		ctx.Error(http.StatusUnprocessableEntity, fmt.Sprintf("Invalid search mode: \"%s\"", mode))
		return
	}

	if ctx.FormString("archived") != "" {
		opts.Archived = util.OptionalBoolOf(ctx.FormBool("archived"))
	}

	if ctx.FormString("is_private") != "" {
		opts.IsPrivate = util.OptionalBoolOf(ctx.FormBool("is_private"))
	}

	sortMode := ctx.FormString("sort")
	if len(sortMode) > 0 {
		sortOrder := ctx.FormString("order")
		if len(sortOrder) == 0 {
			sortOrder = "asc"
		}
		if searchModeMap, ok := repo_model.SearchOrderByMap[sortOrder]; ok {
			if orderBy, ok := searchModeMap[sortMode]; ok {
				opts.OrderBy = orderBy
			} else {
				ctx.Error(http.StatusUnprocessableEntity, fmt.Sprintf("Invalid sort mode: \"%s\"", sortMode))
				return
			}
		} else {
			ctx.Error(http.StatusUnprocessableEntity, fmt.Sprintf("Invalid sort order: \"%s\"", sortOrder))
			return
		}
	}

	repos, count, err := repo_model.SearchRepository(ctx, opts)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, api.SearchError{
			OK:    false,
			Error: err.Error(),
		})
		return
	}

	filteredRepos := repo_model.RepositoryList{}
	if setting.SourceControl.TenantWithRoleModeEnabled {
		if user[0].Type == user_model.UserTypeOrganization {
			filteredRepos = repos
		} else {
			tenantID, err := role_model.GetUserTenantId(ctx, int64(userID))
			if err != nil {
				log.Error("GetUserTenantId: %v", err)
				return
			}
			ten, errGetTenantID := tenant.GetTenantByID(ctx, tenantID)
			if errGetTenantID != nil {
				log.Error("SearchRepo GetUserTenantId failed: %v", errGetTenantID)
				return
			}
			if ten.IsActive {
				signedUser := ctx.Data["SignedUser"].(*user_model.User)
				for _, repo := range repos {
					action := role_model.READ
					if repo.IsPrivate {
						action = role_model.READ_PRIVATE
					}
					allowed, err := role_model.CheckUserPermissionToOrganization(ctx, &user_model.User{ID: signedUser.ID},
						tenantID,
						&organization.Organization{ID: repo.OwnerID},
						action)
					if err != nil {
						log.Error("Error has occurred while checking user's permissions: %v", err)
						ctx.JSON(http.StatusInternalServerError, fmt.Sprintf("Error has occurred while checking user's permissions: %v", err))
						return
					}
					if !allowed {
						allow, err := role_model.CheckUserPermissionToTeam(ctx,
							&user_model.User{ID: signedUser.ID},
							tenantID,
							&organization.Organization{ID: repo.OwnerID},
							repo,
							role_model.ViewBranch.String(),
						)
						if err != nil {
							log.Error("Error has occurred while checking user's permissions: %v", err)
							ctx.JSON(http.StatusInternalServerError, fmt.Sprintf("Error has occurred while checking user's permissions: %v", err))
							return
						}
						if !allow {
							continue
						}
					}
					filteredRepos = append(filteredRepos, repo)
				}
			}

			repos = filteredRepos
			count = int64(len(filteredRepos))
		}

		ctx.SetTotalCountHeader(count)

		// To improve performance when only the count is requested
		if ctx.FormBool("count_only") {
			return
		}

		// collect the latest commit of each repo
		// at most there are dozens of repos (limited by MaxResponseItems), so it's not a big problem at the moment
		repoIDsToLatestCommitSHAs := make(map[int64]string, len(repos))
		for _, repo := range repos {
			commitID, err := repo_service.GetBranchCommitID(ctx, repo, repo.DefaultBranch)
			if err != nil {
				continue
			}
			repoIDsToLatestCommitSHAs[repo.ID] = commitID
		}

		// call the database O(1) times to get the commit statuses for all repos
		repoToItsLatestCommitStatuses, err := git_model.GetLatestCommitStatusForPairs(ctx, repoIDsToLatestCommitSHAs, db.ListOptions{})
		if err != nil {
			log.Error("GetLatestCommitStatusForPairs: %v", err)
			return
		}

		results := make([]*repo_service.WebSearchRepository, len(repos))
		for i, repo := range repos {
			results[i] = &repo_service.WebSearchRepository{
				Repository: &api.Repository{
					ID:       repo.ID,
					FullName: repo.FullName(),
					Fork:     repo.IsFork,
					Private:  repo.IsPrivate,
					Template: repo.IsTemplate,
					Mirror:   repo.IsMirror,
					Stars:    repo.NumStars,
					HTMLURL:  repo.HTMLURL(),
					Link:     repo.Link(),
					Internal: !repo.IsPrivate && repo.Owner.Visibility == api.VisibleTypePrivate,
				},
				LatestCommitStatus: git_model.CalcCommitStatus(repoToItsLatestCommitStatuses[repo.ID]),
			}
		}

		ctx.JSON(http.StatusOK, repo_service.WebSearchResults{
			OK:   true,
			Data: results,
		})
	}
}
