package repo_server

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"code.gitea.io/gitea/models/git/protected_branch"
	"code.gitea.io/gitea/models/role_model"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/sbt/audit"
	"code.gitea.io/gitea/modules/trace"
	"code.gitea.io/gitea/routers/web/user/accesser"
	repo_service "code.gitea.io/gitea/services/repository"
)

// DeleteBranchPost responses for delete merged branch
func (s *Server) DeleteBranchPost(ctx *context.Context) {
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

	defer s.redirect(ctx)
	branchName := ctx.FormString("name")
	auditParams := map[string]string{
		"branch_name": branchName,
	}

	allowed, err := s.orgRequestAccessor.IsAccessGranted(*ctx, accesser.OrgAccessRequest{
		DoerID:         ctx.Doer.ID,
		TargetOrgID:    ctx.Repo.Repository.OwnerID,
		TargetTenantID: ctx.Data["TenantID"].(string),
		Action:         role_model.WRITE,
	})
	if err != nil {
		log.Error("Error has occurred while checking user's permissions: %v", err)
		ctx.Error(http.StatusForbidden, fmt.Sprintf("Error has occurred while checking user's permissions: %v", err))
		return
	}
	if !allowed {
		allow, err := s.repoRequestAccessor.AccessesByCustomPrivileges(*ctx, accesser.RepoAccessRequest{
			DoerID:          ctx.Doer.ID,
			OrgID:           ctx.Repo.Repository.OwnerID,
			TargetTenantID:  ctx.Data["TenantID"].(string),
			RepoID:          ctx.Repo.Repository.ID,
			CustomPrivilege: role_model.ChangeBranch.String(),
		})
		if err != nil {
			log.Error("Error has occurred while checking user's permissions: %v", err)
			ctx.Error(http.StatusForbidden, fmt.Sprintf("Error has occurred while checking user's permissions: %v", err))
			return
		}
		if !allow {
			log.Debug("User is not allowed to delete branch")
			ctx.Error(http.StatusForbidden, fmt.Sprintf("User is not allowed to delete branch: %s", branchName))
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

func (s *Server) redirect(ctx *context.Context) {
	ctx.JSON(http.StatusOK, map[string]interface{}{
		"redirect": ctx.Repo.RepoLink + "/branches?page=" + url.QueryEscape(ctx.FormString("page")),
	})
}
