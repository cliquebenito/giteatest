package models

import (
	"errors"
	"fmt"
)

// ================= Protected Branch Errors =================

// ErrProtectedBranchBranchNameTooLong is returned when a branch name exceeds 255 characters.
type ErrProtectedBranchBranchNameTooLong struct {
	BranchName string
}

func NewErrProtectedBranchBranchNameTooLong(branchName string) ErrProtectedBranchBranchNameTooLong {
	return ErrProtectedBranchBranchNameTooLong{BranchName: branchName}
}

func (e ErrProtectedBranchBranchNameTooLong) Error() string {
	return fmt.Sprintf("branch name '%s' must be shorter than 255 characters", e.BranchName)
}

func IsErrProtectedBranchBranchNameTooLong(err error) bool {
	var target *ErrProtectedBranchBranchNameTooLong
	return errors.As(err, &target)
}

// ErrProtectedBranchBranchNameEmpty is returned when a branch name is empty.
type ErrProtectedBranchBranchNameEmpty struct{}

func NewErrProtectedBranchBranchNameEmpty() ErrProtectedBranchBranchNameEmpty {
	return ErrProtectedBranchBranchNameEmpty{}
}

func (e ErrProtectedBranchBranchNameEmpty) Error() string {
	return "branch name must be non-empty"
}

func IsErrProtectedBranchBranchNameEmpty(err error) bool {
	var target *ErrProtectedBranchBranchNameEmpty
	return errors.As(err, &target)
}

// ErrPushWhitelistRequired is returned when push whitelist is required but not provided.
type ErrPushWhitelistRequired struct{}

func NewErrPushWhitelistRequired() ErrPushWhitelistRequired {
	return ErrPushWhitelistRequired{}
}

func (e ErrPushWhitelistRequired) Error() string {
	return "push_whitelist_usernames must be non-nil when require_push_whitelist is true"
}

func IsErrPushWhitelistRequired(err error) bool {
	var target *ErrPushWhitelistRequired
	return errors.As(err, &target)
}

// ErrPushWhitelistUnexpected is returned when push whitelist is provided but not required.
type ErrPushWhitelistUnexpected struct{}

func NewErrPushWhitelistUnexpected() ErrPushWhitelistUnexpected {
	return ErrPushWhitelistUnexpected{}
}

func (e ErrPushWhitelistUnexpected) Error() string {
	return "push_whitelist_usernames must be nil when require_push_whitelist is false"
}

func IsErrPushWhitelistUnexpected(err error) bool {
	var target *ErrPushWhitelistUnexpected
	return errors.As(err, &target)
}

// ErrForcePushWhitelistRequired is returned when force-push whitelist is required but not provided.
type ErrForcePushWhitelistRequired struct{}

func NewErrForcePushWhitelistRequired() ErrForcePushWhitelistRequired {
	return ErrForcePushWhitelistRequired{}
}

func (e ErrForcePushWhitelistRequired) Error() string {
	return "force_push_whitelist_usernames must be non-nil when require_force_push_whitelist is true"
}

func IsErrForcePushWhitelistRequired(err error) bool {
	var target *ErrForcePushWhitelistRequired
	return errors.As(err, &target)
}

// ErrForcePushWhitelistUnexpected is returned when force-push whitelist is provided but not required.
type ErrForcePushWhitelistUnexpected struct{}

func NewErrForcePushWhitelistUnexpected() ErrForcePushWhitelistUnexpected {
	return ErrForcePushWhitelistUnexpected{}
}

func (e ErrForcePushWhitelistUnexpected) Error() string {
	return "force_push_whitelist_usernames must be nil when require_force_push_whitelist is false"
}

func IsErrForcePushWhitelistUnexpected(err error) bool {
	var target *ErrForcePushWhitelistUnexpected
	return errors.As(err, &target)
}

// ErrDeleteWhitelistRequired is returned when delete whitelist is required but not provided.
type ErrDeleteWhitelistRequired struct{}

func NewErrDeleteWhitelistRequired() ErrDeleteWhitelistRequired {
	return ErrDeleteWhitelistRequired{}
}

func (e ErrDeleteWhitelistRequired) Error() string {
	return "delete_whitelist_usernames must be non-nil when require_delete_whitelist is true"
}

func IsErrDeleteWhitelistRequired(err error) bool {
	var target *ErrDeleteWhitelistRequired
	return errors.As(err, &target)
}

// ErrDeleteWhitelistUnexpected is returned when delete whitelist is provided but not required.
type ErrDeleteWhitelistUnexpected struct{}

func NewErrDeleteWhitelistUnexpected() ErrDeleteWhitelistUnexpected {
	return ErrDeleteWhitelistUnexpected{}
}

func (e ErrDeleteWhitelistUnexpected) Error() string {
	return "delete_whitelist_usernames must be nil when require_delete_whitelist is false"
}

func IsErrDeleteWhitelistUnexpected(err error) bool {
	var target *ErrDeleteWhitelistUnexpected
	return errors.As(err, &target)
}

// ==========================================================
