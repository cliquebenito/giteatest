package custom_creator

import (
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/role_model"
	"code.gitea.io/gitea/models/role_model/custom_casbin_role_manager"
)

type customPrivilegeCreator interface {
	CheckGroupingPolicy(fieldIndex int, fieldName string) bool
	UpdateCustomPrivileges(newPolicies [][]string) error
	CreateCustomGroupingPrivileges(name string, groupingPolicies []string) error
	GrantCustomPrivilegeTeamUser(privilege custom_casbin_role_manager.GrantCustomPrivilege) error
	RemoveCustomPrivileges(teamName string) error
}

type customCreator struct {
	customPrivilegeCreator
}

func NewCustomCreator(customPrivilegeCreator customPrivilegeCreator) *customCreator {
	return &customCreator{customPrivilegeCreator: customPrivilegeCreator}
}

// ConfCustomPrivileges структура для добавления кастномных привилегий к команде
type ConfCustomPrivileges struct {
	Repos                   []*repo_model.Repository
	TeamName                string
	ProjectID               string
	CustomPrivilegesRequest []role_model.CustomPrivilege
	NamePolicy              string
	Repository              *repo_model.Repository
	BranchName              string
}
