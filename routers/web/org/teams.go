// Copyright 2014 The Gogs Authors. All rights reserved.
// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package org

import (
	"net/http"
	"net/url"
	"path"
	"regexp"

	"code.gitea.io/gitea/models"
	avatars_model "code.gitea.io/gitea/models/avatars"
	"code.gitea.io/gitea/models/db"
	org_model "code.gitea.io/gitea/models/organization"
	"code.gitea.io/gitea/models/organization/custom"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/tenant"
	unit_model "code.gitea.io/gitea/models/unit"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/base"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/services/convert"
	"code.gitea.io/gitea/services/forms"
	org_service "code.gitea.io/gitea/services/org"
)

const (
	// tplTeams template path for teams list page
	tplTeams base.TplName = "org/team/teams"
	// tplTeamNew template path for create new team page
	tplTeamNew base.TplName = "org/team/new"
	// tplTeamMembers template path for showing team members page
	tplTeamMembers base.TplName = "org/team/members"
	// tplTeamRepositories template path for showing team repositories page
	tplTeamRepositories base.TplName = "org/team/repositories"
	// tplTeamInvite template path for team invites page
	tplTeamInvite base.TplName = "org/team/invite"
	actionChoose               = 1
)

var (
	// регулярка на проверку корректности названия команды
	regexTeamName = regexp.MustCompile(`^[a-zA-Z0-9._-]{1,30}$`)
)

// Teams render teams list page
func Teams(ctx *context.Context) {
	org := ctx.Org.Organization
	ctx.Data["Title"] = org.FullName
	ctx.Data["PageIsOrgTeams"] = true

	for _, t := range ctx.Org.Teams {
		if err := t.LoadMembers(ctx); err != nil {
			ctx.ServerError("GetMembers", err)
			return
		}
	}

	tenantId, err := tenant.GetTenantByOrgIdOrDefault(ctx, ctx.Org.Organization.ID)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, err.Error())
		return
	}

	ctx.Data["TenantID"] = tenantId
	ctx.Data["Teams"] = ctx.Org.Teams
	ctx.Data["ContextUser"] = ctx.ContextUser

	ctx.HTML(http.StatusOK, tplTeams)
}

// TeamsRepoAction operate team's repository
func TeamsRepoAction(ctx *context.Context) {
	if !ctx.Org.IsOwner {
		ctx.Error(http.StatusNotFound)
		return
	}
	form := web.GetForm(ctx).(*forms.AddReposForTeam)
	var err error
	action := ctx.Params(":action")
	switch action {
	case "add":
		repoName := path.Base(ctx.FormString("repo_name"))
		var repo *repo_model.Repository
		repo, err = repo_model.GetRepositoryByName(ctx.Org.Organization.ID, repoName)
		if err != nil {
			if repo_model.IsErrRepoNotExist(err) {
				ctx.Flash.Error(ctx.Tr("org.teams.add_nonexistent_repo"))
				ctx.Redirect(ctx.Org.OrgLink + "/teams/" + url.PathEscape(ctx.Org.Team.LowerName) + "/repositories")
				return
			}
			ctx.ServerError("GetRepositoryByName", err)
			return
		}
		err = org_service.TeamAddRepository(ctx.Org.Team, repo)
	case "addrepos":
		mapRepoIDRepository, errGetRepositories := repo_model.GetRepositoriesMapByIDs(form.RepoIDs)
		if errGetRepositories != nil {
			log.Error("Error has occurred while getting repositories by repo_ids: %v, err: %v", form.RepoIDs, errGetRepositories)
			ctx.ServerError("Error has occurred while getting repositories: %v", err)
			return
		}

		repositories := make([]*repo_model.Repository, 0, len(mapRepoIDRepository))
		for _, repo := range mapRepoIDRepository {
			repositories = append(repositories, repo)
		}

		for _, repo := range repositories {
			if !org_model.HasTeamRepo(ctx, ctx.Org.Team.OrgID, ctx.Org.Team.ID, repo.ID) {
				if err := models.AddRepository(ctx, ctx.Org.Team, repo); err != nil {
					log.Error("Error has occurred while adding repository")
					ctx.ServerError("Error has occurred while adding repository: %v", err)
					return
				}
			}
		}
	case "remove":
		err = models.RemoveRepository(ctx.Org.Team, ctx.FormInt64("repoid"))
	case "addall":
		err = models.AddAllRepositories(ctx.Org.Team)
	case "removeall":
		err = models.RemoveAllRepositories(ctx.Org.Team)
	}

	if err != nil {
		log.Error("Action(%s): '%s' %v", ctx.Params(":action"), ctx.Org.Team.Name, err)
		ctx.ServerError("TeamsRepoAction", err)
		return
	}

	if action == "addall" || action == "removeall" {
		ctx.JSON(http.StatusOK, map[string]interface{}{
			"redirect": ctx.Org.OrgLink + "/teams/" + url.PathEscape(ctx.Org.Team.LowerName) + "/repositories",
		})
		return
	}
	ctx.Redirect(ctx.Org.OrgLink + "/teams/" + url.PathEscape(ctx.Org.Team.LowerName) + "/repositories")
}

