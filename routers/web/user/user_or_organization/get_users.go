package user_or_organization

import (
	"fmt"
	"net/http"

	avatars_model "code.gitea.io/gitea/models/avatars"
	"code.gitea.io/gitea/models/organization"
	"code.gitea.io/gitea/models/role_model"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/routers/web/user/accesser"
)

// GetUsersByOrgName получаем всех пользователей по проекту
func (s Server) GetUsersByOrgName(ctx *context.Context) {
	orgName := ctx.Params("org")
	if orgName == "" {
		log.Warn("Organization name is empty")
		ctx.JSON(http.StatusBadRequest, "Organization name is empty")
		return
	}

	org, err := organization.GetOrgByName(ctx, orgName)
	if err != nil {
		log.Error("Error has occurred while getting organization by name: %v", err)
		ctx.JSON(http.StatusInternalServerError, fmt.Sprintf("Error has occurred while getting organization by name: %v", err))
		return
	}

	orgUsers, err := organization.GetOrgUsersByOrgID(ctx, &organization.FindOrgMembersOpts{OrgID: org.ID})
	if err != nil {
		log.Error("Error has occurred while getting users by organization id: %v", err)
		ctx.JSON(http.StatusInternalServerError, fmt.Sprintf("Error has occurred while getting users by organization id: %v", err))
		return
	}

	userIDs := make(map[int64]struct{}, len(orgUsers))
	for idx := range orgUsers {
		userIDs[orgUsers[idx].UID] = struct{}{}
	}

	teamName := ctx.Req.URL.Query().Get("team")
	if teamName != "" {
		tenantID, err := role_model.GetUserTenantId(ctx, ctx.Doer.ID)
		if err != nil {
			log.Error("Error has occurred while getting tenant id: %v", err)
			ctx.JSON(http.StatusInternalServerError, fmt.Sprintf("Error has occurred while getting tenant id: %v", err))
			return
		}

		teamIDs, err := organization.GetTeamIDsByNames(org.ID, []string{teamName}, false)
		if err != nil {
			log.Error("Error has occurred while getting team ids: %v", err)
			ctx.JSON(http.StatusInternalServerError, fmt.Sprintf("Error has occurred while getting team ids: %v", err))
			return
		}

		if len(teamIDs) == 0 {
			log.Error("Teams with a such name: %s weren't found", teamName)
			ctx.JSON(http.StatusNotFound, "Team not found")
			return
		}

		team, err := organization.GetTeamByID(ctx, teamIDs[0])
		if err != nil {
			log.Error("Error has occurred while getting team by name: %v", err)
			ctx.JSON(http.StatusInternalServerError, fmt.Sprintf("Error has occurred while getting team by name: %v", err))
			return
		}
		if team == nil {
			log.Error("Team's id: %d wasn't found", teamIDs[0])
			ctx.JSON(http.StatusNotFound, "Team not found")
			return
		}

		for idx := range orgUsers {
			if ok := s.repoRequestAccessor.CheckCustomPrivilegesForUser(accesser.RepoAccessRequest{
				DoerID:         orgUsers[idx].UID,
				OrgID:          org.ID,
				TargetTenantID: tenantID,
				Team:           team,
			}); ok {
				delete(userIDs, orgUsers[idx].UID)
			}
		}
	}

	users := make([]int64, 0, len(userIDs))
	for userID := range userIDs {
		users = append(users, userID)
	}

	if len(users) == 0 {
		ctx.JSON(http.StatusOK, nil)
		return
	}

	userList, err := user_model.GetUsersByIDs(users)
	if err != nil {
		log.Error("Error has occurred while getting users by users ids: %v", err)
		ctx.JSON(http.StatusInternalServerError, fmt.Sprintf("Error has occurred while getting users by users ids: %v", err))
		return
	}

	// добавляем аватарки пользователю
	for _, user := range userList {
		var avatarLink string

		if user.Avatar != "" {
			avatarLink = avatars_model.GenerateUserAvatarImageLink(user.Avatar, 0)
		} else {
			avatarLink = avatars_model.GenerateEmailAvatarFastLink(ctx, user.AvatarEmail, 0)
		}
		user.Avatar = avatarLink
	}
	ctx.JSON(http.StatusOK, userList)
}
