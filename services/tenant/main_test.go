//go:build !correct

package tenant

import (
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/tenant"
	"code.gitea.io/gitea/models/unittest"
	"path/filepath"
	"testing"
)

// иницилизируем бд для тестов
func init() {
	db.RegisterModel(new(tenant.ScTenant))
	db.RegisterModel(new(tenant.ScTenantOrganizations))
}

func TestMain(m *testing.M) {
	unittest.MainTest(m, &unittest.TestOptions{
		GiteaRootPath: filepath.Join("..", ".."),
	})
}
