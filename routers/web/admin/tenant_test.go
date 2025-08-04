//go:build !correct

package admin

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"

	tenat_model "code.gitea.io/gitea/models/tenant"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/test"
	"code.gitea.io/gitea/modules/timeutil"
	"github.com/stretchr/testify/assert"
)

func TestEditTenant_ChangeNameOk(t *testing.T) {
	tenantID := "99246748-c934-4034-9fa1-b6ef9014e672"
	tenant := &tenat_model.ScTenant{
		ID:        tenantID,
		Name:      "test",
		IsActive:  false,
		Default:   false,
		CreatedAt: timeutil.TimeStampNow(),
		UpdatedAt: timeutil.TimeStampNow(),
	}
	unittest.PrepareTestEnv(t)
	ctx := test.MockContext(t, fmt.Sprintf("admin/tenants/%s/edit", tenantID))
	ctx.SetParams("tenantid", tenantID)

	u := unittest.AssertExistsAndLoadBean(t, &user_model.User{
		IsAdmin: true,
		ID:      2,
	})

	ctx.Doer = u

	_, err := tenat_model.InsertTenant(ctx, tenant)
	assert.NoError(t, err)
	unittest.AssertExistsAndLoadBean(t, tenant)

	ctx.Req.Form = url.Values{"name": {"test2"}, "is_active": {"false"}}
	EditTenant(ctx)

	tn, err := tenat_model.GetTenantByID(ctx, tenant.ID)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, ctx.Base.Resp.Status())
	assert.Equal(t, tn.Name, "test2")
	assert.Equal(t, tn.IsActive, false)
}

func TestEditTenant_ChangeActiveOk(t *testing.T) {
	tenantID := "99246748-c934-4034-9fa1-b6ef9014e672"
	tenant := &tenat_model.ScTenant{
		ID:        tenantID,
		Name:      "test",
		IsActive:  false,
		Default:   false,
		CreatedAt: timeutil.TimeStampNow(),
		UpdatedAt: timeutil.TimeStampNow(),
	}
	unittest.PrepareTestEnv(t)
	ctx := test.MockContext(t, fmt.Sprintf("admin/tenants/%s/edit", tenantID))
	ctx.SetParams("tenantid", tenantID)

	u := unittest.AssertExistsAndLoadBean(t, &user_model.User{
		IsAdmin: true,
		ID:      2,
	})

	ctx.Doer = u

	_, err := tenat_model.InsertTenant(ctx, tenant)
	assert.NoError(t, err)
	unittest.AssertExistsAndLoadBean(t, tenant)

	ctx.Req.Form = url.Values{"name": {"test"}, "is_active": {"true"}}
	EditTenant(ctx)

	tn, err := tenat_model.GetTenantByID(ctx, tenant.ID)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, ctx.Base.Resp.Status())
	assert.Equal(t, tn.Name, "test")
	assert.Equal(t, tn.IsActive, true)
}

func TestEditTenant_DoesntExist(t *testing.T) {
	tenantID := "99246748-c934-4034-9fa1-b6ef9014e672"

	unittest.PrepareTestEnv(t)
	ctx := test.MockContext(t, fmt.Sprintf("admin/tenants/%s/edit", tenantID))
	ctx.SetParams("tenantid", tenantID)

	u := unittest.AssertExistsAndLoadBean(t, &user_model.User{
		IsAdmin: true,
		ID:      2,
	})

	ctx.Doer = u

	ctx.Req.Form = url.Values{"name": {"test"}, "is_active": {"true"}}
	EditTenant(ctx)

	assert.Equal(t, http.StatusNotFound, ctx.Base.Resp.Status())
}

func TestEditTenant_InvalidName(t *testing.T) {
	tenantID := "99246748-c934-4034-9fa1-b6ef9014e672"
	tenant := &tenat_model.ScTenant{
		ID:        tenantID,
		Name:      "test",
		IsActive:  false,
		Default:   false,
		CreatedAt: timeutil.TimeStampNow(),
		UpdatedAt: timeutil.TimeStampNow(),
	}
	unittest.PrepareTestEnv(t)
	ctx := test.MockContext(t, fmt.Sprintf("admin/tenants/%s/edit", tenantID))
	ctx.SetParams("tenantid", tenantID)

	u := unittest.AssertExistsAndLoadBean(t, &user_model.User{
		IsAdmin: true,
		ID:      2,
	})

	ctx.Doer = u

	_, err := tenat_model.InsertTenant(ctx, tenant)
	assert.NoError(t, err)
	unittest.AssertExistsAndLoadBean(t, tenant)

	newName := "testtesttesttesttesttesttesttesttesttesttesttesttes"

	ctx.Req.Form = url.Values{"name": {newName}, "is_active": {"true"}}
	EditTenant(ctx)

	assert.Equal(t, http.StatusBadRequest, ctx.Base.Resp.Status())
}

func TestEditTenant_ChangeNameAlreadyExists(t *testing.T) {
	tenantID1 := "99246748-c934-4034-9fa1-b6ef9014e671"
	tenantID2 := "99246748-c934-4034-9fa1-b6ef9014e672"
	tenant1 := &tenat_model.ScTenant{
		ID:        tenantID1,
		Name:      "test1",
		OrgKey:    "test1",
		IsActive:  false,
		Default:   false,
		CreatedAt: timeutil.TimeStampNow(),
		UpdatedAt: timeutil.TimeStampNow(),
	}
	tenant2 := &tenat_model.ScTenant{
		ID:        tenantID2,
		Name:      "test2",
		OrgKey:    "test2",
		IsActive:  false,
		Default:   false,
		CreatedAt: timeutil.TimeStampNow(),
		UpdatedAt: timeutil.TimeStampNow(),
	}
	unittest.PrepareTestEnv(t)
	ctx := test.MockContext(t, fmt.Sprintf("admin/tenants/%s/edit", tenantID2))
	ctx.SetParams("tenantid", tenantID2)

	u := unittest.AssertExistsAndLoadBean(t, &user_model.User{
		IsAdmin: true,
		ID:      2,
	})

	ctx.Doer = u

	_, err := tenat_model.InsertTenant(ctx, tenant1)
	assert.NoError(t, err)
	unittest.AssertExistsAndLoadBean(t, tenant1)
	_, err = tenat_model.InsertTenant(ctx, tenant2)
	assert.NoError(t, err)
	unittest.AssertExistsAndLoadBean(t, tenant2)

	ctx.Req.Form = url.Values{"name": {tenant1.Name}, "is_active": {"false"}}
	EditTenant(ctx)

	assert.Equal(t, http.StatusBadRequest, ctx.Base.Resp.Status())
}
