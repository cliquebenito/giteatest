package custom_creator

import (
	"context"
	"fmt"
	"strconv"

	org_model "code.gitea.io/gitea/models/organization"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/role_model"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/services/forms"
)

// UpdateCustomPrivilegeToTeamUser обновляем информацию о кастомных привилегиях в команде
func (c *customCreator) UpdateCustomPrivilegeToTeamUser(ctx context.Context, ordID int64, teamName string, customPrivilegeForm []forms.CustomPrivileges) error {
	if err := c.RemoveCustomPrivileges(teamName); err != nil {
		log.Error("Error has occurred while removing custom privileges: %v", err)
		return fmt.Errorf("removing custom privileges: %w", err)
	}

	for _, customPrivilege := range customPrivilegeForm {
		if len(customPrivilege.Privileges) == 0 {
			continue
		}
		confCustomPrivilege := ConfCustomPrivileges{
			TeamName:  teamName,
			ProjectID: strconv.Itoa(int(ordID)),
		}
		confCustomPrivilege.CustomPrivilegesRequest = customPrivilege.Privileges

		switch {
		case customPrivilege.AllRepositories:
			repositories, errGetOrgRepositories := org_model.GetOrgRepositories(ctx, ordID)
			if errGetOrgRepositories != nil {
				log.Error("Error has occurred while getting repositories by org_id: %v", errGetOrgRepositories)
				return fmt.Errorf("getting repositories by org: %w", errGetOrgRepositories)
			}
			confCustomPrivilege.Repos = repositories

		default:
			mapGitRepositories, errGetRepositoriesByID := repo_model.GetRepositoriesMapByIDs([]int64{customPrivilege.RepoID})
			if errGetRepositoriesByID != nil {
				log.Error("Error has occurred while getting repositories to map ids by repository_id: %v", errGetRepositoriesByID)
				return fmt.Errorf("getting repository: %w", errGetRepositoriesByID)
			}
			repo := mapGitRepositories[customPrivilege.RepoID]
			confCustomPrivilege.Repos = []*repo_model.Repository{repo}

		}

		confCustomPrivilege.NamePolicy = role_model.ConvertCustomPrivilegeToNameOfPolicy(customPrivilege.Privileges)
		if err := c.CreateOrDeleteCustomPrivileges(ctx, confCustomPrivilege); err != nil {
			log.Error("Error has occurred while creating or removing custom privileges: %v", err)
			return fmt.Errorf("adding custom privileges: %w", err)
		}
	}
	return nil
}
