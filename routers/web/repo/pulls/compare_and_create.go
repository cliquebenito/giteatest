package pulls

import (
	"net/http"
	"strconv"

	issues_model "code.gitea.io/gitea/models/issues"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/unit"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/git"
	issue_template "code.gitea.io/gitea/modules/issue/template"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/sbt/audit"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/upload"
	"code.gitea.io/gitea/modules/util"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/modules/web/middleware"
	"code.gitea.io/gitea/routers/utils"
	web_repo "code.gitea.io/gitea/routers/web/repo"
	"code.gitea.io/gitea/services/forms"
	"code.gitea.io/gitea/services/gitdiff"
	pull_service "code.gitea.io/gitea/services/pull"
)

// CompareAndPullRequestPost response for creating pull request
func (s Server) CompareAndPullRequestPost(ctx *context.Context) {
	form := web.GetForm(ctx).(*forms.CreateIssueForm)
	ctx.Data["Title"] = ctx.Tr("repo.pulls.compare_changes")
	ctx.Data["PageIsComparePull"] = true
	ctx.Data["IsDiffCompare"] = true
	ctx.Data["IsRepoToolbarCommits"] = true
	ctx.Data["PullRequestWorkInProgressPrefixes"] = setting.Repository.PullRequest.WorkInProgressPrefixes
	ctx.Data["IsAttachmentEnabled"] = setting.Attachment.Enabled
	upload.AddUploadContext(ctx, "comment")
	ctx.Data["HasIssuesOrPullsWritePermission"] = ctx.Repo.CanWrite(unit.TypePullRequests)

	var (
		repo        = ctx.Repo.Repository
		attachments []string
	)
	auditParams := map[string]string{
		"repository": repo.Name,
	}

	ci := web_repo.ParseCompareInfo(ctx)
	defer func() {
		if ci != nil && ci.HeadGitRepo != nil {
			ci.HeadGitRepo.Close()
		}
	}()
	if ctx.Written() {
		auditParams["error"] = "Failed to parse compare info"
		audit.CreateAndSendEvent(audit.PRCreateEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}

	labelIDs, assigneeIDs, milestoneID, _ := web_repo.ValidateRepoMetas(ctx, *form, true)
	if ctx.Written() {
		auditParams["error"] = "Failed to validate repo metas"
		audit.CreateAndSendEvent(audit.PRCreateEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}

	if setting.Attachment.Enabled {
		attachments = form.Files
	}

	if ctx.HasError() {
		middleware.AssignForm(form, ctx.Data)

		// This stage is already stop creating new pull request, so it does not matter if it has
		// something to compare or not.
		web_repo.PrepareCompareDiff(ctx, ci,
			gitdiff.GetWhitespaceFlag(ctx.Data["WhitespaceBehavior"].(string)))
		if ctx.Written() {
			auditParams["error"] = "Failed to prepare compare diff"
			audit.CreateAndSendEvent(audit.PRCreateEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}

		if len(form.Title) > 255 {
			var trailer string
			form.Title, trailer = util.SplitStringAtByteN(form.Title, 255)

			form.Content = trailer + "\n\n" + form.Content
		}
		middleware.AssignForm(form, ctx.Data)

		ctx.HTML(http.StatusOK, web_repo.TplCompareDiff)
		auditParams["error"] = "Please compare diff"
		audit.CreateAndSendEvent(audit.PRCreateEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}

	if util.IsEmptyString(form.Title) {
		web_repo.PrepareCompareDiff(ctx, ci,
			gitdiff.GetWhitespaceFlag(ctx.Data["WhitespaceBehavior"].(string)))
		if ctx.Written() {
			auditParams["error"] = "Failed to prepare compare diff"
			audit.CreateAndSendEvent(audit.PRCreateEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}

		ctx.RenderWithErr(ctx.Tr("repo.issues.new.title_empty"), web_repo.TplCompareDiff, form)
		auditParams["error"] = "Title empty"
		audit.CreateAndSendEvent(audit.PRCreateEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}

	content := form.Content
	if filename := ctx.Req.Form.Get("template-file"); filename != "" {
		if template, err := issue_template.UnmarshalFromRepo(ctx.Repo.GitRepo, ctx.Repo.Repository.DefaultBranch, filename); err == nil {
			content = issue_template.RenderToMarkdown(template, ctx.Req.Form)
		}
	}

	pullIssue := &issues_model.Issue{
		RepoID:      repo.ID,
		Repo:        repo,
		Title:       form.Title,
		PosterID:    ctx.Doer.ID,
		Poster:      ctx.Doer,
		MilestoneID: milestoneID,
		IsPull:      true,
		Content:     content,
	}
	pullRequest := &issues_model.PullRequest{
		HeadRepoID:          ci.HeadRepo.ID,
		BaseRepoID:          repo.ID,
		HeadBranch:          ci.HeadBranch,
		BaseBranch:          ci.BaseBranch,
		HeadRepo:            ci.HeadRepo,
		BaseRepo:            repo,
		MergeBase:           ci.CompareInfo.MergeBase,
		Type:                issues_model.PullRequestGitea,
		AllowMaintainerEdit: form.AllowMaintainerEdit,
	}
	// FIXME: check error in the case two people send pull request at almost same time, give nice error prompt
	// instead of 500.

	if err := pull_service.NewPullRequest(ctx, repo, pullIssue, labelIDs, attachments, pullRequest, assigneeIDs); err != nil {
		if repo_model.IsErrUserDoesNotHaveAccessToRepo(err) {
			ctx.Error(http.StatusBadRequest, "UserDoesNotHaveAccessToRepo", err.Error())
			auditParams["error"] = "User does not have access to repo"
			audit.CreateAndSendEvent(audit.PRCreateEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		} else if git.IsErrPushRejected(err) {
			pushrejErr := err.(*git.ErrPushRejected)
			message := pushrejErr.Message
			if len(message) == 0 {
				ctx.Flash.Error(ctx.Tr("repo.pulls.push_rejected_no_message"))
				auditParams["error"] = "Push rejected because no message"
			} else {
				flashError, err := ctx.RenderToString(web_repo.TplAlertDetails, map[string]interface{}{
					"Message": ctx.Tr("repo.pulls.push_rejected"),
					"Summary": ctx.Tr("repo.pulls.push_rejected_summary"),
					"Details": utils.SanitizeFlashErrorString(pushrejErr.Message),
				})
				if err != nil {
					ctx.ServerError("CompareAndPullRequest.HTMLString", err)
					auditParams["error"] = "Error has occurred while rendering template to string"
					audit.CreateAndSendEvent(audit.PRCreateEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
					return
				}
				ctx.Flash.Error(flashError)
				auditParams["error"] = "Push rejected"
			}
			ctx.Redirect(pullIssue.Link())
			audit.CreateAndSendEvent(audit.PRCreateEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}
		ctx.ServerError("NewPullRequest", err)
		return
	}

	if s.taskTrackerEnabled {
		userName := ctx.Doer.LowerName

		if err := s.linkUnitsFromIssue(ctx, pullIssue, userName); err != nil {
			log.Debug("unit_linker: run: %v", err)
		}
	}

	log.Trace("Pull request created: %d/%d", repo.ID, pullIssue.ID)
	ctx.Redirect(pullIssue.Link())

	auditParams["pr_number"] = strconv.FormatInt(pullRequest.Index, 10)
	audit.CreateAndSendEvent(audit.PRCreateEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)
}
