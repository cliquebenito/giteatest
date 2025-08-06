package setting

var SourceControlAuthKeycloakSecret struct {
	// RealmClientSecret clientSecret для realm в keycloak
	RealmClientSecret string
	// RealmClientID clientID для realm в keycloak
	RealmClientID string
	// AdminCliSecret cliSecret для admin в keycloak
	AdminCliSecret string
	// AdminClientIDMasterRealm clientID для admin в keycloak
	AdminClientIDMasterRealm string
	// StoragePath путь к директории с secrets в vault хранилище
	StoragePath string
	// SecretPaths пути к директориям в vault хранилище
	SecretPath string
	// VersionKey версия получения cred
	VersionKey int
}

// loadSourceControlAuthKeycloak функция для загрузки sourcecontrol.vault.auth_keycloak из config
func loadSourceControlAuthKeycloak(rootCfg ConfigProvider) {
	if SourceControl.Enabled && SourceControlWrapVault.Enabled {
		sec := rootCfg.Section("sourcecontrol.vault.auth_keycloak")
		SourceControlAuthKeycloakSecret.RealmClientSecret = sec.Key("REALM_CLIENT_SECRET").MustString("")
		SourceControlAuthKeycloakSecret.RealmClientID = sec.Key("REALM_CLIENT_ID").MustString("")
		SourceControlAuthKeycloakSecret.AdminCliSecret = sec.Key("ADMIN_CLI_SECRET").MustString("")
		SourceControlAuthKeycloakSecret.AdminClientIDMasterRealm = sec.Key("ADMIN_CLIENT_ID_MASTER_REALM").MustString("")
		SourceControlAuthKeycloakSecret.StoragePath = sec.Key("STORAGE_PATH").MustString("")
		SourceControlAuthKeycloakSecret.SecretPath = sec.Key("SECRET_PATH").MustString("")
		SourceControlAuthKeycloakSecret.VersionKey = sec.Key("VERSION_KV").MustInt(0)
	}
}
