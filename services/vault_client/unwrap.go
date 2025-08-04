package vault_client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	vault_model "code.gitea.io/gitea/models/vault_client"
	"code.gitea.io/gitea/modules/log"
)

// unwrapSecret получает secret_id и role_id для авторизации через unwrapped token
func unwrapSecret(client http.Client, config *vault_model.ConfigForWrapToken, wrappedToken string) (*vault_model.LoginPayload, error) {
	request, err := http.NewRequest("POST", config.URL+"/v1/"+config.Namespace+"/sys/wrapping/unwrap", nil)
	if err != nil {
		log.Error("Error has occurred while creating unwrap secret request: %v", err)
		return nil, fmt.Errorf("create unwrapping secret request: %w", err)
	}
	request.Header.Set("X-Vault-Token", wrappedToken)

	response, err := client.Do(request)
	if err != nil {
		if response != nil && response.StatusCode == http.StatusInternalServerError {
			var retryErr error
			response, retryErr = retry(client, request, 0)
			if retryErr != nil {
				log.Error("Error has occurred while making an unwrap secret request to secret storage: timeout or statusCode = 500: %v", retryErr)
				return nil, fmt.Errorf("unwrap secret request failed by timeout or statusCode = 500: %w", retryErr)
			}
		} else {
			log.Error("Error has occurred while making an unwrap secret request: %v", err)
			return nil, fmt.Errorf("unwrap secret request: %w", err)
		}
	}
	defer response.Body.Close()

	byteBody, err := io.ReadAll(response.Body)
	if err != nil {
		log.Error("Error has occurred while reading response body: %v", err)
		return nil, fmt.Errorf("read response body: %w", err)
	}
	var vaultResponse vault_model.UnwrapResponse
	if err = json.Unmarshal(byteBody, &vaultResponse); err != nil {
		log.Error("Error has occurred while unmarshalling json: %v", err)
		return nil, fmt.Errorf("unmarshal json: %w", err)
	}
	loginPayload := &vault_model.LoginPayload{
		RoleID:   vaultResponse.Data[keyRoleID],
		SecretID: vaultResponse.Data[keyForWrappingUnwrappingSecret],
	}
	config.RoleID = vaultResponse.Data[keyRoleID]
	if loginPayload.SecretID != "" && loginPayload.RoleID != "" {
		return loginPayload, nil
	}
	log.Error("Error has occurred while unwrapping secret: %v", errCanNotGetWrappedSecret)
	return nil, errCanNotGetWrappedSecret
}
