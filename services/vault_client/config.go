package vault_client

import (
	"encoding/json"
	"fmt"
	"os"

	vault_model "code.gitea.io/gitea/models/vault_client"
	"code.gitea.io/gitea/modules/log"
)

// readWrappedSecretInFile получаем wrapped_secret_id из файла
func readWrappedSecretInFile(filename string) (string, error) {
	file, err := os.ReadFile(filename)
	if err != nil {
		log.Error("Error has occurred while reading from file '%s': %v", filename, err)
		return "", fmt.Errorf("read from file '%s': %w", filename, err)
	}

	var config vault_model.WrappedConfig
	err = json.Unmarshal(file, &config)
	if err != nil {
		log.Error("Error has occurred while unmarshalling json: %v", err)
		return "", fmt.Errorf("unmarshal json: %w", err)
	}

	return config.WrappedSecretID, nil
}

// updateWrappedSecretInFile обновляем wrapped_secret_id
func updateWrappedSecretInFile(filename, secret string) error {
	config := vault_model.WrappedConfig{WrappedSecretID: secret}
	payload, err := json.Marshal(config)
	if err != nil {
		log.Error("Error has occurred while marshalling json: %v", err)
		return fmt.Errorf("marshal json: %w", err)
	}

	if err = os.WriteFile(filename, payload, 0666); err != nil {
		log.Error("Error has occurred while writing wrapped secret to file: %v", err)
		return fmt.Errorf("write wrapped secret to file: %w", err)
	}
	return nil
}
