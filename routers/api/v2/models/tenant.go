package models

import (
	"fmt"
	"regexp"
	"strings"
)

var tenantNameRegex = regexp.MustCompile("^[a-zA-Z0-9_.-]+$")

// Tenant model for API v2 response
// swagger:response tenantGetResponse
type TenantGetResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	IsActive  bool   `json:"is_active"`
	TenantKey string `json:"tenant_key"`
}

// Tenant model for API v2 response
// swagger:response tenantPostResponse
type TenantPostResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	TenantKey string `json:"tenant_key"`
}

// CreateTenantOptions options to create tenant
type CreateTenantOptions struct {
	TenantKey string `json:"tenant_key" binding:"Required;MaxSize(50)"`
	Name      string `json:"name" binding:"Required;MaxSize(50)"`
}

func (o *CreateTenantOptions) Validate() error {
	if !tenantNameRegex.MatchString(o.Name) {
		return fmt.Errorf("tenant name is not valid")
	}
	if strings.Contains(o.TenantKey, " ") {
		return fmt.Errorf("tenant key is not valid")
	}
	return nil
}
