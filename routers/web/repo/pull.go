// Copyright 2018 The Gitea Authors.
// Copyright 2014 The Gogs Authors.
// All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	contextDefault "context"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"code.gitea.io/gitea/modules/trace"
	"gitlab.com/gitlab-org/gitaly/v16/proto/go/gitalypb"

	"code.gitea.io/gitea/models"
	activities_model "code.gitea.io/gitea/models/activities"
	"code.gitea.io/gitea/models/db"
	git_model "code.gitea.io/gitea/models/git"
	"code.gitea.io/gitea/models/git/protected_branch"
	issues_model "code.gitea.io/gitea/models/issues"
	"code.gitea.io/gitea/models/organization"
	access_model "code.gitea.io/gitea/models/perm/access"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/role_model"
	"code.gitea.io/gitea/models/tenant"
	"code.gitea.io/gitea/models/unit"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/base"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/notification"
	"code.gitea.io/gitea/modules/sbt/audit"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/routers/utils"
	"code.gitea.io/gitea/services/automerge"
	"code.gitea.io/gitea/services/forms"
	pull_service "code.gitea.io/gitea/services/pull"
	repo_service "code.gitea.io/gitea/services/repository"

	"github.com/gobwas/glob"
)

const (
	tplFork        base.TplName = "repo/pulls/fork"
	TplCompareDiff base.TplName = "repo/diff/compare"
)

func getRepository(ctx *context.Context, repoID int64) *repo_model.Repository {
	repo, err := repo_model.GetRepositoryByID(ctx, repoID)
	if err != nil {
		if repo_model.IsErrRepoNotExist(err) {
			ctx.NotFound("GetRepositoryByID", nil)
		} else {
			ctx.ServerError("GetRepositoryByID", err)
		}
		return nil
	}

	perm, err := access_model.GetUserRepoPermission(ctx, repo, ctx.Doer)
	if err != nil {
		ctx.ServerError("GetUserRepoPermission", err)
		return nil
	}

	if !perm.CanRead(unit.TypeCode) {
		log.Trace("Permission Denied: User %-v cannot read %-v of repo %-v\n"+
			"User in repo has Permissions: %-+v",
			ctx.Doer,
			unit.TypeCode,
			ctx.Repo,
			perm)
		ctx.NotFound("getRepository", nil)
		return nil
	}
	return repo
}

func getForkRepository(ctx *context.Context) *repo_model.Repository {
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

	forkRepo := getRepository(ctx, ctx.ParamsInt64(":repoid"))
	if ctx.Written() {
		return nil
	}

	if forkRepo.IsEmpty {
		log.Trace("Empty repository %-v", forkRepo)
		ctx.NotFound("empty repository", nil)
		return nil
	}

	if err := forkRepo.LoadOwner(ctx); err != nil {
		ctx.ServerError("load owner of repository", err)
		return nil
	}

	ctx.Data["repo_name"] = forkRepo.Name
	ctx.Data["description"] = forkRepo.Description
	ctx.Data["IsPrivate"] = forkRepo.IsPrivate || forkRepo.Owner.Visibility == structs.VisibleTypePrivate
	canForkToUser := forkRepo.OwnerID != ctx.Doer.ID && !repo_model.HasForkedRepo(ctx.Doer.ID, forkRepo.ID) && !setting.SourceControl.TenantWithRoleModeEnabled

	ctx.Data["ForkRepo"] = forkRepo

	var clearOrgs []*organization.Organization
	var orgOwner *organization.Organization

	if setting.SourceControl.TenantWithRoleModeEnabled {
		// TenantWithRoleModeEnabled = true получаем проекты по тенатнам
		// Проверьте, является ли пользователь может создать репозиторий.
		tenantID, err := role_model.GetUserTenantId(db.DefaultContext, ctx.Doer.ID)
		if err != nil {
			log.Error("error has occurred while get user tenant id: %v", err)
			ctx.ServerError("error has occurred while get user tenant id", err)
			return nil //todo механика -- беда
		}

		allPrivileges, err := role_model.GetAllPrivileges()
		if err != nil {
			log.Error("error has occurred while get all privileges for user: %v", err)
			ctx.ServerError("error has occurred while get all privileges for user: %v", err)
			return nil
		}
		for _, privilege := range allPrivileges {
			if privilege.User.ID == ctx.Doer.ID {
				// если мы сейчас в этой организации - пропускаем
				if privilege.Org.ID == ctx.Repo.Repository.OwnerID {
					orgOwner = privilege.Org
					continue
				}
				// проверяем, является ли пользователь может создать репозиторий в организации
				if ok, err := role_model.CheckUserPermissionToOrganization(ctx, ctx.Doer, tenantID, privilege.Org, role_model.CREATE); err != nil {
					log.Error("error has occurred while checking user permission to organization: %v", err)
					ctx.ServerError("error has occurred while checking user permission to organization. %v", err)
					return nil //todo механика -- беда
				} else if ok {
					clearOrgs = append(clearOrgs, privilege.Org)
				}
			}
		}
	} else {
		ownedOrgs, err := organization.GetOrgsCanCreateRepoByUserID(ctx.Doer.ID)
		if err != nil {
			log.Error("error has occurred while checking organization can create repository: %v", err)
			ctx.ServerError("error has occurred while checking organization can create repository", err)
			return nil
		}
		for _, org := range ownedOrgs {
			if org.ID == ctx.Repo.Repository.OwnerID {
				orgOwner = org
				continue
			}
			if forkRepo.OwnerID != org.ID && !repo_model.HasForkedRepo(org.ID, forkRepo.ID) {
				clearOrgs = append(clearOrgs, org)
			}
		}
	}

	ctx.Data["CanForkToUser"] = canForkToUser
	ctx.Data["Orgs"] = clearOrgs

	if canForkToUser {
		ctx.Data["ContextUser"] = ctx.Doer
	} else if len(clearOrgs) > 0 {
		ctx.Data["ContextUser"] = orgOwner
	}

	return forkRepo
}

