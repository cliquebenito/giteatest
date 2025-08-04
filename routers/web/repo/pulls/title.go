package pulls

import (
	goctx "context"
	"net/http"

	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/routers/web/repo"
	issue_service "code.gitea.io/gitea/services/issue"
)

// UpdateIssueTitle change issue's title
func (s Server) UpdateIssueTitle(ctx *context.Context) {
	issue := repo.GetActionIssue(ctx)
	if ctx.Written() {
		return
	}

	if !ctx.IsSigned || (!issue.IsPoster(ctx.Doer.ID) && !ctx.Repo.CanWriteIssuesOrPulls(issue.IsPull)) {
		ctx.Error(http.StatusForbidden)
		return
	}

	title := ctx.FormTrim("title")
	if len(title) == 0 {
		ctx.Error(http.StatusNoContent)
		return
	}

	if err := issue_service.ChangeTitle(ctx, issue, ctx.Doer, title); err != nil {
		ctx.ServerError("ChangeTitle", err)
		return
	}

	goCtx := goctx.Background()

	if s.taskTrackerEnabled {
		userName := ctx.Doer.LowerName

		if err := s.linkUnitsFromIssue(goCtx, issue, userName); err != nil {
			log.Debug("unit_linker: run: %v", err)
		}
	}

	ctx.JSON(http.StatusOK, map[string]interface{}{
		"title": issue.Title,
	})
}
