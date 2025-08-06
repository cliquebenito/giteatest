package repo_server

import (
	"net/http"

	git_model "code.gitea.io/gitea/models/git"
	issues_model "code.gitea.io/gitea/models/issues"
	"code.gitea.io/gitea/models/role_model"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/trace"
	"code.gitea.io/gitea/routers/web/repo"
	"code.gitea.io/gitea/routers/web/user/accesser"
)

// ViewPullCommits show commits for a pull request
func (s *Server) ViewPullCommits(ctx *context.Context) {
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

	ctx.Data["PageIsPullList"] = true
	ctx.Data["PageIsPullCommits"] = true

	issue := repo.CheckPullInfo(ctx)
	if ctx.Written() {
		return
	}
	pull := issue.PullRequest

	var prInfo *git.CompareInfo
	if pull.HasMerged {
		prInfo = repo.PrepareMergedViewPullInfo(ctx, issue)
	} else {
		prInfo = repo.PrepareViewPullInfo(ctx, issue)
	}

	if ctx.Written() {
		return
	} else if prInfo == nil {
		ctx.NotFound("ViewPullCommits", nil)
		return
	}

	ctx.Data["Username"] = ctx.Repo.Owner.Name
	ctx.Data["Reponame"] = ctx.Repo.Repository.Name

	commits := git_model.ConvertFromGitCommit(ctx, prInfo.Commits, ctx.Repo.Repository)
	ctx.Data["Commits"] = commits
	ctx.Data["CommitCount"] = len(commits)

	s.getBranchData(ctx, issue)
	ctx.HTML(http.StatusOK, tplPullCommits)
}

func (s *Server) getBranchData(ctx *context.Context, issue *issues_model.Issue) {
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

	ctx.Data["BaseBranch"] = nil
	ctx.Data["HeadBranch"] = nil
	ctx.Data["HeadUserName"] = nil
	ctx.Data["BaseName"] = ctx.Repo.Repository.OwnerName
	if issue.IsPull {
		pull := issue.PullRequest
		ctx.Data["BaseBranch"] = pull.BaseBranch
		ctx.Data["HeadBranch"] = pull.HeadBranch
		ctx.Data["HeadUserName"] = pull.MustHeadUserName(ctx)
	}
	var tenantID string
	if ctx.Data != nil && ctx.Data["TenantID"] != "" {
		tenantID = ctx.Data["TenantID"].(string)
	} else {
		tenantID, err = role_model.GetUserTenantId(ctx, ctx.Doer.ID)
		if err != nil {
			log.Error("Error has occurred while getting tenant id by user: %v", err)
			ctx.ServerError("Error has occurred while getting tenant id by user: %v", err)
			return
		}
	}

	action := role_model.READ
	if ctx.Repo.Repository.IsPrivate {
		action = role_model.READ_PRIVATE
	}

	allowed, err := s.orgRequestAccessor.IsAccessGranted(*ctx, accesser.OrgAccessRequest{
		DoerID:         ctx.Doer.ID,
		TargetOrgID:    ctx.Repo.Repository.OwnerID,
		TargetTenantID: tenantID,
		Action:         action,
	})
	if err != nil {
		log.Error("Error has occurred while checking user's permissions: %v", err)
		ctx.ServerError("Error has occurred while checking user's permissions: %v", err)
		return
	}
	if !allowed {
		allow, err := s.repoRequestAccessor.AccessesByCustomPrivileges(*ctx, accesser.RepoAccessRequest{
			DoerID:          ctx.Doer.ID,
			OrgID:           ctx.Repo.Repository.OwnerID,
			TargetTenantID:  tenantID,
			RepoID:          ctx.Repo.Repository.ID,
			CustomPrivilege: role_model.ViewBranch.String(),
		})
		if err != nil {
			log.Error("Error has occurred while checking user's permissions: %v", err)
			ctx.ServerError("Error has occurred while checking user's permissions: %v", err)
			return
		}
		if !allow {
			ctx.Error(http.StatusForbidden, "You don't have permission to view this issue")
			return
		}
	}
}
