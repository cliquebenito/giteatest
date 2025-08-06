package project

import (
	"errors"
	"fmt"

	"code.gitea.io/gitea/modules/structs"
)

// ErrProjectNameAlreadyUsed represents a "ErrProjectNameAlreadyUsed" kind of error
type ErrProjectNameAlreadyUsed struct {
	Name string
}

// Реализация интерфейса error
func (err ErrProjectNameAlreadyUsed) Error() string {
	return fmt.Sprintf("project name already used [name: %s]", err.Name)
}

// Unwrap возвращает базовую ошибку для сравнения через errors.Is
func (err ErrProjectNameAlreadyUsed) Unwrap() error {
	return errors.New("project name already used")
}

// IsProjectNameAlreadyUsed проверяет, является ли ошибка ErrProjectNameAlreadyUsed
func IsProjectNameAlreadyUsed(err error) bool {
	return errors.As(err, &ErrProjectNameAlreadyUsed{})
}

// ErrVisibilityIncorrect represents a "ErrVisibilityIncorrect" kind of error
type ErrVisibilityIncorrect struct {
	Visibility structs.VisibleType
}

// Реализация интерфейса error
func (err ErrVisibilityIncorrect) Error() string {
	return fmt.Sprintf("Err: wrong visibility[visibility: %s]", err.Visibility)
}

// Unwrap возвращает базовую ошибку для сравнения через errors.Is
func (err ErrVisibilityIncorrect) Unwrap() error {
	return errors.New("Err: wrong visibility")
}

// IsVisibilityIncorrect проверяет, является ли ошибка ErrVisibilityIncorrect
func IsVisibilityIncorrect(err error) bool {
	return errors.As(err, &ErrVisibilityIncorrect{})
}
