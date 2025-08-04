package external_metric_counter_db

import (
	"errors"
	"fmt"
)

type ExternalMetricCounterDoesntExistsError struct {
	RepoID int64
}

func NewExternalMetricCounterDoesntExistsError(repoID int64) *ExternalMetricCounterDoesntExistsError {
	return &ExternalMetricCounterDoesntExistsError{RepoID: repoID}
}

func IsErrExternalMetricCounterDoesntExists(err error) bool {
	newErr := new(ExternalMetricCounterDoesntExistsError)
	return errors.As(err, &newErr)
}

func (e ExternalMetricCounterDoesntExistsError) Error() string {
	return fmt.Sprintf("counter for repo '%d' doesn't exist", e.RepoID)
}
