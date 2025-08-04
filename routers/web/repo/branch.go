// Copyright 2014 The Gogs Authors. All rights reserved.
// Copyright 2018 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"code.gitea.io/gitea/models"
	git_model "code.gitea.io/gitea/models/git"
	"code.gitea.io/gitea/models/git/protected_branch"
	issues_model "code.gitea.io/gitea/models/issues"
	"code.gitea.io/gitea/models/organization"
	"code.gitea.io/gitea/models/organization/custom"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/role_model"
	"code.gitea.io/gitea/models/unit"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/base"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/log"
	repo_module "code.gitea.io/gitea/modules/repository"
	"code.gitea.io/gitea/modules/sbt/audit"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/trace"
	"code.gitea.io/gitea/modules/util"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/routers/utils"
	"code.gitea.io/gitea/routers/web/user/team_server"
	"code.gitea.io/gitea/services/forms"
	release_service "code.gitea.io/gitea/services/release"
	repo_service "code.gitea.io/gitea/services/repository"
	files_service "code.gitea.io/gitea/services/repository/files"
)

const (
	tplBranch base.TplName = "repo/branch/list"
)

// Branch contains the branch information
type Branch struct {
	Name              string
	Commit            *git.Commit
	IsProtected       bool
	IsDeleted         bool
	IsIncluded        bool
	DeletedBranch     *git_model.DeletedBranch
	CommitsAhead      int
	CommitsBehind     int
	LatestPullRequest *issues_model.PullRequest
	MergeMovedOn      bool
}

func New(casbinPermissioner context.RolePermissioner, custom custom.CustomPrivileger, creator *team_server.Server) Server {
	return Server{
		creator:            creator,
		casbinCustomRepo:   custom,
		casbinPermissioner: casbinPermissioner}
}

type Server struct {
	casbinPermissioner context.RolePermissioner
	casbinCustomRepo   custom.CustomPrivileger
	creator            *team_server.Server
}

// Branches render repository branch page
func (s Server) Branches(ctx *context.Context) {
	var (
		canWrite bool
		canPull  bool
		err      error
	)
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

	ctx.Data["Title"] = "Branches"
	ctx.Data["IsRepoToolbarBranches"] = true
	ctx.Data["DefaultBranch"] = ctx.Repo.Repository.DefaultBranch
	ctx.Data["AllowsPulls"] = ctx.Repo.Repository.AllowsPulls()

	canWrite, err = ctx.Repo.CanUseBranchWriteActionButton(ctx, s.casbinPermissioner, ctx.Doer)
	if err != nil {
		log.Error("Error has occurred while check if user can use write branch actions %v", err)
		ctx.Error(http.StatusInternalServerError, "Error has occurred while check if user can use write branch actions")
		return
	}
	ctx.Data["IsWriter"] = canWrite
	ctx.Data["IsMirror"] = ctx.Repo.Repository.IsMirror

	canPull, err = ctx.Repo.CanUseBranchPullActonButton(ctx, s.casbinPermissioner, ctx.Doer)
	if err != nil {
		log.Error("Error has occurred while check if user can use pull branch actions %v", err)
		ctx.Error(http.StatusInternalServerError, "Error has occurred while check if user can use pull branch actions")
		return
	}
	ctx.Data["CanPull"] = canPull
	ctx.Data["PageIsViewCode"] = true
	ctx.Data["PageIsBranches"] = true

	page := ctx.FormInt("page")
	if page <= 1 {
		page = 1
	}
	pageSize := setting.Git.BranchesRangeSize

	skip := (page - 1) * pageSize
	log.Debug("Branches: skip: %d limit: %d", skip, pageSize)
	defaultBranchBranch, branches, branchesCount := loadBranches(ctx, skip, pageSize)
	if ctx.Written() {
		return
	}

	action := role_model.READ
	if ctx.Repo.Repository.IsPrivate {
		action = role_model.READ_PRIVATE
	}

	allowed, err := role_model.CheckUserPermissionToOrganization(ctx, &user_model.User{ID: ctx.Doer.ID},
		ctx.Data["TenantID"].(string),
		&organization.Organization{ID: ctx.Repo.Repository.OwnerID},
		action)
	if err != nil {
		log.Error("Error has occurred while checking user's permission: %v", err)
		ctx.ServerError("Error has occurred while checking user's permission: %v", err)
		return
	}
	if !allowed {
		allow, err := role_model.CheckUserPermissionToTeam(ctx,
			&user_model.User{ID: ctx.Doer.ID},
			ctx.Data["TenantID"].(string),
			&organization.Organization{ID: ctx.Repo.Repository.OwnerID},
			&repo_model.Repository{ID: ctx.Repo.Repository.ID},
			role_model.ViewBranch.String(),
		)
		if err != nil || !allow {
			log.Error("Error has occurred while checking user's permission: %v", err)
			ctx.ServerError("Error has occurred while checking user's permission: %v", err)
			return
		}
	}

	ctx.Data["Branches"] = branches
	ctx.Data["BranchesCount"] = branchesCount
	ctx.Data["DefaultBranchBranch"] = defaultBranchBranch

	pager := context.NewPagination(branchesCount, pageSize, page, 5)
	pager.SetDefaultParams(ctx)
	ctx.Data["Page"] = pager

	ctx.HTML(http.StatusOK, tplBranch)
}

