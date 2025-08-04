//go:build !correct

package tenant

import (
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/tenant"
	"code.gitea.io/gitea/models/unittest"
	"code.gitea.io/gitea/services/forms"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"testing"
)

// TestGetTenantProjects тесты для создание, получения и удаления связи tenant c organizations
func TestGetTenantProjects(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	form := &forms.CreateTenantOrganizationForm{
		TenantID:       uuid.NewString(),
		OrganizationID: int64(1),
	}
	tenProject := &tenant.ScTenantOrganizations{
		ID:             uuid.NewString(),
		OrganizationID: form.OrganizationID,
		TenantID:       form.TenantID,
	}
	err := CreateRelationTenantOrganization(db.DefaultContext, tenProject)
	assert.NoError(t, err)
	tenantProjects, err := TenantOrganizations(db.DefaultContext, tenProject.TenantID)
	assert.NoError(t, err)
	assert.Equal(t, tenantProjects[0].TenantID, tenProject.TenantID)
	assert.Equal(t, tenantProjects[0].OrganizationID, tenProject.OrganizationID)
	assert.Equal(t, tenantProjects[0].ID, tenProject.ID)

	err = RemoveTenantOrganization(db.DefaultContext, tenProject.TenantID, tenProject.OrganizationID)
	assert.NoError(t, err)
}
