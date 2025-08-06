package models

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"code.gitea.io/gitea/modules/json"
	"code.gitea.io/gitea/routers/api"
)

// TrimmedString автоматически обрезает пробелы при JSON-маршалинге
type TrimmedString string

func (s TrimmedString) String() string {
	return string(s)
}

func (s *TrimmedString) UnmarshalJSON(b []byte) error {
	var str string
	if err := json.Unmarshal(b, &str); err != nil {
		return err
	}
	*s = TrimmedString(strings.TrimSpace(str))
	return nil
}

// CreateOrUpdateSonarProjectRequest — запрос на создание или обновление настроек Sonar
// swagger:model
type CreateOrUpdateSonarProjectRequest struct {
	// URL of the SonarQube server (must start with http or https)
	// required: true
	// example: https://sonarqube.example.com
	SonarServerURL TrimmedString `json:"sonar_server_url" validate:"required,max=2048,prefix=http"`

	// Unique project key in SonarQube
	// required: true
	// example: my-project-key
	SonarProjectKey TrimmedString `json:"sonar_project_key" validate:"required,max=50"`

	// Token used for authentication with SonarQube
	// required: true
	// example: your-secret-token
	SonarToken TrimmedString `json:"sonar_token" validate:"required,min=40,max=255"`
}

func (c CreateOrUpdateSonarProjectRequest) Validate() error {
	var errors []string

	if err := api.RequestValidator(c); err != nil {
		if verrs, ok := err.(api.ValidationErrors); ok {
			errors = append(errors, verrs.Errors...)
		} else {
			errors = append(errors, err.Error())
		}
	}

	if len(errors) > 0 {
		return api.ValidationErrors{Errors: errors}
	}

	if _, parseErr := url.ParseRequestURI(c.SonarServerURL.String()); parseErr != nil {
		errors = append(errors, fmt.Sprintf("поле 'sonar_server_url' является недопустимым URL: %v", parseErr))
	}

	if !regexp.MustCompile(`^[a-zA-Z0-9\-._:]+$`).MatchString(c.SonarProjectKey.String()) {
		errors = append(errors, "поле 'sonar_project_key' содержит недопустимые символы")
	}

	if !regexp.MustCompile(`^[a-zA-Z0-9\-._:]+$`).MatchString(c.SonarToken.String()) {
		errors = append(errors, "поле 'sonar_token' содержит недопустимые символы")
	}

	if len(errors) > 0 {
		return api.ValidationErrors{Errors: errors}
	}
	return nil
}
