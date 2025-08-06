package issues

import (
	"net/http"
	"strconv"

	issues_model "code.gitea.io/gitea/models/issues"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/sbt/audit"
	"code.gitea.io/gitea/routers/web/repo"
	issue_service "code.gitea.io/gitea/services/issue"
)

// UpdateIssueStatus change issue's status
func (s Server) UpdateIssueStatus(ctx *context.Context) {
	issues := repo.GetActionIssues(ctx)
	if ctx.Written() {
		return
	}

	var isClosed bool
	switch action := ctx.FormString("action"); action {
	case "open":
		isClosed = false
	case "close":
		isClosed = true
	default:
		log.Warn("Unrecognized action: %s", action)
	}

	auditParams := map[string]string{
		"repository_id": strconv.FormatInt(ctx.Repo.Repository.ID, 10),
	}

	if _, err := issues_model.IssueList(issues).LoadRepositories(ctx); err != nil {
		ctx.ServerError("Error has occurred while loading repositories", err)
		return
	}
	for _, issue := range issues {
		if issue.IsClosed == isClosed {
			continue
		}
		if err := issue_service.ChangeStatus(issue, ctx.Doer, "", isClosed); err != nil {
			if issues_model.IsErrDependenciesLeft(err) {
				ctx.JSON(http.StatusPreconditionFailed, map[string]interface{}{
					"error": ctx.Tr("repo.issues.dependency.issue_batch_close_blocked", issue.Index),
				})
				return
			}
			ctx.ServerError("Error has occurred while changing status of pull request", err)
			return
		}
		auditParams["issue_id"] = strconv.FormatInt(issue.ID, 10)
		auditParams["status"] = strconv.FormatBool(isClosed)
		auditParams["pull_request_id"] = strconv.FormatInt(issue.PullRequest.ID, 10)
		if s.taskTrackerEnabled {
			userName := ctx.Doer.LowerName

			if err := s.updateIssueStatus(ctx, issue, userName); err != nil {
				log.Error("updating status pr: run: %v", err)
				auditParams["error"] = "Error has occurred trying to change issue status from unit"
				audit.CreateAndSendEvent(audit.PullRequestsUpdateEvent, userName, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			}
			audit.CreateAndSendEvent(audit.PullRequestsUpdateEvent, userName, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)
		}
	}
	ctx.JSON(http.StatusOK, map[string]interface{}{
		"ok": true,
	})
}
