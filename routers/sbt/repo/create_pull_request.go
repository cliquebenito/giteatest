package repo

import (
	issuesModel "code.gitea.io/gitea/models/issues"
	accessModel "code.gitea.io/gitea/models/perm/access"
	repoModel "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/unit"
	userModel "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/timeutil"
	"code.gitea.io/gitea/modules/web"
	apiError "code.gitea.io/gitea/routers/sbt/apierror"
	"code.gitea.io/gitea/routers/sbt/logger"
	"code.gitea.io/gitea/routers/sbt/request"
	"code.gitea.io/gitea/services/convert"
	pullService "code.gitea.io/gitea/services/pull"
	"fmt"
	"net/http"
	"strings"
)

// CreatePullRequest создает запрос на слияние веток
func CreatePullRequest(ctx *context.Context) {
	req := web.GetForm(ctx).(*request.CreatePullRequest)

	log := logger.Logger{}
	log.SetTraceId(ctx)

	if req.Head == req.Base {
		log.Debug("Wrong pull request. Content for branches %s and %s is identical for /%s/%s/", ctx.Repo.Repository.OwnerName, ctx.Repo.Repository.Name, req.Head, req.Base)
		ctx.JSON(http.StatusBadRequest, apiError.BranchesAreIdentical())
		return
	}

	var (
		repo        = ctx.Repo.Repository
		labelIDs    []int64
		milestoneID int64
	)

	// собираем информацию о репозитории и ветках
	gitRepo, compareInfo, baseBranch, headBranch := parseCompareInfo(ctx, req, log)
	if ctx.Written() {
		return
	}
	defer closeGitRepo(gitRepo, log)

	// проверяем нет ли уже аналогичного запроса на слияние
	existingPr, err := issuesModel.GetUnmergedPullRequest(ctx, repo.ID, ctx.Repo.Repository.ID, headBranch, baseBranch, issuesModel.PullRequestFlowGithub)
	if err != nil {
		// Если ошибка ErrPullRequestNotExist, то это нормальная ситуация. Аналогичный PR отсутствует и мы продолжаем исполнение кода
		if !issuesModel.IsErrPullRequestNotExist(err) {
			log.Error("Unknown error type has occurred, error: %v", err)
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
			return
		}
	} else {
		log.Info("Pull request for branches %s and %s repository: /%s/%s already exists - existing pr index %d", baseBranch, headBranch, ctx.Repo.Repository.OwnerName, repo.Name, existingPr.Index)
		ctx.JSON(http.StatusBadRequest, apiError.PullRequestAlreadyExist(existingPr.Index))
		return
	}

	if len(req.Labels) > 0 {
		labels, err := issuesModel.GetLabelsInRepoByIDs(ctx, ctx.Repo.Repository.ID, req.Labels)
		if err != nil {
			log.Error("Unknown error type has occurred while checking labels:, error: %v", labels, err)
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
			return
		}

		labelIDs = make([]int64, len(req.Labels))

		for i := range labels {
			labelIDs[i] = labels[i].ID
		}
	}

	if req.Milestone > 0 {
		milestone, err := issuesModel.GetMilestoneByRepoID(ctx, ctx.Repo.Repository.ID, req.Milestone)
		if err != nil {
			log.Error("Unknown error type has occurred, error: %v", err)
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
			return
		}

		milestoneID = milestone.ID
	}

	var deadlineUnix timeutil.TimeStamp
	if req.Deadline != nil {
		deadlineUnix = timeutil.TimeStamp(req.Deadline.Unix())
	}

	prIssue := &issuesModel.Issue{
		RepoID:       repo.ID,
		Title:        req.Title,
		PosterID:     ctx.Doer.ID,
		Poster:       ctx.Doer,
		MilestoneID:  milestoneID,
		IsPull:       true,
		Content:      req.Body,
		DeadlineUnix: deadlineUnix,
	}
	pr := &issuesModel.PullRequest{
		HeadRepoID: repo.ID,
		BaseRepoID: repo.ID,
		HeadBranch: headBranch,
		BaseBranch: baseBranch,
		HeadRepo:   repo,
		BaseRepo:   repo,
		MergeBase:  compareInfo.MergeBase,
		Type:       issuesModel.PullRequestGitea,
	}

	// Проверяем идентификаторы пользователей
	assigneeIDs, err := issuesModel.MakeIDsFromAPIAssigneesToAdd(ctx, "", req.Assignees)
	if err != nil {
		if userModel.IsErrUserNotExist(err) {
			userErr := err.(userModel.ErrUserNotExist)
			log.Debug("User not found %s while checking assignee in pull request: %s", userErr.Name, req)
			ctx.JSON(http.StatusBadRequest, apiError.UserNotFoundByNameError(userErr.Name))
		} else {
			log.Error("Unknown error type has occurred, error: %v", err)
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		}
		return
	}
	// Проверяем доступность пользователей
	for _, aID := range assigneeIDs {
		assignee, err := userModel.GetUserByID(ctx, aID)
		if err != nil {
			log.Error("Unknown error type has occurred, error: %v", err)
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
			return
		}

		valid, err := accessModel.CanBeAssigned(ctx, assignee, repo, true)
		if err != nil {
			log.Error("Unknown error type has occurred, error: %v", err)
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
			return
		}
		if !valid {
			log.Debug("User %s is not authorized to assign to pull request for repository /%s/%s/", assignee.Name, ctx.Repo.Repository.OwnerName, ctx.Repo.Repository.Name)
			ctx.JSON(http.StatusBadRequest, apiError.UserInsufficientPermission(assignee.Name, "assign to pull request"))
			return
		}
	}

	if err := pullService.NewPullRequest(ctx, repo, prIssue, labelIDs, []string{}, pr, assigneeIDs); err != nil {
		if repoModel.IsErrUserDoesNotHaveAccessToRepo(err) {
			log.Debug("Users %s are not authorized to assign to pull request for repository /%s/%s/", assigneeIDs, ctx.Repo.Repository.OwnerName, ctx.Repo.Repository.Name)
			ctx.JSON(http.StatusBadRequest, apiError.UserInsufficientPermission(strings.Trim(strings.Replace(fmt.Sprint(assigneeIDs), " ", ",", -1), "[]"), "read repository"))
			return
		}
		log.Error("Unknown error type has occurred, error: %v", err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		return
	}

	ctx.JSON(http.StatusCreated, convert.ToAPIPullRequest(ctx, pr, ctx.Doer))
}

func parseCompareInfo(ctx *context.Context, req *request.CreatePullRequest, log logger.Logger) (*git.Repository, *git.CompareInfo, string, string) {
	repo := ctx.Repo.Repository

	baseBranch := req.Base
	headBranch := req.Head

	var (
		headOwner *userModel.User
		err       error
	)

	headOwner = ctx.Repo.Owner
	gitRepo := ctx.Repo.GitRepo

	if !gitRepo.IsBranchExist(baseBranch) {
		log.Debug("Branch: %s not exist in repository: /%s/%s", baseBranch, headOwner.Name, repo.Name)
		ctx.JSON(http.StatusBadRequest, apiError.BranchNotExist(baseBranch))
		return nil, nil, "", ""
	}

	if !gitRepo.IsBranchExist(headBranch) {
		log.Debug("Branch: %s not exist in repository: /%s/%s", headBranch, headOwner.Name, repo.Name)
		ctx.JSON(http.StatusBadRequest, apiError.BranchNotExist(headBranch))
		return nil, nil, "", ""
	}

	// проверяем доступы
	permBase, err := accessModel.GetUserRepoPermission(ctx, repo, ctx.Doer)
	if err != nil {
		closeGitRepo(gitRepo, log)
		log.Error("Unknown error type has occurred due to check repo permissions, error: %v", err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		return nil, nil, "", ""
	}
	if !permBase.CanReadIssuesOrPulls(true) || !permBase.CanRead(unit.TypeCode) {
		if log.IsTrace() {
			log.Trace("Permission Denied: User %-v cannot create/read pull requests or cannot read code in Repo %-v\nUser in baseRepo has Permissions: %-+v",
				ctx.Doer,
				repo,
				permBase)
		}
		closeGitRepo(gitRepo, log)
		log.Debug("User %s is not authorized to create pull request for repository /%s/%s/", ctx.Doer, ctx.Repo.Repository.OwnerName, ctx.Repo.Repository.Name)
		ctx.JSON(http.StatusBadRequest, apiError.UserInsufficientPermission(ctx.Doer.Name, "create pull request"))
		return nil, nil, "", ""
	}

	compareInfo, err := gitRepo.GetCompareInfo(repoModel.RepoPath(repo.Owner.Name, repo.Name), baseBranch, headBranch, false, false)
	if err != nil {
		closeGitRepo(gitRepo, log)
		log.Error("Unknown error type has occurred due to check repo permissions, error: %v", err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		return nil, nil, "", ""
	}

	return gitRepo, compareInfo, baseBranch, headBranch
}

func closeGitRepo(gitRepo *git.Repository, log logger.Logger) {
	err := gitRepo.Close()
	if err != nil {
		log.Error("Not able ti close git repository by path: %s, error: %v", gitRepo.Path, err)
	}
}
