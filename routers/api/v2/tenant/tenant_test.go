//go:build !correct

package tenant

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	tenat_model "code.gitea.io/gitea/models/tenant"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/test"
	"code.gitea.io/gitea/modules/timeutil"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/routers/api/v2/models"
)

var tenantServer = NewTenantServer()

func TestGetTenant_Ok(t *testing.T) {
	unittest.PrepareTestEnv(t)
	tenantID := "99246748-c934-4034-9fa1-b6ef9014e672"
	tenantID2 := "99246748-c934-4034-9fa1-b6ef9014e673"
	tenantKey := "tenantKey"
	tenant := &tenat_model.ScTenant{
		ID:        tenantID,
		Name:      "test",
		IsActive:  false,
		Default:   true,
		CreatedAt: timeutil.TimeStampNow(),
		UpdatedAt: timeutil.TimeStampNow(),
	}
	tenant2 := &tenat_model.ScTenant{
		ID:        tenantID2,
		OrgKey:    tenantKey,
		Name:      "test2",
		IsActive:  false,
		Default:   false,
		CreatedAt: timeutil.TimeStampNow(),
		UpdatedAt: timeutil.TimeStampNow(),
	}

	ctx := test.MockAPIContext(t, fmt.Sprintf("api/v2/tenants/get?tenant_key=%s", tenantKey))
	ctx.SetFormString("tenant_key", tenantKey)

	_, err := tenat_model.InsertTenant(ctx, tenant)
	assert.NoError(t, err)
	unittest.AssertExistsAndLoadBean(t, tenant)
	_, err = tenat_model.InsertTenant(ctx, tenant2)
	assert.NoError(t, err)
	unittest.AssertExistsAndLoadBean(t, tenant2)

	test.LoadUser(t, ctx, 1)

	u := unittest.AssertExistsAndLoadBean(t, &user_model.User{
		IsAdmin: true,
		ID:      1,
	})
	ctx.Doer = u

	tenantServer.GetTenantByKey(ctx)
	assert.Equal(t, http.StatusOK, ctx.Resp.Status())
}

func TestGetTenant_TenantNotExist(t *testing.T) {
	unittest.PrepareTestEnv(t)
	tenantKey := "tenantKey"

	ctx := test.MockAPIContext(t, fmt.Sprintf("api/v2/tenants/get?tenant_key%s", tenantKey))
	ctx.SetFormString("tenant_key", tenantKey)

	test.LoadUser(t, ctx, 1)

	u := unittest.AssertExistsAndLoadBean(t, &user_model.User{
		IsAdmin: true,
		ID:      1,
	})
	ctx.Doer = u

	tenantServer.GetTenantByKey(ctx)
	assert.Equal(t, http.StatusNotFound, ctx.Resp.Status())
}

func TestCreateTenant_Ok(t *testing.T) {
	unittest.PrepareTestEnv(t)
	tenantKey := "test"
	tenantName := "test"
	tenantOptions := &models.CreateTenantOptions{
		Name:      tenantName,
		TenantKey: tenantKey,
	}

	ctx := test.MockAPIContext(t, "api/v2/tenants/create")
	web.SetForm(ctx, tenantOptions)

	test.LoadUser(t, ctx, 1)

	u := unittest.AssertExistsAndLoadBean(t, &user_model.User{
		IsAdmin: true,
		ID:      1,
	})
	ctx.Doer = u

	tenantServer.CreateTenant(ctx)
	assert.Equal(t, http.StatusCreated, ctx.Resp.Status())
}

func TestCreateTenant_TenantExists(t *testing.T) {
	unittest.PrepareTestEnv(t)
	tenantID := "99246748-c934-4034-9fa1-b6ef9014e672"
	tenantKey := "test"
	tenantName := "test"
	tenant := &tenat_model.ScTenant{
		ID:        tenantID,
		Name:      "test",
		IsActive:  false,
		Default:   true,
		CreatedAt: timeutil.TimeStampNow(),
		UpdatedAt: timeutil.TimeStampNow(),
	}
	tenantOptions := &models.CreateTenantOptions{
		Name:      tenantName,
		TenantKey: tenantKey,
	}

	ctx := test.MockAPIContext(t, "api/v2/tenants/create")
	web.SetForm(ctx, tenantOptions)

	_, err := tenat_model.InsertTenant(ctx, tenant)
	assert.NoError(t, err)
	unittest.AssertExistsAndLoadBean(t, tenant)

	test.LoadUser(t, ctx, 1)

	u := unittest.AssertExistsAndLoadBean(t, &user_model.User{
		IsAdmin: true,
		ID:      1,
	})
	ctx.Doer = u

	tenantServer.CreateTenant(ctx)
	assert.Equal(t, http.StatusConflict, ctx.Resp.Status())
}
