package pulls

import (
	goctx "context"
	"fmt"

	issues_model "code.gitea.io/gitea/models/issues"
	"code.gitea.io/gitea/models/pull_request_sender"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/routers/private/pull_request_task_creator"
)

func (s Server) updatePullRequestStatusWhileMerge(ctx goctx.Context, issue *issues_model.Issue, userName string) error {
	if s.isIssueNone(issue) {
		log.Debug("issue or pull request is empty during an updating issue")
		return nil
	}
	active, err := s.checkStatusOfActivityPr(ctx, issue.PullRequest.ID)
	if err != nil {
		log.Error("Error has occurred while checking for pull request id: %v", err)
		return fmt.Errorf("check unit links for pull request: %w", err)
	}
	if !active {
		log.Debug("There are not active units links for pr in %s", issue.PullRequest.ID)
		return nil
	}

	prID := issue.PullRequest.ID
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
		PullRequestStatus: pull_request_sender.PRStatusMerged,
	}

	log.Debug("try to update status of pull request %d from delete handler", prID, issue.Index)

	if err := s.pullRequestSender.UpdateStatusOfPullRequest(ctx, request); err != nil {
		return fmt.Errorf("update pull request status - merge: %w", err)
	}
	return nil
}

func (s Server) isIssueNone(issue *issues_model.Issue) bool {
	return issue == nil || !issue.IsPull || issue.PullRequest == nil
}

func (s Server) checkStatusOfActivityPr(ctx goctx.Context, prID int64) (bool, error) {
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
