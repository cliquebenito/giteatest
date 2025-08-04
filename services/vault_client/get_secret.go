package vault_client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	vault_model "code.gitea.io/gitea/models/vault_client"
	"code.gitea.io/gitea/modules/log"
)

// KvV1Get получаем cred в виде key_value из хранилища секретов типа V1 или Default
func KvV1Get(config *VaultClient, client *vault_model.ConfigForWrapToken, configKvGet *vault_model.KeyValueConfigForGetSecrets) (*vault_model.SecretVaultResponse, error) {
	request, err := http.NewRequest("GET", config.URL+"/v1/"+config.Namespace+"/"+configKvGet.StoragePath+"/"+configKvGet.SecretPath, nil)
	if err != nil {
		log.Error("Error has occurred while creating request to secret storage v1: %v", err)
		return nil, fmt.Errorf("create request to secret storage v1: %w", err)
	}
	request.Header.Set("X-Vault-Token", config.ClientToken)

	response, err := client.HttpClient.Do(request)
	if err != nil {
		if response.StatusCode == http.StatusInternalServerError {
			var retryErr error
			response, retryErr = retry(client.HttpClient, request, 0)
			if retryErr != nil {
				log.Error("Error has occurred while getting response from secret storage, got 500 code instead: %v", retryErr)
				return nil, fmt.Errorf("get response from secret storage, got 500 code instead: %w", retryErr)
			}
		} else {
			log.Error("Error has occurred while receiving response from secret storage: %v", err)
			return nil, fmt.Errorf("receive response from secret storage: %w", err)
		}
	}

	defer response.Body.Close()

	byteBody, err := io.ReadAll(response.Body)
	if err != nil {
		log.Error("Error has occurred while reading response body: %v", err)
		return nil, fmt.Errorf("read response body: %w", err)
	}

	var secret vault_model.SecretVaultResponse
	err = json.Unmarshal(byteBody, &secret)
	if err != nil {
		log.Error("Error has occurred while unmarshalling json: %v", err)
		return nil, fmt.Errorf("unmarshal json: %w", err)
	}

	return &secret, nil
}

// KvV2Get получаем cred в виде key_value из хранилища секретов типа V2
func KvV2Get(config *VaultClient, client *vault_model.ConfigForWrapToken, configKvGet *vault_model.KeyValueConfigForGetSecrets) (*vault_model.SecretVaultResponse, error) {
	request, err := http.NewRequest("GET", config.URL+"/v1/"+config.Namespace+"/"+configKvGet.StoragePath+"/data/"+configKvGet.SecretPath, nil)
	if err != nil {
		log.Error("Error has occurred while creating request to secret storage v2: %v", err)
		return nil, fmt.Errorf("create request to secret storage v2: %w", err)
	}
	request.Header.Set("X-Vault-Token", config.ClientToken)

	response, err := client.HttpClient.Do(request)
	if err != nil {
		if response.StatusCode == http.StatusInternalServerError {
			var retryErr error
			response, retryErr = retry(client.HttpClient, request, 0)
			if retryErr != nil {
				log.Error("Error has occurred while getting response from secret storage, got 500 code instead: %v", retryErr)
				return nil, fmt.Errorf("get response from secret strage, got 500 code instead: %w", retryErr)
			}
		} else {
			log.Error("Error has occurred while receiving response from secret storage: %v", err)
			return nil, fmt.Errorf("receive response from secret storage: %w", err)
		}
	}

	defer response.Body.Close()

	byteBody, err := io.ReadAll(response.Body)
	if err != nil {
		log.Error("Error has occurred while reading response body: %v", err)
		return nil, fmt.Errorf("read response body: %w", err)
	}

	var secret vault_model.SecretVaultResponse
	if err = json.Unmarshal(byteBody, &secret); err != nil {
		log.Error("Error has occurred while unmarshalling json: %v", err)
		return nil, fmt.Errorf("unmarshal json: %w", err)
	}

	return &secret, nil
}
