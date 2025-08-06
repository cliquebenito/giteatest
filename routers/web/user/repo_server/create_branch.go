package repo_server

import (
	"fmt"
	"net/http"
	"strconv"

	"code.gitea.io/gitea/models"
	"code.gitea.io/gitea/models/role_model"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/sbt/audit"
	"code.gitea.io/gitea/modules/trace"
	"code.gitea.io/gitea/modules/util"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/routers/utils"
	"code.gitea.io/gitea/routers/web/repo"
	"code.gitea.io/gitea/routers/web/user/accesser"
	"code.gitea.io/gitea/services/forms"
	release_service "code.gitea.io/gitea/services/release"
	repo_service "code.gitea.io/gitea/services/repository"
)

// CreateBranch creates new branch in repository
func (s *Server) CreateBranch(ctx *context.Context) {
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

	form := web.GetForm(ctx).(*forms.NewBranchForm)
	auditParams := map[string]string{
		"branch_name": form.NewBranchName,
	}

	allowed, err := s.orgRequestAccessor.IsAccessGranted(*ctx, accesser.OrgAccessRequest{
		DoerID:         ctx.Doer.ID,
		TargetOrgID:    ctx.Org.Organization.ID,
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
			OrgID:           ctx.Org.Organization.ID,
			TargetTenantID:  ctx.Data["TenantID"].(string),
			RepoID:          ctx.Repo.Repository.ID,
			CustomPrivilege: role_model.ViewBranch.String(),
		})
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
				flashError, err := ctx.RenderToString(repo.TplAlertDetails, map[string]interface{}{
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
