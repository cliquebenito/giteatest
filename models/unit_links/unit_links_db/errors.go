package unit_links_db

import "fmt"

type PullRequestNotFoundError struct {
	PullRequestID int64
}

func NewPullRequestNotFoundError(pullRequestID int64) *PullRequestNotFoundError {
	return &PullRequestNotFoundError{PullRequestID: pullRequestID}
}

func (e *PullRequestNotFoundError) Error() string {
	return fmt.Sprintf("pull request '%d' not found", e.PullRequestID)
}
