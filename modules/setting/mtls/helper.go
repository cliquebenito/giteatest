package mtls

import (
	"code.gitea.io/gitea/modules/setting"
)

// CheckMTLSConfigSecManEnabled проверяем, что для блок с конфигом был добавлен, mtls включен, и client для sec man был успешно создан
func CheckMTLSConfigSecManEnabled(nameMTLSConn string) bool {
	configMtls, ok := setting.MTLSConnectionAvailable[nameMTLSConn]
	return ok && configMtls.Enabled && setting.CheckSettingsForIntegrationWithSecMan() && setting.SourceControlSecretMTLS.MTLSEnabled
}
