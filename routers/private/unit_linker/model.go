package unit_linker

import (
	"fmt"

	"code.gitea.io/gitea/routers/private/pull_request_reader"
)

type PullRequestLinkRequest struct {
	BranchName        string
	PullRequestID     int64
	PullRequestStatus pull_request_reader.PullRequestStatus
	UserName          string
}

func (p PullRequestLinkRequest) Validate() error {
	if p.PullRequestID < 1 {
		return fmt.Errorf("pull request ID should be a positive number")
	}

	if len(p.BranchName) == 0 {
		return fmt.Errorf("branch name is required")
	}

	if len(p.UserName) == 0 {
		return fmt.Errorf("user name is required")
	}

	return nil
}

type BranchLinkRequest struct {
	BranchName string
}
