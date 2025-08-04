package custom_creator

import (
	"context"
	"fmt"

	"code.gitea.io/gitea/models"
	org_model "code.gitea.io/gitea/models/organization"
	"code.gitea.io/gitea/models/role_model/custom_casbin_role_manager"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/log"
)

// AddUserToTeam добавляем пользователя в команду с добавлением кастномных привилегий
func (c *customCreator) AddUserToTeam(ctx context.Context, teamID, orgID int64, tenantID string, userIDs []int64) error {
	team, err := org_model.GetTeamByID(ctx, teamID)
	if err != nil {
		log.Error("Error has occurred while getting team by ID: %v", err)
		return fmt.Errorf("get team by id: %w", err)
	}

	for _, userID := range userIDs {
		if err := models.AddTeamMember(team, userID); err != nil {
			log.Error("Error has occurred while adding team user: %v", err)
			return fmt.Errorf("add member to team: %w", err)
		}

		if err := c.GrantCustomPrivilegeTeamUser(custom_casbin_role_manager.GrantCustomPrivilege{
			User:     &user_model.User{ID: userID},
			Org:      &org_model.Organization{ID: orgID},
			Team:     team,
			TenantID: tenantID,
		}); err != nil {
			log.Error("Error has occurred while grant custom privileges for user and team: %v", err)
			return fmt.Errorf("grant custom priviges: %w", err)
		}
	}
	return nil
}
