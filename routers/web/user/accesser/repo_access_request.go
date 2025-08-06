package accesser

import "code.gitea.io/gitea/models/organization"

// RepoAccessRequest структура для проверки кастомных привилегий в репозиторий
type RepoAccessRequest struct {
	DoerID, RepoID, OrgID           int64
	TargetTenantID, CustomPrivilege string
	Team                            *organization.Team
}

// RepoCustomParamsRequest структура для проверки политик casbin
type RepoCustomParamsRequest struct {
	FieldIdx  int
	FieldName string
}
