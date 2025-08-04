package review_settings_db

import (
	"errors"
	"fmt"
)

type ReviewSettingsDoesntExistsError struct {
	BranchName string
	RepoID     int64
}

func NewReviewSettingsDoesntExistsError(repoID int64, branchName string) *ReviewSettingsDoesntExistsError {
	return &ReviewSettingsDoesntExistsError{RepoID: repoID, BranchName: branchName}
}

func IsErrReviewSettingsDoesntExistsError(err error) bool {
	newErr := new(ReviewSettingsDoesntExistsError)
	return errors.As(err, &newErr)
}

func (e ReviewSettingsDoesntExistsError) Error() string {
	return fmt.Sprintf("review setting for repo '%d' with branch name '%s' doesn't exist", e.RepoID, e.BranchName)
}
