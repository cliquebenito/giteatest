package convert

import (
	"code.gitea.io/gitea/models/organization"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/routers/sbt/logger"
	"code.gitea.io/gitea/routers/sbt/response"
	"code.gitea.io/sdk/gitea"
	"fmt"

	issuesModel "code.gitea.io/gitea/models/issues"
	"code.gitea.io/gitea/models/perm"
	accessModel "code.gitea.io/gitea/models/perm/access"
	userModel "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/git"
)

// ToPullRequest конвертирует model.PullRequest в response.PullRequest
func ToPullRequest(ctx *context.Context, pr *issuesModel.PullRequest, doer *userModel.User, log logger.Logger) (*response.PullRequest, error) {
	var (
		baseBranch *git.Branch
		headBranch *git.Branch
		baseCommit *git.Commit
		err        error
	)

	if err = pr.Issue.LoadRepo(ctx); err != nil {
		return nil, err
	}

	apiIssue := ToIssue(ctx, pr.Issue, log)
	if err := pr.LoadBaseRepo(ctx); err != nil {
		return nil, err
	}

	if err := pr.LoadHeadRepo(ctx); err != nil {
		return nil, err
	}

	p, err := accessModel.GetUserRepoPermission(ctx, pr.BaseRepo, doer)
	if err != nil {
		p.AccessMode = perm.AccessModeNone
	}

	reviewers, err := getRepoReviewers(ctx, pr.IssueID)
	if err != nil {
		return nil, err
	}

	apiPullRequest := &response.PullRequest{
		ID:        pr.ID,
		Index:     pr.Index,
		Poster:    apiIssue.Poster,
		Title:     apiIssue.Title,
		Body:      apiIssue.Body,
		Labels:    apiIssue.Labels,
		Milestone: apiIssue.Milestone,
		Reviewers: reviewers,
		State:     apiIssue.State,
		IsLocked:  apiIssue.IsLocked,
		HasMerged: pr.HasMerged,
		MergeBase: pr.MergeBase,
		Mergeable: pr.Mergeable(),
		Deadline:  apiIssue.Deadline,
		Created:   pr.Issue.CreatedUnix.AsTimePtr(),
		Updated:   pr.Issue.UpdatedUnix.AsTimePtr(),

		AllowMaintainerEdit: pr.AllowMaintainerEdit,

		Base: &response.PRBranchInfo{
			Name:       pr.BaseBranch,
			Ref:        pr.BaseBranch,
			RepoID:     pr.BaseRepoID,
			Repository: ToRepo(ctx, pr.BaseRepo, p.AccessMode, log),
		},
		Head: &response.PRBranchInfo{
			Name:   pr.HeadBranch,
			Ref:    fmt.Sprintf("%s%d/head", git.PullPrefix, pr.Index),
			RepoID: -1,
		},
	}

	if pr.Issue.ClosedUnix != 0 {
		apiPullRequest.Closed = pr.Issue.ClosedUnix.AsTimePtr()
	}

	gitRepo, err := git.OpenRepository(ctx, pr.BaseRepo.OwnerName, pr.BaseRepo.Name, pr.BaseRepo.RepoPath())
	if err != nil {
		return nil, err
	}
	defer gitRepo.Close()

	baseBranch, err = gitRepo.GetBranch(pr.BaseBranch)
	if err != nil && !git.IsErrBranchNotExist(err) {
		return nil, err
	}

	if err == nil {
		baseCommit, err = baseBranch.GetCommit()
		if err != nil && !git.IsErrNotExist(err) {
			return nil, err
		}

		if err == nil {
			apiPullRequest.Base.Sha = baseCommit.ID.String()
		}
	}

	if pr.Flow == issuesModel.PullRequestFlowAGit {
		gitRepo, err := git.OpenRepository(ctx, pr.BaseRepo.OwnerName, pr.BaseRepo.Name, pr.BaseRepo.RepoPath())
		if err != nil {
			return nil, err
		}
		defer gitRepo.Close()

		apiPullRequest.Head.Sha, err = gitRepo.GetRefCommitID(pr.GetGitRefName())
		if err != nil {
			return nil, err
		}
		apiPullRequest.Head.RepoID = pr.BaseRepoID
		apiPullRequest.Head.Repository = apiPullRequest.Base.Repository
		apiPullRequest.Head.Name = ""
	}

	if pr.HeadRepo != nil && pr.Flow == issuesModel.PullRequestFlowGithub {
		p, err := accessModel.GetUserRepoPermission(ctx, pr.HeadRepo, doer)
		if err != nil {
			p.AccessMode = perm.AccessModeNone
		}

		apiPullRequest.Head.RepoID = pr.HeadRepo.ID
		apiPullRequest.Head.Repository = ToRepo(ctx, pr.HeadRepo, p.AccessMode, log)

		headGitRepo, err := git.OpenRepository(ctx, pr.HeadRepo.OwnerName, pr.HeadRepo.Name, pr.HeadRepo.RepoPath())
		if err != nil {
			return nil, err
		}
		defer headGitRepo.Close()

		headBranch, err = headGitRepo.GetBranch(pr.HeadBranch)
		if err != nil && !git.IsErrBranchNotExist(err) {
			return nil, err
		}

		if git.IsErrBranchNotExist(err) {
			headCommitID, err := headGitRepo.GetRefCommitID(apiPullRequest.Head.Ref)
			if err != nil && !git.IsErrNotExist(err) {
				return nil, err
			}
			if err == nil {
				apiPullRequest.Head.Sha = headCommitID
			}
		} else {
			commit, err := headBranch.GetCommit()
			if err != nil && !git.IsErrNotExist(err) {
				return nil, err
			}

			if err == nil {
				apiPullRequest.Head.Ref = pr.HeadBranch
				apiPullRequest.Head.Sha = commit.ID.String()
			}
		}
	}

	if len(apiPullRequest.Head.Sha) == 0 && len(apiPullRequest.Head.Ref) != 0 {
		baseGitRepo, err := git.OpenRepository(ctx, pr.BaseRepo.OwnerName, pr.BaseRepo.Name, pr.BaseRepo.RepoPath())
		if err != nil {
			return nil, err
		}
		defer baseGitRepo.Close()
		refs, err := baseGitRepo.GetRefsFiltered(apiPullRequest.Head.Ref)
		if err != nil {
			return nil, err
		} else if len(refs) > 0 {
			apiPullRequest.Head.Sha = refs[0].Object.String()
		}
	}

	if pr.HasMerged {
		apiPullRequest.Merged = pr.MergedUnix.AsTimePtr()
		apiPullRequest.MergedCommitID = &pr.MergedCommitID
		apiPullRequest.MergedBy = ToUser(ctx, pr.Merger, nil)
	}

	return apiPullRequest, nil
}

