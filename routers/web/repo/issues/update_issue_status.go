package issues

import (
	goctx "context"
	"fmt"

	"golang.org/x/net/context"

	issues_model "code.gitea.io/gitea/models/issues"
	"code.gitea.io/gitea/models/pull_request_sender"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/routers/private/pull_request_task_creator"
)

func (s Server) updateIssueStatus(ctx goctx.Context, issue *issues_model.Issue, userName string) error {
	if s.isIssueNone(issue) {
		log.Debug("issue or pull request is empty during an updating pr")
		return nil
	}

	active, err := s.checkStatusOfActivityPullRequest(ctx, issue.PullRequest.ID)
	if err != nil {
		log.Error("Error has occurred while checking for pull request id: %v", err)
		return fmt.Errorf("check unit links for pull request: %w", err)
	}
	if !active {
		log.Debug("There are not active units links for pr in %s", issue.PullRequest.ID)
		return nil
	}

	prID := issue.PullRequest.ID

	pullRequestStatus := pull_request_sender.PRStatusOpen

	if issue.IsClosed {
		pullRequestStatus = pull_request_sender.PRStatusClosed
	}

	prURL := fmt.Sprintf(
		"/%s/%s/pulls/%d",
		issue.Repo.OwnerName,
		issue.Repo.LowerName,
		issue.Index,
	)

	request := pull_request_task_creator.PullRequestUpdateStatus{
		UserName:          userName,
		PullRequestID:     prID,
		PullRequestURL:    prURL,
		PullRequestStatus: pullRequestStatus,
	}

	log.Debug("try to update status of pull request %v", prID)

	if err := s.pullRequestSender.UpdateStatusOfPullRequest(ctx, request); err != nil {
		log.Error("Error has occurred while updating pull request status %v", prID, err)
		return fmt.Errorf("update pull request status: %w", err)
	}

	log.Info("update status of pull request %v: success", prID)

	return nil
}

func (s Server) isIssueNone(issue *issues_model.Issue) bool {
	return issue == nil || !issue.IsPull || issue.PullRequest == nil
}

func (s Server) checkStatusOfActivityPullRequest(ctx context.Context, prID int64) (bool, error) {
	active, err := s.pullRequestSender.IsActiveOfPullRequestStatus(ctx, prID)
	if err != nil {
		log.Error("Error has occurred while getting unit links: %v", err)
		return false, fmt.Errorf("checking activity of a pr status: %w", err)
	}
	if !active {
		return false, nil
	}
	return true, nil
}
