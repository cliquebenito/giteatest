//go:build !correct

package role_model

import (
	"context"
	"testing"

	"code.gitea.io/gitea/models/organization"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/setting"

	"github.com/stretchr/testify/assert"
)

/*
	Для запуска тестов необходимо добавить в настройках тестов "Go tool arguments": -tags sqlite,sqlite_unlock_notify
	Для запуска тестов из консоли использовать команду go test code.gitea.io/gitea/models/role_model -tags sqlite,sqlite_unlock_notify
*/

var user = &user_model.User{ID: 123}
var org = &organization.Organization{ID: 345}

// TestNoInitRoleModelIfDisabled проверяет что инициализация не запускается, если SBT_TENANT_WITH_ROLE_MODEL_ENABLED = false
func TestNoInitRoleModelIfDisabled(t *testing.T) {
	setting.SourceControl.TenantWithRoleModeEnabled = false
	assert.NoError(t, InitRoleModel())
	assert.Empty(t, securityEnforcer)
}

// TestInitRoleModel проверяет что инициализация запускается, если SBT_TENANT_WITH_ROLE_MODEL_ENABLED = true
func TestInitRoleModel(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	setting.SourceControl.TenantWithRoleModeEnabled = true
	assert.NoError(t, InitRoleModel())
	assert.NotEmpty(t, securityEnforcer)
}

// TestCheckUserPermissionToOrganizationWithoutPolicy проверяет что проверка на доступ не успешна, если не был выдан доступ
func TestCheckUserPermissionToOrganizationWithoutPolicy(t *testing.T) {
	PrepareRoleModel(t)

	permitted, err := CheckUserPermissionToOrganization(context.Background(), user, "tenant", org, WRITE)
	assert.NoError(t, err)
	assert.False(t, permitted)
}

// TestCheckUserPermissionToOrganizationWithPolicyInOtherTenant проверяет что проверка на доступ не успешна, если был выдан доступ в другом тенанте
func TestCheckUserPermissionToOrganizationWithPolicyInOtherTenant(t *testing.T) {
	PrepareRoleModel(t)
	assert.NoError(t, GrantUserPermissionToOrganization(user, "tenant", org, OWNER))

	permitted, err := CheckUserPermissionToOrganization(context.Background(), user, "tenant_1", org, WRITE)
	assert.NoError(t, err)
	assert.False(t, permitted)
	assert.NoError(t, RevokeUserPermissionToOrganization(user, "tenant", org, OWNER, true))
}

// TestCheckUserPermissionToOrganizationWithSufficientPrivileges проверяет что проверка на доступ успешна, если был выдан доступ
func TestCheckUserPermissionToOrganizationWithSufficientPrivileges(t *testing.T) {
	PrepareRoleModel(t)
	assert.NoError(t, GrantUserPermissionToOrganization(user, "tenant", org, OWNER))

	permitted, err := CheckUserPermissionToOrganization(context.Background(), user, "tenant", org, WRITE)
	assert.NoError(t, err)
	assert.True(t, permitted)
	assert.NoError(t, RevokeUserPermissionToOrganization(user, "tenant", org, OWNER, true))
}

// TestCheckUserPermissionToOrganizationWithInsufficientPrivileges проверяем что проверка на доступы успешна, привилегии были выданы выше запрашиваемых
func TestCheckUserPermissionToOrganizationWithInsufficientPrivileges(t *testing.T) {
	PrepareRoleModel(t)
	assert.NoError(t, GrantUserPermissionToOrganization(user, "tenant", org, READER))

	permitted, err := CheckUserPermissionToOrganization(context.Background(), user, "tenant", org, WRITE)
	assert.NoError(t, err)
	assert.False(t, permitted)
	assert.NoError(t, RevokeUserPermissionToOrganization(user, "tenant", org, READER, true))
}

// TestCheckUserPermissionToOrganizationWithNotSufficientPrivileges проверяет что проверка на доступ не успешна, если в выданном доступе недостаточно привилегий
func TestCheckUserPermissionToOrganizationWithNotSufficientPrivileges(t *testing.T) {
	PrepareRoleModel(t)
	assert.NoError(t, GrantUserPermissionToOrganization(user, "tenant", org, MANAGER))
	permitted, err := CheckUserPermissionToOrganization(context.Background(), user, "tenant", org, OWN)
	assert.NoError(t, err)
	assert.False(t, permitted)
	assert.NoError(t, RevokeUserPermissionToOrganization(user, "tenant", org, MANAGER, true))
}

// TestCheckUserPermissionToOrganizationAfterRevokeSufficientPrivileges проверяет что проверка на доступ не успешна, если доступ был снят
func TestCheckUserPermissionToOrganizationAfterRevokeSufficientPrivileges(t *testing.T) {
	PrepareRoleModel(t)
	assert.NoError(t, GrantUserPermissionToOrganization(user, "tenant", org, WRITER))

	permitted, err := CheckUserPermissionToOrganization(context.Background(), user, "tenant", org, READ)
	assert.NoError(t, err)
	assert.True(t, permitted)

	assert.NoError(t, RevokeUserPermissionToOrganization(user, "tenant", org, WRITER, true))

	permitted, err = CheckUserPermissionToOrganization(context.Background(), user, "tenant", org, WRITE)
	assert.NoError(t, err)
	assert.False(t, permitted)
}

// PrepareRoleModel подготавливает тестовую базу и инициализирует ролевую модель
func PrepareRoleModel(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	setting.SourceControl.TenantWithRoleModeEnabled = true
	assert.NoError(t, InitRoleModel())
}
