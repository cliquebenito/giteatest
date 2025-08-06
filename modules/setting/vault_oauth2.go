package setting

// SourceControlVaultOauth2  получение secret для oauth2
var SourceControlVaultOauth2 struct {
	// JwtSecret
	JwtSecret string
	// StoragePath путь к директории с secrets в vault хранилище
	StoragePath string
	// SecretPaths пути к директориям в vault хранилище
	SecretPath string
	// VersionKey версия получения cred
	VersionKey int
}

// loadSourceControlVaultOauth2 функция для загрузки sourcecontrol.vault.oauth2 из config
func loadSourceControlVaultOauth2(rootCfg ConfigProvider) {
	if SourceControl.Enabled && SourceControlWrapVault.Enabled {
		sec := rootCfg.Section("sourcecontrol.vault.oauth2")
		SourceControlVaultOauth2.JwtSecret = sec.Key("JWT_SECRET").MustString("")
		SourceControlVaultOauth2.StoragePath = sec.Key("STORAGE_PATH").MustString("")
		SourceControlVaultOauth2.SecretPath = sec.Key("SECRET_PATH").MustString("")
		SourceControlVaultOauth2.VersionKey = sec.Key("VERSION_KV").MustInt(0)
	}
}