// NewTeam render create new team page
func NewTeam(ctx *context.Context) {
	tenantId, err := tenant.GetTenantByOrgIdOrDefault(ctx, ctx.Org.Organization.ID)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, err.Error())
		return
	}

	ctx.Data["TenantID"] = tenantId
	ctx.Data["Title"] = ctx.Org.Organization.FullName
	ctx.Data["PageIsOrgTeams"] = true
	ctx.Data["PageIsOrgTeamsNew"] = true
	ctx.Data["Team"] = &org_model.Team{}
	ctx.Data["Units"] = unit_model.Units
	ctx.HTML(http.StatusOK, tplTeamNew)
}

//// NewTeamPost response for create new team
//func NewTeamPost(ctx *context.Context) {
//	form := web.GetForm(ctx).(*forms.CreateTeamForm)
//
//	if form != nil && !regexTeamName.MatchString(form.TeamName) {
//		log.Error("Team name is incorrect: %s", form.TeamName)
//		ctx.Error(http.StatusBadRequest, ctx.Tr("org.teams.invalid_team_name"))
//		return
//	}
//
//	if form != nil && form.Description != "" && utf8.RuneCountInString(form.Description) > 255 {
//		log.Error("Description is too long: %s", form.Description)
//		ctx.Error(http.StatusBadRequest, ctx.Tr("org.teams.invalid_description"))
//		return
//	}
//
//	includesAllRepositories := form.RepoAccess == "all"
//	p := perm.ParseAccessMode(form.Permission)
//	unitPerms := getUnitPerms(ctx.Req.Form, p)
//	if p < perm.AccessModeAdmin {
//		// if p is less than admin accessmode, then it should be general accessmode,
//		// so we should calculate the minial accessmode from units accessmodes.
//		p = unit_model.MinUnitAccessMode(unitPerms)
//	}
//	auditParams := map[string]string{
//		"project":    ctx.Org.Organization.Name,
//		"project_id": strconv.FormatInt(ctx.Org.Organization.ID, 10),
//		"team":       form.TeamName,
//	}
//
//	type auditValue struct {
//		OrgID                   int64
//		Name                    string
//		Description             string
//		AccessMode              string
//		IncludesAllRepositories bool
//		CanCreateOrgRepo        bool
//		AccessModeForTypes      map[string]string
//		CustomPrivileges        []forms.CustomPrivileges
//		UserIDs                 []int64
//	}
//
//	newAuditValue := auditValue{
//		OrgID:                   ctx.Org.Organization.ID,
//		Name:                    form.TeamName,
//		Description:             form.Description,
//		AccessMode:              p.String(),
//		IncludesAllRepositories: includesAllRepositories,
//		CanCreateOrgRepo:        form.CanCreateOrgRepo,
//		AccessModeForTypes:      make(map[string]string),
//		CustomPrivileges:        form.CustomPrivileges,
//		UserIDs:                 form.UserIDs,
//	}
//
//	t := &org_model.Team{
//		OrgID:                   ctx.Org.Organization.ID,
//		Name:                    form.TeamName,
//		Description:             form.Description,
//		AccessMode:              p,
//		IncludesAllRepositories: includesAllRepositories,
//		CanCreateOrgRepo:        form.CanCreateOrgRepo,
//	}
//
//	units := make([]*org_model.TeamUnit, 0, len(unitPerms))
//	for tp, perm := range unitPerms {
//		units = append(units, &org_model.TeamUnit{
//			OrgID:      ctx.Org.Organization.ID,
//			Type:       tp,
//			AccessMode: perm,
//		})
//	}
//	t.Units = units
//
//	if t.Units != nil {
//		for _, unit := range t.Units {
//			newAuditValue.AccessModeForTypes[unit.Type.String()] = unit.AccessMode.String()
//		}
//	}
//	newAuditValueBytes, _ := json.Marshal(newAuditValue)
//	auditParams["new_value"] = string(newAuditValueBytes)
//
//	tenantId, err := tenant.GetTenantByOrgIdOrDefault(ctx, ctx.Org.Organization.ID)
//	if err != nil {
//		ctx.Error(http.StatusInternalServerError, err.Error())
//		return
//	}
//
//	ctx.Data["TenantID"] = tenantId
//	ctx.Data["Title"] = ctx.Org.Organization.FullName
//	ctx.Data["PageIsOrgTeams"] = true
//	ctx.Data["PageIsOrgTeamsNew"] = true
//	ctx.Data["Units"] = unit_model.Units
//	ctx.Data["Team"] = t
//
//	if ctx.HasError() {
//		ctx.HTML(http.StatusOK, tplTeamNew)
//		auditParams["error"] = "Error occurs in form validation"
//		audit.CreateAndSendEvent(audit.TeamAddToProjectEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
//		return
//	}
//
//	if t.AccessMode < perm.AccessModeAdmin && len(unitPerms) == 0 {
//		ctx.RenderWithErr(ctx.Tr("form.team_no_units_error"), tplTeamNew, &form)
//		auditParams["error"] = "No access to at least one repository section"
//		audit.CreateAndSendEvent(audit.TeamAddToProjectEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
//		return
//	}
//
//	if err := models.NewTeam(t); err != nil {
//		ctx.Data["Err_TeamName"] = true
//		switch {
//		case org_model.IsErrTeamAlreadyExist(err):
//			ctx.JSON(http.StatusBadRequest, map[string]string{"message": ctx.Tr("org.teams.team_is_already_exists")})
//			auditParams["error"] = "Team name been taken"
//		default:
//			ctx.ServerError("NewTeam", err)
//			auditParams["error"] = "Error has occurred while creating team"
//		}
//		audit.CreateAndSendEvent(audit.TeamAddToProjectEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
//		return
//	}
//	audit.CreateAndSendEvent(audit.TeamAddToProjectEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)
//	log.Trace("Team created: %s/%s", ctx.Org.Organization.Name, t.Name)
//	// если обновляем персональную команду для пользователя то, не переходим на страницу команды
//	customActivate := false
//	if ctx.Data["CustomActivate"] != nil {
//		customActivate = true
//	}
//	if len(form.CustomPrivileges) > 0 {
//		if err := team.AddCustomPrivilegeToTeamUser(ctx, ctx.Org.Organization.ID, t.Name, form.CustomPrivileges); err != nil {
//			log.Error("Error has occurred while adding custom privileges for user: %v", err)
//			ctx.ServerError("Error has occurred while adding custom privileges for user: %v", err)
//			auditParams["error"] = "Error has occurred while adding custom privileges to team members"
//			return
//		}
//
//		if err = team.InsertCustomPrivilegeToTeamUser(t.Name, form.CustomPrivileges); err != nil {
//			log.Error("Error has occurred while inserting custom privileges for user: %v", err)
//			ctx.ServerError("Error has occurred while inserting custom privileges for user: %v", err)
//			auditParams["error"] = "Error has occurred while inserting custom privileges to team members"
//			return
//		}
//	}
//
//	if !customActivate {
//		ctx.Redirect(ctx.Org.OrgLink + "/teams/" + url.PathEscape(t.LowerName))
//	}
//}

