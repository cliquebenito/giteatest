package setting

var SourceControlSecretDB struct {
	// StoragePath путь к директории с secrets в vault хранилище
	StoragePath string
	// SecretPaths пути к директориям в vault хранилище
	SecretPath string
	// UserKey имя поля user для подключения к бд
	UserKey string
	// PasswordKey пароль для полключения к бд
	PasswordKey string
	// VersionKey версия получения cred
	VersionKey int
}

func loadSourceControlVaultDB(rootCgf ConfigProvider) {
	if SourceControl.Enabled && SourceControlWrapVault.Enabled {
		sec := rootCgf.Section("sourcecontrol.vault.db")
		SourceControlSecretDB.StoragePath = sec.Key("STORAGE_PATH").MustString("")
		SourceControlSecretDB.SecretPath = sec.Key("SECRET_PATH").MustString("")
		SourceControlSecretDB.UserKey = sec.Key("USER_KEY").MustString("")
		SourceControlSecretDB.PasswordKey = sec.Key("PASSWORD_KEY").MustString("")
		SourceControlSecretDB.VersionKey = sec.Key("VERSION_KV").MustInt(0)
	}
}
