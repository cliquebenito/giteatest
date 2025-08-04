package tenant

import (
	"errors"
	"fmt"
)

// ErrorTenantDoesntExists кастномная ошибка типа
type ErrorTenantDoesntExists struct {
	TenantID string
}

// IsErrorTenantNotExists проверяет, является ли ошибка ErrorTenantDoesntExists
func IsErrorTenantNotExists(err error) bool {
	_, ok := err.(ErrorTenantDoesntExists)
	return ok
}

func (e ErrorTenantDoesntExists) Error() string {
	return fmt.Sprintf("Err: tenant with id %s doesn't exist", e.TenantID)
}

// ErrTenantOrganizationsNotExists представляет собой ошибку типа "ErrTenantOrganizationsNotExists"
type ErrTenantOrganizationsNotExists struct {
	OrgKey     string
	ProjectKey string
}

// IsTenantOrganizationsNotExists проверяет, является ли ошибка ErrOrgKeysByKeysNotExists.
func IsTenantOrganizationsNotExists(err error) bool {
	_, ok := err.(ErrTenantOrganizationsNotExists)
	return ok
}

// реализуем интерфейс error
func (err ErrTenantOrganizationsNotExists) Error() string {
	return fmt.Sprintf("Project does not exists for tenant_key '%s' and project_key '%s'", err.OrgKey, err.ProjectKey)
}

// ErrTenantByKeysNotExists представляет собой ошибку типа "ErrTenantByKeysNotExists"
type ErrTenantByKeysNotExists struct {
	OrgKey     string
	ProjectKey string
}

// IsErrTenantByKeysNotExists проверяет, является ли ошибка ErrTenantByKeysNotExists.
func IsErrTenantByKeysNotExists(err error) bool {
	_, ok := err.(ErrTenantByKeysNotExists)
	return ok
}

func (err ErrTenantByKeysNotExists) Error() string {
	return fmt.Sprintf("Err: tenant does not exists for orgKey '%s' and projectKey '%s'", err.OrgKey, err.ProjectKey)
}

type ErrTenantOrganizationNotExists struct {
	OrgID int64
}

// IsErrTenantOrganizationNotExists проверяет, является ли ошибка ErrTenantOrganizationNotExists.
func IsErrTenantOrganizationNotExists(err error) bool {
	_, ok := err.(ErrTenantOrganizationNotExists)
	return ok
}

func (err ErrTenantOrganizationNotExists) Error() string {
	return fmt.Sprintf("Err: tenant organizations with orgId: %d doesn't exist", err.OrgID)
}

// ErrProjectKeyAlreadyUsed represents a "ErrProjectKeyAlreadyUsed" kind of error
type ErrProjectKeyAlreadyUsed struct {
	ProjectKey string
}

// Реализация интерфейса error
func (err ErrProjectKeyAlreadyUsed) Error() string {
	return fmt.Sprintf("project key already exists [project_key: %s]", err.ProjectKey)
}

// Unwrap возвращает базовую ошибку для сравнения через errors.Is
func (err ErrProjectKeyAlreadyUsed) Unwrap() error {
	return errors.New("user login name already used")
}

// IsProjectKeyAlreadyUsed проверяет, является ли ошибка ErrProjectKeyAlreadyUsed
func IsProjectKeyAlreadyUsed(err error) bool {
	return errors.As(err, &ErrProjectKeyAlreadyUsed{})
}

// ErrTenantNotActive represents a "ErrTenantNotActive" kind of error
type ErrTenantNotActive struct {
	TenantKey string
}

// Реализация интерфейса error
func (err ErrTenantNotActive) Error() string {
	return fmt.Sprintf("Err: tenant key is not active [tenant_key: %s]", err.TenantKey)
}

// Unwrap возвращает базовую ошибку для сравнения через errors.Is
func (err ErrTenantNotActive) Unwrap() error {
	return errors.New("tenant key is not active")
}

// IsTenantNotActive проверяет, является ли ошибка ErrTenantNotActive
func IsTenantNotActive(err error) bool {
	return errors.As(err, &ErrTenantNotActive{})
}

// ErrTenantKeyNotExists represents a "ErrTenantKeyNotExists" kind of error
type ErrTenantKeyNotExists struct {
	TenantKey string
}

// Реализация интерфейса error
func (err ErrTenantKeyNotExists) Error() string {
	return fmt.Sprintf("Err: tenant key is not exists [tenant_key: %s]", err.TenantKey)
}

// Unwrap возвращает базовую ошибку для сравнения через errors.Is
func (err ErrTenantKeyNotExists) Unwrap() error {
	return errors.New("tenant key is not exists")
}

// IsTenantKeyNotExists проверяет, является ли ошибка ErrTenantKeyNotExists
func IsTenantKeyNotExists(err error) bool {
	return errors.As(err, &ErrTenantKeyNotExists{})
}
