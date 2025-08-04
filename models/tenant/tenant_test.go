//go:build !correct

package tenant

import (
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/modules/timeutil"
	"fmt"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"testing"
)

// TestInsertTenant тестируем вставку, получение tenant
func TestInsertTenant(t *testing.T) {
	tenant := &ScTenant{
		ID:        uuid.NewString(),
		Name:      "test",
		IsActive:  false,
		Default:   false,
		CreatedAt: timeutil.TimeStampNow(),
		UpdatedAt: timeutil.TimeStampNow(),
	}
	_, err := InsertTenant(db.DefaultContext, tenant)
	assert.NoError(t, err, "tenant_model.InsertTenant failed")
	tenants, err := GetTenants(db.DefaultContext)
	if assert.Len(t, tenants, 1) {
		assert.False(t, tenants[0].Default)
		assert.False(t, tenants[0].IsActive)
		assert.Equal(t, tenants[0].Name, "test")
		assert.NotEqual(t, tenants[0].ID, "")
	}
}

// TestUpdateTenant тестируем обновление информации о tenant
func TestUpdateTenant(t *testing.T) {
	tenant := &ScTenant{
		ID:        uuid.NewString(),
		Name:      "test",
		IsActive:  true,
		Default:   false,
		CreatedAt: timeutil.TimeStampNow(),
		UpdatedAt: timeutil.TimeStampNow(),
	}
	_, err := InsertTenant(db.DefaultContext, tenant)
	assert.NoError(t, err, "tenant_model.InsertTenant failed")
	tenantEdit := &ScTenant{
		ID:        tenant.ID,
		Name:      "test2",
		IsActive:  true,
		Default:   true,
		UpdatedAt: timeutil.TimeStampNow(),
	}
	err = UpdateTenant(db.DefaultContext, tenantEdit)
	assert.NoError(t, err, fmt.Sprintf("failed UpdateTenant: %v", err))
	tenant, err = GetTenantByID(db.DefaultContext, tenant.ID)
	assert.Equal(t, tenant.Name, "test2")
	assert.Equal(t, tenant.IsActive, true)
	assert.Equal(t, tenant.Default, false)
}

// TestDeleteTenant тестируем удаление tenant
func TestDeleteTenant(t *testing.T) {
	tenant := &ScTenant{
		ID:        uuid.NewString(),
		Name:      "test",
		IsActive:  true,
		Default:   false,
		CreatedAt: timeutil.TimeStampNow(),
		UpdatedAt: timeutil.TimeStampNow(),
	}
	_, err := InsertTenant(db.DefaultContext, tenant)
	assert.NoError(t, err, "tenant_model.InsertTenant failed")
	err = DeleteTenant(db.DefaultContext, tenant.ID, []int64{})
	assert.NoError(t, err, fmt.Sprintf("failed DeleteTenant: %v", err))
	_, err = GetTenantByID(db.DefaultContext, tenant.ID)
	assert.Error(t, err)
}
