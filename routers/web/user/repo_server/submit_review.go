package repo_server

import (
	"fmt"
	"net/http"

	issues_model "code.gitea.io/gitea/models/issues"
	"code.gitea.io/gitea/models/role_model"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/trace"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/routers/web/repo"
	"code.gitea.io/gitea/routers/web/user/accesser"
	"code.gitea.io/gitea/services/forms"
	pull_service "code.gitea.io/gitea/services/pull"
)

// SubmitReview creates a review out of the existing pending review or creates a new one if no pending review exist
func (s *Server) SubmitReview(ctx *context.Context) {
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

	allowed, err := s.orgRequestAccessor.IsAccessGranted(ctx, accesser.OrgAccessRequest{
		DoerID:         ctx.Doer.ID,
		TargetOrgID:    ctx.Repo.Repository.OwnerID,
		TargetTenantID: ctx.Data["TenantID"].(string),
		Action:         role_model.WRITE,
	})
	if err != nil {
		log.Error("Error has occurred while checking user's permissions: %v", err)
		ctx.Error(http.StatusForbidden, fmt.Sprintf("User does not have enough custom privileges to merge pr."))
		return
	}
	if !allowed {
		allow, err := s.repoRequestAccessor.AccessesByCustomPrivileges(ctx, accesser.RepoAccessRequest{
			DoerID:          ctx.Doer.ID,
			OrgID:           ctx.Repo.Repository.OwnerID,
			TargetTenantID:  ctx.Data["TenantID"].(string),
			RepoID:          ctx.Repo.Repository.ID,
			CustomPrivilege: role_model.ApprovePR.String(),
		})
		if err != nil || !allow {
			log.Error("Error has occurred while checking user's permissions: %v", err)
			ctx.Error(http.StatusForbidden, fmt.Sprintf("User does not have enough custom privileges to merge pr."))
			return
		}
	}

	form := web.GetForm(ctx).(*forms.SubmitReviewForm)
	issue := repo.GetActionIssue(ctx)
	if !issue.IsPull {
		return
	}
	if ctx.Written() {
		return
	}
	if ctx.HasError() {
		ctx.Flash.Error(ctx.Data["ErrorMsg"].(string))
		ctx.Redirect(fmt.Sprintf("%s/pulls/%d/files", ctx.Repo.RepoLink, issue.Index))
		return
	}

	reviewType := form.ReviewType()
	switch reviewType {
	case issues_model.ReviewTypeUnknown:
		ctx.ServerError("ReviewType", fmt.Errorf("unknown ReviewType: %s", form.Type))
		return

	// can not approve/reject your own PR
	case issues_model.ReviewTypeApprove, issues_model.ReviewTypeReject:
		if issue.IsPoster(ctx.Doer.ID) {
			var translated string
			if reviewType == issues_model.ReviewTypeApprove {
				translated = ctx.Tr("repo.Issues.review.self.approval")
			} else {
				translated = ctx.Tr("repo.Issues.review.self.rejection")
			}

			ctx.Flash.Error(translated)
			ctx.Redirect(fmt.Sprintf("%s/pulls/%d/files", ctx.Repo.RepoLink, issue.Index))
			return
		}
	}

	var attachments []string
	if setting.Attachment.Enabled {
		attachments = form.Files
	}

	_, comm, err := pull_service.SubmitReview(ctx, ctx.Doer, ctx.Repo.GitRepo, issue, reviewType, form.Content, form.CommitID, attachments)
	if err != nil {
		if issues_model.IsContentEmptyErr(err) {
			ctx.Flash.Error(ctx.Tr("repo.issues.review.content.empty"))
			ctx.Redirect(fmt.Sprintf("%s/pulls/%d/files", ctx.Repo.RepoLink, issue.Index))
		} else {
			ctx.ServerError("SubmitReview", err)
		}
		return
	}

	ctx.Redirect(fmt.Sprintf("%s/pulls/%d#%s", ctx.Repo.RepoLink, issue.Index, comm.HashTag()))
}
