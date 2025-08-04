package setting

import (
	"fmt"

	vault_model "code.gitea.io/gitea/models/vault_client"
	"code.gitea.io/gitea/modules/sbt/audit"
	"code.gitea.io/gitea/services/vault_client"
)

// CheckSettingsForIntegrationWithSecMan проверка включения режима для ингреции с sec man
func CheckSettingsForIntegrationWithSecMan() bool {
	return SourceControl.Enabled && SourceControlWrapVault.Enabled && SourceControlVaultClient != nil && SourceControlVaultClient.ClientToken != ""
}

type getCredForSecMan struct {
}

func NewGetterForSecMan() getCredForSecMan {
	return getCredForSecMan{}
}

// GetCredFromSecManByVersionKey получение cred из sec man по версии хранилища
func (g getCredForSecMan) GetCredFromSecManByVersionKey(configForKvGet *vault_model.KeyValueConfigForGetSecrets) (*vault_model.SecretVaultResponse, error) {
	var resp *vault_model.SecretVaultResponse
	var errGetKV error
	switch configForKvGet.VersionKey {
	case 2:
		resp, errGetKV = vault_client.KvV2Get(SourceControlVaultClient, SourceControlHttpClient, configForKvGet)
		if errGetKV != nil {
			return nil, fmt.Errorf("get cred from SecMan v2: %w", errGetKV)
		}
	default:
		resp, errGetKV = vault_client.KvV1Get(SourceControlVaultClient, SourceControlHttpClient, configForKvGet)
		if errGetKV != nil {
			return nil, fmt.Errorf("get cred from SecMan v1: %w", errGetKV)
		}
	}
	return resp, nil
}

// GetResponseNotNil проверка ответа и поле data в нем на nil
func GetResponseNotNil(resp *vault_model.SecretVaultResponse) bool {
	return resp != nil && resp.Data != nil
}

func CheckIfSecretIsEmptyAndReportToAudit(secretName string, secretValue string, errMsg string, log func(format string, v ...any)) {
	auditParams := map[string]string{
		"secret_name": secretName,
	}
	auditStatus := audit.StatusSuccess

	if secretValue == "" {
		auditParams["error"] = errMsg
		auditStatus = audit.StatusFailure
		defer log(errMsg)
	}

	audit.CreateAndSendEvent(
		audit.SecManReadSecretEvent,
		audit.EmptyRequiredField,
		audit.EmptyRequiredField,
		auditStatus,
		audit.EmptyRequiredField,
		auditParams,
	)
}
