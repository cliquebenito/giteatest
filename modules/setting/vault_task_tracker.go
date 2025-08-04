package setting

var SourceControlTaskTracker struct {
	// TokenKey token для получения доступа к task tracker
	APIToken string
	// StoragePath путь к директории с secrets в vault хранилище
	StoragePath string
	// SecretPath пути к директориям в vault хранилище
	SecretPath string
	// VersionKey версия получения cred
	VersionKey int
}

func loadSourceControlTaskTracker(rootCgf ConfigProvider) {
	if SourceControl.Enabled && SourceControlWrapVault.Enabled {
		sec := rootCgf.Section("sourcecontrol.vault.task_tracker")
		SourceControlTaskTracker.APIToken = sec.Key("API_TOKEN").MustString("")
		SourceControlTaskTracker.StoragePath = sec.Key("STORAGE_PATH").MustString("")
		SourceControlTaskTracker.SecretPath = sec.Key("SECRET_PATH").MustString("")
		SourceControlTaskTracker.VersionKey = sec.Key("VERSION_KV").MustInt(0)
	}
}
