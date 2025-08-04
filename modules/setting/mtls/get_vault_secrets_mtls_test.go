//go:build !correct

package mtls

import (
	"strings"
	"testing"

	vault_model "code.gitea.io/gitea/models/vault_client"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/setting/mocks"

	"github.com/stretchr/testify/assert"
)

func TestGetMTLSCertsFromSecMan(t *testing.T) {
	setting.MTLSConnectionAvailable["task_tracker"] = &setting.MTLSConfig{
		StoragePath: "secret/path",
		SecretPath:  "task_tracker",
		VersionKey:  1,
		CaCertPath:  "ca_crt",
		CertPath:    "client_crt",
		KeyPath:     "client_key",
		Enabled:     true,
	}
	configMtls := setting.MTLSConnectionAvailable["task_tracker"]

	configForKvGet := &vault_model.KeyValueConfigForGetSecrets{
		SecretPath:  strings.TrimSpace(configMtls.SecretPath),
		StoragePath: strings.TrimSpace(configMtls.StoragePath),
		VersionKey:  configMtls.VersionKey,
	}

	getSecretMan := mocks.NewGetCredSecMan(t)

	getSecretMan.On("GetCredFromSecManByVersionKey", configForKvGet).Return(
		&vault_model.SecretVaultResponse{
			DefaultResponse: vault_model.DefaultResponse{},
			Data: map[string]string{
				"ca_crt":     "-----BEGIN CERTIFICATE-----\nMIIDQTCCAkeCCQ...snip...\n-----END CERTIFICATE-----\n",
				"client_crt": "-----BEGIN CERTIFICATE-----\nMIIDQTCCAkeCCQ...snip...\n-----END CERTIFICATE-----\n",
				"client_key": "-----BEGIN PRIVATE KEY-----\nMIIEpAIBAAKCAQE...snip...\n-----END PRIVATE KEY-----\n",
			},
		}, nil)

	mtlsClientCerts := GetMTLSCertsFromSecMan("task_tracker", getSecretMan)
	assert.NotNil(t, mtlsClientCerts)
}
