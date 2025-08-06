package repo

import (
	"errors"
	"fmt"
)

// ErrorRepoKeyDoesntExists кастомная ошибка типа
type ErrorRepoKeyDoesntExists struct {
	RepoKey string
	RepoID  string
}

func (e ErrorRepoKeyDoesntExists) Error() string {
	return fmt.Sprintf("Repo key with key %s and id %s doesn't exist", e.RepoKey, e.RepoID)
}

func IsErrorRepoKeyDoesntExists(err error) bool {
	return errors.As(err, &ErrorRepoKeyDoesntExists{})
}

// ErrorOrgDoestExist кастомная ошибка типа
type ErrorOrgDoestExist struct {
	ProjectKey string
	TenantKey  string
}

func (e ErrorOrgDoestExist) Error() string {
	return fmt.Sprintf("Err: project does not exist for tenant_key %s and project_key %s", e.TenantKey, e.ProjectKey)
}

func IsErrorOrgDoestExist(err error) bool {
	return errors.As(err, &ErrorRepoKeyDoesntExists{})
}
