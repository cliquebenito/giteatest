package setting

import "strings"

// MTLSConfig конфиг для получения creds из Sec Man
type MTLSConfig struct {
	// StoragePath путь к директории с secrets в vault хранилище
	StoragePath string `json:"storagePath"`
	// SecretPaths пути к директориям в vault хранилище
	SecretPath string `json:"secretPath"`
	// CaCertPath путь к сертификату CA
	CaCertPath string `json:"caCertPath"`
	// CertPath путь к клиентскому сертификату
	CertPath string `json:"certPath"`
	// KeyPath путь к клиентскому ключу
	KeyPath string `json:"keyPath"`
	// VersionKey версия получения cred
	VersionKey int `json:"versionKey"`
	// Enabled включение mtls
	Enabled bool `json:"enabled"`
}

// MTLSConnectionAvailable map с названия клиента для установления mtls-connection и creds для получения данных из Sec Man
var MTLSConnectionAvailable = map[string]*MTLSConfig{}

// SourceControlSecretMTLS установка mtls-connection
var SourceControlSecretMTLS struct {
	// MTLSEnabled включения режима для установки mtls-connection
	MTLSEnabled bool
}

func loadSourceControlVaultMTLS(rootCgf ConfigProvider) {
	if SourceControl.Enabled && SourceControlWrapVault.Enabled {
		sec := rootCgf.Section("sourcecontrol.vault.mtls")
		SourceControlSecretMTLS.MTLSEnabled = sec.Key("MTLS_ENABLED").MustBool(false)
		if SourceControlSecretMTLS.MTLSEnabled {
			for _, childSection := range sec.ChildSections() {
				key := strings.TrimPrefix(childSection.Name(), "sourcecontrol.vault.mtls.")
				mtlsConfig := &MTLSConfig{
					Enabled:     childSection.Key("ENABLED").MustBool(false),
					StoragePath: childSection.Key("STORAGE_PATH").MustString(""),
					SecretPath:  childSection.Key("SECRET_PATH").MustString(""),
					CaCertPath:  childSection.Key("CA_CERT_PATH").MustString(""),
					CertPath:    childSection.Key("CERT_PATH").MustString(""),
					KeyPath:     childSection.Key("KEY_PATH").MustString(""),
					VersionKey:  childSection.Key("VERSION_KEY").MustInt(0),
				}
				MTLSConnectionAvailable[key] = mtlsConfig
			}
		}
	}
}
