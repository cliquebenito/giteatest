package issues

import (
	goctx "context"
	"fmt"

	issues_model "code.gitea.io/gitea/models/issues"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/routers/private/pull_request_reader"
	"code.gitea.io/gitea/routers/private/unit_linker"
)

func (s Server) unlinkUnitsFromIssue(ctx goctx.Context, issue *issues_model.Issue, userName string) error {
	if s.isIssueNone(issue) {
		log.Debug("issue or pull request is empty during an adding pr")
		return nil
	}

	prID := issue.PullRequest.ID

	request := unit_linker.PullRequestLinkRequest{
		UserName:          userName,
		BranchName:        issue.PullRequest.HeadBranch,
		PullRequestID:     prID,
		PullRequestStatus: pull_request_reader.MergedPullRequestStatus,
	}

	log.Debug("try to unlink pull request %d from delete handler", prID, issue.Index)

	if err := s.unitLinker.UnlinkPullRequest(ctx, request); err != nil {
		log.Error("Error has occurred while unlink pull request %d from delete handler", prID, err)
		return fmt.Errorf("link pull request: %w", err)
	}

	return nil
}
