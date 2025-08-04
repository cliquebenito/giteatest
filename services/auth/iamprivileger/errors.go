package iamprivileger

import "fmt"

type ErrorTenantNotFound struct {
	TenantName string

	error
}

func NewErrTenantNotFound(tenantName string, err error) error {
	return &ErrorTenantNotFound{TenantName: tenantName, error: err}
}

func (e *ErrorTenantNotFound) Error() string {
	return fmt.Sprintf("tenant %s not found: %s", e.TenantName, e.error.Error())
}

type ErrorOrganizationNotFound struct {
	OrganizationName string

	error
}

func NewErrorOrganizationNotFound(organizationName string, err error) error {
	return &ErrorOrganizationNotFound{OrganizationName: organizationName, error: err}
}

func (e *ErrorOrganizationNotFound) Error() string {
	return fmt.Sprintf("organization %s not found: %s", e.OrganizationName, e.error.Error())
}
