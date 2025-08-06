//go:build !correct

package tenant

import (
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/tenant"
	"code.gitea.io/gitea/models/unittest"
	"code.gitea.io/gitea/modules/timeutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"testing"
)

// TestGetTenants тесты для вставки, получения и удаления tenant или tenants
func TestGetTenants(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	ten := &tenant.ScTenant{
		ID:        uuid.NewString(),
		Name:      "test",
		Default:   false,
		IsActive:  true,
		CreatedAt: timeutil.TimeStampNow(),
		UpdatedAt: timeutil.TimeStampNow(),
	}
	err := AddTenant(db.DefaultContext, ten)
	assert.NoError(t, err)

	tenants, err := GetTenants(db.DefaultContext)
	assert.NoError(t, err)
	assert.Equal(t, tenants[0].ID, ten.ID)
	assert.Equal(t, tenants[0].Name, ten.Name)
	assert.Equal(t, tenants[0].IsActive, ten.IsActive)
	assert.Equal(t, tenants[0].Default, ten.Default)

	tenantGet, err := TenantByID(db.DefaultContext, ten.ID)
	assert.NoError(t, err)
	assert.Equal(t, tenantGet.ID, ten.ID)
	assert.Equal(t, tenantGet.Name, ten.Name)
	assert.Equal(t, tenantGet.IsActive, ten.IsActive)
	assert.Equal(t, tenantGet.Default, ten.Default)

	tenEdit := &tenant.ScTenant{
		Name:      "test2",
		IsActive:  true,
		UpdatedAt: timeutil.TimeStampNow(),
	}
	err = UpdateTenant(db.DefaultContext, tenEdit)
	assert.NoError(t, err)

	err = RemoveTenantByID(db.DefaultContext, ten.ID)
	assert.NoError(t, err)
}
