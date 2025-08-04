package pull_request_reader

import (
	"context"
	"fmt"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/issues"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/modules/git"
)

type PullRequestStatus int

const (
	MergedPullRequestStatus   PullRequestStatus = iota
	UnmergedPullRequestStatus PullRequestStatus = iota
)

type PullRequestHeader struct {
	PullRequestURL  string
	PullRequestName string
	BranchName      string
	CommitNames     []string
}

type PullRequestReader struct {
	db db.Engine
}

func NewReader(db db.Engine) PullRequestReader {
	return PullRequestReader{db: db}
}

// ReadByID метод собирает информацию о МР по идентификатору
func (p PullRequestReader) ReadByID(
	ctx context.Context,
	pullRequestID int64,
	PullRequestStatus PullRequestStatus,
) (PullRequestHeader, error) {
	pullRequest, err := p.readPullRequest(ctx, pullRequestID)
	if err != nil {
		return PullRequestHeader{}, fmt.Errorf("read pull request: %w", err)
	}

	if pullRequest.Issue == nil || len(pullRequest.Issue.Title) == 0 {
		return PullRequestHeader{}, fmt.Errorf("empty repo title")
	}

	var isPullRequestMerged bool
	if PullRequestStatus == MergedPullRequestStatus {
		isPullRequestMerged = true
	}

	commits, err := p.readCommits(ctx, pullRequest, isPullRequestMerged)
	if err != nil {
		return PullRequestHeader{}, fmt.Errorf("read commits: %w", err)
	}

	repoID := pullRequest.Issue.RepoID

	repo, err := repo_model.GetRepositoryByID(ctx, repoID)
	if err != nil {
		return PullRequestHeader{}, fmt.Errorf("get repo by id: %w", err)
	}

	url := fmt.Sprintf(
		"/%s/%s/pulls/%d",
		repo.OwnerName,
		repo.LowerName,
		pullRequest.Issue.Index,
	)

	prHeader := PullRequestHeader{
		CommitNames:     commits,
		PullRequestURL:  url,
		PullRequestName: pullRequest.Issue.Title,
	}

	return prHeader, nil
}

func (p PullRequestReader) readPullRequest(ctx context.Context, pullRequestID int64) (*issues.PullRequest, error) {
	pullRequest, err := issues.GetPullRequestByID(ctx, pullRequestID)
	if err != nil {
		return nil, fmt.Errorf("get pull request by id: %w", err)
	}

	if err = pullRequest.LoadBaseRepo(ctx); err != nil {
		return nil, fmt.Errorf("load base repo: %w", err)
	}

	if err = pullRequest.LoadHeadRepo(ctx); err != nil {
		return nil, fmt.Errorf("load head repo: %w", err)
	}

	if err = pullRequest.LoadIssue(ctx); err != nil {
		return nil, fmt.Errorf("load issue: %w", err)
	}

	return pullRequest, nil
}

func (p PullRequestReader) readCommits(
	ctx context.Context,
	pullRequest *issues.PullRequest,
	isPullRequestMerged bool,
) ([]string, error) {
	baseGitRepo, err := git.OpenRepository(ctx, pullRequest.BaseRepo.OwnerName, pullRequest.BaseRepo.Name, pullRequest.BaseRepo.RepoPath())
	if err != nil {
		return nil, fmt.Errorf("open repo: %w", err)
	}

	baseBranch := pullRequest.BaseBranch

	if isPullRequestMerged {
		baseBranch = pullRequest.MergeBase
	}

	repoPath := pullRequest.BaseRepo.RepoPath()
	headBranch := pullRequest.GetGitRefName()

	const (
		directComp = false
		fileOnly   = false
	)

	prInfo, err := baseGitRepo.GetCompareInfo(repoPath, baseBranch, headBranch, directComp, fileOnly)
	if err != nil {
		return nil, fmt.Errorf("get compare info: %w", err)
	}

	var commits []string
	for _, commit := range prInfo.Commits {
		commits = append(commits, commit.CommitMessage)
	}

	return commits, nil
}
