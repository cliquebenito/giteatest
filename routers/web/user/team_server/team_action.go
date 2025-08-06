package team_server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"code.gitea.io/gitea/models"
	org_model "code.gitea.io/gitea/models/organization"
	"code.gitea.io/gitea/models/role_model"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/sbt/audit"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/routers/utils"
	"code.gitea.io/gitea/routers/web/user/accesser"
	org_service "code.gitea.io/gitea/services/org"
)

const (
	actionChoose = 1
)

// TeamsAction response for join, leave, remove, add operations to team
func (s *Server) TeamsAction(ctx *context.Context) {
	page := ctx.FormString("page")

	type auditValue struct {
		AccessMode              string
		RepositoryIds           string
		IncludesAllRepositories bool
		CanCreateOrgRepo        bool
		AccessModeForTypes      map[string]string
	}

	team := ctx.Org.Team

	auditParams := map[string]string{
		"project":    ctx.Org.Organization.Name,
		"project_id": strconv.FormatInt(ctx.Org.Organization.ID, 10),
		"team":       team.Name,
		"team_id":    strconv.FormatInt(team.ID, 10),
	}

	_ = team.LoadUnits(ctx)
	_ = team.LoadRepositories(ctx)

	rightsAuditValue := auditValue{
		AccessMode:              team.AccessMode.String(),
		IncludesAllRepositories: team.IncludesAllRepositories,
		CanCreateOrgRepo:        team.CanCreateOrgRepo,
		AccessModeForTypes:      make(map[string]string),
	}

	if team.Repos != nil {
		var repoIds []string
		for _, repo := range team.Repos {
			repoIds = append(repoIds, strconv.FormatInt(repo.ID, 10))
		}
		rightsAuditValue.RepositoryIds = strings.Join(repoIds, ",")
	}

	if team.Units != nil {
		for _, unit := range team.Units {
			rightsAuditValue.AccessModeForTypes[unit.Type.String()] = unit.AccessMode.String()
		}
	}
	rightsAuditValueBytes, _ := json.Marshal(rightsAuditValue)
	uid := ctx.FormInt64("uid")
	tenantID, errGetTenantID := role_model.GetUserTenantId(ctx, uid)
	if errGetTenantID != nil {
		log.Error("Error has occurred while getting tenant_id by user_id: %v", errGetTenantID)
		ctx.Error(http.StatusNotFound, fmt.Sprintf("Error has occurred while getting tenant_id by user_id: %v", errGetTenantID))
		return
	}
	var events []audit.Event
	// получаем метод для изменения положения пользователя в команде
	action := strings.Split(ctx.Link, "/action/")
	if len(action) < 1 {
		log.Error("Error has occurred while splitting action we receive action which is empty")
		ctx.Error(http.StatusBadRequest, "action is empty")
		return
	}
	var err error
	switch action[actionChoose] {
	case "join":
		events = append(events, audit.UserAddToProjectTeamEvent, audit.ProjectTeamRightsGrantedEvent)
		auditParams["new_value"] = string(rightsAuditValueBytes)
		if !ctx.Org.IsOwner {
			ctx.Error(http.StatusNotFound)
			auditParams["error"] = "User is not the owner"
			for _, event := range events {
				audit.CreateAndSendEvent(event, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			}
			return
		}
		err = models.AddTeamMember(team, ctx.Doer.ID)
	case "leave":
		events = append(events, audit.UserRemoveFromProjectTeamEvent, audit.ProjectTeamRightsRemoveEvent)
		auditParams["old_value"] = string(rightsAuditValueBytes)
		err = models.RemoveTeamMember(team, ctx.Doer.ID)
		if err != nil {
			if org_model.IsErrLastOrgOwner(err) {
				ctx.Flash.Error(ctx.Tr("form.last_org_owner"))
				auditParams["error"] = "Cannot remove the last user from the 'owners' team"
			} else {
				log.Error("Action(%s): %v", ctx.Params(":action"), err)
				ctx.JSON(http.StatusOK, map[string]interface{}{
					"ok":  false,
					"err": err.Error(),
				})
				auditParams["error"] = "Error has occurred while removing team member"
				for _, event := range events {
					audit.CreateAndSendEvent(event, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
				}
				return
			}
		}

		redirect := ctx.Org.OrgLink + "/teams/"
		if isOrgMember, err := org_model.IsOrganizationMember(ctx, ctx.Org.Organization.ID, ctx.Doer.ID); err != nil {
			ctx.ServerError("IsOrganizationMember", err)
			auditParams["error"] = "Error has occurred while checking organization member"
			for _, event := range events {
				audit.CreateAndSendEvent(event, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			}
			return
		} else if !isOrgMember {
			redirect = setting.AppSubURL + "/"
		}
		for _, event := range events {
			audit.CreateAndSendEvent(event, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)
		}
		ctx.JSON(http.StatusOK,
			map[string]interface{}{
				"redirect": redirect,
			})
		return
	case "remove":
		events = append(events, audit.UserRemoveFromProjectTeamEvent, audit.ProjectTeamRightsRemoveEvent)
		auditParams["old_value"] = string(rightsAuditValueBytes)
		auditParams["affected_user_id"] = strconv.FormatInt(ctx.FormInt64("uid"), 10)
		if !ctx.Org.IsOwner {
			ctx.Error(http.StatusNotFound)
			auditParams["error"] = "User is not the owner"
			for _, event := range events {
				audit.CreateAndSendEvent(event, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			}
			return
		}

		uid := ctx.FormInt64("uid")
		if uid == 0 {
			ctx.Redirect(ctx.Org.OrgLink + "/teams")
			auditParams["error"] = "User id is empty"
			for _, event := range events {
				audit.CreateAndSendEvent(event, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			}
			return
		}

		err = models.RemoveTeamMember(team, uid)
		if err != nil {
			if org_model.IsErrLastOrgOwner(err) {
				ctx.Flash.Error(ctx.Tr("form.last_org_owner"))
				auditParams["error"] = "Cannot remove the last user from the 'owners' team"
			} else {
				log.Error("Action(%s): %v", ctx.Params(":action"), err)
				ctx.JSON(http.StatusOK, map[string]interface{}{
					"ok":  false,
					"err": err.Error(),
				})
				auditParams["error"] = "Error has occurred while removing team member"
				for _, event := range events {
					audit.CreateAndSendEvent(event, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
				}
				return
			}
		}

		if err := s.repoRequestAccessor.RemoveCustomPrivilege(accesser.RepoAccessRequest{
			DoerID:         uid,
			TargetTenantID: tenantID,
			OrgID:          ctx.Org.Organization.ID,
			Team:           team,
		}); err != nil {
			log.Error("TeamsAction role_model.RemoveUserFromTeamCustomPrivilege failed while removing user's custom privileges from team : %v", err)
			ctx.Error(http.StatusNotFound, fmt.Sprintf("TeamsAction role_model.RemoveUserFromTeamCustomPrivilege failed: %v", err))
			return
		}

		for _, event := range events {
			audit.CreateAndSendEvent(event, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)
		}
		ctx.JSON(http.StatusOK,
			map[string]interface{}{
				"redirect": ctx.Org.OrgLink + "/teams/" + url.PathEscape(team.LowerName),
			})
		return
	case "add":
		events = append(events, audit.UserAddToProjectTeamEvent, audit.ProjectTeamRightsGrantedEvent)
		auditParams["new_value"] = string(rightsAuditValueBytes)

		uname := utils.RemoveUsernameParameterSuffix(strings.ToLower(ctx.FormString("uname")))
		auditParams["affected_user"] = uname

		if !ctx.Org.IsOwner {
			ctx.Error(http.StatusNotFound)
			auditParams["error"] = "User is not the owner"
			for _, event := range events {
				audit.CreateAndSendEvent(event, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			}
			return
		}
		var u *user_model.User
		u, err = user_model.GetUserByName(ctx, uname)
		if err != nil {
			if user_model.IsErrUserNotExist(err) {
				u, err = user_model.GetAndCreateUserByNameOrEmailFromKeycloak(uname, ctx.Locale, ctx) // попробуем найти в keycloak если это возможно
				if err != nil {
					switch true {
					case user_model.IsErrUserNotExist(err):
						ctx.Flash.Error(ctx.Tr("form.user_not_exist"))
						ctx.Redirect(setting.AppSubURL + ctx.Org.OrgLink + "/teams/" + url.PathEscape(team.LowerName))
						auditParams["error"] = "User not exist"
						for _, event := range events {
							audit.CreateAndSendEvent(event, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
						}
						return
					case user_model.IsErrEmailAddressNotExist(err):
						ctx.Flash.Error(ctx.Tr("form.email_is_empty"))
						ctx.Redirect(setting.AppSubURL + ctx.Req.URL.EscapedPath())
						auditParams["error"] = "Email is empty"
						for _, event := range events {
							audit.CreateAndSendEvent(event, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
						}
						return
					default:
						log.Error("Error has occurred while try get and add user from keycloak to db with name: %s, err: %v", uname, err)
						ctx.ServerError("GetUserByName", err)
						auditParams["error"] = "Error has occurred while try get and add user from keycloak to db"
						for _, event := range events {
							audit.CreateAndSendEvent(event, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
						}
						return
					}
				}
				if setting.MailService != nil && user_model.ValidateEmail(uname) == nil {
					if err := org_service.CreateTeamInvite(ctx, ctx.Doer, team, uname); err != nil {
						if org_model.IsErrTeamInviteAlreadyExist(err) {
							ctx.Flash.Error(ctx.Tr("form.duplicate_invite_to_team"))
							auditParams["error"] = "Duplicate invite to team"
							for _, event := range events {
								audit.CreateAndSendEvent(event, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
							}
						} else if org_model.IsErrUserEmailAlreadyAdded(err) {
							ctx.Flash.Error(ctx.Tr("org.teams.add_duplicate_users"))
							auditParams["error"] = "User already added"
							for _, event := range events {
								audit.CreateAndSendEvent(event, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
							}
						} else {
							ctx.ServerError("CreateTeamInvite", err)
							auditParams["error"] = "Error has occurred while creating team invite"
							for _, event := range events {
								audit.CreateAndSendEvent(event, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
							}
							return
						}
					}
				}
			} else {
				log.Error("Error has occurred while try get user from db with name: %s, err: %v", uname, err)
				ctx.ServerError("GetUserByName", err)
				auditParams["error"] = "Error has occurred while trying get user from db"
				for _, event := range events {
					audit.CreateAndSendEvent(event, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
				}
				return
			}
		}

		auditParams["affected_user_id"] = strconv.FormatInt(u.ID, 10)

		if u.IsOrganization() {
			ctx.Flash.Error(ctx.Tr("form.cannot_add_org_to_team"))
			ctx.Redirect(ctx.Org.OrgLink + "/teams/" + url.PathEscape(team.LowerName))
			auditParams["error"] = "Cannot add organization to team"
			for _, event := range events {
				audit.CreateAndSendEvent(event, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			}
			return
		}

		if team.IsMember(u.ID) {
			ctx.Flash.Error(ctx.Tr("org.teams.add_duplicate_users"))
			auditParams["error"] = "User already added"
			for _, event := range events {
				audit.CreateAndSendEvent(event, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			}
		} else {
			if err = models.AddTeamMember(team, u.ID); err != nil {
				log.Error("TeamsAction models.AddTeamMember failed while adding user to team: %v", err)
				ctx.ServerError("TeamsAction models.AddTeamMember failed while adding user to team: %v", err)
				return
			}
			if err := s.repoRequestAccessor.GrantCustomPrivilege(accesser.RepoAccessRequest{
				DoerID:         u.ID,
				TargetTenantID: tenantID,
				OrgID:          ctx.Org.Organization.ID,
				Team:           team,
			}); err != nil {
				log.Error("TeamsAction role_model.GrantCustomPrivilegeTeamUser failed while grant custom privileges for team and user: %v", err)
				ctx.Error(http.StatusInternalServerError, fmt.Sprintf("TeamsAction role_model.GrantCustomPrivilegeTeamUser failed: %v", err))
				return
			}
		}

		page = "team"
	case "remove_invite":
		if !ctx.Org.IsOwner {
			ctx.Error(http.StatusNotFound)
			return
		}

		iid := ctx.FormInt64("iid")
		if iid == 0 {
			ctx.Redirect(ctx.Org.OrgLink + "/teams/" + url.PathEscape(team.LowerName))
			return
		}

		if err := org_model.RemoveInviteByID(ctx, iid, team.ID); err != nil {
			log.Error("Action(%s): %v", ctx.Params(":action"), err)
			ctx.ServerError("RemoveInviteByID", err)
			return
		}

		page = "team"
	}

	if err != nil {
		if org_model.IsErrLastOrgOwner(err) {
			ctx.Flash.Error(ctx.Tr("form.last_org_owner"))
			auditParams["error"] = "Cannot remove the last user from the 'owners' team"
		} else {
			log.Error("Action(%s): %v", ctx.Params(":action"), err)
			ctx.JSON(http.StatusOK, map[string]interface{}{
				"ok":  false,
				"err": err.Error(),
			})
			auditParams["error"] = "Error has occurred while changing team members"
		}
		for _, event := range events {
			audit.CreateAndSendEvent(event, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		}
		return
	}

	// проверка на то, что ошибок не было
	if !ctx.Written() {
		for _, event := range events {
			audit.CreateAndSendEvent(event, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)
		}
	}
	// если создаем персональную команду для пользователя то, не переходим на страницу команды
	customActivate := false
	if ctx.Data["CustomActivate"] != nil {
		customActivate = true
	}
	if !customActivate {
		switch page {
		case "team":
			ctx.Redirect(ctx.Org.OrgLink + "/teams/" + url.PathEscape(team.LowerName))
		case "home":
			ctx.Redirect(ctx.Org.Organization.AsUser().HomeLink())
		default:
			ctx.Redirect(ctx.Org.OrgLink + "/teams")
		}
	}
}
