package util

import (
	"strings"
)

// CheckMTLSParameterForAudit определения раздела для аудита в зависимости от интеграции по mtls
func CheckMTLSParameterForAudit(nameMTLSConn string) string {
	auditKey := "client_name"
	if strings.Contains(nameMTLSConn, "server") {
		auditKey = "server_name"
	}
	return auditKey
}
