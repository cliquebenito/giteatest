//go:build !correct

package repo

import (
	"fmt"
	"net/http"
	"strconv"
	"testing"

	"code.gitea.io/gitea/models/db"
	"github.com/stretchr/testify/assert"

	"code.gitea.io/gitea/models/organization"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/role_model"
	tenat_model "code.gitea.io/gitea/models/tenant"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/test"
	"code.gitea.io/gitea/modules/timeutil"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/routers/api/v2/models/repo"
)

var defaultTenantId = "99246748-c934-4034-9fa1-b6ef9014e673"
var defaultPermission = true

func TestGetOrgRepo_Ok(t *testing.T) {
	unittest.PrepareTestEnv(t)
	var orgId int64 = 2
	var repoId int64 = 1
	var repoKey = "1234"
	tenantID := "99246748-c934-4034-9fa1-b6ef9014e672"
	tenantKey := "tenant"
	projectKey := "project"
	tenant := &tenat_model.ScTenant{
		ID:        tenantID,
		Name:      "test",
		IsActive:  false,
		Default:   false,
		CreatedAt: timeutil.TimeStampNow(),
		UpdatedAt: timeutil.TimeStampNow(),
	}
	tenantOrg := &tenat_model.ScTenantOrganizations{
		TenantID:       defaultTenantId,
		OrganizationID: orgId,
		OrgKey:         tenantKey,
		ProjectKey:     projectKey,
	}
	scRepoKey := &repo_model.ScRepoKey{
		RepoID:  strconv.FormatInt(repoId, 10),
		RepoKey: repoKey,
	}

	ctx := test.MockAPIContext(t, fmt.Sprintf("api/v2/project/repos?repo_key=%s&tenant_key=%s&project_key=%s", repoKey, tenantKey, projectKey))
	ctx.SetFormString("repo_key", repoKey)
	ctx.SetFormString("tenant_key", tenantKey)
	ctx.SetFormString("project_key", projectKey)
	dbEngine := db.GetEngine(ctx)
	repoKeyDB := repo_model.NewRepoKeyDB(dbEngine)
	var repoServer = NewRepoServer(getUserPermissionFunc, repoKeyDB)

	_, err := tenat_model.InsertTenant(ctx, tenant)
	assert.NoError(t, err)
	unittest.AssertExistsAndLoadBean(t, tenant)
	err = tenat_model.InsertTenantOrganization(ctx, tenantOrg)
	assert.NoError(t, err)
	unittest.AssertExistsAndLoadBean(t, tenantOrg)
	err = repoKeyDB.InsertRepoKey(ctx, scRepoKey)
	assert.NoError(t, err)
	unittest.AssertExistsAndLoadBean(t, scRepoKey)

	test.LoadRepo(t, ctx, repoId)
	test.LoadUser(t, ctx, 1)

	u := unittest.AssertExistsAndLoadBean(t, &user_model.User{
		IsAdmin: true,
		ID:      1,
	})
	unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{
		ID: repoId,
	})
	ctx.Doer = u

	repoServer.GetOrgRepo(ctx)
	assert.Equal(t, http.StatusOK, ctx.Resp.Status())
}

func TestGetOrgRepo_NoPermission(t *testing.T) {
	unittest.PrepareTestEnv(t)
	var orgId int64 = 2
	var repoId int64 = 1
	var repoKey = "1234"
	tenantKey := "tenant"
	projectKey := "project"
	tenantID := "99246748-c934-4034-9fa1-b6ef9014e672"
	tenant := &tenat_model.ScTenant{
		ID:        tenantID,
		Name:      "test",
		IsActive:  false,
		Default:   false,
		CreatedAt: timeutil.TimeStampNow(),
		UpdatedAt: timeutil.TimeStampNow(),
	}
	tenantOrg := &tenat_model.ScTenantOrganizations{
		TenantID:       defaultTenantId,
		OrganizationID: orgId,
		OrgKey:         tenantKey,
		ProjectKey:     projectKey,
	}
	scRepoKey := &repo_model.ScRepoKey{
		RepoID:  strconv.FormatInt(repoId, 10),
		RepoKey: repoKey,
	}

	ctx := test.MockAPIContext(t, fmt.Sprintf("api/v2/project/repos?repo_key=%s&tenant_key=%s&project_key=%s", repoKey, tenantKey, projectKey))
	ctx.SetFormString("repo_key", repoKey)
	ctx.SetFormString("tenant_key", tenantKey)
	ctx.SetFormString("project_key", projectKey)
	dbEngine := db.GetEngine(ctx)
	repoKeyDB := repo_model.NewRepoKeyDB(dbEngine)
	var repoServer = NewRepoServer(getUserPermissionFunc, repoKeyDB)

	_, err := tenat_model.InsertTenant(ctx, tenant)
	assert.NoError(t, err)
	unittest.AssertExistsAndLoadBean(t, tenant)
	err = tenat_model.InsertTenantOrganization(ctx, tenantOrg)
	assert.NoError(t, err)
	unittest.AssertExistsAndLoadBean(t, tenantOrg)
	err = repoKeyDB.InsertRepoKey(ctx, scRepoKey)
	assert.NoError(t, err)
	unittest.AssertExistsAndLoadBean(t, scRepoKey)

	test.LoadRepo(t, ctx, repoId)
	test.LoadUser(t, ctx, 1)
	defaultPermission = false

	u := unittest.AssertExistsAndLoadBean(t, &user_model.User{
		IsAdmin: true,
		ID:      1,
	})
	unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{
		ID: repoId,
	})
	ctx.Doer = u

	repoServer.GetOrgRepo(ctx)
	assert.Equal(t, http.StatusNotFound, ctx.Resp.Status())
}

