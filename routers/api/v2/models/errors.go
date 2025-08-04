package models

import (
	"errors"
	"fmt"
)

// ErrInvalidHookType - ошибка при проверке хука на корректный тип
type ErrInvalidHookType struct {
	HookType string
}

func (err ErrInvalidHookType) Error() string {
	return fmt.Sprintf("Err: hook type is invalid [type: %s]", err.HookType)

}
func IsErrInvalidHookType(err error) bool {
	return errors.As(err, &ErrInvalidHookType{})
}

type ErrInvalidHookContentType struct {
	HookContentType string
}

func (err ErrInvalidHookContentType) Error() string {
	return fmt.Sprintf("Err: content type is invalid [type: %s]", err.HookContentType)

}
func IsErrInvalidHookContentType(err error) bool {
	return errors.As(err, &ErrInvalidHookContentType{})
}

type ErrInvalidHookID struct {
	HookID string
}

func (err ErrInvalidHookID) Error() string {
	return fmt.Sprintf("Err: incorrect hook [id: %s]", err.HookID)
}
func IsErrInvalidHookID(err error) bool {
	return errors.As(err, &ErrInvalidHookID{})
}
