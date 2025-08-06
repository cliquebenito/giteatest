package issues

import (
	"fmt"
	"net/http"
	"strconv"

	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/json"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/sbt/audit"
	"code.gitea.io/gitea/routers/web/repo"
	issue_service "code.gitea.io/gitea/services/issue"
)

// DeleteIssue deletes an issue
func (s Server) DeleteIssue(ctx *context.Context) {
	issue := repo.GetActionIssue(ctx)
	auditParams := map[string]string{}
	var doerName, doerID, remoteAddress string

	if ctx.Doer != nil {
		doerName = ctx.Doer.Name
		doerID = strconv.FormatInt(ctx.Doer.ID, 10)
	} else {
		doerName = audit.EmptyRequiredField
		doerID = audit.EmptyRequiredField
	}

	if ctx.Req != nil {
		remoteAddress = ctx.Req.RemoteAddr
	} else {
		remoteAddress = audit.EmptyRequiredField
	}

	if ctx.Written() {
		if issue == nil || issue.IsPull {
			auditParams["error"] = "Failed to get action issue"
			audit.CreateAndSendEvent(audit.PRDeleteEvent, doerName, doerID, audit.StatusFailure, remoteAddress, auditParams)
		}
		return
	}

	auditParams["repository"] = issue.Repo.Name

	type auditValue struct {
		ID     int64
		RepoID int64
		Title  string
	}

	oldValue := auditValue{
		ID:     issue.ID,
		RepoID: issue.RepoID,
		Title:  issue.Title,
	}

	oldValueBytes, _ := json.Marshal(oldValue)
	auditParams["old_value"] = string(oldValueBytes)

	if s.taskTrackerEnabled {
		userName := ctx.Doer.LowerName

		if err := s.unlinkUnitsFromIssue(ctx, issue, userName); err != nil {
			log.Debug("unit_linker: run: %v", err)
			auditParams["error"] = "Error has occurred trying to untie MR from unit"
			audit.CreateAndSendEvent(audit.PullRequestLinksDeleteEvent, doerName, doerID, audit.StatusFailure, remoteAddress, auditParams)
		}
		audit.CreateAndSendEvent(audit.PullRequestLinksDeleteEvent, doerName, doerID, audit.StatusSuccess, remoteAddress, auditParams)
	}

	auditParams["pr_number"] = strconv.FormatInt(issue.Index, 10)
	if err := issue_service.DeleteIssue(ctx, ctx.Doer, ctx.Repo.GitRepo, issue); err != nil {
		ctx.ServerError("DeleteIssueByID", err)
		if issue.IsPull {
			auditParams["error"] = "Cannot delete issue"
			audit.CreateAndSendEvent(audit.PRDeleteEvent, doerName, doerID, audit.StatusFailure, remoteAddress, auditParams)
		}

		return
	}

	if issue.IsPull {
		ctx.Redirect(fmt.Sprintf("%s/pulls", ctx.Repo.Repository.Link()), http.StatusSeeOther)

		audit.CreateAndSendEvent(audit.PRDeleteEvent, doerName, doerID, audit.StatusSuccess, remoteAddress, auditParams)

		return
	}

	ctx.Redirect(fmt.Sprintf("%s/issues", ctx.Repo.Repository.Link()), http.StatusSeeOther)
}
