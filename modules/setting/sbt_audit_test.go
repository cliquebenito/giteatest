//go:build !correct

package setting

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

// TestAuditSettingsWithoutConfig проверяет настройки аудита без конфигурации секции [sbt.audit]
func TestAuditSettingsWithoutConfig(t *testing.T) {
	cfg, err := NewConfigProviderFromData(``)
	assert.NoError(t, err)
	loadAuditSbtGlobalFrom(cfg)

	assert.Equal(t, AppWorkPath, auditConfig.Path)
	assert.Equal(t, defaultAuditFileName, auditConfig.FileName)
	assert.NotEmpty(t, GetAuditWriterName())
}

// TestAuditSettingsWithEmptyConfig проверяет настройки аудита с пустой конфигурацией секции [sbt.audit]
func TestAuditSettingsWithEmptyConfig(t *testing.T) {
	cfg, err := NewConfigProviderFromData(`
[sbt.audit]
`)
	assert.NoError(t, err)
	loadAuditSbtGlobalFrom(cfg)

	assert.Equal(t, AppWorkPath, auditConfig.Path)
	assert.Equal(t, defaultAuditFileName, auditConfig.FileName)
	assert.NotEmpty(t, GetAuditWriterName())
}

// TestAuditSettingsWithConfiguredPath проверяет настройки аудита при указании аудит директории в конфигурации
func TestAuditSettingsWithConfiguredPath(t *testing.T) {
	cfg, err := NewConfigProviderFromData(`
[sbt.audit]
AUDIT_PATH=sbt_audit
`)
	assert.NoError(t, err)
	loadAuditSbtGlobalFrom(cfg)

	assert.Equal(t, AppWorkPath+"/sbt_audit", auditConfig.Path)
	assert.Equal(t, defaultAuditFileName, auditConfig.FileName)
	assert.NotEmpty(t, GetAuditWriterName())
}

// TestAuditSettingsWithConfiguredAbsolutePath проверяет настройки аудита при указании абсолютного пути к аудит директории в конфигурации
func TestAuditSettingsWithConfiguredAbsolutePath(t *testing.T) {
	cfg, err := NewConfigProviderFromData(`
[sbt.audit]
AUDIT_PATH=` + AppWorkPath + `/sbt_audit
`)
	assert.NoError(t, err)
	loadAuditSbtGlobalFrom(cfg)

	assert.Equal(t, AppWorkPath+"/sbt_audit", auditConfig.Path)
	assert.Equal(t, defaultAuditFileName, auditConfig.FileName)
	assert.NotEmpty(t, GetAuditWriterName())
}

// TestAuditSettingsWithConfiguredFileName проверяет настройки аудита при указании имени аудит файла в конфигурации
func TestAuditSettingsWithConfiguredFileName(t *testing.T) {
	cfg, err := NewConfigProviderFromData(`
[sbt.audit]
AUDIT_FILE_NAME=audit123
`)
	assert.NoError(t, err)
	loadAuditSbtGlobalFrom(cfg)

	assert.Equal(t, AppWorkPath, auditConfig.Path)
	assert.Equal(t, "audit123", auditConfig.FileName)
	assert.NotEmpty(t, GetAuditWriterName())
}

// TestAuditSettingsWithFullConfig проверяет настройки аудита при указании имени аудит файла и директории в конфигурации
func TestAuditSettingsWithFullConfig(t *testing.T) {
	cfg, err := NewConfigProviderFromData(`
[sbt.audit]
AUDIT_PATH=sbt_audit
AUDIT_FILE_NAME=audit123
`)
	assert.NoError(t, err)
	loadAuditSbtGlobalFrom(cfg)

	assert.Equal(t, AppWorkPath+"/sbt_audit", auditConfig.Path)
	assert.Equal(t, "audit123", auditConfig.FileName)
	assert.NotEmpty(t, GetAuditWriterName())
}
