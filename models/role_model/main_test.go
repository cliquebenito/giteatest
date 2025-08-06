//go:build !correct

package role_model

import (
	_ "code.gitea.io/gitea/models"
)

/*
TestMain метод необходимый для создания тестовой среды
Автоматически вызывается перед запуском тестов
*/
//func TestMain(m *testing.M) {
//	unittest.MainTest(m, &unittest.TestOptions{
//		GiteaRootPath: filepath.Join("..", ".."),
//	})
//}
