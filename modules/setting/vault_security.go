package setting

// SourceControlVaultSecurity получение secret для security
var SourceControlVaultSecurity struct {
	// InternalToken
	InternalToken string
	// StoragePath путь к директории с secrets в vault хранилище
	StoragePath string
	// SecretPaths пути к директориям в vault хранилище
	SecretPath string
	// VersionKey версия получения cred
	VersionKey int
}

// loadSourceControlVaultSecurity функция для загрузки sourcecontrol.vault.security из config
func loadSourceControlVaultSecurity(rootCfg ConfigProvider) {
	if SourceControl.Enabled && SourceControlWrapVault.Enabled {
		sec := rootCfg.Section("sourcecontrol.vault.security")
		SourceControlVaultSecurity.InternalToken = sec.Key("INTERNAL_TOKEN").MustString("")
		SourceControlVaultSecurity.StoragePath = sec.Key("STORAGE_PATH").MustString("")
		SourceControlVaultSecurity.SecretPath = sec.Key("SECRET_PATH").MustString("")
		SourceControlVaultSecurity.VersionKey = sec.Key("VERSION_KV").MustInt(0)
	}
}
