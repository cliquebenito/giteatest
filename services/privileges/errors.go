package privileges

import (
	"errors"
	"fmt"
)

// ErrProjectNameAlreadyUsed represents a "ErrProjectNameAlreadyUsed" kind of error
type ErrWrongPrivelegeGroup struct {
	Name string
}

// Реализация интерфейса error
func (err ErrWrongPrivelegeGroup) Error() string {
	return fmt.Sprintf("Err: role not exists")
}

// Unwrap возвращает базовую ошибку для сравнения через errors.Is
func (err ErrWrongPrivelegeGroup) Unwrap() error {
	return errors.New("Err: wrong privelege_group")
}

// IsProjectNameAlreadyUsed проверяет, является ли ошибка ErrProjectNameAlreadyUsed
func IsProjectNameAlreadyUsed(err error) bool {
	return errors.As(err, &ErrWrongPrivelegeGroup{})
}
