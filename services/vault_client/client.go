package vault_client

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	vault_model "code.gitea.io/gitea/models/vault_client"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/sbt/audit"
)

const (
	keyForWrappingUnwrappingSecret = "secret_id"
	keyRoleID                      = "role_id"
	maxRetries                     = 5
)

// errCanNotGetWrappedSecret ошибка при wrapping или unwrapping token
var errCanNotGetWrappedSecret = errors.New("err can not get wrapped secret")

// VaultClient сткутура клиента для получения cred
type VaultClient struct {
	URL         string
	Namespace   string
	ClientToken string
}

// autoReWrapCycle ticker для прогрева token
func autoReWrapCycle(ticker *time.Ticker, config *vault_model.ConfigForWrapToken) {
	var wrapError error
	for {
		select {
		case <-ticker.C:
			_, wrapError = autoReWrap(config)
			if wrapError != nil {
				log.Error("Error has occurred while trying to re-wrap secret: %v", wrapError)
				return
			}
			log.Info("autoReWrap token was executed success")
		}
	}
}

/*
autoReWrap функция для:
-получения wrap_secret_id из файла
-unwrapping token
-получения token для авторизации клиента
-wrapping нового token для unwrapping
-записываем новый token в файл, из которого его достали изначально
*/
func autoReWrap(config *vault_model.ConfigForWrapToken) (*VaultClient, error) {
	wrappedSecretId, err := readWrappedSecretInFile(config.WrappedTokenFile)
	if err != nil {
		log.Error("Error has occurred while reading wrapped secret from file: %v", err)
		return nil, fmt.Errorf("could not get wrapped secret from file: %w", err)
	}

	loginPayload, err := unwrapSecret(config.HttpClient, config, wrappedSecretId)
	if err != nil {
		log.Error("Error has occurred while unwrapping secret: %v", err)
		return nil, fmt.Errorf("unwrap secret: %w", err)
	}

	auditParams := map[string]string{
		"secret_name": "client authorization token",
	}
	clientToken, err := login(config.HttpClient, config, loginPayload)
	if err != nil {
		log.Error("Error has occurred while logging in in secret storage: %v", err)
		auditParams["error"] = "Failed to load client authorization token"
		audit.CreateAndSendEvent(
			audit.SecManApplySecretEvent,
			audit.EmptyRequiredField,
			audit.EmptyRequiredField,
			audit.StatusFailure,
			audit.EmptyRequiredField,
			auditParams,
		)
		return nil, fmt.Errorf("login in secret storage: %w", err)
	}
	audit.CreateAndSendEvent(
		audit.SecManApplySecretEvent,
		audit.EmptyRequiredField,
		audit.EmptyRequiredField,
		audit.StatusSuccess,
		audit.EmptyRequiredField,
		auditParams,
	)

	wrappedSecret, err := wrapSecret(config.HttpClient, config, clientToken, loginPayload.SecretID)
	if err != nil {
		log.Error("Error has occurred while wrapping secret: %v", err)
		return nil, fmt.Errorf("wrap secret: %w", err)
	}

	err = updateWrappedSecretInFile(config.WrappedTokenFile, wrappedSecret)
	if err != nil {
		log.Error("Error has occurred while updating wrapped secret: %v", err)
		return nil, fmt.Errorf("update wrapped secret: %w", err)
	}

	return &VaultClient{
		ClientToken: clientToken,
		URL:         config.URL,
		Namespace:   config.Namespace,
	}, nil
}

// GetClient функция получения клинета для хранилища секретов
func GetClient(config *vault_model.ConfigForWrapToken) (*VaultClient, *vault_model.ConfigForWrapToken, error) {
	periodTTL, err := time.ParseDuration(strconv.Itoa(config.PeriodTime) + "s")
	if err != nil {
		log.Error("Error has occurred while parsing time duration: %v", err)
		return nil, nil, fmt.Errorf("parse time duration: %w", err)
	}
	client, err := autoReWrap(config)
	if err != nil {
		log.Error("Error has occurred while trying to re-wrap secret: %v", err)
		return nil, nil, fmt.Errorf("re-wrap secret: %w", err)
	}
	ticker := time.NewTicker(periodTTL)

	go autoReWrapCycle(ticker, config)

	return client, config, nil
}
