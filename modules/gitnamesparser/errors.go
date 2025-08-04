package gitnamesparser

import "fmt"

type UnitCodeNotFoundError struct {
	RawName string
}

func NewUnitCodeNotFoundError(rawName string) *UnitCodeNotFoundError {
	return &UnitCodeNotFoundError{RawName: rawName}
}

func (e *UnitCodeNotFoundError) Error() string {
	return fmt.Sprintf("unit '%s' not found", e.RawName)
}

type EmptyCommitLinksError struct{}

func NewEmptyCommitLinksError() *EmptyCommitLinksError {
	return &EmptyCommitLinksError{}
}

func (e *EmptyCommitLinksError) Error() string {
	return fmt.Sprintf("empty commit links")
}