// DeleteBranchPost responses for delete merged branch
func DeleteBranchPost(ctx *context.Context) {
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

	defer redirect(ctx)
	branchName := ctx.FormString("name")
	auditParams := map[string]string{
		"branch_name": branchName,
	}

	allowed, err := role_model.CheckUserPermissionToOrganization(ctx, &user_model.User{ID: ctx.Doer.ID},
		ctx.Data["TenantID"].(string),
		&organization.Organization{ID: ctx.Repo.Repository.OwnerID},
		role_model.WRITE)
	if err != nil {
		log.Error("Error has occurred while checking user's permissions: %v", err)
		ctx.Error(http.StatusForbidden, fmt.Sprintf("Error has occurred while checking user's permissions: %v", err))
		return
	}
	if !allowed {
		allow, err := role_model.CheckUserPermissionToTeam(ctx,
			&user_model.User{ID: ctx.Doer.ID},
			ctx.Data["TenantID"].(string),
			&organization.Organization{ID: ctx.Repo.Repository.OwnerID},
			&repo_model.Repository{ID: ctx.Repo.Repository.ID},
			role_model.ChangeBranch.String(),
		)
		if err != nil || !allow {
			log.Error("Error has occurred while checking user's permissions: %v", err)
			ctx.Error(http.StatusForbidden, fmt.Sprintf("Error has occurred while checking user's permissions: %v", err))
			return
		}
	}

	if err := repo_service.DeleteBranch(ctx, ctx.Doer, ctx.Repo.Repository, ctx.Repo.GitRepo, branchName); err != nil {
		switch {
		case git.IsErrBranchNotExist(err):
			log.Debug("DeleteBranch: Can't delete non existing branch '%s'", branchName)
			auditParams["error"] = "Can't delete non existing branch"
			ctx.Flash.Error(ctx.Tr("repo.branch.deletion_failed", branchName))
		case errors.Is(err, repo_service.ErrBranchIsDefault):
			log.Debug("DeleteBranch: Can't delete default branch '%s'", branchName)
			auditParams["error"] = "Can't delete default branch"
			ctx.Flash.Error(ctx.Tr("repo.branch.default_deletion_failed", branchName))
		case protected_branch.IsBranchIsProtectedError(err):
			log.Debug("DeleteBranch: Can't delete protected branch '%s'", branchName)
			auditParams["error"] = "Can't delete protected branch"
			ctx.Flash.Error(ctx.Tr("repo.branch.protected_deletion_failed", branchName))
		default:
			log.Error("DeleteBranch: %v", err)
			auditParams["error"] = "Error has occurred while deleting branch"
			ctx.Flash.Error(ctx.Tr("repo.branch.deletion_failed", branchName))
		}
		audit.CreateAndSendEvent(audit.BranchDeleteEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}
	ctx.Flash.Success(ctx.Tr("repo.branch.deletion_success", branchName))
	audit.CreateAndSendEvent(audit.BranchDeleteEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)
}

// RestoreBranchPost responses for delete merged branch
func RestoreBranchPost(ctx *context.Context) {
	defer redirect(ctx)

	branchID := ctx.FormInt64("branch_id")
	branchName := ctx.FormString("name")

	deletedBranch, err := git_model.GetDeletedBranchByID(ctx, ctx.Repo.Repository.ID, branchID)
	if err != nil {
		log.Error("GetDeletedBranchByID: %v", err)
		ctx.Flash.Error(ctx.Tr("repo.branch.restore_failed", branchName))
		return
	} else if deletedBranch == nil {
		log.Debug("RestoreBranch: Can't restore branch[%d] '%s', as it does not exist", branchID, branchName)
		ctx.Flash.Error(ctx.Tr("repo.branch.restore_failed", branchName))
		return
	}
	err = repo_service.CreateNewBranchFromCommit(ctx, ctx.Doer, ctx.Repo.Repository, deletedBranch.Commit, branchName)
	if err != nil {
		log.Error("CreateNewBranchFromCommit: %v", err)
	}
	err = git_model.RemoveDeletedBranchByID(ctx, ctx.Repo.Repository.ID, branchID)
	if err != nil {
	}

	// Don't return error below this
	if err := repo_service.PushUpdate(
		&repo_module.PushUpdateOptions{
			RefFullName:  git.BranchPrefix + deletedBranch.Name,
			OldCommitID:  git.EmptySHA,
			NewCommitID:  deletedBranch.Commit,
			PusherID:     ctx.Doer.ID,
			PusherName:   ctx.Doer.Name,
			RepoUserName: ctx.Repo.Owner.Name,
			RepoName:     ctx.Repo.Repository.Name,
		}); err != nil {
		log.Error("RestoreBranch: Update: %v", err)
	}

	ctx.Flash.Success(ctx.Tr("repo.branch.restore_success", deletedBranch.Name))
}

func redirect(ctx *context.Context) {
	ctx.JSON(http.StatusOK, map[string]interface{}{
		"redirect": ctx.Repo.RepoLink + "/branches?page=" + url.QueryEscape(ctx.FormString("page")),
	})
}

// loadBranches loads branches from the repository limited by page & pageSize.
// NOTE: May write to context on error.
func loadBranches(ctx *context.Context, skip, limit int) (*Branch, []*Branch, int) {
	defaultBranch, err := ctx.Repo.GitRepo.GetBranch(ctx.Repo.Repository.DefaultBranch)
	if err != nil {
		if !git.IsErrBranchNotExist(err) {
			log.Error("loadBranches: get default branch: %v", err)
			ctx.ServerError("GetDefaultBranch", err)
			return nil, nil, 0
		}
		log.Warn("loadBranches: missing default branch %s for %-v", ctx.Repo.Repository.DefaultBranch, ctx.Repo.Repository)
	}

	rawBranches, totalNumOfBranches, err := ctx.Repo.GitRepo.GetBranches(skip, limit)
	if err != nil {
		log.Error("GetBranches: %v", err)
		ctx.ServerError("GetBranches", err)
		return nil, nil, 0
	}

	rules, err := git_model.FindRepoProtectedBranchRules(ctx, ctx.Repo.Repository.ID)
	if err != nil {
		ctx.ServerError("FindRepoProtectedBranchRules", err)
		return nil, nil, 0
	}

	repoIDToRepo := map[int64]*repo_model.Repository{}
	repoIDToRepo[ctx.Repo.Repository.ID] = ctx.Repo.Repository

	repoIDToGitRepo := map[int64]*git.Repository{}
	repoIDToGitRepo[ctx.Repo.Repository.ID] = ctx.Repo.GitRepo

	var branches []*Branch
	for i := range rawBranches {
		if defaultBranch != nil && rawBranches[i].Name == defaultBranch.Name {
			// Skip default branch
			continue
		}

		branch := loadOneBranch(ctx, rawBranches[i], defaultBranch, &rules, repoIDToRepo, repoIDToGitRepo)
		if branch == nil {
			return nil, nil, 0
		}

		branches = append(branches, branch)
	}

	var defaultBranchBranch *Branch
	if defaultBranch != nil {
		// Always add the default branch
		log.Debug("loadOneBranch: load default: '%s'", defaultBranch.Name)
		defaultBranchBranch = loadOneBranch(ctx, defaultBranch, defaultBranch, &rules, repoIDToRepo, repoIDToGitRepo)
		branches = append(branches, defaultBranchBranch)
	}

	if ctx.Repo.CanWrite(unit.TypeCode) {
		deletedBranches, err := getDeletedBranches(ctx)
		if err != nil {
			ctx.ServerError("getDeletedBranches", err)
			return nil, nil, 0
		}
		branches = append(branches, deletedBranches...)
	}

	return defaultBranchBranch, branches, totalNumOfBranches
}

func loadOneBranch(ctx *context.Context, rawBranch, defaultBranch *git.Branch,
	protectedBranches *protected_branch.ProtectedBranchRules,
	repoIDToRepo map[int64]*repo_model.Repository,
	repoIDToGitRepo map[int64]*git.Repository,
) *Branch {
	log.Trace("loadOneBranch: '%s'", rawBranch.Name)

	commit, err := rawBranch.GetCommit()
	if err != nil {
		ctx.ServerError("GetCommit", err)
		return nil
	}

	branchName := rawBranch.Name
	p := git_model.GetFirstMatched(*protectedBranches, branchName)
	isProtected := p != nil

	divergence := &git.DivergeObject{
		Ahead:  -1,
		Behind: -1,
	}
	if defaultBranch != nil {
		divergence, err = files_service.CountDivergingCommits(ctx, ctx.Repo.Repository, git.BranchPrefix+branchName)
		if err != nil {
			log.Error("CountDivergingCommits", err)
		}
	}

	pr, err := issues_model.GetLatestPullRequestByHeadInfo(ctx.Repo.Repository.ID, branchName)
	if err != nil {
		ctx.ServerError("GetLatestPullRequestByHeadInfo", err)
		return nil
	}
	headCommit := commit.ID.String()

	mergeMovedOn := false
	if pr != nil {
		pr.HeadRepo = ctx.Repo.Repository
		if err := pr.LoadIssue(ctx); err != nil {
			ctx.ServerError("LoadIssue", err)
			return nil
		}
		if repo, ok := repoIDToRepo[pr.BaseRepoID]; ok {
			pr.BaseRepo = repo
		} else if err := pr.LoadBaseRepo(ctx); err != nil {
			ctx.ServerError("LoadBaseRepo", err)
			return nil
		} else {
			repoIDToRepo[pr.BaseRepoID] = pr.BaseRepo
		}
		pr.Issue.Repo = pr.BaseRepo

		if pr.HasMerged {
			baseGitRepo, ok := repoIDToGitRepo[pr.BaseRepoID]
			if !ok {
				baseGitRepo, err = git.OpenRepository(ctx, pr.BaseRepo.OwnerName, pr.BaseRepo.Name, pr.BaseRepo.RepoPath())
				if err != nil {
					ctx.ServerError("OpenRepository", err)
					return nil
				}
				defer baseGitRepo.Close()
				repoIDToGitRepo[pr.BaseRepoID] = baseGitRepo
			}
			pullCommit, err := baseGitRepo.GetRefCommitID(pr.GetGitRefName())
			if err != nil && !git.IsErrNotExist(err) {
				ctx.ServerError("GetBranchCommitID", err)
				return nil
			}
			if err == nil && headCommit != pullCommit {
				// the head has moved on from the merge - we shouldn't delete
				mergeMovedOn = true
			}
		}
	}

	isIncluded := divergence.Ahead == 0 && ctx.Repo.Repository.DefaultBranch != branchName
	return &Branch{
		Name:              branchName,
		Commit:            commit,
		IsProtected:       isProtected,
		IsIncluded:        isIncluded,
		CommitsAhead:      divergence.Ahead,
		CommitsBehind:     divergence.Behind,
		LatestPullRequest: pr,
		MergeMovedOn:      mergeMovedOn,
	}
}

func getDeletedBranches(ctx *context.Context) ([]*Branch, error) {
	branches := []*Branch{}

	deletedBranches, err := git_model.GetDeletedBranches(ctx, ctx.Repo.Repository.ID)
	if err != nil {
		return branches, err
	}

	for i := range deletedBranches {
		deletedBranches[i].LoadUser(ctx)
		branches = append(branches, &Branch{
			Name:          deletedBranches[i].Name,
			IsDeleted:     true,
			DeletedBranch: deletedBranches[i],
		})
	}

	return branches, nil
}

// CreateBranch creates new branch in repository
func CreateBranch(ctx *context.Context) {
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

	form := web.GetForm(ctx).(*forms.NewBranchForm)
	auditParams := map[string]string{
		"branch_name": form.NewBranchName,
	}
	var err error

	allowed, err := role_model.CheckUserPermissionToOrganization(ctx, &user_model.User{ID: ctx.Doer.ID},
		ctx.Data["TenantID"].(string),
		&organization.Organization{ID: ctx.Repo.Repository.OwnerID},
		role_model.WRITE)
	if err != nil {
		log.Error("Error has occurred while checking user's permissions: %v", err)
		ctx.Error(http.StatusForbidden, fmt.Sprintf("Error has occurred while checking user's permissions: %v", err))
		return
	}
	if !allowed {
		allow, err := role_model.CheckUserPermissionToTeam(ctx,
			&user_model.User{ID: ctx.Doer.ID},
			ctx.Data["TenantID"].(string),
			&organization.Organization{ID: ctx.Repo.Repository.OwnerID},
			&repo_model.Repository{ID: ctx.Repo.Repository.ID},
			role_model.ChangeBranch.String(),
		)
		if err != nil || !allow {
			log.Error("Error has occurred while checking user's permissions: %v", err)
			ctx.Error(http.StatusForbidden, fmt.Sprintf("Error has occurred while checking user's permissions: %v", err))
			return
		}
	}

	if !ctx.Repo.CanCreateBranch() {
		ctx.NotFound("CreateBranch", nil)
		auditParams["error"] = "Repository is not editable or the user does not have the proper access level"
		audit.CreateAndSendEvent(audit.BranchCreateEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}

	if ctx.HasError() {
		ctx.Flash.Error(ctx.GetErrMsg())
		ctx.Redirect(ctx.Repo.RepoLink + "/src/" + ctx.Repo.BranchNameSubURL())
		auditParams["error"] = "Error occurs in form validation"
		audit.CreateAndSendEvent(audit.BranchCreateEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}

	if form.CreateTag {
		commit := ctx.Repo.CommitID
		target := ctx.Repo.BranchName
		err = release_service.CreateNewTag(ctx, ctx.Doer, ctx.Repo.Repository, commit, target, form.NewBranchName, "")
	} else if ctx.Repo.IsViewBranch {
		err = repo_service.CreateNewBranch(ctx, ctx.Doer, ctx.Repo.Repository, ctx.Repo.BranchName, form.NewBranchName)
	} else {
		err = repo_service.CreateNewBranchFromCommit(ctx, ctx.Doer, ctx.Repo.Repository, ctx.Repo.CommitID, form.NewBranchName)
	}
	if err != nil {
		if models.IsErrProtectedTagName(err) {
			ctx.Flash.Error(ctx.Tr("repo.release.tag_name_protected"))
			ctx.Redirect(ctx.Repo.RepoLink + "/src/" + ctx.Repo.BranchNameSubURL())
			return
		}

		if models.IsErrTagAlreadyExists(err) {
			e := err.(models.ErrTagAlreadyExists)
			ctx.Flash.Error(ctx.Tr("repo.branch.tag_collision", e.TagName))
			ctx.Redirect(ctx.Repo.RepoLink + "/src/" + ctx.Repo.BranchNameSubURL())
			return
		}
		if models.IsErrBranchAlreadyExists(err) || git.IsErrPushOutOfDate(err) {
			ctx.Flash.Error(ctx.Tr("repo.branch.branch_already_exists", form.NewBranchName))
			ctx.Redirect(ctx.Repo.RepoLink + "/src/" + ctx.Repo.BranchNameSubURL())
			auditParams["error"] = "Branch already exists"
			audit.CreateAndSendEvent(audit.BranchCreateEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}
		if models.IsErrBranchNameConflict(err) {
			e := err.(models.ErrBranchNameConflict)
			ctx.Flash.Error(ctx.Tr("repo.branch.branch_name_conflict", form.NewBranchName, e.BranchName))
			ctx.Redirect(ctx.Repo.RepoLink + "/src/" + ctx.Repo.BranchNameSubURL())
			auditParams["error"] = "Branch name conflict"
			audit.CreateAndSendEvent(audit.BranchCreateEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}
		if git.IsErrPushRejected(err) {
			e := err.(*git.ErrPushRejected)
			if len(e.Message) == 0 {
				ctx.Flash.Error(ctx.Tr("repo.editor.push_rejected_no_message"))
				auditParams["error"] = "Push rejected because no message"
			} else {
				flashError, err := ctx.RenderToString(TplAlertDetails, map[string]interface{}{
					"Message": ctx.Tr("repo.editor.push_rejected"),
					"Summary": ctx.Tr("repo.editor.push_rejected_summary"),
					"Details": utils.SanitizeFlashErrorString(e.Message),
				})
				if err != nil {
					ctx.ServerError("UpdatePullRequest.HTMLString", err)
					auditParams["error"] = "Error has occurred while render template to string"
					audit.CreateAndSendEvent(audit.BranchCreateEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
					return
				}
				ctx.Flash.Error(flashError)
				auditParams["error"] = "Push rejected"
			}
			ctx.Redirect(ctx.Repo.RepoLink + "/src/" + ctx.Repo.BranchNameSubURL())
			audit.CreateAndSendEvent(audit.BranchCreateEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}

		ctx.ServerError("CreateNewBranch", err)
		auditParams["error"] = "Error has occurred while creating new branch"
		audit.CreateAndSendEvent(audit.BranchCreateEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}

	if form.CreateTag {
		ctx.Flash.Success(ctx.Tr("repo.tag.create_success", form.NewBranchName))
		ctx.Redirect(ctx.Repo.RepoLink + "/src/tag/" + util.PathEscapeSegments(form.NewBranchName))
		return
	}

	ctx.Flash.Success(ctx.Tr("repo.branch.create_success", form.NewBranchName))
	ctx.Redirect(ctx.Repo.RepoLink + "/src/branch/" + util.PathEscapeSegments(form.NewBranchName) + "/" + util.PathEscapeSegments(form.CurrentPath))

	audit.CreateAndSendEvent(audit.BranchCreateEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)
}
