package setting

var SourceControlOneWorkToken struct {
	// Получаем захэшированный token для индентификации one work
	OneWorkToken string
	// StoragePath путь к директории с secrets в vault хранилище
	StoragePath string
	// SecretPaths пути к директориям в vault хранилище
	SecretPath string
	// VersionKey версия получения cred
	VersionKey int
}

// loadSourceControlOneWork загружаем параметры из конфига
func loadSourceControlOneWork(rootCfg ConfigProvider) {
	if SourceControl.Enabled && SourceControlWrapVault.Enabled {
		sec := rootCfg.Section("sourcecontrol.vault.one_work")
		SourceControlOneWorkToken.OneWorkToken = sec.Key("ONE_WORK_TOKEN").MustString("")
		SourceControlOneWorkToken.StoragePath = sec.Key("STORAGE_PATH").MustString("")
		SourceControlOneWorkToken.SecretPath = sec.Key("SECRET_PATH").MustString("")
		SourceControlOneWorkToken.VersionKey = sec.Key("VERSION_KEY").MustInt(0)
	}
}
