package setting

// SourceControlWrapVault настройки для wrap токена и получение кредов из vault хранилища
var SourceControlWrapVault struct {
	// WrapTokenPath путь до файла с wrap_secret_id
	WrapTokenPath string
	// PeriodWrappingToken время прогрева токена
	PeriodWrappingToken int
	// NameTenantSecMan название тенанта в sec man
	NameTenantSecManWrap string
	// UrlSecMan url для подключения к sec man
	UrlSecMan string
	// TtlWrapToken период жизни токена
	TtlWrapToken int
	// Enabled включен или выключен режим для работы с sec man
	Enabled bool
}

// loadSourceControlWrapVault загружаем переменные из конфига для Wrapping
func loadSourceControlWrapVault(rootCfg ConfigProvider) {
	sec := rootCfg.Section("sourcecontrol.wrap.vault")
	SourceControlWrapVault.Enabled = sec.Key("VAULT_ENABLED").MustBool(false)
	if SourceControlWrapVault.Enabled && SourceControl.Enabled {
		SourceControlWrapVault.WrapTokenPath = sec.Key("WRAP_TOKEN_PATH").MustString("")
		SourceControlWrapVault.PeriodWrappingToken = sec.Key("PERIOD_WRAPPING_TOKEN").MustInt(0)
		SourceControlWrapVault.TtlWrapToken = sec.Key("TTL_WRAP_TOKEN").MustInt(0)
		SourceControlWrapVault.NameTenantSecManWrap = sec.Key("NAME_TENANT_SEC_MAN_WRAP").MustString("")
		SourceControlWrapVault.UrlSecMan = sec.Key("URL_SEC_MAN").MustString("")
	}
}
