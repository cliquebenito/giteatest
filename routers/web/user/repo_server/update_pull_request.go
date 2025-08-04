package repo_server

import (
	"fmt"

	"code.gitea.io/gitea/models"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/routers/utils"
	"code.gitea.io/gitea/routers/web/repo"
	pull_service "code.gitea.io/gitea/services/pull"
)

func (s *Server) UpdatePullRequest(ctx *context.Context) {
	issue := repo.CheckPullInfo(ctx)
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
			flashError, err := ctx.RenderToString(repo.TplAlertDetails, map[string]interface{}{
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
			flashError, err := ctx.RenderToString(repo.TplAlertDetails, map[string]interface{}{
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
