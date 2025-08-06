//go:build !correct

package tenant

import (
	"code.gitea.io/gitea/models/db"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"testing"
)

// TestGetTenantOrganizations тестируем получение organizations для конкретного tenant
func TestGetTenantOrganizations(t *testing.T) {
	tenantOrganization := &ScTenantOrganizations{
		ID:             uuid.NewString(),
		TenantID:       uuid.NewString(),
		OrganizationID: int64(1),
	}
	err := InsertTenantOrganization(db.DefaultContext, tenantOrganization)
	assert.NoError(t, err)
	tenProject, err := GetTenantOrganizations(db.DefaultContext, tenantOrganization.TenantID)
	assert.NoError(t, err)
	assert.Equal(t, tenProject[0].OrganizationID, tenantOrganization.OrganizationID)
	assert.Equal(t, tenProject[0].TenantID, tenantOrganization.TenantID)
	assert.Equal(t, tenProject[0].ID, tenantOrganization.ID)
}

// TestDeleteTenantOrganization тестируем вставку и удаление связи tenant c organization
func TestDeleteTenantOrganization(t *testing.T) {
	tenantOrganization := &ScTenantOrganizations{
		ID:             uuid.NewString(),
		TenantID:       uuid.NewString(),
		OrganizationID: int64(1),
	}
	err := InsertTenantOrganization(db.DefaultContext, tenantOrganization)
	assert.NoError(t, err)
	err = DeleteTenantOrganization(db.DefaultContext, tenantOrganization.TenantID, []int64{tenantOrganization.OrganizationID})
	assert.NoError(t, err)
}