// Fork render repository fork page
func Fork(ctx *context.Context) {
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

	ctx.Data["Title"] = ctx.Tr("new_fork")

	// если у нас включена ролевая модель SourceControl, то запускается проверка привилегий на создание репозитория
	if setting.SourceControl.TenantWithRoleModeEnabled {
		tenantId, err := tenant.GetTenantByOrgIdOrDefault(ctx, ctx.Repo.Repository.OwnerID)
		if err != nil {
			ctx.NotFound(ctx.Req.URL.RequestURI(), nil)
			return
		}

		allowed, err := role_model.CheckUserPermissionToOrganization(ctx, ctx.Doer, tenantId, &organization.Organization{ID: ctx.Repo.Repository.OwnerID}, role_model.CREATE)
		if err != nil || !allowed {
			ctx.NotFound(ctx.Req.URL.RequestURI(), nil)
			return
		}
	}

	if ctx.Doer.CanForkRepo() {
		ctx.Data["CanForkRepo"] = true
	} else {
		maxCreationLimit := ctx.Doer.MaxCreationLimit()
		msg := ctx.TrN(maxCreationLimit, "repo.form.reach_limit_of_creation_1", "repo.form.reach_limit_of_creation_n", maxCreationLimit)
		ctx.Data["Flash"] = ctx.Flash
		ctx.Flash.Error(msg)
	}

	getForkRepository(ctx)
	if ctx.Written() {
		return
	}

	ctx.HTML(http.StatusOK, tplFork)
}

