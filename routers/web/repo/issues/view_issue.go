package issues

import (
	"fmt"
	"net/http"
	"strconv"

	issues_model "code.gitea.io/gitea/models/issues"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/log"
	repo_module "code.gitea.io/gitea/modules/repository"
	"code.gitea.io/gitea/modules/sbt/audit"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/routers/web/repo"
	"code.gitea.io/gitea/services/forms"
	issue_service "code.gitea.io/gitea/services/issue"
	pull_service "code.gitea.io/gitea/services/pull"
)

// NewComment create a comment for issue
func (s Server) NewComment(ctx *context.Context) {
	form := web.GetForm(ctx).(*forms.CreateCommentForm)
	issue := repo.GetActionIssue(ctx)
	if ctx.Written() {
		return
	}

	if !ctx.IsSigned || (ctx.Doer.ID != issue.PosterID && !ctx.Repo.CanReadIssuesOrPulls(issue.IsPull)) {
		if log.IsTrace() {
			if ctx.IsSigned {
				issueType := "issues"
				if issue.IsPull {
					issueType = "pulls"
				}
				log.Trace("Permission Denied: User %-v not the Poster (ID: %d) and cannot read %s in Repo %-v.\n"+
					"User in Repo has Permissions: %-+v",
					ctx.Doer,
					issue.PosterID,
					issueType,
					ctx.Repo.Repository,
					ctx.Repo.Permission)
			} else {
				log.Trace("Permission Denied: Not logged in")
			}
		}

		ctx.Error(http.StatusForbidden)
		return
	}

	if issue.IsLocked && !ctx.Repo.CanWriteIssuesOrPulls(issue.IsPull) && !ctx.Doer.IsAdmin {
		ctx.Flash.Error(ctx.Tr("repo.issues.comment_on_locked"))
		ctx.Redirect(issue.Link())
		return
	}

	var attachments []string
	if setting.Attachment.Enabled {
		attachments = form.Files
	}

	if ctx.HasError() {
		ctx.Flash.Error(ctx.Data["ErrorMsg"].(string))
		ctx.Redirect(issue.Link())
		return
	}

	var comment *issues_model.Comment
	defer func() {
		// Check if issue admin/poster changes the status of issue.
		if (ctx.Repo.CanWriteIssuesOrPulls(issue.IsPull) || (ctx.IsSigned && issue.IsPoster(ctx.Doer.ID))) &&
			(form.Status == "reopen" || form.Status == "close") &&
			!(issue.IsPull && issue.PullRequest.HasMerged) {

			// Duplication and conflict check should apply to reopen pull request.
			var pr *issues_model.PullRequest

			auditParams := map[string]string{
				"repository": ctx.Repo.Repository.Name,
				"pr_number":  strconv.FormatInt(issue.Index, 10),
			}

			if form.Status == "reopen" && issue.IsPull {
				pull := issue.PullRequest
				var err error
				pr, err = issues_model.GetUnmergedPullRequest(ctx, pull.HeadRepoID, pull.BaseRepoID, pull.HeadBranch, pull.BaseBranch, pull.Flow)
				if err != nil {
					if !issues_model.IsErrPullRequestNotExist(err) {
						ctx.Flash.Error(ctx.Tr("repo.issues.dependency.pr_close_blocked"))
						ctx.Redirect(fmt.Sprintf("%s/pulls/%d", ctx.Repo.RepoLink, pull.Index))
						auditParams["error"] = "Pull request not exist"
						audit.CreateAndSendEvent(audit.PRReopenEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
						return
					}
				}

				// Regenerate patch and test conflict.
				if pr == nil {
					issue.PullRequest.HeadCommitID = ""
					pull_service.AddToTaskQueue(issue.PullRequest)
				}

				// check whether the ref of PR <refs/pulls/pr_index/head> in base repo is consistent with the head commit of head branch in the head repo
				// get head commit of PR
				prHeadRef := pull.GetGitRefName()
				if err := pull.LoadBaseRepo(ctx); err != nil {
					ctx.ServerError("Unable to load base repo", err)
					auditParams["error"] = "Unable to load base repo"
					audit.CreateAndSendEvent(audit.PRReopenEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
					return
				}
				prHeadCommitID, err := git.GetFullCommitID(ctx, pull.BaseRepo.RepoPath(), prHeadRef)
				if err != nil {
					ctx.ServerError("Get head commit Id of pr fail", err)
					auditParams["error"] = "Failed to get head commit id of pull request"
					audit.CreateAndSendEvent(audit.PRReopenEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
					return
				}

				// get head commit of branch in the head repo
				if err := pull.LoadHeadRepo(ctx); err != nil {
					ctx.ServerError("Unable to load head repo", err)
					auditParams["error"] = "Unable to load head repo"
					audit.CreateAndSendEvent(audit.PRReopenEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
					return
				}
				if ok := git.IsBranchExist(ctx, pull.HeadRepo.OwnerName, pull.HeadRepo.Name, pull.HeadRepo.RepoPath(), pull.BaseBranch); !ok {
					// todo localize
					ctx.Flash.Error("The origin branch is delete, cannot reopen.")
					ctx.Redirect(fmt.Sprintf("%s/pulls/%d", ctx.Repo.RepoLink, pull.Index))
					auditParams["error"] = "The origin branch is delete"
					audit.CreateAndSendEvent(audit.PRReopenEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
					return
				}
				headBranchRef := pull.GetGitHeadBranchRefName()
				headBranchCommitID, err := git.GetFullCommitID(ctx, pull.HeadRepo.RepoPath(), headBranchRef)
				if err != nil {
					ctx.ServerError("Get head commit Id of head branch fail", err)
					auditParams["error"] = "Failed to get head commit id of head branch"
					audit.CreateAndSendEvent(audit.PRReopenEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
					return
				}

				err = pull.LoadIssue(ctx)
				if err != nil {
					ctx.ServerError("load the issue of pull request error", err)
					auditParams["error"] = "Error has occurred while loading issue of pull request"
					audit.CreateAndSendEvent(audit.PRReopenEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
					return
				}

				if prHeadCommitID != headBranchCommitID {
					// force push to base repo
					err := git.Push(ctx, pull.HeadRepo.RepoPath(), git.PushOptions{
						Remote: pull.BaseRepo.RepoPath(),
						Branch: pull.HeadBranch + ":" + prHeadRef,
						Force:  true,
						Env:    repo_module.InternalPushingEnvironment(pull.Issue.Poster, pull.BaseRepo),
					})
					if err != nil {
						ctx.ServerError("force push error", err)
						auditParams["error"] = "Error has occurred while force pushing"
						audit.CreateAndSendEvent(audit.PRReopenEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
						return
					}
				}
			}

			if pr != nil {
				ctx.Flash.Info(ctx.Tr("repo.pulls.open_unmerged_pull_exists", pr.Index))
				auditParams["error"] = "Unmerged pull request with same changes already exists"
				audit.CreateAndSendEvent(audit.PRReopenEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			} else {
				isClosed := form.Status == "close"
				if err := issue_service.ChangeStatus(issue, ctx.Doer, "", isClosed); err != nil {
					log.Error("ChangeStatus: %v", err)

					if issue.IsPull {
						auditParams["error"] = "Error has occurred while changing status of pull request"
						if isClosed {
							audit.CreateAndSendEvent(audit.PRCloseEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
						} else {
							audit.CreateAndSendEvent(audit.PRReopenEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
						}
					}

					if issues_model.IsErrDependenciesLeft(err) {
						if issue.IsPull {
							ctx.Flash.Error(ctx.Tr("repo.issues.dependency.pr_close_blocked"))
							ctx.Redirect(fmt.Sprintf("%s/pulls/%d", ctx.Repo.RepoLink, issue.Index))
						} else {
							ctx.Flash.Error(ctx.Tr("repo.issues.dependency.issue_close_blocked"))
							ctx.Redirect(fmt.Sprintf("%s/issues/%d", ctx.Repo.RepoLink, issue.Index))
						}
						return
					}
				} else {
					if err := repo.StopTimerIfAvailable(ctx.Doer, issue); err != nil {
						ctx.ServerError("CreateOrStopIssueStopwatch", err)

						auditParams["error"] = "Error has occurred while stopping timer"
						if isClosed {
							audit.CreateAndSendEvent(audit.PRCloseEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
						} else {
							audit.CreateAndSendEvent(audit.PRReopenEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
						}
						return
					}

					if isClosed {
						audit.CreateAndSendEvent(audit.PRCloseEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)
					} else {
						audit.CreateAndSendEvent(audit.PRReopenEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)
					}
					log.Trace("Issue [%d] status changed to closed: %v", issue.ID, issue.IsClosed)
				}
			}

		}

		// Redirect to comment hashtag if there is any actual content.
		typeName := "issues"
		if issue.IsPull {
			typeName = "pulls"
		}
		if comment != nil {
			ctx.Redirect(fmt.Sprintf("%s/%s/%d#%s", ctx.Repo.RepoLink, typeName, issue.Index, comment.HashTag()))
		} else {
			ctx.Redirect(fmt.Sprintf("%s/%s/%d", ctx.Repo.RepoLink, typeName, issue.Index))
		}
	}()
	auditParams := map[string]string{
		"repository_id": strconv.FormatInt(ctx.Repo.Repository.ID, 10),
	}
	auditParams["issue_id"] = strconv.FormatInt(issue.ID, 10)
	auditParams["pull_request_id"] = strconv.FormatInt(issue.PullRequest.ID, 10)
	if s.taskTrackerEnabled {
		userName := ctx.Doer.LowerName
		issue.IsClosed = true
		if form.Status == "reopen" {
			issue.IsClosed = false
		}
		if err := s.updateIssueStatus(ctx, issue, userName); err != nil {
			log.Error("updating status pr: run: %v", err)
			auditParams["error"] = "Error has occurred trying to change issue status from unit"
			audit.CreateAndSendEvent(audit.PullRequestsUpdateEvent, userName, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		}
		audit.CreateAndSendEvent(audit.PullRequestsUpdateEvent, userName, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)
	}
	// Fix #321: Allow empty comments, as long as we have attachments.
	if len(form.Content) == 0 && len(attachments) == 0 {
		return
	}

	comment, err := issue_service.CreateIssueComment(ctx, ctx.Doer, ctx.Repo.Repository, issue, form.Content, attachments)
	if err != nil {
		ctx.ServerError("CreateIssueComment", err)
		return
	}

	log.Trace("Comment created: %d/%d/%d", ctx.Repo.Repository.ID, issue.ID, comment.ID)
}
