package repo

import (
	"fmt"
	"regexp"
	"strings"
)

var nameBranchRegex = regexp.MustCompile("^[a-zA-Z0-9_.-]+$")

// Repository model for API v2 get response
// swagger:response repositoryGetResponse
type RepositoryGetResponse struct {
	ID            string `json:"id"`
	TenantKey     string `json:"tenant_key"`
	ProjectKey    string `json:"project_key"`
	RepositoryKey string `json:"repository_key"`
	Name          string `json:"name"`
	DefaultBranch string `json:"default_branch"`
	Private       bool   `json:"private"`
	URI           string `json:"uri"`
}

// RepositoryGetOptions модель для запроса репозитория
type RepositoryGetOptions struct {
	RepoKey    string `json:"repo_key"`
	TenantKey  string `json:"tenant_key"`
	ProjectKey string `json:"project_key"`
}

// Repository model for API v2 post response
// swagger:response repositoryPostResponse
type RepositoryPostResponse struct {
	ID            string `json:"id"`
	TenantKey     string `json:"tenant_key"`
	ProjectKey    string `json:"project_key"`
	RepositoryKey string `json:"repository_key"`
	DefaultBranch string `json:"default_branch"`
	Name          string `json:"name"`
	Private       bool   `json:"private"`
	URI           string `json:"uri"`
}

// CreateRepoOptions represents options to create repo
type CreateRepoOptions struct {
	TenantKey     string `json:"tenant_key" binding:"Required"`
	ProjectKey    string `json:"project_key" binding:"Required"`
	RepositoryKey string `json:"repository_key" binding:"Required;MaxSize(255)"`
	DefaultBranch string `json:"default_branch" binding:"SbtMaxSize(100)"`
	Description   string `json:"description" binding:"MaxSize(255)"`
	Name          string `json:"name" binding:"Required;MaxSize(100)"`
	Private       *bool  `json:"private" binding:"Required"`
}

func (o *CreateRepoOptions) Validate() error {
	if strings.Contains(o.TenantKey, " ") {
		return fmt.Errorf("tenant key is not valid")
	}
	if strings.Contains(o.ProjectKey, " ") {
		return fmt.Errorf("project key is not valid")
	}
	if strings.Contains(o.RepositoryKey, " ") {
		return fmt.Errorf("repository key is not valid")
	}
	if !nameBranchRegex.MatchString(o.DefaultBranch) {
		return fmt.Errorf("default branch name is not valid")
	}
	if !nameBranchRegex.MatchString(o.Name) {
		return fmt.Errorf("repository name is not valid")
	}
	return nil
}

// SetMarkRequest - структура ручки для проставления метки репозитория
// swagger:model SetMarkRequest
type SetMarkRequest struct {
	TenantKey  string `json:"tenant_key" binding:"Required"`
	RepoKey    string `json:"repo_key" binding:"Required"`
	ProjectKey string `json:"project_key" binding:"Required"`
}

func (s SetMarkRequest) Validate() error {
	if strings.Contains(s.TenantKey, " ") {
		return fmt.Errorf("tenant key is not valid")
	}
	if strings.Contains(s.RepoKey, " ") {
		return fmt.Errorf("repository key is not valid")
	}
	if strings.Contains(s.ProjectKey, " ") {
		return fmt.Errorf("project key is not valid")
	}
	return nil
}

// DeleteMarkRequest - структура ручки для удаления метки репозитория
// swagger:model DeleteMarkRequest
type DeleteMarkRequest struct {
	TenantKey  string `json:"tenant_key" binding:"Required"`
	RepoKey    string `json:"repo_key" binding:"Required"`
	ProjectKey string `json:"project_key" binding:"Required"`
}

func (s DeleteMarkRequest) Validate() error {
	if strings.Contains(s.TenantKey, " ") {
		return fmt.Errorf("tenant key is not valid")
	}
	if strings.Contains(s.RepoKey, " ") {
		return fmt.Errorf("repository key is not valid")
	}
	if strings.Contains(s.ProjectKey, " ") {
		return fmt.Errorf("project key is not valid")
	}
	return nil
}