// ForkPost response for forking a repository
func ForkPost(ctx *context.Context) {
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
	ctx.Data["Title"] = ctx.Tr("new_fork")
	ctx.Data["CanForkRepo"] = true

	auditParams := map[string]string{
		"repository":           form.RepoName,
		"forked_repository_id": strconv.FormatInt(ctx.ParamsInt64(":repoid"), 10),
	}

	ctxUser := checkContextUser(ctx, form.UID)
	if ctx.Written() {
		auditParams["error"] = "Error has occurred while checking context user"
		audit.CreateAndSendEvent(audit.RepositoryForkEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}
	// если у нас включена ролевая модель SourceControl, то запускается проверка привилегий на создание репозитория
	if setting.SourceControl.TenantWithRoleModeEnabled {
		tenantId, err := tenant.GetTenantByOrgIdOrDefault(ctx, ctxUser.ID)
		if err != nil {
			ctx.NotFound(ctx.Req.URL.RequestURI(), nil)
			auditParams["error"] = "Error has occurred while checking context user"
			audit.CreateAndSendEvent(audit.RepositoryForkEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}

		allowed, err := role_model.CheckUserPermissionToOrganization(ctx, ctx.Doer, tenantId, &organization.Organization{ID: ctxUser.ID}, role_model.CREATE)
		if err != nil || !allowed {
			ctx.NotFound(ctx.Req.URL.RequestURI(), nil)
			auditParams["error"] = "Error has occurred while checking context user"
			audit.CreateAndSendEvent(audit.RepositoryForkEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}
	}

	auditParams["affected_user_id"] = strconv.FormatInt(ctxUser.ID, 10)
	auditParams["affected_user"] = ctxUser.Name

	forkRepo := getForkRepository(ctx)
	if ctx.Written() {
		auditParams["error"] = "Error has occurred while checking context user"
		audit.CreateAndSendEvent(audit.RepositoryForkEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}
	auditParams["forked_repository_id"] = strconv.FormatInt(forkRepo.ID, 10)
	auditParams["forked_repository"] = forkRepo.Name

	ctx.Data["ContextUser"] = ctxUser

	if ctx.HasError() {
		auditParams["error"] = "Error occurs in form validation"
		audit.CreateAndSendEvent(audit.RepositoryForkEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		ctx.HTML(http.StatusOK, tplFork)
		return
	}

	traverseParentRepo := forkRepo
	for {
		if ctxUser.ID == traverseParentRepo.OwnerID {
			auditParams["error"] = "The new owner already has a repository with same name"
			audit.CreateAndSendEvent(audit.RepositoryForkEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			ctx.RenderWithErr(ctx.Tr("repo.settings.new_owner_has_same_repo"), tplFork, &form)
			return
		}
		repo := repo_model.GetForkedRepo(ctxUser.ID, traverseParentRepo.ID)
		if repo != nil {
			auditParams["error"] = "User has already forked a repository with given repository id"
			audit.CreateAndSendEvent(audit.RepositoryForkEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			ctx.Redirect(ctxUser.HomeLink() + "/" + url.PathEscape(repo.Name))
			return
		}
		if !traverseParentRepo.IsFork {
			break
		}
		traverseParentRepo, err = repo_model.GetRepositoryByID(ctx, traverseParentRepo.ForkID)
		if err != nil {
			auditParams["error"] = "Error has occurred while getting repository by id"
			audit.CreateAndSendEvent(audit.RepositoryForkEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			ctx.ServerError("GetRepositoryByID", err)
			return
		}
	}

	// Check if user is allowed to create repo's on the organization.
	if ctxUser.IsOrganization() {
		isAllowedToFork, err := organization.OrgFromUser(ctxUser).CanCreateOrgRepo(ctx.Doer.ID)
		if err != nil {
			auditParams["error"] = "Error has occurred while checking possibility creating repo's on the organization"
			audit.CreateAndSendEvent(audit.RepositoryForkEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			ctx.ServerError("CanCreateOrgRepo", err)
			return
		} else if !isAllowedToFork {
			auditParams["error"] = "User cannot create repo in organization"
			audit.CreateAndSendEvent(audit.RepositoryForkEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			ctx.Error(http.StatusForbidden)
			return
		}
	}

	repo, err := repo_service.ForkRepository(ctx, ctx.Doer, ctxUser, repo_service.ForkRepoOptions{
		BaseRepo:    forkRepo,
		Name:        form.RepoName,
		Description: form.Description,
	})
	if err != nil {
		ctx.Data["Err_RepoName"] = true
		switch {
		case repo_model.IsErrReachLimitOfRepo(err):
			maxCreationLimit := ctxUser.MaxCreationLimit()
			msg := ctx.TrN(maxCreationLimit, "repo.form.reach_limit_of_creation_1", "repo.form.reach_limit_of_creation_n", maxCreationLimit)
			auditParams["error"] = "Reached limit of repositories"
			ctx.RenderWithErr(msg, tplFork, &form)
		case repo_model.IsErrCreateUserRepo(err):
			ctx.RenderWithErr(ctx.Tr("repo.create_user_repo_not_allowed"), tplFork, &form)
		case repo_model.IsErrRepoAlreadyExist(err):
			auditParams["error"] = "Repository name been taken"
			ctx.RenderWithErr(ctx.Tr("repo.settings.new_owner_has_same_repo"), tplFork, &form)
		case repo_model.IsErrRepoFilesAlreadyExist(err):
			auditParams["error"] = "Files already exist for this repository"
			switch {
			case ctx.IsUserSiteAdmin() || (setting.Repository.AllowAdoptionOfUnadoptedRepositories && setting.Repository.AllowDeleteOfUnadoptedRepositories):
				ctx.RenderWithErr(ctx.Tr("form.repository_files_already_exist.adopt_or_delete"), tplFork, form)
			case setting.Repository.AllowAdoptionOfUnadoptedRepositories:
				ctx.RenderWithErr(ctx.Tr("form.repository_files_already_exist.adopt"), tplFork, form)
			case setting.Repository.AllowDeleteOfUnadoptedRepositories:
				ctx.RenderWithErr(ctx.Tr("form.repository_files_already_exist.delete"), tplFork, form)
			default:
				ctx.RenderWithErr(ctx.Tr("form.repository_files_already_exist"), tplFork, form)
			}
		case db.IsErrNameReserved(err):
			auditParams["error"] = "Name reserved"
			ctx.RenderWithErr(ctx.Tr("repo.form.name_reserved", err.(db.ErrNameReserved).Name), tplFork, &form)
		case db.IsErrNamePatternNotAllowed(err):
			auditParams["error"] = "Name pattern not allowed"
			ctx.RenderWithErr(ctx.Tr("repo.form.name_pattern_not_allowed", err.(db.ErrNamePatternNotAllowed).Pattern), tplFork, &form)
		default:
			auditParams["error"] = "Error has occurred while forking repository"
			ctx.ServerError("ForkPost", err)
		}
		audit.CreateAndSendEvent(audit.RepositoryForkEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}

	audit.CreateAndSendEvent(audit.RepositoryForkEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)
	log.Trace("Repository forked[%d]: %s/%s", forkRepo.ID, ctxUser.Name, repo.Name)
	ctx.Redirect(ctxUser.HomeLink() + "/" + url.PathEscape(repo.Name))
}

func CheckPullInfo(ctx *context.Context) *issues_model.Issue {
	issue, err := issues_model.GetIssueByIndex(ctx.Repo.Repository.ID, ctx.ParamsInt64(":index"))
	if err != nil {
		if issues_model.IsErrIssueNotExist(err) {
			ctx.NotFound("GetIssueByIndex", err)
		} else {
			ctx.ServerError("GetIssueByIndex", err)
		}
		return nil
	}
	if err = issue.LoadPoster(ctx); err != nil {
		ctx.ServerError("LoadPoster", err)
		return nil
	}
	if err := issue.LoadRepo(ctx); err != nil {
		ctx.ServerError("LoadRepo", err)
		return nil
	}
	ctx.Data["Title"] = fmt.Sprintf("#%d - %s", issue.Index, issue.Title)
	ctx.Data["Issue"] = issue

	if !issue.IsPull {
		ctx.NotFound("ViewPullCommits", nil)
		return nil
	}

	if err = issue.LoadPullRequest(ctx); err != nil {
		ctx.ServerError("LoadPullRequest", err)
		return nil
	}

	if err = issue.PullRequest.LoadHeadRepo(ctx); err != nil {
		ctx.ServerError("LoadHeadRepo", err)
		return nil
	}

	if ctx.IsSigned {
		// Update issue-user.
		if err = activities_model.SetIssueReadBy(ctx, issue.ID, ctx.Doer.ID); err != nil {
			ctx.ServerError("ReadBy", err)
			return nil
		}
	}

	return issue
}

func setMergeTarget(ctx *context.Context, pull *issues_model.PullRequest) {
	if ctx.Repo.Owner.Name == pull.MustHeadUserName(ctx) {
		ctx.Data["HeadTarget"] = pull.HeadBranch
	} else if pull.HeadRepo == nil {
		ctx.Data["HeadTarget"] = pull.MustHeadUserName(ctx) + ":" + pull.HeadBranch
	} else {
		ctx.Data["HeadTarget"] = pull.MustHeadUserName(ctx) + "/" + pull.HeadRepo.Name + ":" + pull.HeadBranch
	}
	ctx.Data["BaseTarget"] = pull.BaseBranch
	ctx.Data["HeadBranchLink"] = pull.GetHeadBranchLink()
	ctx.Data["BaseBranchLink"] = pull.GetBaseBranchLink()
}

// PrepareMergedViewPullInfo show meta information for a merged pull request view page
func PrepareMergedViewPullInfo(ctx *context.Context, issue *issues_model.Issue) *git.CompareInfo {
	pull := issue.PullRequest

	setMergeTarget(ctx, pull)
	ctx.Data["HasMerged"] = true

	compareInfo := new(git.CompareInfo)
	compareInfo.MergeBase = pull.MergeBase
	compareInfo.BaseCommitID = pull.MergeBase

	var err error
	compareInfo.Commits, err = findMergedPullCommits(ctx, ctx.Repo.GitRepo, pull.MergedCommitID, pull.MergeBase)
	if err != nil {
		ctx.ServerError("GetCompareInfo", err)
		return nil
	}

	if len(compareInfo.Commits) == 0 {
		ctx.Data["IsPullRequestBroken"] = true
		ctx.Data["BaseTarget"] = pull.BaseBranch
		ctx.Data["NumCommits"] = 0
		ctx.Data["NumFiles"] = 0
		return nil
	}

	// need for clear merge commit from pr commit tree
	if len(compareInfo.Commits) > 1 {
		defaultMergeMessage, _, err := pull_service.GetDefaultMergeMessage(ctx, ctx.Repo.GitRepo, pull, "")
		if err != nil {
			ctx.ServerError("GetDefaultMergeMessage", err)
			return nil
		}
		if strings.Contains(compareInfo.Commits[0].CommitMessage, defaultMergeMessage) {
			compareInfo.Commits = compareInfo.Commits[1:]
		}
	}
	compareInfo.HeadCommitID = compareInfo.Commits[0].ID.String()
	diffStats, err := ctx.Repo.GitRepo.DiffClient.DiffStats(ctx.Repo.GitRepo.Ctx, &gitalypb.DiffStatsRequest{
		Repository:    ctx.Repo.GitRepo.GitalyRepo,
		LeftCommitId:  compareInfo.BaseCommitID,
		RightCommitId: compareInfo.HeadCommitID,
	})
	if err != nil {
		ctx.ServerError("GetCompareInfo", err)
		return nil
	}

	canRead := true
	for canRead {
		diffStatsResp, err := diffStats.Recv()
		if err != nil && err != io.EOF {
			ctx.ServerError("GetCompareInfo", err)
			return nil
		}
		if diffStatsResp == nil {
			canRead = false
		} else {
			compareInfo.NumFiles += len(diffStatsResp.GetStats())
		}
	}
	ctx.Data["NumCommits"] = len(compareInfo.Commits)
	ctx.Data["NumFiles"] = compareInfo.NumFiles

	if len(compareInfo.Commits) != 0 {
		sha := compareInfo.Commits[0].ID.String()
		commitStatuses, _, err := git_model.GetLatestCommitStatus(ctx, ctx.Repo.Repository.ID, sha, db.ListOptions{})
		if err != nil {
			ctx.ServerError("GetLatestCommitStatus", err)
			return nil
		}
		if len(commitStatuses) != 0 {
			ctx.Data["LatestCommitStatuses"] = commitStatuses
			ctx.Data["LatestCommitStatus"] = git_model.CalcCommitStatus(commitStatuses)
		}
	}

	return compareInfo
}

func findMergedPullCommits(ctx *context.Context, gitRepo *git.Repository, startOid string, baseCommit string) ([]*git.Commit, error) {
	ctxWithCancel, cancel := contextDefault.WithCancel(gitRepo.Ctx)
	defer cancel()

	commits := make([]*git.Commit, 0)
	listCommitsClient, err := gitRepo.CommitClient.ListCommitsByOid(ctxWithCancel, &gitalypb.ListCommitsByOidRequest{
		Repository: gitRepo.GitalyRepo,
		Oid:        []string{startOid},
	})
	if err != nil {
		return commits, err
	}

	listCommitsMerged, err := listCommitsClient.Recv()
	if err != nil {
		return commits, err
	}

	commits, err = gitRepo.ParseGitCommitsToCommit(listCommitsMerged.GetCommits())

	parentOids := make([]string, 0)
	for _, comm := range listCommitsMerged.GetCommits() {
		parentOids = append(parentOids, comm.ParentIds...)
	}

	for _, parentOid := range parentOids {
		if parentOid != baseCommit {
			parentCommits, err := findMergedPullCommits(ctx, gitRepo, parentOid, baseCommit)
			if err != nil {
				return commits, err
			}
			commits = append(commits, parentCommits...)
		}
	}
	return commits, nil
}

// PrepareViewPullInfo show meta information for a pull request preview page
func PrepareViewPullInfo(ctx *context.Context, issue *issues_model.Issue) *git.CompareInfo {
	ctx.Data["PullRequestWorkInProgressPrefixes"] = setting.Repository.PullRequest.WorkInProgressPrefixes

	repo := ctx.Repo.Repository
	pull := issue.PullRequest

	if err := pull.LoadHeadRepo(ctx); err != nil {
		ctx.ServerError("LoadHeadRepo", err)
		return nil
	}

	if err := pull.LoadBaseRepo(ctx); err != nil {
		ctx.ServerError("LoadBaseRepo", err)
		return nil
	}

	setMergeTarget(ctx, pull)

	pb, err := git_model.GetMergeMatchProtectedBranchRule(ctx, repo.ID, pull.BaseBranch)
	if err != nil {
		log.Error("Err: get merge match prottected branch with repo id - %d, base branch - %s: %v", repo.ID, pull.BaseBranch, err)
		ctx.ServerError("LoadProtectedBranch", err)
		return nil
	}
	ctx.Data["EnableStatusCheck"] = pb != nil && pb.EnableStatusCheck

	var baseGitRepo *git.Repository
	if pull.BaseRepoID == ctx.Repo.Repository.ID && ctx.Repo.GitRepo != nil {
		baseGitRepo = ctx.Repo.GitRepo
	} else {
		baseGitRepo, err := git.OpenRepository(ctx, pull.BaseRepo.OwnerName, pull.BaseRepo.Name, pull.BaseRepo.RepoPath())
		if err != nil {
			ctx.ServerError("OpenRepository", err)
			return nil
		}
		defer baseGitRepo.Close()
	}

	if !baseGitRepo.IsBranchExist(pull.BaseBranch) {
		ctx.Data["IsPullRequestBroken"] = true
		ctx.Data["BaseTarget"] = pull.BaseBranch
		ctx.Data["HeadTarget"] = pull.HeadBranch

		sha, err := baseGitRepo.GetRefCommitID(pull.GetGitRefName())
		if err != nil {
			ctx.ServerError(fmt.Sprintf("GetRefCommitID(%s)", pull.GetGitRefName()), err)
			return nil
		}
		commitStatuses, _, err := git_model.GetLatestCommitStatus(ctx, repo.ID, sha, db.ListOptions{})
		if err != nil {
			ctx.ServerError("GetLatestCommitStatus", err)
			return nil
		}
		if len(commitStatuses) > 0 {
			ctx.Data["LatestCommitStatuses"] = commitStatuses
			ctx.Data["LatestCommitStatus"] = git_model.CalcCommitStatus(commitStatuses)
		}

		compareInfo, err := baseGitRepo.GetCompareInfo(pull.BaseRepo.RepoPath(),
			pull.MergeBase, pull.GetGitRefName(), false, false)
		if err != nil {
			if strings.Contains(err.Error(), "fatal: Not a valid object name") {
				ctx.Data["IsPullRequestBroken"] = true
				ctx.Data["BaseTarget"] = pull.BaseBranch
				ctx.Data["NumCommits"] = 0
				ctx.Data["NumFiles"] = 0
				return nil
			}

			ctx.ServerError("GetCompareInfo", err)
			return nil
		}

		ctx.Data["NumCommits"] = len(compareInfo.Commits)
		ctx.Data["NumFiles"] = compareInfo.NumFiles
		return compareInfo
	}

	var headBranchExist bool
	var headBranchSha string
	// HeadRepo may be missing
	if pull.HeadRepo != nil {
		headGitRepo, err := git.OpenRepository(ctx, pull.HeadRepo.OwnerName, pull.HeadRepo.Name, pull.HeadRepo.RepoPath())
		if err != nil {
			ctx.ServerError("OpenRepository", err)
			return nil
		}
		defer headGitRepo.Close()

		if pull.Flow == issues_model.PullRequestFlowGithub {
			headBranchExist = headGitRepo.IsBranchExist(pull.HeadBranch)
		} else {
			headBranchExist = git.IsReferenceExist(ctx, baseGitRepo.Owner, baseGitRepo.Name, baseGitRepo.Path, pull.GetGitRefName())
		}

		if headBranchExist {
			if pull.Flow != issues_model.PullRequestFlowGithub {
				headBranchSha, err = baseGitRepo.GetRefCommitID(pull.GetGitRefName())
			} else {
				headBranchSha, err = headGitRepo.GetBranchCommitID(pull.HeadBranch)
			}
			if err != nil {
				ctx.ServerError("GetBranchCommitID", err)
				return nil
			}
		}
	}

	if headBranchExist {
		var err error
		ctx.Data["UpdateAllowed"], ctx.Data["UpdateByRebaseAllowed"], err = pull_service.IsUserAllowedToUpdate(ctx, pull, ctx.Doer)
		if err != nil {
			ctx.ServerError("IsUserAllowedToUpdate", err)
			return nil
		}
		ctx.Data["GetCommitMessages"] = pull_service.GetSquashMergeCommitMessages(ctx, pull)
	} else {
		ctx.Data["GetCommitMessages"] = ""
	}

	sha, err := baseGitRepo.GetRefCommitID(pull.GetGitHeadBranchRefName())
	if err != nil && err != io.EOF {
		if git.IsErrNotExist(err) {
			ctx.Data["IsPullRequestBroken"] = true
			if pull.IsSameRepo() {
				ctx.Data["HeadTarget"] = pull.HeadBranch
			} else if pull.HeadRepo == nil {
				ctx.Data["HeadTarget"] = "<deleted>:" + pull.HeadBranch
			} else {
				ctx.Data["HeadTarget"] = pull.HeadRepo.OwnerName + ":" + pull.HeadBranch
			}
			ctx.Data["BaseTarget"] = pull.BaseBranch
			ctx.Data["NumCommits"] = 0
			ctx.Data["NumFiles"] = 0
			return nil
		}
		ctx.ServerError(fmt.Sprintf("GetRefCommitID(%s)", pull.GetGitRefName()), err)
		return nil
	}

	commitStatuses, _, err := git_model.GetLatestCommitStatus(ctx, repo.ID, sha, db.ListOptions{})
	if err != nil {
		ctx.ServerError("GetLatestCommitStatus", err)
		return nil
	}
	if len(commitStatuses) > 0 {
		ctx.Data["LatestCommitStatuses"] = commitStatuses
		ctx.Data["LatestCommitStatus"] = git_model.CalcCommitStatus(commitStatuses)
	}

	if pb != nil && pb.EnableStatusCheck {
		ctx.Data["is_context_required"] = func(context string) bool {
			for _, c := range pb.StatusCheckContexts {
				if gp, err := glob.Compile(c); err == nil && gp.Match(context) {
					return true
				}
			}
			return false
		}
		ctx.Data["RequiredStatusCheckState"] = pull_service.MergeRequiredContextsCommitStatus(commitStatuses, pb.StatusCheckContexts)
	}

	ctx.Data["HeadBranchMovedOn"] = headBranchSha != sha
	ctx.Data["HeadBranchCommitID"] = headBranchSha
	ctx.Data["PullHeadCommitID"] = sha

	if pull.HeadRepo == nil || !headBranchExist || (!pull.Issue.IsClosed && (headBranchSha != sha)) {
		ctx.Data["IsPullRequestBroken"] = true
		if pull.IsSameRepo() {
			ctx.Data["HeadTarget"] = pull.HeadBranch
		} else if pull.HeadRepo == nil {
			ctx.Data["HeadTarget"] = "<deleted>:" + pull.HeadBranch
		} else {
			ctx.Data["HeadTarget"] = pull.HeadRepo.OwnerName + ":" + pull.HeadBranch
		}
	}

	compareInfo, err := baseGitRepo.GetCompareInfo(pull.BaseRepo.RepoPath(),
		git.BranchPrefix+pull.BaseBranch, pull.GetGitRefName(), false, false)
	if err != nil {
		if strings.Contains(err.Error(), "fatal: Not a valid object name") {
			ctx.Data["IsPullRequestBroken"] = true
			ctx.Data["BaseTarget"] = pull.BaseBranch
			ctx.Data["NumCommits"] = 0
			ctx.Data["NumFiles"] = 0
			return nil
		}

		ctx.ServerError("GetCompareInfo", err)
		return nil
	}

	if compareInfo.HeadCommitID == compareInfo.MergeBase {
		ctx.Data["IsNothingToCompare"] = true
		compareInfo.NumFiles = 0
	}

	if pull.IsWorkInProgress() {
		ctx.Data["IsPullWorkInProgress"] = true
		ctx.Data["WorkInProgressPrefix"] = pull.GetWorkInProgressPrefix(ctx)
	}

	if pull.IsFilesConflicted() {
		ctx.Data["IsPullFilesConflicted"] = true
		ctx.Data["ConflictedFiles"] = pull.ConflictedFiles
	}

	ctx.Data["NumCommits"] = len(compareInfo.Commits)
	ctx.Data["NumFiles"] = compareInfo.NumFiles
	return compareInfo
}

// UpdatePullRequest merge PR's baseBranch into headBranch
func UpdatePullRequest(ctx *context.Context) {
	issue := CheckPullInfo(ctx)
	if ctx.Written() {
		return
	}
	if issue.IsClosed {
		ctx.NotFound("MergePullRequest", nil)
		return
	}
	if issue.PullRequest.HasMerged {
		ctx.NotFound("MergePullRequest", nil)
		return
	}

	rebase := ctx.FormString("style") == "rebase"

	if err := issue.PullRequest.LoadBaseRepo(ctx); err != nil {
		ctx.ServerError("LoadBaseRepo", err)
		return
	}
	if err := issue.PullRequest.LoadHeadRepo(ctx); err != nil {
		ctx.ServerError("LoadHeadRepo", err)
		return
	}

	allowedUpdateByMerge, allowedUpdateByRebase, err := pull_service.IsUserAllowedToUpdate(ctx, issue.PullRequest, ctx.Doer)
	if err != nil {
		ctx.ServerError("IsUserAllowedToMerge", err)
		return
	}

	// ToDo: add check if maintainers are allowed to change branch ... (need migration & co)
	if (!allowedUpdateByMerge && !rebase) || (rebase && !allowedUpdateByRebase) {
		ctx.Flash.Error(ctx.Tr("repo.pulls.update_not_allowed"))
		ctx.Redirect(issue.Link())
		return
	}

	// default merge commit message
	message := fmt.Sprintf("Merge branch '%s' into %s", issue.PullRequest.BaseBranch, issue.PullRequest.HeadBranch)

	if err = pull_service.Update(ctx, issue.PullRequest, ctx.Doer, message, rebase); err != nil {
		if models.IsErrMergeConflicts(err) {
			conflictError := err.(models.ErrMergeConflicts)
			flashError, err := ctx.RenderToString(TplAlertDetails, map[string]interface{}{
				"Message": ctx.Tr("repo.pulls.merge_conflict"),
				"Summary": ctx.Tr("repo.pulls.merge_conflict_summary"),
				"Details": utils.SanitizeFlashErrorString(conflictError.StdErr) + "<br>" + utils.SanitizeFlashErrorString(conflictError.StdOut),
			})
			if err != nil {
				ctx.ServerError("UpdatePullRequest.HTMLString", err)
				return
			}
			ctx.Flash.Error(flashError)
			ctx.Redirect(issue.Link())
			return
		} else if models.IsErrRebaseConflicts(err) {
			conflictError := err.(models.ErrRebaseConflicts)
			flashError, err := ctx.RenderToString(TplAlertDetails, map[string]interface{}{
				"Message": ctx.Tr("repo.pulls.rebase_conflict", utils.SanitizeFlashErrorString(conflictError.CommitSHA)),
				"Summary": ctx.Tr("repo.pulls.rebase_conflict_summary"),
				"Details": utils.SanitizeFlashErrorString(conflictError.StdErr) + "<br>" + utils.SanitizeFlashErrorString(conflictError.StdOut),
			})
			if err != nil {
				ctx.ServerError("UpdatePullRequest.HTMLString", err)
				return
			}
			ctx.Flash.Error(flashError)
			ctx.Redirect(issue.Link())
			return

		}
		ctx.Flash.Error(err.Error())
		ctx.Redirect(issue.Link())
		return
	}

	ctx.Flash.Success(ctx.Tr("repo.pulls.update_branch_success"))
	ctx.Redirect(issue.Link())
}

// CancelAutoMergePullRequest cancels a scheduled pr
func CancelAutoMergePullRequest(ctx *context.Context) {
	issue := CheckPullInfo(ctx)
	if ctx.Written() {
		return
	}

	if err := automerge.RemoveScheduledAutoMerge(ctx, ctx.Doer, issue.PullRequest); err != nil {
		if db.IsErrNotExist(err) {
			ctx.Flash.Error(ctx.Tr("repo.pulls.auto_merge_not_scheduled"))
			ctx.Redirect(fmt.Sprintf("%s/pulls/%d", ctx.Repo.RepoLink, issue.Index))
			return
		}
		ctx.ServerError("RemoveScheduledAutoMerge", err)
		return
	}
	ctx.Flash.Success(ctx.Tr("repo.pulls.auto_merge_canceled_schedule"))
	ctx.Redirect(fmt.Sprintf("%s/pulls/%d", ctx.Repo.RepoLink, issue.Index))
}

func StopTimerIfAvailable(user *user_model.User, issue *issues_model.Issue) error {
	if issues_model.StopwatchExists(user.ID, issue.ID) {
		if err := issues_model.CreateOrStopIssueStopwatch(user, issue); err != nil {
			return err
		}
	}

	return nil
}

// CleanUpPullRequest responses for delete merged branch when PR has been merged
func CleanUpPullRequest(ctx *context.Context) {
	issue := CheckPullInfo(ctx)
	if ctx.Written() {
		return
	}

	pr := issue.PullRequest

	// Don't cleanup unmerged and unclosed PRs
	if !pr.HasMerged && !issue.IsClosed {
		ctx.NotFound("CleanUpPullRequest", nil)
		return
	}

	// Don't cleanup when there are other PR's that use this branch as head branch.
	exist, err := issues_model.HasUnmergedPullRequestsByHeadInfo(ctx, pr.HeadRepoID, pr.HeadBranch)
	if err != nil {
		ctx.ServerError("HasUnmergedPullRequestsByHeadInfo", err)
		return
	}
	if exist {
		ctx.NotFound("CleanUpPullRequest", nil)
		return
	}

	if err := pr.LoadHeadRepo(ctx); err != nil {
		ctx.ServerError("LoadHeadRepo", err)
		return
	} else if pr.HeadRepo == nil {
		// Forked repository has already been deleted
		ctx.NotFound("CleanUpPullRequest", nil)
		return
	} else if err = pr.LoadBaseRepo(ctx); err != nil {
		ctx.ServerError("LoadBaseRepo", err)
		return
	} else if err = pr.HeadRepo.LoadOwner(ctx); err != nil {
		ctx.ServerError("HeadRepo.LoadOwner", err)
		return
	}

	perm, err := access_model.GetUserRepoPermission(ctx, pr.HeadRepo, ctx.Doer)
	if err != nil {
		ctx.ServerError("GetUserRepoPermission", err)
		return
	}
	if !perm.CanWrite(unit.TypeCode) {
		ctx.NotFound("CleanUpPullRequest", nil)
		return
	}

	fullBranchName := pr.HeadRepo.Owner.Name + "/" + pr.HeadBranch

	var gitBaseRepo *git.Repository

	// Assume that the base repo is the current context (almost certainly)
	if ctx.Repo != nil && ctx.Repo.Repository != nil && ctx.Repo.Repository.ID == pr.BaseRepoID && ctx.Repo.GitRepo != nil {
		gitBaseRepo = ctx.Repo.GitRepo
	} else {
		// If not just open it
		gitBaseRepo, err = git.OpenRepository(ctx, pr.BaseRepo.OwnerName, pr.BaseRepo.Name, pr.BaseRepo.RepoPath())
		if err != nil {
			ctx.ServerError(fmt.Sprintf("OpenRepository[%s]", pr.BaseRepo.RepoPath()), err)
			return
		}
		defer gitBaseRepo.Close()
	}

	// Now assume that the head repo is the same as the base repo (reasonable chance)
	gitRepo := gitBaseRepo
	// But if not: is it the same as the context?
	if pr.BaseRepoID != pr.HeadRepoID && ctx.Repo != nil && ctx.Repo.Repository != nil && ctx.Repo.Repository.ID == pr.HeadRepoID && ctx.Repo.GitRepo != nil {
		gitRepo = ctx.Repo.GitRepo
	} else if pr.BaseRepoID != pr.HeadRepoID {
		// Otherwise just load it up
		gitRepo, err = git.OpenRepository(ctx, pr.HeadRepo.OwnerName, pr.HeadRepo.Name, pr.HeadRepo.RepoPath())
		if err != nil {
			ctx.ServerError(fmt.Sprintf("OpenRepository[%s]", pr.HeadRepo.RepoPath()), err)
			return
		}
		defer gitRepo.Close()
	}

	defer func() {
		ctx.JSON(http.StatusOK, map[string]interface{}{
			"redirect": issue.Link(),
		})
	}()

	// Check if branch has no new commits
	headCommitID, err := gitBaseRepo.GetRefCommitID(pr.GetGitRefName())
	if err != nil {
		log.Error("GetRefCommitID: %v", err)
		ctx.Flash.Error(ctx.Tr("repo.branch.deletion_failed", fullBranchName))
		return
	}
	branchCommitID, err := gitRepo.GetBranchCommitID(pr.HeadBranch)
	if err != nil {
		log.Error("GetBranchCommitID: %v", err)
		ctx.Flash.Error(ctx.Tr("repo.branch.deletion_failed", fullBranchName))
		return
	}
	if headCommitID != branchCommitID {
		ctx.Flash.Error(ctx.Tr("repo.branch.delete_branch_has_new_commits", fullBranchName))
		return
	}

	DeleteBranch(ctx, pr, gitRepo)
}

func DeleteBranch(ctx *context.Context, pr *issues_model.PullRequest, gitRepo *git.Repository) {
	fullBranchName := pr.HeadRepo.FullName() + ":" + pr.HeadBranch
	auditBranchParams := map[string]string{
		"repository":  ctx.Repo.Repository.Name,
		"branch_name": pr.HeadBranch,
	}

	if err := repo_service.DeleteBranch(ctx, ctx.Doer, pr.HeadRepo, gitRepo, pr.HeadBranch); err != nil {
		switch {
		case git.IsErrBranchNotExist(err):
			ctx.Flash.Error(ctx.Tr("repo.branch.deletion_failed", fullBranchName))
			auditBranchParams["error"] = "Branch not exist"
		case errors.Is(err, repo_service.ErrBranchIsDefault):
			ctx.Flash.Error(ctx.Tr("repo.branch.deletion_failed", fullBranchName))
			auditBranchParams["error"] = "Branch is default"
		case protected_branch.IsBranchIsProtectedError(err):
			ctx.Flash.Error(ctx.Tr("repo.branch.deletion_failed", fullBranchName))
			auditBranchParams["error"] = "Branch is protected"
		default:
			log.Error("DeleteBranch: %v", err)
			ctx.Flash.Error(ctx.Tr("repo.branch.deletion_failed", fullBranchName))
			auditBranchParams["error"] = "Error has occurred while deleting branch"
		}
		audit.CreateAndSendEvent(audit.BranchDeleteEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditBranchParams)
		return
	}

	if err := issues_model.AddDeletePRBranchComment(ctx, ctx.Doer, pr.BaseRepo, pr.IssueID, pr.HeadBranch); err != nil {
		// Do not fail here as branch has already been deleted
		log.Error("DeleteBranch: %v", err)
	}

	ctx.Flash.Success(ctx.Tr("repo.branch.deletion_success", fullBranchName))
	audit.CreateAndSendEvent(audit.BranchDeleteEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditBranchParams)
}

// DownloadPullDiff render a pull's raw diff
func DownloadPullDiff(ctx *context.Context) {
	DownloadPullDiffOrPatch(ctx, false)
}

// DownloadPullPatch render a pull's raw patch
func DownloadPullPatch(ctx *context.Context) {
	DownloadPullDiffOrPatch(ctx, true)
}

// DownloadPullDiffOrPatch render a pull's raw diff or patch
func DownloadPullDiffOrPatch(ctx *context.Context, patch bool) {
	pr, err := issues_model.GetPullRequestByIndex(ctx, ctx.Repo.Repository.ID, ctx.ParamsInt64(":index"))
	if err != nil {
		if issues_model.IsErrPullRequestNotExist(err) {
			ctx.NotFound("GetPullRequestByIndex", err)
		} else {
			ctx.ServerError("GetPullRequestByIndex", err)
		}
		return
	}

	binary := ctx.FormBool("binary")

	if err := pull_service.DownloadDiffOrPatch(ctx, pr, ctx, patch, binary); err != nil {
		ctx.ServerError("DownloadDiffOrPatch", err)
		return
	}
}

// UpdatePullRequestTarget change pull request's target branch
func UpdatePullRequestTarget(ctx *context.Context) {
	issue := GetActionIssue(ctx)
	pr := issue.PullRequest
	if ctx.Written() {
		return
	}
	if !issue.IsPull {
		ctx.Error(http.StatusNotFound)
		return
	}

	if !ctx.IsSigned || (!issue.IsPoster(ctx.Doer.ID) && !ctx.Repo.CanWriteIssuesOrPulls(issue.IsPull)) {
		ctx.Error(http.StatusForbidden)
		return
	}

	targetBranch := ctx.FormTrim("target_branch")
	if len(targetBranch) == 0 {
		ctx.Error(http.StatusNoContent)
		return
	}

	if err := pull_service.ChangeTargetBranch(ctx, pr, ctx.Doer, targetBranch); err != nil {
		if issues_model.IsErrPullRequestAlreadyExists(err) {
			err := err.(issues_model.ErrPullRequestAlreadyExists)

			RepoRelPath := ctx.Repo.Owner.Name + "/" + ctx.Repo.Repository.Name
			errorMessage := ctx.Tr("repo.pulls.has_pull_request", html.EscapeString(ctx.Repo.RepoLink+"/pulls/"+strconv.FormatInt(err.IssueID, 10)), html.EscapeString(RepoRelPath), err.IssueID) // FIXME: Creates url inside locale string

			ctx.Flash.Error(errorMessage)
			ctx.JSON(http.StatusConflict, map[string]interface{}{
				"error":      err.Error(),
				"user_error": errorMessage,
			})
		} else if issues_model.IsErrIssueIsClosed(err) {
			errorMessage := ctx.Tr("repo.pulls.is_closed")

			ctx.Flash.Error(errorMessage)
			ctx.JSON(http.StatusConflict, map[string]interface{}{
				"error":      err.Error(),
				"user_error": errorMessage,
			})
		} else if models.IsErrPullRequestHasMerged(err) {
			errorMessage := ctx.Tr("repo.pulls.has_merged")

			ctx.Flash.Error(errorMessage)
			ctx.JSON(http.StatusConflict, map[string]interface{}{
				"error":      err.Error(),
				"user_error": errorMessage,
			})
		} else if models.IsErrBranchesEqual(err) {
			errorMessage := ctx.Tr("repo.pulls.nothing_to_compare")

			ctx.Flash.Error(errorMessage)
			ctx.JSON(http.StatusBadRequest, map[string]interface{}{
				"error":      err.Error(),
				"user_error": errorMessage,
			})
		} else {
			ctx.ServerError("UpdatePullRequestTarget", err)
		}
		return
	}
	notification.NotifyPullRequestChangeTargetBranch(ctx, ctx.Doer, pr, targetBranch)

	ctx.JSON(http.StatusOK, map[string]interface{}{
		"base_branch": pr.BaseBranch,
	})
}

// SetAllowEdits allow edits from maintainers to PRs
func SetAllowEdits(ctx *context.Context) {
	form := web.GetForm(ctx).(*forms.UpdateAllowEditsForm)

	pr, err := issues_model.GetPullRequestByIndex(ctx, ctx.Repo.Repository.ID, ctx.ParamsInt64(":index"))
	if err != nil {
		if issues_model.IsErrPullRequestNotExist(err) {
			ctx.NotFound("GetPullRequestByIndex", err)
		} else {
			ctx.ServerError("GetPullRequestByIndex", err)
		}
		return
	}

	if err := pull_service.SetAllowEdits(ctx, ctx.Doer, pr, form.AllowMaintainerEdit); err != nil {
		if errors.Is(pull_service.ErrUserHasNoPermissionForAction, err) {
			ctx.Error(http.StatusForbidden)
			return
		}
		ctx.ServerError("SetAllowEdits", err)
		return
	}

	ctx.JSON(http.StatusOK, map[string]interface{}{
		"allow_maintainer_edit": pr.AllowMaintainerEdit,
	})
}

// GetPullRequestStatusState Получение статуса сборки последнего коммита пулл-реквеста с датой
func GetPullRequestStatusState(ctx *context.Context) {
	pr, err := issues_model.GetPullRequestByIndex(ctx, ctx.Repo.Repository.ID, ctx.ParamsInt64(":index"))
	if err != nil {
		if issues_model.IsErrIssueNotExist(err) {
			ctx.NotFound("GetIssueByIndex", err)
		} else {
			ctx.ServerError("GetIssueByIndex", err)
		}
		ctx.JSON(http.StatusInternalServerError, "")
	}

	lastCommitStatus, lastCommitStatusDate, err := pull_service.GetPullRequestCommitStatusStateWithDate(ctx, pr)
	if len(lastCommitStatus) == 0 {
		lastCommitStatus = "unknown"
	}

	ctx.JSON(http.StatusOK, map[string]string{
		"state":   string(lastCommitStatus),
		"updated": lastCommitStatusDate.UTC().String()},
	)
}
