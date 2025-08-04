package repo

import (
	issuesModel "code.gitea.io/gitea/models/issues"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/git"
	repoModule "code.gitea.io/gitea/modules/repository"
	"code.gitea.io/gitea/modules/web"
	apiError "code.gitea.io/gitea/routers/sbt/apierror"
	"code.gitea.io/gitea/routers/sbt/logger"
	"code.gitea.io/gitea/routers/sbt/request"
	issueService "code.gitea.io/gitea/services/issue"
	pullService "code.gitea.io/gitea/services/pull"
	"net/http"
)

// ChangePullRequestStatus меняет статус PR
func ChangePullRequestStatus(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	req := web.GetForm(ctx).(*request.ChangePullRequestStatus)
	pr := GetPr(ctx, log)
	if ctx.Written() {
		return
	}

	if (ctx.Doer.ID != pr.PosterID && !ctx.Repo.CanReadIssuesOrPulls(true)) ||
		(pr.IsLocked && !ctx.Repo.CanWriteIssuesOrPulls(true) && !ctx.Doer.IsAdmin) {
		log.Debug("User with userId: %d is not authorized to manage PRs in repository: /%s", ctx.Doer.ID, ctx.Repo.Repository.FullName())
		ctx.JSON(http.StatusBadRequest, apiError.UserUnauthorized())
		return
	}

	if ctx.HasError() {
		log.Error("Unknown error type has occurred, error: %v", ctx.Err())
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		return
	}
	if (ctx.Repo.CanWriteIssuesOrPulls(true) || pr.IsPoster(ctx.Doer.ID)) &&
		(req.Status == "reopen" || req.Status == "close") &&
		!(pr.PullRequest.HasMerged) {

		var existingPr *issuesModel.PullRequest

		if req.Status == "reopen" {
			pull := pr.PullRequest
			var err error
			existingPr, err = issuesModel.GetUnmergedPullRequest(ctx, pull.HeadRepoID, pull.BaseRepoID, pull.HeadBranch, pull.BaseBranch, pull.Flow)
			if err != nil {
				if !issuesModel.IsErrPullRequestNotExist(err) {
					log.Debug("Unmerged pull request with index: %d not found in repository: /%s, error: %v", pr.Index, ctx.Repo.Repository.FullName())
					ctx.JSON(http.StatusBadRequest, apiError.PullRequestNotFound(pr.Index))
					return
				}
			}

			// Regenerate patch and test conflict.
			if existingPr == nil {
				pr.PullRequest.HeadCommitID = ""
				pullService.AddToTaskQueue(pr.PullRequest)
			} else {
				log.Debug("Pull request for branches %s and %s repository: /%s already exists - existing pr index %d", pull.BaseBranch, pull.HeadBranch, ctx.Repo.Repository.FullName(), existingPr.Index)
				ctx.JSON(http.StatusBadRequest, apiError.PullRequestAlreadyExist(existingPr.Index))
				return
			}

			// check whether the ref of PR <refs/pulls/pr_index/head> in base repo is consistent with the head commit of head branch in the head repo
			// get head commit of PR
			prHeadRef := pull.GetGitRefName()
			if err := pull.LoadBaseRepo(ctx); err != nil {
				log.Error("Unable to load base repo, error: %v", err)
				ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
				return
			}

			prHeadCommitID, err := git.GetFullCommitID(ctx, pull.BaseRepo.RepoPath(), prHeadRef)
			if err != nil {
				log.Error("Get head commit Id of pr fail, error: %v", err)
				ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
				return
			}

			// get head commit of branch in the head repo
			if err := pull.LoadHeadRepo(ctx); err != nil {
				log.Error("Unable to load head repo, error: %v", err)
				ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
				return
			}
			if ok := git.IsBranchExist(ctx, pull.HeadRepo.OwnerName, pull.HeadRepo.Name, pull.HeadRepo.RepoPath(), pull.BaseBranch); !ok {
				log.Debug("Not able to reopen PR, branch: %s not exists in repository: /%s", pull.BaseBranch, ctx.Repo.Repository.FullName())
				ctx.JSON(http.StatusBadRequest, apiError.BranchNotExist(pull.BaseBranch))
				return
			}
			headBranchRef := pull.GetGitHeadBranchRefName()
			headBranchCommitID, err := git.GetFullCommitID(ctx, pull.HeadRepo.RepoPath(), headBranchRef)
			if err != nil {
				log.Error("Get head commit Id of head branch fail, error: %v", err)
				ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
				return
			}

			err = pull.LoadIssue(ctx)
			if err != nil {
				log.Error("Not able to load pull request, error: %v", err)
				ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
				return
			}

			if prHeadCommitID != headBranchCommitID {
				// force push to base repo
				err := git.Push(ctx, pull.HeadRepo.RepoPath(), git.PushOptions{
					Remote: pull.BaseRepo.RepoPath(),
					Branch: pull.HeadBranch + ":" + prHeadRef,
					Force:  true,
					Env:    repoModule.InternalPushingEnvironment(pull.Issue.Poster, pull.BaseRepo),
				})
				if err != nil {
					log.Error("Not able to push, error: %v", err)
					ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
					return
				}
			}
		}

		if existingPr == nil {
			isClosed := req.Status == "close"
			if err := issueService.ChangeStatus(pr, ctx.Doer, "", isClosed); err != nil {
				if issuesModel.IsErrPullWasClosed(err) {
					log.Debug("Pull request with index: %d already closed in repository: /%s", pr.Index, ctx.Repo.Repository.FullName())
					ctx.JSON(http.StatusBadRequest, apiError.PullRequestAlreadyClosed(pr.Index))
				} else {
					log.Error("Not able to change PR status, error: %v", err)
					ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
				}
				return
			}
		}
	}

	if req.Comment != "" {
		_, err := issueService.CreateIssueComment(ctx, ctx.Doer, ctx.Repo.Repository, pr, req.Comment, []string{})
		if err != nil {
			log.Error("Not able to create comment for PR, error: %v", err)
			ctx.JSON(http.StatusInternalServerError, apiError.CommentWasNotAdded())
			return
		}
	}

	ctx.Status(http.StatusOK)
}

// GetPr возвращает PR по его индексу.
func GetPr(ctx *context.Context, log logger.Logger) *issuesModel.Issue {
	index := ctx.ParamsInt64(":index")
	pr, err := issuesModel.GetIssueByIndex(ctx.Repo.Repository.ID, index)
	if err != nil {
		if issuesModel.IsErrPullRequestNotExist(err) || issuesModel.IsErrIssueNotExist(err) {
			log.Debug("Pull request with index: %d in repository: /%s not found", index, ctx.Repo.Repository.FullName())
			ctx.JSON(http.StatusBadRequest, apiError.PullRequestNotFound(index))
		} else {
			log.Error("Unknown error type has occurred, error: %v", ctx.Err())
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		}
		return nil
	}
	pr.Repo = ctx.Repo.Repository

	if err = pr.LoadAttributes(ctx); err != nil {
		log.Error("Unable to load attributes for pull request with index: %d in repository: /%s, error: %v", index, ctx.Repo.Repository.FullName(), err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		return nil
	}
	return pr
}
