package role_model

import (
	"errors"
	"fmt"
)

// ErrRoleAlreadyExists представляет собой ошибку типа "ErrRoleAlreadyExists"
type ErrRoleAlreadyExists struct {
	UserID   int64
	TenantID string
	OrgID    int64
	Role     string
}

// IsErrRoleAlreadyExists проверяет, является ли ошибка ErrRoleAlreadyExists.
func IsErrRoleAlreadyExists(err error) bool {
	errRoleAlreadyExists := new(ErrRoleAlreadyExists)
	return errors.As(err, &errRoleAlreadyExists)
}

func (err ErrRoleAlreadyExists) Error() string {
	return fmt.Sprintf("UserId %d already have role '%s' in orgId %d under tenantId: %s", err.UserID, err.Role, err.OrgID, err.TenantID)
}

// ErrNonExistentRole представляет собой ошибку типа "ErrNonExistentRole"
type ErrNonExistentRole struct {
	Role string
}

// IsErrNonExistentRole проверяет, является ли ошибка ErrNonExistentRole.
func IsErrNonExistentRole(err error) bool {
	_, ok := err.(ErrNonExistentRole)
	return ok
}

func (err ErrNonExistentRole) Error() string {
	return fmt.Sprintf("Role %s does not exist", err.Role)
}

// ErrCustomGroupNotFound представляет собой ошибку типа "ErrCustomGroupNotFound"
type ErrCustomGroupNotFound struct {
	Group string
}

// IsErrCustomGroupNotFound проверяет, является ли ошибка ErrCustomGroupNotFound.
func IsErrCustomGroupNotFound(err error) bool {
	errCustomGroupNotFound := new(ErrCustomGroupNotFound)
	return errors.As(err, &errCustomGroupNotFound)
}

func (err ErrCustomGroupNotFound) Error() string {
	return fmt.Sprintf("Custom Group %s not found", err.Group)
}
