package internal_metric_counter_db

import "fmt"

type CodeHubCounterDoesntExistsError struct {
	RepoID int64
}

func NewCodeHubCounterDoesntExistsError(repoID int64) *CodeHubCounterDoesntExistsError {
	return &CodeHubCounterDoesntExistsError{RepoID: repoID}
}

func (e CodeHubCounterDoesntExistsError) Error() string {
	return fmt.Sprintf("counter for repo '%d' doesn't exist", e.RepoID)
}
