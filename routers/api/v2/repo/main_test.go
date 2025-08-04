//go:build !correct

package repo

import (
	"path/filepath"
	"testing"

	"code.gitea.io/gitea/models/unittest"
	"code.gitea.io/gitea/modules/setting"
	webhook_service "code.gitea.io/gitea/services/webhook"
)

func TestMain(m *testing.M) {
	unittest.MainTest(m, &unittest.TestOptions{
		GiteaRootPath: filepath.Join("..", "..", "..", ".."),
		SetUp: func() error {
			setting.LoadQueueSettings()
			return webhook_service.Init()
		},
	})
}
