package setting

import (
	"os"
	"path/filepath"

	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/sbt/audit/writers"
	"code.gitea.io/gitea/modules/util"
)

const (
	defaultAuditFileName = "audit"
	auditFileFormat      = ".log"
)

// auditSbtConfig структура описывающая конфигурацию аудита
type auditSbtConfig struct {
	Path         string               // путь до файла
	FileName     string               // имя файла
	WriterOption log.WriterFileOption // опции аудирования
}

var auditConfig auditSbtConfig

// loadAuditSbtGlobalFrom функция считывания параметров аудита из конфигурации
func loadAuditSbtGlobalFrom(rootCfg ConfigProvider) {
	sec := rootCfg.Section("sbt.audit")

	auditConfig.Path = sec.Key("AUDIT_PATH").MustString(AppWorkPath)
	if !filepath.IsAbs(auditConfig.Path) {
		auditConfig.Path = filepath.Join(AppWorkPath, auditConfig.Path)
	}
	auditConfig.Path = util.FilePathJoinAbs(auditConfig.Path)
	if err := os.MkdirAll(auditConfig.Path, 0o755); err != nil {
		log.Fatal("Cannot create destination: %v, please check parameter [sbt.audit] AUDIT_PATH", err)
	}

	auditConfig.FileName = sec.Key("AUDIT_FILE_NAME").MustString(defaultAuditFileName)

	writerOption := log.WriterFileOption{}
	writerOption.FileName = auditConfig.Path + "/" + auditConfig.FileName + auditFileFormat
	writerOption.LogRotate = ConfigInheritedKey(sec, "AUDIT_ROTATE").MustBool(true)
	writerOption.MaxSize = 1 << uint(ConfigInheritedKey(sec, "MAX_SIZE_SHIFT").MustInt(28))
	writerOption.DailyRotate = ConfigInheritedKey(sec, "DAILY_ROTATE").MustBool(true)
	writerOption.MaxDays = ConfigInheritedKey(sec, "MAX_DAYS").MustInt(7)
	writerOption.Compress = ConfigInheritedKey(sec, "COMPRESS").MustBool(true)
	writerOption.CompressionLevel = ConfigInheritedKey(sec, "COMPRESSION_LEVEL").MustInt(-1)
	auditConfig.WriterOption = writerOption

	writers.NewAuditWriter(writerOption)
}
