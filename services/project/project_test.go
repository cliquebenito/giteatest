//go:build !correct

package project

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/organization"
	"code.gitea.io/gitea/models/tenant"
	"code.gitea.io/gitea/models/unittest"
	"code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/services/forms"
)

func TestMain(m *testing.M) {
	unittest.MainTest(m, &unittest.TestOptions{
		GiteaRootPath: filepath.Join("..", ".."),
	})
}
func TestCreateProject_Success(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	tenants := &tenant.ScTenant{
		ID:       "test_tenant_id",
		OrgKey:   "test_tenant_key",
		IsActive: true,
	}
	_, err := db.GetEngine(db.DefaultContext).Insert(tenants)
	assert.NoError(t, err)

	projectRequest := forms.CreateProjectRequest{
		TenantKey:   tenants.OrgKey,
		ProjectKey:  "test_project_key",
		Name:        "TestProject",
		Description: "This is a test project",
		Visibility:  structs.VisibleTypePrivate,
	}

	ctx := context.Background()
	response, err := CreateProject(ctx, projectRequest)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, projectRequest.Name, response.Name)
	assert.Equal(t, projectRequest.ProjectKey, response.ProjectKey)
	assert.Equal(t, projectRequest.Visibility, response.Visibility)

	org, err := organization.GetOrganizationByIDs(ctx, []int64{response.Id})
	assert.NoError(t, err)
	assert.NotNil(t, org)
	assert.Equal(t, projectRequest.Name, org[0].Name)
	assert.Equal(t, projectRequest.Visibility, org[0].Visibility)

	_, err = db.GetEngine(db.DefaultContext).Delete(&organization.Organization{ID: org[0].ID})
	assert.NoError(t, err)
	_, err = db.GetEngine(db.DefaultContext).Delete(&tenant.ScTenantOrganizations{ProjectKey: projectRequest.ProjectKey})
	assert.NoError(t, err)
	_, err = db.GetEngine(db.DefaultContext).Delete(tenants)
	assert.NoError(t, err)
}
