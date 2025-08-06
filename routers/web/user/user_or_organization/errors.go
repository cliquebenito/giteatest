package user_or_organization

import "fmt"

type InternalServerError struct {
	err error
}

func NewInternalServerError(err error) *InternalServerError {
	return &InternalServerError{err: err}
}

func (e *InternalServerError) Error() string {
	return fmt.Sprintf("internal server error: %s", e.err.Error())
}

func (e *InternalServerError) Unwrap() error {
	return e.err
}
