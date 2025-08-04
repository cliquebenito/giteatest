package domain

import (
	user_model "code.gitea.io/gitea/models/user"
)

type CreateOrUpdateSonarProjectRequest struct {
	SonarServerURL  string
	SonarToken      string
	SonarProjectKey string
	RepoId          int64
	TenantKey       string
	Project         *user_model.User
}

type SonarSettingsResponse struct {
	SonarServerURL  string `json:"sonar_server_url" `
	SonarProjectKey string `json:"sonar_project_key" `
	SonarToken      string `json:"sonar_token" `
}
