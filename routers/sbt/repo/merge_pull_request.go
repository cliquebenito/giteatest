package repo

import (
	"errors"
	"net/http"
	"strings"

	"code.gitea.io/gitea/models"
	activitiesModel "code.gitea.io/gitea/models/activities"
	"code.gitea.io/gitea/models/git/protected_branch"
	issuesModel "code.gitea.io/gitea/models/issues"
	pullModel "code.gitea.io/gitea/models/pull"
	repoModel "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/web"
	apiError "code.gitea.io/gitea/routers/sbt/apierror"
	"code.gitea.io/gitea/routers/sbt/logger"
	"code.gitea.io/gitea/routers/sbt/request"
	asymkeyService "code.gitea.io/gitea/services/asymkey"
	"code.gitea.io/gitea/services/automerge"
	pullService "code.gitea.io/gitea/services/pull"
	repoService "code.gitea.io/gitea/services/repository"
)

/*
CreateMergePullRequest метод создания слияния пулл реквеста
TODO Разобраться с мердж стилями MergeStyle
TODO Зачем указывать HeadCommitID (в любом случае происходит мердж, но при указании HeadCommitID происходит создание временной папки слияния - зачем?)
*/
func CreateMergePullRequest(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	req := web.GetForm(ctx).(*request.MergePullRequest)

	index := ctx.ParamsInt64(":index")
	iss := getIssueByIndex(ctx, index)
	if ctx.Written() {
		return
	}
	pr := iss.PullRequest

	if err := pr.LoadHeadRepo(ctx); err != nil {
		log.Error("Error has occurred while loading pull request's HEAD. Pull request: %d in repository: %s, error: %v", pr.ID, ctx.Repo.Repository.FullName(), err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		return
	}

	if err := pr.LoadIssue(ctx); err != nil {
		log.Error("Error has occurred while loading pull request's issue. Pull request: %d in repository: %s, error: %v", pr.ID, ctx.Repo.Repository.FullName(), err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		return
	}
	pr.Issue.Repo = ctx.Repo.Repository

	if err := activitiesModel.SetIssueReadBy(ctx, pr.Issue.ID, ctx.Doer.ID); err != nil {
		log.Error("Error has occurred while updating pull request's issue. Pull request: %d in repository: %s,  error: %v", pr.ID, ctx.Repo.Repository.FullName(), err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		return
	}

	manuallyMerged := repoModel.MergeStyle(*req.Do) == repoModel.MergeStyleManuallyMerged

	mergeCheckType := pullService.MergeCheckTypeGeneral
	if req.MergeWhenChecksSucceed != nil && *req.MergeWhenChecksSucceed {
		mergeCheckType = pullService.MergeCheckTypeAuto
	}
	if manuallyMerged {
		mergeCheckType = pullService.MergeCheckTypeManually
	}

	//ForceMerge - проверить
	if req.ForceMerge == nil {
		var defaultBool bool
		req.ForceMerge = &defaultBool
	}

	// Проверяем что эту ветку можно смерджить
	if err := pullService.CheckPullMergable(ctx, ctx.Doer, &ctx.Repo.Permission, pr, mergeCheckType, *req.ForceMerge); err != nil {
		if errors.Is(err, pullService.ErrIsClosed) {
			log.Debug("Pull request: %d in repository: %s has been closed, error: %v", pr.Index, ctx.Repo.Repository.FullName(), err)
			ctx.JSON(http.StatusBadRequest, apiError.PullRequestAlreadyClosed(pr.ID))

		} else if errors.Is(err, pullService.ErrUserNotAllowedToMerge) {
			log.Debug("User %s is not authorized to merge pull request: %d for repository %s", ctx.Doer.Name, pr.ID, ctx.Repo.Repository.FullName())
			ctx.JSON(http.StatusBadRequest, apiError.UserInsufficientPermission(ctx.Doer.Name, "merge commit"))

		} else if errors.Is(err, pullService.ErrHasMerged) {
			log.Debug("Pull request: %d in repository: %s already has been merged, error: %v", pr.Index, ctx.Repo.Repository.FullName(), err)
			ctx.JSON(http.StatusBadRequest, apiError.PullRequestAlreadyMerged())

		} else if errors.Is(err, pullService.ErrIsWorkInProgress) {
			log.Debug("Pull request: %d in  repository: %s in progress and can't be merged, error: %v", pr.Index, ctx.Repo.Repository.FullName(), err)
			ctx.JSON(http.StatusBadRequest, apiError.PullRequestWorkInProgress())

		} else if errors.Is(err, pullService.ErrNotMergableState) {
			log.Debug("Pull request: %d in repository: %s can't be merged because not in mergable state, error: %v", pr.Index, ctx.Repo.Repository.FullName(), err)
			ctx.JSON(http.StatusBadRequest, apiError.PullRequestNotMergableState())

		} else if models.IsErrDisallowedToMerge(err) {
			log.Debug("Pull request: %d  in repository: %s can't be merged because branch: %s is protected and the current user: %s is not allowed to modify it, error: %v", pr.Index, ctx.Repo.Repository.FullName(), pr.BaseBranch, ctx.Doer.Name, err)
			ctx.JSON(http.StatusBadRequest, apiError.BranchIsProtected(pr.BaseBranch))

		} else if asymkeyService.IsErrWontSign(err) {
			log.Debug("Pull request: %d in repository: %s can't be merged because branch: %s is protected and requires sign commit but this merge would not be signed, error: %v", pr.Index, ctx.Repo.Repository.FullName(), pr.BaseBranch, ctx.Doer.Name, err)
			ctx.JSON(http.StatusBadRequest, apiError.BranchIsProtected(pr.BaseBranch))

		} else {
			log.Error("Unknown error type has occurred while checking mergable of pull request: %d in repository %s, error: %v", pr.ID, ctx.Repo.Repository.FullName(), err)
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		}
		return
	}

	if req.MergeCommitID == nil {
		var emptyStr string
		req.MergeCommitID = &emptyStr
	}
	// Пометить пулл реквест как слитый вручную. Не используется
	if manuallyMerged {
		if err := pullService.MergedManually(pr, ctx.Doer, ctx.Repo.GitRepo, *req.MergeCommitID); err != nil {
			switch {
			case models.IsErrInvalidMergeStyle(err):
				log.Debug("Invalid merge style: %v", err)
				ctx.JSON(http.StatusBadRequest, apiError.InvalidMergeStyle())

			case strings.Contains(err.Error(), "Wrong commit ID"):
				log.Debug("Wrong commit Id: %s", *req.MergeCommitID)
				ctx.JSON(http.StatusBadRequest, apiError.CommitNotExist(*req.MergeCommitID))

			default:
				log.Error("Unknown error type has occurred while manually merging pull request: %d in repository %s, error: %v", pr.ID, ctx.Repo.Repository.FullName(), err)
				ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())

			}
			return
		}

		ctx.Status(http.StatusOK)
		return
	}

	var (
		err          error
		mergeMessage string
	)

	if req.MergeTitleField != nil {
		mergeMessage = strings.TrimSpace(*req.MergeTitleField)
	}

	if len(mergeMessage) == 0 {
		mergeMessage, _, err = pullService.GetDefaultMergeMessage(ctx, ctx.Repo.GitRepo, pr, repoModel.MergeStyle(*req.Do))
		if err != nil {
			log.Error("Unknown error type has occurred while getting default merge message in repository %s, error: %v", ctx.Repo.Repository.FullName(), err)
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())

			return
		}
	}

	if req.MergeMessageField != nil && len(strings.TrimSpace(*req.MergeMessageField)) > 0 {
		mergeMessage += "\n\n" + strings.TrimSpace(*req.MergeMessageField)
	}

	if req.MergeWhenChecksSucceed != nil && *req.MergeWhenChecksSucceed {
		_ = pullModel.DeleteScheduledAutoMerge(ctx, pr.ID)
		scheduled, err := automerge.ScheduleAutoMerge(ctx, ctx.Doer, pr, repoModel.MergeStyle(*req.Do), mergeMessage)
		if err != nil {
			log.Error("Unknown error type has occurred while scheduling auto merge pull request: %d in repository %s, error: %v", pr.ID, ctx.Repo.Repository.FullName(), err)
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())

			return
		} else if scheduled {
			ctx.Status(http.StatusOK)
			return
		}
	}

	if req.HeadCommitID == nil {
		var emptyStr string
		req.HeadCommitID = &emptyStr
	}

	// Мердж ветки
	if err := pullService.Merge(ctx, pr, ctx.Doer, ctx.Repo.GitRepo, repoModel.MergeStyle(*req.Do), *req.HeadCommitID, mergeMessage, false); err != nil {
		if models.IsErrInvalidMergeStyle(err) {
			log.Debug("Invalid merge style: %v while merging pull request: %d in repository: %s", err, pr.ID, ctx.Repo.Repository.FullName())
			ctx.JSON(http.StatusBadRequest, apiError.InvalidMergeStyle())

		} else if models.IsErrMergeConflicts(err) {
			log.Debug("Merge conflict has occurred: %v while merging pull request: %d in repository: %s", err, pr.ID, ctx.Repo.Repository.FullName())
			ctx.JSON(http.StatusBadRequest, apiError.MergeConflict())

		} else if models.IsErrRebaseConflicts(err) {
			log.Debug("Rebase conflict has occurred: %v while merging pull request: %d in repository: %s", err, pr.ID, ctx.Repo.Repository.FullName())
			ctx.JSON(http.StatusBadRequest, apiError.RebaseConflict())

		} else if models.IsErrMergeUnrelatedHistories(err) {
			log.Debug("Merge unrelated histories: %v in pull request: %d in repository: %s", err, pr.ID, ctx.Repo.Repository.FullName())
			ctx.JSON(http.StatusBadRequest, apiError.UnrelatedHistories())

		} else if git.IsErrPushOutOfDate(err) {
			log.Debug("Merge unrelated histories because merge push out of date: %v in pull request: %d in repository: %s", err, pr.ID, ctx.Repo.Repository.FullName())
			ctx.JSON(http.StatusBadRequest, apiError.UnrelatedHistories())

		} else if models.IsErrSHADoesNotMatch(err) {
			log.Debug("Can not merge pull request: %d in repository: %s because SHA does not match: %v", err, pr.ID, ctx.Repo.Repository.FullName())
			ctx.JSON(http.StatusBadRequest, apiError.SHADoesNotMatch())

		} else if git.IsErrPushRejected(err) {
			log.Debug("Can not merge pull request: %d in repository: %s because push was rejected: %v", pr.ID, ctx.Repo.Repository.FullName(), err)
			ctx.JSON(http.StatusBadRequest, apiError.MergePushRejected())

		} else {
			log.Error("Unknown error type has occurred while merging pull request: %d in repository %s, error: %v", pr.ID, ctx.Repo.Repository.FullName(), err)
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		}

		return
	}
	log.Debug("Pull request has been merged: %d", pr.ID)

	// Удаление ветки после слияния. Сначала проверяем что на этой ветке нет никаких других не слитых коммитов
	if req.DeleteBranchAfterMerge != nil && *req.DeleteBranchAfterMerge {
		// Don't cleanup when there are other PR's that use this branch as head branch.
		exist, err := issuesModel.HasUnmergedPullRequestsByHeadInfo(ctx, pr.HeadRepoID, pr.HeadBranch)
		if err != nil {
			log.Error("Error has occurred while checkin unmerged pull request: %d by head repo ID: %d in repository: %s, error: %v", pr.ID, pr.HeadRepoID, ctx.Repo.Repository.FullName(), err)
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())

			return
		}
		if exist {
			ctx.Status(http.StatusOK)

			return
		}

		var headRepo *git.Repository
		if ctx.Repo != nil && ctx.Repo.Repository != nil && ctx.Repo.Repository.ID == pr.HeadRepoID && ctx.Repo.GitRepo != nil {
			headRepo = ctx.Repo.GitRepo
		} else {
			headRepo, err = git.OpenRepository(ctx, pr.HeadRepo.OwnerName, pr.HeadRepo.Name, pr.HeadRepo.RepoPath())
			if err != nil {
				log.Error("Error has occurred while opening git repository: %s, error: %v", pr.HeadRepo.RepoPath(), err)
				ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())

				return
			}
			defer headRepo.Close()
		}
		if err := repoService.DeleteBranch(ctx, ctx.Doer, pr.HeadRepo, headRepo, pr.HeadBranch); err != nil {
			switch {
			case git.IsErrBranchNotExist(err):
				log.Debug("Branch: %s not exist in repository: %s", pr.HeadBranch, ctx.Repo.Repository.FullName())
				ctx.JSON(http.StatusBadRequest, apiError.BranchNotExist(pr.HeadBranch))

			case errors.Is(err, repoService.ErrBranchIsDefault):
				log.Debug("Can not delete branch: %s in repository: %s because branch is default", pr.HeadBranch, ctx.Repo.Repository.FullName())
				ctx.JSON(http.StatusBadRequest, apiError.BranchIsDefault(pr.HeadBranch))

			case protected_branch.IsBranchIsProtectedError(err):
				log.Debug("Can not delete branch: %s in repository: %s because branch is protected", pr.HeadBranch, ctx.Repo.Repository.OwnerName, ctx.Repo.Repository.Name)
				ctx.JSON(http.StatusBadRequest, apiError.BranchIsProtected(pr.HeadBranch))

			default:
				log.Error("An error has occurred while try to delete branch: %s in repository: %s, error: %v", pr.HeadBranch, ctx.Repo.Repository.FullName(), err)
				ctx.JSON(http.StatusInternalServerError, apiError.BranchWasNotDeletedInternalServerError)
			}
			return
		}
		if err := issuesModel.AddDeletePRBranchComment(ctx, ctx.Doer, pr.BaseRepo, pr.Issue.ID, pr.HeadBranch); err != nil {
			// Do not fail here as branch has already been deleted
			log.Error("An error has occurred while adding commit after deleting branch: %s in repository: %s, error: %v", pr.HeadBranch, ctx.Repo.Repository.FullName(), err)
		}
	}

	ctx.Status(http.StatusOK)
}
