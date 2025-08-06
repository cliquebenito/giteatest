package iamtokenparser

import "fmt"

type ErrorIAMTenantNotFound struct {
	error
}

func NewErrorIAMTenantNotFound(err error) *ErrorIAMTenantNotFound {
	return &ErrorIAMTenantNotFound{error: err}
}

func (e *ErrorIAMTenantNotFound) Error() string {
	return fmt.Sprintf("tenant from token not found: %s", e.error.Error())
}

func (e *ErrorIAMTenantNotFound) Unwrap() error {
	return e.error
}

type ErrorIAMClaimNotExists struct {
	error
}

func NewErrorIAMClaimNotExists(err error) *ErrorIAMClaimNotExists {
	return &ErrorIAMClaimNotExists{error: err}
}

func (e *ErrorIAMClaimNotExists) Error() string {
	return e.error.Error()
}

func (e *ErrorIAMClaimNotExists) Unwrap() error {
	return e.error
}