// TeamMembers render team members page
func TeamMembers(ctx *context.Context) {
	tenantId, err := tenant.GetTenantByOrgIdOrDefault(ctx, ctx.Org.Organization.ID)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, err.Error())
		return
	}

	dbEngine := db.GetEngine(ctx)
	customDb := custom.NewCustomDB(dbEngine)

	ctx.Data["TenantID"] = tenantId
	ctx.Data["Title"] = ctx.Org.Team.Name
	ctx.Data["PageIsOrgTeams"] = true
	ctx.Data["PageIsOrgTeamMembers"] = true
	if err := ctx.Org.Team.LoadMembers(ctx); err != nil {
		ctx.ServerError("GetMembers", err)
		return
	}

	members := ctx.Org.Team.Members
	for idx := range members {
		var avatarLink string

		if members[idx].Avatar != "" {
			avatarLink = avatars_model.GenerateUserAvatarImageLink(members[idx].Avatar, 0)
		} else {
			avatarLink = avatars_model.GenerateEmailAvatarFastLink(ctx, members[idx].AvatarEmail, 0)
		}
		members[idx].Avatar = avatarLink
	}

	ctx.Org.Team.Members = members
	ctx.Data["Units"] = unit_model.Units

	invites, err := org_model.GetInvitesByTeamID(ctx, ctx.Org.Team.ID)
	if err != nil {
		ctx.ServerError("GetInvitesByTeamID", err)
		return
	}

	if err := ctx.Org.Team.LoadRepositories(ctx); err != nil {
		log.Error("Error has occurred while loading repositories for team: %v", err)
		ctx.ServerError("Error has occurred while loading repositories for team: %v", err)
		return
	}

	ctx.Data["Invites"] = invites
	ctx.Data["IsEmailInviteEnabled"] = setting.MailService != nil
	customPrivileges, err := customDb.GetCustomPrivilegesByTeam(ctx.Org.Team.Name)
	if err != nil {
		log.Error("Error has occurred while getting custom privileges: %v", err)
		ctx.ServerError("Error has occurred while getting custom privileges: %v", err)
		return
	}
	ctx.Data["CustomPrivilegesUnits"] = customPrivileges
	ctx.HTML(http.StatusOK, tplTeamMembers)
}

