package custom_casbin_role_manager

import (
	"code.gitea.io/gitea/models/organization"
	user_model "code.gitea.io/gitea/models/user"
)

// ConfCustomPrivilege структура для проверки доступа пользователя
type ConfCustomPrivilege struct {
	User                      *user_model.User
	Org                       *organization.Organization
	RepoID                    int64
	CustomPrivilege, TenantID string
}

// GrantCustomPrivilege структура для назначения пользователю кастомных привилегий
type GrantCustomPrivilege struct {
	User                      *user_model.User
	Org                       *organization.Organization
	Team                      *organization.Team
	RepoID                    int64
	CustomPrivilege, TenantID string
}
