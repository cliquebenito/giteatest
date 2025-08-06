package custom_creator

import (
	"context"
	"fmt"
	"strconv"

	"code.gitea.io/gitea/models/organization"
	"code.gitea.io/gitea/models/role_model"
	"code.gitea.io/gitea/modules/log"
	org_service "code.gitea.io/gitea/services/org"
)

// CreateOrDeleteCustomPrivileges добавляем или удаляем кастомные привилегии для команды
func (c *customCreator) CreateOrDeleteCustomPrivileges(ctx context.Context, customPrivileges ConfCustomPrivileges) error {
	newCustomprivileges := make([]role_model.CustomPrivilege, 0, len(customPrivileges.CustomPrivilegesRequest))
	for _, v := range customPrivileges.CustomPrivilegesRequest {
		if v != 0 {
			newCustomprivileges = append(newCustomprivileges, v)
		}
	}
	if len(newCustomprivileges) == 0 {
		return fmt.Errorf("incorrect privileges request")
	}
	customPrivileges.CustomPrivilegesRequest = newCustomprivileges

	projectID, err := strconv.ParseInt(customPrivileges.ProjectID, 10, 64)
	if err != nil {
		log.Error("Error has occurred while trying to parse project to int: %v", err)
		return fmt.Errorf("parsing org id: %w", err)
	}

	teamIDs, err := organization.GetTeamIDsByNames(projectID, []string{customPrivileges.TeamName}, false)
	if err != nil {
		log.Error("Error has occurred while getting team id by team's name: %v", err)
		return fmt.Errorf("getting team by name: %w", err)
	}
	if len(teamIDs) == 0 {
		log.Error("Error has occurred while getting team id by team's name: no team found")
		return fmt.Errorf("no team found")
	}

	team, err := organization.GetTeamByID(ctx, teamIDs[0])
	if err != nil {
		log.Error("Error has occurred while getting team by id: %v", err)
		return fmt.Errorf("getting team by id: %w", err)
	}

	// все репозитории сравнить с количеством в проекте
	// includesOldRepo сравнивать перед апдейтом

	if err = c.checkPolicyAlreadyExists(customPrivileges.NamePolicy,
		getCustomPrivilegesNotExistsRequest(customPrivileges.CustomPrivilegesRequest)); err != nil {
		log.Error("Error has occurred while checking policy existence: %v", err)
		return fmt.Errorf("checking existing custom privileges: %w", err)
	}

	newPolicies := make([][]string, 0)
	for _, gitRepository := range customPrivileges.Repos {
		customPrivileges.Repository = gitRepository
		errAddTeamRepository := org_service.TeamAddRepository(team, gitRepository)
		if errAddTeamRepository != nil {
			log.Error("Error has occurred while adding team to repository: %v", err)
			return fmt.Errorf("adding repository to team: %w", errAddTeamRepository)
		}
		newPolicies = append(newPolicies, []string{
			customPrivileges.TeamName,
			customPrivileges.ProjectID,
			strconv.Itoa(int(gitRepository.ID)),
			customPrivileges.NamePolicy})
	}

	if err := c.UpdateCustomPrivileges(newPolicies); err != nil {
		log.Error("Error has occurred while updating custom privileges: %v", err)
		return fmt.Errorf("updating custom privileges: %w", err)
	}

	return nil
}

func (c *customCreator) RemoveCustomPrivilegesByTeam(ctx context.Context, teamName string) error {
	if err := c.RemoveCustomPrivileges(teamName); err != nil {
		log.Error("Error has occurred while removing custom privileges: %v", err)
		return fmt.Errorf("removing custom privileges: %w", err)
	}

	return nil
}

// checkPolicyAlreadyExists проверяем привилегии, которые уже назначены
func (c *customCreator) checkPolicyAlreadyExists(policyName string, customPrivileges []string) error {
	has := c.CheckGroupingPolicy(0, policyName)
	if !has {
		if err := c.CreateCustomGroupingPrivileges(
			policyName,
			customPrivileges,
		); err != nil {
			log.Error("Error has occurred while creating custom grouping privileges: %v", err)
			return fmt.Errorf("creating grouping policies: %w", err)
		}
	}
	return nil
}

func getCustomPrivilegesNotExistsRequest(customPrivileges []role_model.CustomPrivilege) []string {

	customPrivilegesExists := make([]string, 0, len(role_model.Privileges))
	for _, privilege := range customPrivileges {
		customPrivilegesExists = append(customPrivilegesExists, role_model.Privileges[privilege])
	}

	return customPrivilegesExists
}
