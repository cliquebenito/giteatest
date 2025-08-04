package vault_client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	vault_model "code.gitea.io/gitea/models/vault_client"
	"code.gitea.io/gitea/modules/log"
)

// wrapSecret перезаписывем token для unwrapping через http Client
func wrapSecret(client http.Client, config *vault_model.ConfigForWrapToken, clientToken, secretID string) (string, error) {
	secret := map[string]string{
		keyForWrappingUnwrappingSecret: secretID,
		keyRoleID:                      config.RoleID,
	}
	postBody, err := json.Marshal(secret)
	if err != nil {
		log.Error("Error has occurred while marshalling json: %v", err)
		return "", fmt.Errorf("marshal json: %w", err)
	}
	request, err := http.NewRequest("POST", config.URL+"/v1/"+config.Namespace+"/sys/wrapping/wrap", bytes.NewBuffer(postBody))
	if err != nil {
		log.Error("Error has occurred while creating request: %v", err)
		return "", fmt.Errorf("create request: %w", err)
	}
	request.Header.Set("X-Vault-Wrap-TTL", strconv.Itoa(config.TtlWrapToken))
	request.Header.Set("X-Vault-Token", clientToken)

	response, err := client.Do(request)
	if err != nil {
		if response != nil && response.StatusCode == http.StatusInternalServerError {
			var retryErr error
			response, retryErr = retry(client, request, 0)
			if retryErr != nil {
				log.Error("Error has occurred while getting response from secret storage on wrap secret request by timeout or statusCode = 500: %v", retryErr)
				return "", fmt.Errorf("get response from secret storage on wrap secret request by timeout or statusCode = 500: %w", retryErr)
			}
		} else {
			log.Error("Error has occurred while getting response from secret storage on wrap secret request: %v", err)
			return "", fmt.Errorf("get response from secret storage on wrap secret request: %w", err)
		}
	}
	defer response.Body.Close()

	byteBody, err := io.ReadAll(response.Body)
	if err != nil {
		log.Error("Error has occurred while reading response body: %v", err)
		return "", fmt.Errorf("read from response body: %w", err)
	}

	var vaultResponse vault_model.DefaultResponse
	if err = json.Unmarshal(byteBody, &vaultResponse); err != nil {
		log.Error("Error has occurred while unmarshalling json: %v", err)
		return "", fmt.Errorf("unmarshal json: %w", err)
	}
	return vaultResponse.WrapInfo.Token, nil
}
