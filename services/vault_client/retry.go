package vault_client

import (
	"errors"
	"net/http"

	"code.gitea.io/gitea/modules/log"
)

var errRetrySecretStorageRequest = errors.New("max count of retries were sent to secret storage")

// retry функция для повторения запроса при ошибке возникшей со стороны sec man, максимальное количество попыток ограничена maxRetries
func retry(client http.Client, req *http.Request, currentCountRetries int) (*http.Response, error) {
	for currentCountRetries < maxRetries {
		resp, err := client.Do(req)
		if (resp != nil && resp.StatusCode == http.StatusInternalServerError) || err != nil {
			log.Error("Error has occurred while making %v try of request: %s", currentCountRetries+1, req.URL.RequestURI())
		} else {
			log.Info("Request to secret storage was successful")
			return resp, nil
		}
		currentCountRetries++
	}
	log.Error("Error has occurred while making request to secret storage: max number of retries exceeded")
	return nil, errRetrySecretStorageRequest
}