// TeamRepositories show the repositories of team
func TeamRepositories(ctx *context.Context) {
	tenantId, err := tenant.GetTenantByOrgIdOrDefault(ctx, ctx.Org.Organization.ID)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, err.Error())
		return
	}
	dbEngine := db.GetEngine(ctx)
	customDb := custom.NewCustomDB(dbEngine)

	if err := ctx.Org.Team.LoadMembers(ctx); err != nil {
		log.Error("Error has occurred while loading members for team: %v", err)
		ctx.ServerError("Error has occurred while loading members for team: %v", err)
		return
	}

	ctx.Data["TenantID"] = tenantId
	ctx.Data["Title"] = ctx.Org.Team.Name
	ctx.Data["PageIsOrgTeams"] = true
	ctx.Data["PageIsOrgTeamRepos"] = true
	if err := ctx.Org.Team.LoadRepositories(ctx); err != nil {
		ctx.ServerError("GetRepositories", err)
		return
	}
	ctx.Data["Units"] = unit_model.Units

	customPrivileges, err := customDb.GetCustomPrivilegesByTeam(ctx.Org.Team.Name)
	if err != nil {
		log.Error("Error has occurred while getting custom privileges: %v", err)
		ctx.ServerError("Error has occurred while getting custom privileges: %v", err)
		return
	}
	ctx.Data["CustomPrivilegesUnits"] = customPrivileges

	ctx.HTML(http.StatusOK, tplTeamRepositories)
}

// SearchTeam api for searching teams
func SearchTeam(ctx *context.Context) {
	listOptions := db.ListOptions{
		Page:     ctx.FormInt("page"),
		PageSize: convert.ToCorrectPageSize(ctx.FormInt("limit")),
	}

	opts := &org_model.SearchTeamOptions{
		// UserID is not set because the router already requires the doer to be an org admin. Thus, we don't need to restrict to teams that the user belongs in
		Keyword:     ctx.FormTrim("q"),
		OrgID:       ctx.Org.Organization.ID,
		IncludeDesc: ctx.FormString("include_desc") == "" || ctx.FormBool("include_desc"),
		ListOptions: listOptions,
	}

	teams, maxResults, err := org_model.SearchTeam(opts)
	if err != nil {
		log.Error("SearchTeam failed: %v", err)
		ctx.JSON(http.StatusInternalServerError, map[string]interface{}{
			"ok":    false,
			"error": "SearchTeam internal failure",
		})
		return
	}

	apiTeams, err := convert.ToTeams(ctx, teams, false)
	if err != nil {
		log.Error("convert ToTeams failed: %v", err)
		ctx.JSON(http.StatusInternalServerError, map[string]interface{}{
			"ok":    false,
			"error": "SearchTeam failed to get units",
		})
		return
	}

	ctx.SetTotalCountHeader(maxResults)
	ctx.JSON(http.StatusOK, map[string]interface{}{
		"ok":   true,
		"data": apiTeams,
	})
}

