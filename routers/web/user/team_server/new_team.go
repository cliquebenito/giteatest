package team_server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"unicode/utf8"

	"code.gitea.io/gitea/models"
	org_model "code.gitea.io/gitea/models/organization"
	"code.gitea.io/gitea/models/perm"
	"code.gitea.io/gitea/models/tenant"
	unit_model "code.gitea.io/gitea/models/unit"
	"code.gitea.io/gitea/modules/base"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/sbt/audit"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/services/forms"
	"code.gitea.io/gitea/services/team"
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
)

var (
	// регулярка на проверку корректности названия команды
	regexTeamName = regexp.MustCompile(`^[a-zA-Z0-9._-]{1,30}$`)
)

// NewTeamPost response for create new team
func (s *Server) NewTeamPost(ctx *context.Context) {
	form := web.GetForm(ctx).(*forms.CreateTeamForm)

	if form != nil && !regexTeamName.MatchString(form.TeamName) {
		log.Error("Team name is incorrect: %s", form.TeamName)
		ctx.Error(http.StatusBadRequest, ctx.Tr("org.teams.invalid_team_name"))
		return
	}

	if form != nil && form.Description != "" && utf8.RuneCountInString(form.Description) > 255 {
		log.Error("Description is too long: %s", form.Description)
		ctx.Error(http.StatusBadRequest, ctx.Tr("org.teams.invalid_description"))
		return
	}

	includesAllRepositories := form.RepoAccess == "all"
	p := perm.ParseAccessMode(form.Permission)
	unitPerms := getUnitPerms(ctx.Req.Form, p)
	if p < perm.AccessModeAdmin {
		// if p is less than admin accessmode, then it should be general accessmode,
		// so we should calculate the minial accessmode from units accessmodes.
		p = unit_model.MinUnitAccessMode(unitPerms)
	}
	auditParams := map[string]string{
		"project":    ctx.Org.Organization.Name,
		"project_id": strconv.FormatInt(ctx.Org.Organization.ID, 10),
		"team":       form.TeamName,
	}

	type auditValue struct {
		OrgID                   int64
		Name                    string
		Description             string
		AccessMode              string
		IncludesAllRepositories bool
		CanCreateOrgRepo        bool
		AccessModeForTypes      map[string]string
		CustomPrivileges        []forms.CustomPrivileges
		UserIDs                 []int64
	}

	newAuditValue := auditValue{
		OrgID:                   ctx.Org.Organization.ID,
		Name:                    form.TeamName,
		Description:             form.Description,
		AccessMode:              p.String(),
		IncludesAllRepositories: includesAllRepositories,
		CanCreateOrgRepo:        form.CanCreateOrgRepo,
		AccessModeForTypes:      make(map[string]string),
		CustomPrivileges:        form.CustomPrivileges,
		UserIDs:                 form.UserIDs,
	}

	t := &org_model.Team{
		OrgID:                   ctx.Org.Organization.ID,
		Name:                    form.TeamName,
		Description:             form.Description,
		AccessMode:              p,
		IncludesAllRepositories: includesAllRepositories,
		CanCreateOrgRepo:        form.CanCreateOrgRepo,
	}

	units := make([]*org_model.TeamUnit, 0, len(unitPerms))
	for tp, perm := range unitPerms {
		units = append(units, &org_model.TeamUnit{
			OrgID:      ctx.Org.Organization.ID,
			Type:       tp,
			AccessMode: perm,
		})
	}
	t.Units = units

	if t.Units != nil {
		for _, unit := range t.Units {
			newAuditValue.AccessModeForTypes[unit.Type.String()] = unit.AccessMode.String()
		}
	}
	newAuditValueBytes, _ := json.Marshal(newAuditValue)
	auditParams["new_value"] = string(newAuditValueBytes)

	tenantId, err := tenant.GetTenantByOrgIdOrDefault(ctx, ctx.Org.Organization.ID)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, err.Error())
		return
	}

	ctx.Data["TenantID"] = tenantId
	ctx.Data["Title"] = ctx.Org.Organization.FullName
	ctx.Data["PageIsOrgTeams"] = true
	ctx.Data["PageIsOrgTeamsNew"] = true
	ctx.Data["Units"] = unit_model.Units
	ctx.Data["Team"] = t

	if ctx.HasError() {
		ctx.HTML(http.StatusOK, tplTeamNew)
		auditParams["error"] = "Error occurs in form validation"
		audit.CreateAndSendEvent(audit.TeamAddToProjectEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}

	if t.AccessMode < perm.AccessModeAdmin && len(unitPerms) == 0 {
		ctx.RenderWithErr(ctx.Tr("form.team_no_units_error"), tplTeamNew, &form)
		auditParams["error"] = "No access to at least one repository section"
		audit.CreateAndSendEvent(audit.TeamAddToProjectEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}

	if err := models.NewTeam(t); err != nil {
		ctx.Data["Err_TeamName"] = true
		switch {
		case org_model.IsErrTeamAlreadyExist(err):
			ctx.JSON(http.StatusBadRequest, map[string]string{"message": ctx.Tr("org.teams.team_is_already_exists")})
			auditParams["error"] = "Team name been taken"
		default:
			ctx.ServerError("NewTeam", err)
			auditParams["error"] = "Error has occurred while creating team"
		}
		audit.CreateAndSendEvent(audit.TeamAddToProjectEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}
	audit.CreateAndSendEvent(audit.TeamAddToProjectEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)
	log.Trace("Team created: %s/%s", ctx.Org.Organization.Name, t.Name)
	// если обновляем персональную команду для пользователя то, не переходим на страницу команды
	customActivate := false
	if ctx.Data["CustomActivate"] != nil {
		customActivate = true
	}
	if len(form.CustomPrivileges) > 0 {
		if err := s.customCreator.AddCustomPrivilegeToTeamUser(ctx, ctx.Org.Organization.ID, t.Name, form.CustomPrivileges); err != nil {
			auditParams["error"] = "Error has occurred while adding custom privileges to team members"
			audit.CreateAndSendEvent(audit.AddCustomPrivilegesEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			log.Error("Error has occurred while adding custom privileges for user: %v", err)
			ctx.ServerError("Error has occurred while adding custom privileges for user: %v", err)
			return
		}

		if err = team.InsertCustomPrivilegeToTeamUser(t.Name, form.CustomPrivileges); err != nil {
			auditParams["error"] = "Error has occurred while inserting custom privileges to team members"
			audit.CreateAndSendEvent(audit.AddCustomPrivilegesEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			log.Error("Error has occurred while inserting custom privileges for user: %v", err)
			ctx.ServerError("Error has occurred while inserting custom privileges for user: %v", err)
			return
		}
		audit.CreateAndSendEvent(audit.AddCustomPrivilegesEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
	}

	if !customActivate {
		ctx.Redirect(ctx.Org.OrgLink + "/teams/" + url.PathEscape(t.LowerName))
	}
}

func getUnitPerms(forms url.Values, teamPermission perm.AccessMode) map[unit_model.Type]perm.AccessMode {
	unitPerms := make(map[unit_model.Type]perm.AccessMode)
	for _, ut := range unit_model.AllRepoUnitTypes {
		// Default accessmode is none
		unitPerms[ut] = perm.AccessModeNone

		v, ok := forms[fmt.Sprintf("unit_%d", ut)]
		if ok {
			vv, _ := strconv.Atoi(v[0])
			if teamPermission >= perm.AccessModeAdmin {
				unitPerms[ut] = teamPermission
				// Don't allow `TypeExternal{Tracker,Wiki}` to influence this as they can only be set to READ perms.
				if ut == unit_model.TypeExternalTracker || ut == unit_model.TypeExternalWiki {
					unitPerms[ut] = perm.AccessModeRead
				}
			} else {
				unitPerms[ut] = perm.AccessMode(vv)
				if unitPerms[ut] >= perm.AccessModeAdmin {
					unitPerms[ut] = perm.AccessModeWrite
				}
			}
		}
	}
	return unitPerms
}
