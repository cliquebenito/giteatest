package setting

// SourceControlSecretServer получение secret для server
var SourceControlSecretServer struct {
	// LfsJwtSecret jwt secret
	LfsJwtSecret string
	// StoragePath путь к директории с secrets в vault хранилище
	StoragePath string
	// SecretPaths пути к директориям в vault хранилище
	SecretPath string
	// VersionKey версия получения cred
	VersionKey int
}

// loadSourceControlVaultServer функция для загрузки sourcecontrol.vault.server из config
func loadSourceControlVaultServer(rootCfg ConfigProvider) {
	if SourceControl.Enabled && SourceControlWrapVault.Enabled {
		sec := rootCfg.Section("sourcecontrol.vault.server")
		SourceControlSecretServer.LfsJwtSecret = sec.Key("LFS_JWT_SECRET").MustString("")
		SourceControlSecretServer.StoragePath = sec.Key("STORAGE_PATH").MustString("")
		SourceControlSecretServer.SecretPath = sec.Key("SECRET_PATH").MustString("")
		SourceControlSecretServer.VersionKey = sec.Key("VERSION_KV").MustInt(0)
	}
}