// EditTeam render team edit page
func EditTeam(ctx *context.Context) {
	tenantId, err := tenant.GetTenantByOrgIdOrDefault(ctx, ctx.Org.Organization.ID)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, err.Error())
		return
	}

	dbEngine := db.GetEngine(ctx)
	customDb := custom.NewCustomDB(dbEngine)

	ctx.Data["TenantID"] = tenantId
	ctx.Data["Title"] = ctx.Org.Organization.FullName
	ctx.Data["PageIsOrgTeams"] = true
	if err := ctx.Org.Team.LoadUnits(ctx); err != nil {
		ctx.ServerError("LoadUnits", err)
		return
	}
	ctx.Data["Team"] = ctx.Org.Team
	ctx.Data["Units"] = unit_model.Units

	customPrivileges, err := customDb.GetCustomPrivilegesByTeam(ctx.Org.Team.Name)
	if err != nil {
		log.Error("Error has occurred while getting custom privileges: %v", err)
		ctx.ServerError("Error has occurred while getting custom privileges: %v", err)
		return
	}
	ctx.Data["CustomPrivilegesUnits"] = customPrivileges

	ctx.HTML(http.StatusOK, tplTeamNew)
}

// TeamInvite renders the team invite page
func TeamInvite(ctx *context.Context) {
	invite, org, team, inviter, err := getTeamInviteFromContext(ctx)
	if err != nil {
		if org_model.IsErrTeamInviteNotFound(err) {
			ctx.NotFound("ErrTeamInviteNotFound", err)
		} else {
			ctx.ServerError("getTeamInviteFromContext", err)
		}
		return
	}
	tenantId, err := tenant.GetTenantByOrgIdOrDefault(ctx, ctx.Org.Organization.ID)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, err.Error())
		return
	}

	ctx.Data["TenantID"] = tenantId
	ctx.Data["Title"] = ctx.Tr("org.teams.invite_team_member", team.Name)
	ctx.Data["Invite"] = invite
	ctx.Data["Organization"] = org
	ctx.Data["Team"] = team
	ctx.Data["Inviter"] = inviter

	ctx.HTML(http.StatusOK, tplTeamInvite)
}

// TeamInvitePost handles the team invitation
func TeamInvitePost(ctx *context.Context) {
	invite, org, team, _, err := getTeamInviteFromContext(ctx)
	if err != nil {
		if org_model.IsErrTeamInviteNotFound(err) {
			ctx.NotFound("ErrTeamInviteNotFound", err)
		} else {
			ctx.ServerError("getTeamInviteFromContext", err)
		}
		return
	}

	if err := models.AddTeamMember(team, ctx.Doer.ID); err != nil {
		ctx.ServerError("AddTeamMember", err)
		return
	}

	if err := org_model.RemoveInviteByID(ctx, invite.ID, team.ID); err != nil {
		log.Error("RemoveInviteByID: %v", err)
	}

	ctx.Redirect(org.OrganisationLink() + "/teams/" + url.PathEscape(team.LowerName))
}

func getTeamInviteFromContext(ctx *context.Context) (*org_model.TeamInvite, *org_model.Organization, *org_model.Team, *user_model.User, error) {
	invite, err := org_model.GetInviteByToken(ctx, ctx.Params("token"))
	if err != nil {
		return nil, nil, nil, nil, err
	}

	inviter, err := user_model.GetUserByID(ctx, invite.InviterID)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	team, err := org_model.GetTeamByID(ctx, invite.TeamID)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	org, err := user_model.GetUserByID(ctx, team.OrgID)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	return invite, org_model.OrgFromUser(org), team, inviter, nil
}
