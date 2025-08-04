package pulls

import (
	goctx "context"
	"fmt"

	issues_model "code.gitea.io/gitea/models/issues"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/routers/private/pull_request_reader"
	"code.gitea.io/gitea/routers/private/unit_linker"
)

func (s Server) linkUnitsFromIssue(ctx goctx.Context, issue *issues_model.Issue, userName string) error {
	if s.isIssueNone(issue) {
		return nil
	}

	prID := issue.PullRequest.ID

	request := unit_linker.PullRequestLinkRequest{
		UserName:          userName,
		BranchName:        issue.PullRequest.HeadBranch,
		PullRequestID:     prID,
		PullRequestStatus: pull_request_reader.MergedPullRequestStatus,
	}

	log.Debug("try to link pull request %d from pulls create/title handler, request: %v", request)

	if err := s.unitLinker.LinkPullRequest(ctx, request); err != nil {
		return fmt.Errorf("link pull request: %w", err)
	}

	return nil
}
