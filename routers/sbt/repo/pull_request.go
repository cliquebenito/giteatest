package repo

import (
	issuesModel "code.gitea.io/gitea/models/issues"
	repoModel "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/unit"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/util"
	apiError "code.gitea.io/gitea/routers/sbt/apierror"
	"code.gitea.io/gitea/routers/sbt/logger"
	"net/http"
	"net/url"
)

// getIssueByIndex возвращает PR по его индексу.
func getIssueByIndex(ctx *context.Context, index int64) *issuesModel.Issue {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	prIssue, err := issuesModel.GetIssueByIndex(ctx.Repo.Repository.ID, index)
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
	prIssue.Repo = ctx.Repo.Repository

	if err = prIssue.LoadAttributes(ctx); err != nil {
		log.Error("Unable to load attributes for pull request with index: %d in repository: /%s, error: %v", index, ctx.Repo.Repository.FullName(), err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		return nil
	}
	return prIssue
}

// checkIssueReadRights проверяет права на чтение к пулл реквесту
func checkIssueReadRights(ctx *context.Context, issue *issuesModel.Issue) {
	log := logger.Logger{}
	log.SetTraceId(ctx)
	if ctx.Doer.ID != issue.PosterID && !ctx.Repo.CanReadIssuesOrPulls(issue.IsPull) && !ctx.Doer.IsAdmin {
		log.Debug("User with userId: %d is not authorized to read PR or issue in repository: /%s", ctx.Doer.ID, ctx.Repo.Repository.FullName())
		ctx.JSON(http.StatusUnauthorized, apiError.UserUnauthorized())
		return
	}
}

// checkIssueReadRights проверяет права на запись к пулл реквесту
func checkIssueWriteRights(ctx *context.Context, issue *issuesModel.Issue) {
	log := logger.Logger{}
	log.SetTraceId(ctx)
	if ctx.Doer.ID != issue.PosterID && !ctx.Repo.CanWriteIssuesOrPulls(issue.IsPull) && !ctx.Doer.IsAdmin {
		log.Debug("User with userId: %d is not authorized to write PR or issue in repository: /%s", ctx.Doer.ID, ctx.Repo.Repository.FullName())
		ctx.JSON(http.StatusUnauthorized, apiError.UserUnauthorized())
		return
	}
}

// MustAllowPulls проверяет включена ли работа с пулл реквестами и может ли пользователь работать с ним
func MustAllowPulls(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	if !ctx.Repo.Repository.CanEnablePulls() || !ctx.Repo.CanRead(unit.TypePullRequests) {
		log.Debug("Pull requests are not accessible in repository: /%s", ctx.Repo.Repository.FullName())
		ctx.JSON(http.StatusUnauthorized, apiError.UserUnauthorized())
		ctx.NotFound("MustAllowPulls", nil)
		return
	}

	// User can send pull request if owns a forked repository.
	if ctx.IsSigned && repoModel.HasForkedRepo(ctx.Doer.ID, ctx.Repo.Repository.ID) {
		ctx.Repo.PullRequest.Allowed = true
		ctx.Repo.PullRequest.HeadInfoSubURL = url.PathEscape(ctx.Doer.Name) + ":" + util.PathEscapeSegments(ctx.Repo.BranchName)
	}
}
