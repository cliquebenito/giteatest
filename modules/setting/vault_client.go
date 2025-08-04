package setting

import (
	vault_model "code.gitea.io/gitea/models/vault_client"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/services/vault_client"
	"sync"
)

// KeyValueTypeForSecMan specifies type for getting cred
type KeyValueTypeForSecMan int //revive:disable-line:exported

const (
	TypeKeyValueDefault KeyValueTypeForSecMan = iota // 0
	// TypeKeyValueUnknownFirst KeyValue with type 1
	TypeKeyValueUnknownFirst
	// TypeKeyValueUnknownSecond KeyValue with type 2
	TypeKeyValueUnknownSecond
)

// SourceControlVaultClient VaultClient для авторизации в Sec Man
var SourceControlVaultClient *vault_client.VaultClient

// SourceControlHttpClient http client для выполнения запросов в Sec Man и получения секретов
var SourceControlHttpClient *vault_model.ConfigForWrapToken
var once sync.Once

// loadSourceControlVaultClient получаем VaultClient для изъятия cred
func loadSourceControlVaultClient() {
	if SourceControlWrapVault.Enabled && SourceControl.Enabled {
		var err error
		config := &vault_model.ConfigForWrapToken{
			URL:              SourceControlWrapVault.UrlSecMan,
			Namespace:        SourceControlWrapVault.NameTenantSecManWrap,
			WrappedTokenFile: SourceControlWrapVault.WrapTokenPath,
			PeriodTime:       SourceControlWrapVault.PeriodWrappingToken,
			TtlWrapToken:     SourceControlWrapVault.TtlWrapToken,
		}
		once.Do(func() {
			SourceControlVaultClient, SourceControlHttpClient, err = vault_client.GetClient(config)
			if err != nil {
				log.Error("loadSourceControlVaultClient vault_client.GetClient failed: %v", err)
			}
		})
	}
}
