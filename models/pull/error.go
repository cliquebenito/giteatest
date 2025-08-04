package pull

import (
	"errors"
	"fmt"
)

// ErrMergeProcessNotExist will be merge process not exist for this push
type ErrMergeProcessNotExist struct {
	RepoId     int64
	UserId     int64
	BaseBranch string
}

// IsErrMergeProcessNotExist checks if an error is a ErrMergeProcessNotExist.
func IsErrMergeProcessNotExist(err error) bool {
	errMergeProcessNotExist := new(ErrMergeProcessNotExist)
	return errors.As(err, &errMergeProcessNotExist)
}

// Error returns the error message
func (err ErrMergeProcessNotExist) Error() string {
	return fmt.Sprintf("Merge process not exist for user %d push to branch %s in repository %d", err.UserId, err.BaseBranch, err.RepoId)
}
