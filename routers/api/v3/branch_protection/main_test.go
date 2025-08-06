package protected_branch

import (
	"os"
	"testing"

	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/sbt/audit/writers"
)

func TestMain(m *testing.M) {
	// Создаем нужный WriterFileOption
	writerOption := log.WriterFileOption{
		FileName:         "test_audit.log",
		MaxSize:          10 * 1024 * 1024, // 10 MB
		LogRotate:        true,
		DailyRotate:      true,
		MaxDays:          7,
		Compress:         true,
		CompressionLevel: 5,
	}

	// Инициализация аудит-логгера
	writers.NewAuditWriter(writerOption)

	// Запуск тестов
	code := m.Run()
	os.Remove(writerOption.FileName)

	os.Exit(code)
}
