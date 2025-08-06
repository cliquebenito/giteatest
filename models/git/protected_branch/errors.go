package protected_branch

import (
	"errors"
	"fmt"
)

// BranchIsProtectedError is returned when attempting to modify a protected branch.
type BranchIsProtectedError struct {
	BranchName string
}

func NewBranchIsProtectedError(branchName string) *BranchIsProtectedError {
	return &BranchIsProtectedError{BranchName: branchName}
}

func (e *BranchIsProtectedError) Error() string {
	return fmt.Sprintf("branch '%s' is protected", e.BranchName)
}

// Check error is BranchIsProtectedError
func IsBranchIsProtectedError(err error) bool {
	var branchError *BranchIsProtectedError
	return errors.As(err, &branchError)
}
