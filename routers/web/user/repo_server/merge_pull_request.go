package repo_server

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"

	"code.gitea.io/gitea/modules/trace"

	"code.gitea.io/gitea/models"
	issues_model "code.gitea.io/gitea/models/issues"
	"code.gitea.io/gitea/models/organization"
	pull_model "code.gitea.io/gitea/models/pull"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/role_model"
	"code.gitea.io/gitea/models/unit"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/sbt/audit"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/routers/utils"
	"code.gitea.io/gitea/routers/web/repo"
	"code.gitea.io/gitea/routers/web/user/accesser"
	asymkey_service "code.gitea.io/gitea/services/asymkey"
	"code.gitea.io/gitea/services/automerge"
	"code.gitea.io/gitea/services/forms"
	pull_service "code.gitea.io/gitea/services/pull"
)

// MergePullRequest response for merging pull request
func (s *Server) MergePullRequest(ctx *context.Context) {
	logTracer := trace.NewLogTracer()
	traceMessage := logTracer.CreateTraceMessage(ctx)
	err := logTracer.Trace(traceMessage)
	if err != nil {
		log.Errorf("Error has occurred while creating trace message: %v", err)
	}
	defer func() {
		err = logTracer.TraceTime(traceMessage)
		if err != nil {
			log.Errorf("Error has occurred while creating trace time message: %v", err)
		}
	}()

	repoUnit, err := ctx.Repo.Repository.GetUnit(ctx, unit.TypePullRequests)
	if err != nil {
		log.Error(fmt.Sprintf("Error has occurred  while getting units for repository with id %v: %v", ctx.Repo.Repository.ID, err))
		ctx.ServerError(fmt.Sprintf("Error has occurred  while getting units for repository with id %v: %v", ctx.Repo.Repository.ID, err), nil)
		return
	}

	if repoUnit.PullRequestsConfig().AdminCanMergeWithoutChecks {
		allow, err := role_model.CheckUserPermissionToOrganization(ctx, ctx.Doer, ctx.Data["TenantID"].(string), &organization.Organization{ID: ctx.Repo.Repository.OwnerID}, role_model.MERGE_WITHOUT_CHECK)
		if err != nil || !allow {
			log.Errorf("Error has occurred while checking user's permissions: %v", err)
			ctx.Error(http.StatusForbidden, "User does not have enough custom privileges to create config pr.")
			return
		}
	}

	form := web.GetForm(ctx).(*forms.MergePullRequestForm)
	issue := repo.CheckPullInfo(ctx)
	auditParams := map[string]string{
		"repository": ctx.Repo.Repository.Name,
		"pr_number":  strconv.FormatInt(issue.Index, 10),
	}
	allowed, err := s.orgRequestAccessor.IsAccessGranted(ctx, accesser.OrgAccessRequest{
		DoerID:         ctx.Doer.ID,
		TargetOrgID:    ctx.Repo.Repository.OwnerID,
		TargetTenantID: ctx.Data["TenantID"].(string),
		Action:         role_model.WRITE,
	})
	if err != nil {
		log.Errorf("Error has occurred while checking user's permissions: %v", err)
		ctx.Error(http.StatusForbidden, "User does not have enough custom privileges to merge pr.")
		return
	}
	if !allowed {
		allow, err := s.repoRequestAccessor.AccessesByCustomPrivileges(ctx, accesser.RepoAccessRequest{
			DoerID:          ctx.Doer.ID,
			OrgID:           ctx.Repo.Repository.OwnerID,
			TargetTenantID:  ctx.Data["TenantID"].(string),
			RepoID:          ctx.Repo.Repository.ID,
			CustomPrivilege: role_model.MergePR.String(),
		})
		if err != nil || !allow {
			log.Errorf("Error has occurred while checking user's permissions: %v", err)
			ctx.Error(http.StatusForbidden, "User does not have enough custom privileges to merge pr.")
			return
		}
	}
	if ctx.Written() {
		auditParams["error"] = "Failed to check pull info"
		audit.CreateAndSendEvent(audit.PRMergeEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}

	pr := issue.PullRequest
	pr.Issue = issue
	pr.Issue.Repo = ctx.Repo.Repository

	manuallyMerged := repo_model.MergeStyle(form.Do) == repo_model.MergeStyleManuallyMerged

	mergeCheckType := pull_service.MergeCheckTypeGeneral
	if form.MergeWhenChecksSucceed {
		mergeCheckType = pull_service.MergeCheckTypeAuto
	}
	if manuallyMerged {
		mergeCheckType = pull_service.MergeCheckTypeManually
	}

	// start with merging by checking
	if err := pull_service.CheckPullMergable(ctx, ctx.Doer, &ctx.Repo.Permission, pr, mergeCheckType, form.ForceMerge); err != nil {
		switch {
		case errors.Is(err, pull_service.ErrIsClosed):
			if issue.IsPull {
				ctx.Flash.Error(ctx.Tr("repo.pulls.is_closed"))
				auditParams["error"] = "Pull request is closed"
			} else {
				ctx.Flash.Error(ctx.Tr("repo.Issues.closed_title"))
				auditParams["error"] = "Issue closed title"
			}
		case errors.Is(err, pull_service.ErrUserNotAllowedToMerge):
			ctx.Flash.Error(ctx.Tr("repo.pulls.update_not_allowed"))
			auditParams["error"] = "Pull request update not allowed"
		case errors.Is(err, pull_service.ErrHasMerged):
			ctx.Flash.Error(ctx.Tr("repo.pulls.has_merged"))
			auditParams["error"] = "Pull request has merged"
		case errors.Is(err, pull_service.ErrIsWorkInProgress):
			ctx.Flash.Error(ctx.Tr("repo.pulls.no_merge_wip"))
			auditParams["error"] = "Pull request status is 'work in progress'"
		case errors.Is(err, pull_service.ErrNotMergableState):
			ctx.Flash.Error(ctx.Tr("repo.pulls.no_merge_not_ready"))
			auditParams["error"] = "Pull request is not ready to merge"
		case models.IsErrDisallowedToMerge(err):
			ctx.Flash.Error(ctx.Tr("repo.pulls.no_merge_not_ready"))
			auditParams["error"] = "Pull request is not ready to merge"
		case asymkey_service.IsErrWontSign(err):
			ctx.Flash.Error(err.Error()) // has no translation ...
			auditParams["error"] = "Commit would not be signed"
		case errors.Is(err, pull_service.ErrDependenciesLeft):
			ctx.Flash.Error(ctx.Tr("repo.Issues.dependency.pr_close_blocked"))
			auditParams["error"] = "Pull request close blocked"
		default:
			ctx.ServerError("WebCheck", err)
			auditParams["error"] = "Error has occurred while checking pull mergable"
			audit.CreateAndSendEvent(audit.PRMergeEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}

		audit.CreateAndSendEvent(audit.PRMergeEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		ctx.Redirect(issue.Link())
		return
	}

	// handle manually-merged mark
	if manuallyMerged {
		if err := pull_service.MergedManually(pr, ctx.Doer, ctx.Repo.GitRepo, form.MergeCommitID); err != nil {
			switch {

			case models.IsErrInvalidMergeStyle(err):
				ctx.Flash.Error(ctx.Tr("repo.pulls.invalid_merge_option"))
				auditParams["error"] = "Invalid merge option"
			case strings.Contains(err.Error(), "Wrong commit ID"):
				ctx.Flash.Error(ctx.Tr("repo.pulls.wrong_commit_id"))
				auditParams["error"] = "Wrong commit ID"
			default:
				ctx.ServerError("MergedManually", err)
				auditParams["error"] = "Error has occurred while merging manually"
				audit.CreateAndSendEvent(audit.PRMergeEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
				return
			}
		}

		audit.CreateAndSendEvent(audit.PRMergeEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		ctx.Redirect(issue.Link())
		return
	}

	message := strings.TrimSpace(form.MergeTitleField)
	if len(message) == 0 {
		var err error
		message, _, err = pull_service.GetDefaultMergeMessage(ctx, ctx.Repo.GitRepo, pr, repo_model.MergeStyle(form.Do))
		if err != nil {
			ctx.ServerError("GetDefaultMergeMessage", err)
			auditParams["error"] = "Error has occurred while getting default message"
			audit.CreateAndSendEvent(audit.PRMergeEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}
	}

	form.MergeMessageField = strings.TrimSpace(form.MergeMessageField)
	if len(form.MergeMessageField) > 0 {
		message += "\n\n" + form.MergeMessageField
	}

	if form.MergeWhenChecksSucceed {
		// delete all scheduled auto merges
		_ = pull_model.DeleteScheduledAutoMerge(ctx, pr.ID)
		// schedule auto merge
		scheduled, err := automerge.ScheduleAutoMerge(ctx, ctx.Doer, pr, repo_model.MergeStyle(form.Do), message)
		if err != nil {
			ctx.ServerError("ScheduleAutoMerge", err)
			auditParams["error"] = "Error has occurred while scheduling auto merge"
			audit.CreateAndSendEvent(audit.PRMergeEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		} else if scheduled {
			// nothing more to do ...
			ctx.Flash.Success(ctx.Tr("repo.pulls.auto_merge_newly_scheduled"))
			ctx.Redirect(fmt.Sprintf("%s/pulls/%d", ctx.Repo.RepoLink, pr.Index))
			return
		}
	}

	pr.HeadCommitID = form.HeadCommitID

	if err := pull_service.Merge(ctx, pr, ctx.Doer, ctx.Repo.GitRepo, repo_model.MergeStyle(form.Do), form.HeadCommitID, message, false); err != nil {
		if models.IsErrInvalidMergeStyle(err) {
			ctx.Flash.Error(ctx.Tr("repo.pulls.invalid_merge_option"))
			auditParams["error"] = "Invalid merge option"
			ctx.Redirect(issue.Link())
		} else if models.IsErrMergeConflicts(err) {
			conflictError := err.(models.ErrMergeConflicts)
			flashError, err := ctx.RenderToString(repo.TplAlertDetails, map[string]interface{}{
				"Message": ctx.Tr("repo.pulls.merge_conflict"),
				"Summary": ctx.Tr("repo.pulls.merge_conflict_summary"),
				"Details": utils.SanitizeFlashErrorString(conflictError.StdErr) + "<br>" + utils.SanitizeFlashErrorString(conflictError.StdOut),
			})
			if err != nil {
				ctx.ServerError("MergePullRequest.HTMLString", err)
				auditParams["error"] = "Error has occurred while rendering template to string"
				audit.CreateAndSendEvent(audit.PRMergeEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
				return
			}
			ctx.Flash.Error(flashError)
			auditParams["error"] = "Merge conflict"
			ctx.Redirect(issue.Link())
		} else if models.IsErrRebaseConflicts(err) {
			conflictError := err.(models.ErrRebaseConflicts)
			flashError, err := ctx.RenderToString(repo.TplAlertDetails, map[string]interface{}{
				"Message": ctx.Tr("repo.pulls.rebase_conflict", utils.SanitizeFlashErrorString(conflictError.CommitSHA)),
				"Summary": ctx.Tr("repo.pulls.rebase_conflict_summary"),
				"Details": utils.SanitizeFlashErrorString(conflictError.StdErr) + "<br>" + utils.SanitizeFlashErrorString(conflictError.StdOut),
			})
			if err != nil {
				ctx.ServerError("MergePullRequest.HTMLString", err)
				auditParams["error"] = "Error has occurred while rendering template to string"
				audit.CreateAndSendEvent(audit.PRMergeEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
				return
			}
			ctx.Flash.Error(flashError)
			auditParams["error"] = "Rebase conflict"
			ctx.Redirect(issue.Link())
		} else if models.IsErrMergeUnrelatedHistories(err) {
			log.Debugf("MergeUnrelatedHistories error: %v", err)
			ctx.Flash.Error(ctx.Tr("repo.pulls.unrelated_histories"))
			auditParams["error"] = "Unrelated histories"
			ctx.Redirect(issue.Link())
		} else if git.IsErrPushOutOfDate(err) {
			log.Debugf("MergePushOutOfDate error: %v", err)
			ctx.Flash.Error(ctx.Tr("repo.pulls.merge_out_of_date"))
			auditParams["error"] = "Merge out of date"
			ctx.Redirect(issue.Link())
		} else if models.IsErrSHADoesNotMatch(err) {
			log.Debugf("MergeHeadOutOfDate error: %v", err)
			ctx.Flash.Error(ctx.Tr("repo.pulls.head_out_of_date"))
			auditParams["error"] = "Head out of date"
			ctx.Redirect(issue.Link())
		} else if git.IsErrPushRejected(err) {
			log.Debugf("MergePushRejected error: %v", err)
			pushrejErr := err.(*git.ErrPushRejected)
			message := pushrejErr.Message
			if len(message) == 0 {
				ctx.Flash.Error(ctx.Tr("repo.pulls.push_rejected_no_message"))
				auditParams["error"] = "Push rejected because no message"
			} else {
				flashError, err := ctx.RenderToString(repo.TplAlertDetails, map[string]interface{}{
					"Message": ctx.Tr("repo.pulls.push_rejected"),
					"Summary": ctx.Tr("repo.pulls.push_rejected_summary"),
					"Details": utils.SanitizeFlashErrorString(pushrejErr.Message),
				})
				if err != nil {
					ctx.ServerError("MergePullRequest.HTMLString", err)
					auditParams["error"] = "Error has occurred while rendering template to string"
					audit.CreateAndSendEvent(audit.PRMergeEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
					return
				}
				auditParams["error"] = "Push rejected"
				ctx.Flash.Error(flashError)
			}
			ctx.Redirect(issue.Link())
		} else {
			ctx.ServerError("Merge", err)
			auditParams["error"] = "Error has occurred while merging pull request"
		}
		audit.CreateAndSendEvent(audit.PRMergeEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}
	log.Tracef("Pull request merged: %d", pr.ID)

	if err := repo.StopTimerIfAvailable(ctx.Doer, issue); err != nil {
		ctx.ServerError("CreateOrStopIssueStopwatch", err)
		auditParams["error"] = "Error has occurred while stopping timer"
		audit.CreateAndSendEvent(audit.PRMergeEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}

	log.Tracef("Pull request merged: %d", pr.ID)

	if form.DeleteBranchAfterMerge {
		// Don't cleanup when other pr use this branch as head branch
		auditBranchParams := map[string]string{
			"repository":  ctx.Repo.Repository.Name,
			"branch_name": pr.HeadBranch,
		}
		exist, err := issues_model.HasUnmergedPullRequestsByHeadInfo(ctx, pr.HeadRepoID, pr.HeadBranch)
		if err != nil {
			ctx.ServerError("HasUnmergedPullRequestsByHeadInfo", err)
			auditBranchParams["error"] = "Error has occurred while checking unmerged pull requests by head info"
			audit.CreateAndSendEvent(audit.BranchDeleteEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditBranchParams)
			return
		}
		if exist {
			ctx.Redirect(issue.Link())
			auditBranchParams["error"] = "Other PR use this branch as head branch"
			audit.CreateAndSendEvent(audit.BranchDeleteEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditBranchParams)
			return
		}

		var headRepo *git.Repository
		if ctx.Repo != nil && ctx.Repo.Repository != nil && pr.HeadRepoID == ctx.Repo.Repository.ID && ctx.Repo.GitRepo != nil {
			headRepo = ctx.Repo.GitRepo
		} else {
			headRepo, err = git.OpenRepository(ctx, pr.HeadRepo.OwnerName, pr.HeadRepo.Name, pr.HeadRepo.RepoPath())
			if err != nil {
				ctx.ServerError(fmt.Sprintf("OpenRepository[%s]", pr.HeadRepo.RepoPath()), err)
				auditBranchParams["error"] = "Error has occurred while opening repository"
				audit.CreateAndSendEvent(audit.BranchDeleteEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditBranchParams)
				return
			}
			defer headRepo.Close()
		}
		repo.DeleteBranch(ctx, pr, headRepo)
	}

	audit.CreateAndSendEvent(audit.PRMergeEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)
	ctx.Redirect(issue.Link())
}
