package pull_request_task_creator

import (
	"fmt"

	"code.gitea.io/gitea/models/pull_request_sender"
)

// PullRequestUpdateStatus отправка статуса об обновлении pr
type PullRequestUpdateStatus struct {
	PullRequestID     int64
	UserName          string
	PullRequestURL    string
	PullRequestStatus pull_request_sender.FromUnitStatusPr
}

// Validate валидация request
func (p PullRequestUpdateStatus) Validate() error {

	if p.PullRequestID < 1 {
		return fmt.Errorf("pull request ID should be a positive number")
	}

	if len(p.UserName) == 0 {
		return fmt.Errorf("user name is required")
	}

	return nil
}
