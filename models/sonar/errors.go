package sonar

import (
	"errors"
	"fmt"
)

// ErrSonarSettingsAlreadyExists represents a "ErrSonarSettingsAlreadyExists" kind of error
type ErrSonarSettingsAlreadyExists struct {
	SonarProjectKey string
}

// Реализация интерфейса error
func (err ErrSonarSettingsAlreadyExists) Error() string {
	return fmt.Sprintf("Err: sonar settings already exists [sonar_project_key: %s]", err.SonarProjectKey)
}

// Unwrap возвращает базовую ошибку для сравнения через errors.Is
func (err ErrSonarSettingsAlreadyExists) Unwrap() error {
	return errors.New("tenant key is not exists")
}

// IsSonarSettingsAlreadyExists проверяет, является ли ошибка ErrSonarSettingsAlreadyExists
func IsSonarSettingsAlreadyExists(err error) bool {
	return errors.As(err, &ErrSonarSettingsAlreadyExists{})
}

// ErrSonarSettingsNotFound represents a "ErrSonarSettingsNotFound" kind of error
type ErrSonarSettingsNotFound struct {
	SonarProjectKey string
}

// Реализация интерфейса error
func (err ErrSonarSettingsNotFound) Error() string {
	return fmt.Sprintf("Err: sonar settings not found")
}

// Unwrap возвращает базовую ошибку для сравнения через errors.Is
func (err ErrSonarSettingsNotFound) Unwrap() error {
	return errors.New("sonar settings not found")
}

// IsSonarSettingsNotFound проверяет, является ли ошибка ErrSonarSettingsNotFound
func IsSonarSettingsNotFound(err error) bool {
	return errors.As(err, &ErrSonarSettingsNotFound{})
}

// ErrSonarSettingsNotExist represents a "ErrSonarSettingsNotExist" kind of error
type ErrSonarSettingsNotExist struct {
	RepoID int64
}

// Реализация интерфейса error
func (err ErrSonarSettingsNotExist) Error() string {
	return fmt.Sprintf("Err: sonar settings not exist")
}

// Unwrap возвращает базовую ошибку для сравнения через errors.Is
func (err ErrSonarSettingsNotExist) Unwrap() error {
	return errors.New("sonar settings not found")
}

// IsSonarSettingsNotExist проверяет, является ли ошибка ErrSonarSettingsNotExist
func IsSonarSettingsNotExist(err error) bool {
	return errors.As(err, &ErrSonarSettingsNotExist{})
}
