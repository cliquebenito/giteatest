//go:build !correct

package templates

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"code.gitea.io/gitea/models/role_model"
	"code.gitea.io/gitea/modules/setting"
)

func TestCheckPrivilegesByRoleAndCustom(t *testing.T) {
	setting.SourceControl.TenantWithRoleModeEnabled = true
	assert.NoError(t, role_model.InitRoleModel())

	tests := []struct {
		name           string
		userId         int64
		tenantId       string
		orgId          int64
		repoID         int64
		actionCustom   string
		expectedResult bool
	}{
		{
			name:           "User  has privileges",
			userId:         1,
			tenantId:       "tenant_test",
			orgId:          1,
			repoID:         1,
			actionCustom:   "viewBranch",
			expectedResult: true,
		},
		{
			name:           "User  does not have privileges but has custom privileges",
			userId:         1,
			tenantId:       "tenant_test",
			orgId:          1,
			repoID:         1,
			actionCustom:   "changeBranch",
			expectedResult: true,
		},
		{
			name:           "User  does not have any privileges",
			userId:         1,
			tenantId:       "tenant_test",
			orgId:          1,
			repoID:         1,
			actionCustom:   "changeBranch",
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) { // Вызываем тестируемую функцию
			result := CheckPrivilegesByRoleAndCustom(tt.userId, tt.tenantId, tt.orgId, tt.repoID, tt.actionCustom)

			// Проверяем результат
			assert.Equal(t, tt.expectedResult, result)

		})
	}
}