func TestCreateTenantOrgRepo_Ok(t *testing.T) {
	unittest.PrepareTestEnv(t)
	var orgId int64 = 2
	var repoName = "test-repo"
	tenantID := "99246748-c934-4034-9fa1-b6ef9014e672"
	orgKey := "tenant"
	projectKey := "project"
	repoKey := "1234"
	defTenant := &tenat_model.ScTenant{
		ID:        tenantID,
		Name:      "default",
		OrgKey:    orgKey,
		IsActive:  true,
		Default:   true,
		CreatedAt: timeutil.TimeStampNow(),
		UpdatedAt: timeutil.TimeStampNow(),
	}

	tenantOrg := &tenat_model.ScTenantOrganizations{
		TenantID:       tenantID,
		OrganizationID: orgId,
		OrgKey:         orgKey,
		ProjectKey:     projectKey,
	}
	repoOptions := &repo.CreateRepoOptions{
		TenantKey:     orgKey,
		ProjectKey:    projectKey,
		RepositoryKey: repoKey,
		DefaultBranch: "main",
		Description:   "",
		Name:          repoName,
	}

	ctx := test.MockAPIContext(t, "api/v2/projects/repos/create")
	web.SetForm(ctx, repoOptions)
	dbEngine := db.GetEngine(ctx)
	repoKeyDB := repo_model.NewRepoKeyDB(dbEngine)
	var repoServer = NewRepoServer(getUserPermissionFunc, repoKeyDB)

	_, err := tenat_model.InsertTenant(ctx, defTenant)
	assert.NoError(t, err)
	unittest.AssertExistsAndLoadBean(t, defTenant)
	err = tenat_model.InsertTenantOrganization(ctx, tenantOrg)
	assert.NoError(t, err)
	unittest.AssertExistsAndLoadBean(t, tenantOrg)

	test.LoadUser(t, ctx, 1)

	u := unittest.AssertExistsAndLoadBean(t, &user_model.User{
		IsAdmin: true,
		ID:      1,
	})
	ctx.Doer = u

	repoServer.CreateTenantOrgRepo(ctx)
	assert.Equal(t, http.StatusCreated, ctx.Resp.Status())
}

func TestCreateTenantOrgRepo_NoPermission(t *testing.T) {
	unittest.PrepareTestEnv(t)
	var orgId int64 = 2
	var repoName = "test-repo"
	tenantID := "99246748-c934-4034-9fa1-b6ef9014e672"
	orgKey := "tenant"
	projectKey := "project"
	repoKey := "1234"
	defTenant := &tenat_model.ScTenant{
		ID:        tenantID,
		Name:      "default",
		OrgKey:    orgKey,
		IsActive:  true,
		Default:   true,
		CreatedAt: timeutil.TimeStampNow(),
		UpdatedAt: timeutil.TimeStampNow(),
	}

	tenantOrg := &tenat_model.ScTenantOrganizations{
		TenantID:       tenantID,
		OrganizationID: orgId,
		OrgKey:         orgKey,
		ProjectKey:     projectKey,
	}
	repoOptions := &repo.CreateRepoOptions{
		TenantKey:     orgKey,
		ProjectKey:    projectKey,
		RepositoryKey: repoKey,
		DefaultBranch: "main",
		Description:   "",
		Name:          repoName,
	}

	ctx := test.MockAPIContext(t, "api/v2/projects/repos/create")
	web.SetForm(ctx, repoOptions)
	dbEngine := db.GetEngine(ctx)
	repoKeyDB := repo_model.NewRepoKeyDB(dbEngine)
	var repoServer = NewRepoServer(getUserPermissionFunc, repoKeyDB)

	_, err := tenat_model.InsertTenant(ctx, defTenant)
	assert.NoError(t, err)
	unittest.AssertExistsAndLoadBean(t, defTenant)
	err = tenat_model.InsertTenantOrganization(ctx, tenantOrg)
	assert.NoError(t, err)
	unittest.AssertExistsAndLoadBean(t, tenantOrg)

	test.LoadUser(t, ctx, 1)
	defaultPermission = false

	u := unittest.AssertExistsAndLoadBean(t, &user_model.User{
		IsAdmin: true,
		ID:      1,
	})
	ctx.Doer = u

	repoServer.CreateTenantOrgRepo(ctx)
	assert.Equal(t, http.StatusNotFound, ctx.Resp.Status())
}

