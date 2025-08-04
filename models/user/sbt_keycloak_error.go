package user

import (
	"fmt"
	"net/http"
)

// ErrKeycloakWrongHttpStatus ошибка в случае если получен неверный статус ответа на запрос в Keycloak
type ErrKeycloakWrongHttpStatus struct {
	StatusCode int
	Request    *http.Request
}

// IsErrKeycloakWrongHttpStatus checks if an error is a ErrKeycloakWrongHttpStatus.
func IsErrKeycloakWrongHttpStatus(err error) bool {
	_, ok := err.(ErrKeycloakWrongHttpStatus)
	return ok
}

func (err ErrKeycloakWrongHttpStatus) Error() string {
	return fmt.Sprintf("Keycloak wrong response http StatusCode: %d on request: %v", err.StatusCode, &err.Request)
}

// ErrKeycloakWrongHttpRequest ошибка в случае если получена ошибка в процессе отправки запроса в Keycloak
type ErrKeycloakWrongHttpRequest struct {
	ReasonErr error
}

// IsErrKeycloakWrongHttpRequest checks if an error is a ErrKeycloakWrongHttpRequest.
func IsErrKeycloakWrongHttpRequest(err error) bool {
	_, ok := err.(ErrKeycloakWrongHttpRequest)
	return ok
}

func (err ErrKeycloakWrongHttpRequest) Error() string {
	return fmt.Sprintf("Keycloak wrong http request: error %v", err.ReasonErr)
}
