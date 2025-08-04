//go:build !correct

// Copyright 2018 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package tests

import (
	"path/filepath"
	"testing"

	"code.gitea.io/gitea/models/unittest"
	"code.gitea.io/gitea/modules/setting"
	webhook_service "code.gitea.io/gitea/services/webhook"
)

/*
TestMain метод необходимый для создания тестовой среды
Автоматичсеки вызывается перед запуском тестов
*/
func TestMain(m *testing.M) {
	unittest.MainTest(m, &unittest.TestOptions{
		GiteaRootPath: filepath.Join("..", "..", "..", ".."), //путь до models/fixtures, где хранятся тестовые данные
		SetUp: func() error {
			setting.LoadQueueSettings()
			return webhook_service.Init()
		},
	})
}