func TestCreateTenantOrgRepo_RepoNameExists(t *testing.T) {
	unittest.PrepareTestEnv(t)
	var orgId int64 = 2
	var repoName = "test-repo"
	tenantID := "99246748-c934-4034-9fa1-b6ef9014e672"
	orgKey := "tenant"
	projectKey := "project"
	repoKey := "1234"
	defTenant := &tenat_model.ScTenant{
		ID:        tenantID,
		Name:      "default",
		OrgKey:    orgKey,
		IsActive:  true,
		Default:   true,
		CreatedAt: timeutil.TimeStampNow(),
		UpdatedAt: timeutil.TimeStampNow(),
	}

	tenantOrg := &tenat_model.ScTenantOrganizations{
		TenantID:       tenantID,
		OrganizationID: orgId,
		OrgKey:         orgKey,
		ProjectKey:     projectKey,
	}
	repoOptions := &repo.CreateRepoOptions{
		TenantKey:     orgKey,
		ProjectKey:    projectKey,
		RepositoryKey: repoKey,
		DefaultBranch: "main",
		Description:   "",
		Name:          repoName,
	}

	ctx := test.MockAPIContext(t, "api/v2/projects/repos/create")
	web.SetForm(ctx, repoOptions)
	dbEngine := db.GetEngine(ctx)
	repoKeyDB := repo_model.NewRepoKeyDB(dbEngine)
	var repoServer = NewRepoServer(getUserPermissionFunc, repoKeyDB)

	_, err := tenat_model.InsertTenant(ctx, defTenant)
	assert.NoError(t, err)
	unittest.AssertExistsAndLoadBean(t, defTenant)
	err = tenat_model.InsertTenantOrganization(ctx, tenantOrg)
	assert.NoError(t, err)
	unittest.AssertExistsAndLoadBean(t, tenantOrg)

	test.LoadUser(t, ctx, 1)
	test.LoadRepo(t, ctx, 1)
	r := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{
		ID: 1,
	})
	repoOptions.Name = r.Name

	u := unittest.AssertExistsAndLoadBean(t, &user_model.User{
		IsAdmin: true,
		ID:      1,
	})
	ctx.Doer = u

	repoServer.CreateTenantOrgRepo(ctx)
	assert.Equal(t, http.StatusConflict, ctx.Resp.Status())
}

func TestCreateTenantOrgRepo_RepoKeyExists(t *testing.T) {
	unittest.PrepareTestEnv(t)
	var orgId int64 = 2
	var repoName = "test-repo"
	tenantID := "99246748-c934-4034-9fa1-b6ef9014e672"
	orgKey := "tenant"
	projectKey := "project"
	repoKey := "1234"
	defTenant := &tenat_model.ScTenant{
		ID:        tenantID,
		Name:      "default",
		OrgKey:    orgKey,
		IsActive:  true,
		Default:   true,
		CreatedAt: timeutil.TimeStampNow(),
		UpdatedAt: timeutil.TimeStampNow(),
	}

	tenantOrg := &tenat_model.ScTenantOrganizations{
		TenantID:       tenantID,
		OrganizationID: orgId,
		OrgKey:         orgKey,
		ProjectKey:     projectKey,
	}
	repoOptions := &repo.CreateRepoOptions{
		TenantKey:     orgKey,
		ProjectKey:    projectKey,
		RepositoryKey: repoKey,
		DefaultBranch: "main",
		Description:   "",
		Name:          repoName,
	}
	scRepoKey := &repo_model.ScRepoKey{
		RepoKey: repoKey,
	}

	ctx := test.MockAPIContext(t, "api/v2/projects/repos/create")
	web.SetForm(ctx, repoOptions)
	dbEngine := db.GetEngine(ctx)
	repoKeyDB := repo_model.NewRepoKeyDB(dbEngine)
	var repoServer = NewRepoServer(getUserPermissionFunc, repoKeyDB)

	_, err := tenat_model.InsertTenant(ctx, defTenant)
	assert.NoError(t, err)
	unittest.AssertExistsAndLoadBean(t, defTenant)
	err = tenat_model.InsertTenantOrganization(ctx, tenantOrg)
	assert.NoError(t, err)
	unittest.AssertExistsAndLoadBean(t, tenantOrg)
	err = repoKeyDB.InsertRepoKey(ctx, scRepoKey)
	assert.NoError(t, err)
	unittest.AssertExistsAndLoadBean(t, defTenant)

	test.LoadUser(t, ctx, 1)

	u := unittest.AssertExistsAndLoadBean(t, &user_model.User{
		IsAdmin: true,
		ID:      1,
	})
	ctx.Doer = u

	repoServer.CreateTenantOrgRepo(ctx)
	assert.Equal(t, http.StatusConflict, ctx.Resp.Status())
}

func getUserPermissionFunc(trace string, sub *user_model.User, tenantId string, org *organization.Organization, action role_model.Action) (bool, error) {
	return defaultPermission, nil
}
