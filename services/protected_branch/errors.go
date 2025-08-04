package protected_brancher

import (
	"errors"
	"fmt"
)

// ProtectedBranchNotFoundError is returned when a protected branch is not found.
type ProtectedBranchNotFoundError struct{}

func NewProtectedBranchNotFoundError() *ProtectedBranchNotFoundError {
	return &ProtectedBranchNotFoundError{}
}

func (e *ProtectedBranchNotFoundError) Error() string {
	return "protected branch not found"
}

func IsProtectedBranchNotFoundError(err error) bool {
	var target *ProtectedBranchNotFoundError
	return errors.As(err, &target)
}

// ProtectedBranchAlreadyExistError is returned when a protected branch already exists.
type ProtectedBranchAlreadyExistError struct {
	BranchName string
}

func NewProtectedBranchAlreadyExistError(branchName string) *ProtectedBranchAlreadyExistError {
	return &ProtectedBranchAlreadyExistError{BranchName: branchName}
}

func (e *ProtectedBranchAlreadyExistError) Error() string {
	return fmt.Sprintf("protected branch '%s' already exists", e.BranchName)
}

func IsProtectedBranchAlreadyExistError(err error) bool {
	var target *ProtectedBranchAlreadyExistError
	return errors.As(err, &target)
}
