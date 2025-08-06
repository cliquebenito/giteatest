package repo

import (
	"net/http"

	issuesModel "code.gitea.io/gitea/models/issues"
	userModel "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/web"
	apiError "code.gitea.io/gitea/routers/sbt/apierror"
	"code.gitea.io/gitea/routers/sbt/logger"
	"code.gitea.io/gitea/routers/sbt/request"
	issueService "code.gitea.io/gitea/services/issue"
)

// ChangePullRequestReviewer метод добавление/удаления ревьюера в запрос на слияние.
// Метод написан по аналогии с методом repo.UpdatePullReviewRequest.
// В оригинальном методе приходит список issuesIds и в range проводится проверка IsPull,
// а так же устанавливаются ревьюеры в range для каждого issueId.
// В данном методе мы берем issue по pulls/{index}
// В качестве ревьюера можно назначить соавторов и команды, если репозиторий принадлежит организации
func ChangePullRequestReviewer(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	pr := GetPr(ctx, log)
	if ctx.Written() {
		return
	}

	req := web.GetForm(ctx).(*request.ChangePullRequestReviewer)

	if (ctx.Doer.ID != pr.PosterID && !ctx.Repo.CanReadIssuesOrPulls(true)) ||
		(pr.IsLocked && !ctx.Repo.CanWriteIssuesOrPulls(true) && !ctx.Doer.IsAdmin) {
		log.Debug("User with userId: %d is not authorized to manage PRs in repository: %s", ctx.Doer.ID, ctx.Repo.Repository.FullName())
		ctx.JSON(http.StatusBadRequest, apiError.UserUnauthorized())
		return
	}

	// Если ревьюер пользователь, а не команда
	reviewer, err := userModel.GetUserByID(ctx, req.ReviewerId)
	if err != nil {
		if userModel.IsErrUserNotExist(err) {
			log.Debug("Reviewer with userId: %d does not exist", req.ReviewerId)
			ctx.JSON(http.StatusBadRequest, apiError.UserDoesNotExist())
		} else {
			log.Error("Error has occurred while getting reviewer with userId: %d for PR with index: %d, error: %v", req.ReviewerId, pr.Index, err)
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		}
		return
	}

	err = issueService.IsValidReviewRequest(ctx, reviewer, ctx.Doer, req.Action == "attach", pr, nil)
	if err != nil {
		if issuesModel.IsErrNotValidReviewRequest(err) {
			log.Debug("User with userId: %d not valid reviewer for PR with index: %d", req.ReviewerId, pr.Index)
			ctx.JSON(http.StatusBadRequest, apiError.NotValidPullRequestReviewer())
		} else {
			log.Error("Error has occurred while checking valid reviewer userId: %d for PR with index: %d, error: %v", req.ReviewerId, pr.Index, err)
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		}
		return
	}

	if _, err = issueService.ReviewRequest(ctx, pr, ctx.Doer, reviewer, req.Action == "attach"); err != nil {
		log.Error("Error has occurred while %s-ing reviewer userId: %d for PR with index: %d, error: %v", req.Action, req.ReviewerId, pr.Index, err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		return
	}

	ctx.Status(http.StatusOK)
}
