package vault_client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	vault_model "code.gitea.io/gitea/models/vault_client"
	"code.gitea.io/gitea/modules/log"
)

// login авторизация в хранилище секретов по appRole, получаем токен
func login(client http.Client, config *vault_model.ConfigForWrapToken, loginPayload *vault_model.LoginPayload) (string, error) {
	postBody, err := json.Marshal(loginPayload)
	if err != nil {
		log.Error("Error has occurred while marshalling json: %v", err)
		return "", fmt.Errorf("marshal json: %w", err)
	}

	request, err := http.NewRequest("POST", config.URL+"/v1/"+config.Namespace+"/auth/approle/login", bytes.NewBuffer(postBody))
	if err != nil {
		log.Error("Error has occurred while creating login request: %v", err)
		return "", fmt.Errorf("create login request: %w", err)
	}

	response, err := client.Do(request)
	if err != nil {
		if response != nil && response.StatusCode == http.StatusInternalServerError {
			var retryErr error
			response, retryErr = retry(client, request, 0)
			if retryErr != nil {
				log.Error("Error has occurred while making a login request to secret storage: timeout or statusCode = 500: %v", retryErr)
				return "", fmt.Errorf("login request to secret storage failed by timeout or statusCode = 500: %w", retryErr)
			}
		} else {
			log.Error("Error has occurred while making login request to secret storage: %v", err)
			return "", fmt.Errorf("login request to secret storage: %w", err)
		}
	}
	defer response.Body.Close()

	byteBody, err := io.ReadAll(response.Body)
	if err != nil {
		log.Error("Error has occurred while reading secret storage response: %v", err)
		return "", fmt.Errorf("read secret storage response: %w", err)
	}

	var vaultResponse vault_model.DefaultResponse
	if err = json.Unmarshal(byteBody, &vaultResponse); err != nil {
		log.Error("Error has occurred while unmarshalling json: %v", err)
		return "", fmt.Errorf("unmarshal json: %w", err)
	}

	return vaultResponse.Auth.ClientToken, nil
}