// getRepoReviewers Метод возвращает список ревьюеров запроса на слияние
func getRepoReviewers(ctx *context.Context, issueId int64) ([]*response.RepoReviewer, error) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	reviews, err := issuesModel.GetReviewersByIssueID(issueId)
	if err != nil {
		log.Error("Unknown error has occurred while getting reviewer list by issueId: %d, error: %v", issueId, err)
		return nil, err
	}

	if len(reviews) == 0 {
		return nil, nil
	}

	reviewerList := make([]*response.RepoReviewer, 0, len(reviews))
	for _, item := range reviews {
		var newReviewer *response.RepoReviewer

		if item.ReviewerID > 0 {
			if err = item.LoadReviewer(ctx); err != nil {
				if userModel.IsErrUserNotExist(err) {
					continue
				}
				log.Error("Unknown error has occurred while getting reviewer with userId: %d, error: %v", item.ReviewerID, err)
				return nil, err
			}

			newReviewer = &response.RepoReviewer{ReviewerUser: ToUser(ctx, item.Reviewer, ctx.Doer)}
		} else if item.ReviewerTeamID > 0 {
			if err = item.LoadReviewerTeam(ctx); err != nil {
				if organization.IsErrTeamNotExist(err) {
					continue
				}
				log.Error("Unknown error has occurred while getting reviewer with teamId: %d, error: %v", item.ReviewerTeamID, err)
				return nil, err
			}
			newReviewer = &response.RepoReviewer{ReviewerTeam: item.ReviewerTeam}

		} else {
			continue
		}

		switch item.Type {
		case issuesModel.ReviewTypeApprove:
			newReviewer.ReviewState = gitea.ReviewStateApproved
		case issuesModel.ReviewTypeReject:
			newReviewer.ReviewState = gitea.ReviewStateRequestChanges
		case issuesModel.ReviewTypeRequest:
			newReviewer.ReviewState = gitea.ReviewStateRequestReview
		default:
			newReviewer.ReviewState = gitea.ReviewStateUnknown
		}

		reviewerList = append(reviewerList, newReviewer)
	}

	return reviewerList, nil
}
