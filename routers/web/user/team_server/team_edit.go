package team_server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"code.gitea.io/gitea/models"
	"code.gitea.io/gitea/models/db"
	org_model "code.gitea.io/gitea/models/organization"
	"code.gitea.io/gitea/models/organization/custom"
	"code.gitea.io/gitea/models/perm"
	"code.gitea.io/gitea/models/tenant"
	unit_model "code.gitea.io/gitea/models/unit"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/sbt/audit"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/services/forms"
	"code.gitea.io/gitea/services/team"
)

// EditTeamPost response for modify team information
func (s *Server) EditTeamPost(ctx *context.Context) {
	form := web.GetForm(ctx).(*forms.CreateTeamForm)

	if form != nil && !regexTeamName.MatchString(form.TeamName) {
		log.Error("Team name is incorrect: %s", form.TeamName)
		ctx.Error(http.StatusBadRequest, ctx.Tr("org.teams.invalid_team_name"))
		return
	}

	if form != nil && form.Description != "" && len(form.Description) > 255 {
		log.Error("Description is too long: %s", form.Description)
		ctx.Error(http.StatusBadRequest, ctx.Tr("org.teams.invalid_description"))
		return
	}

	dbEngine := db.GetEngine(ctx)
	customDb := custom.NewCustomDB(dbEngine)

	t := ctx.Org.Team
	newAccessMode := perm.ParseAccessMode(form.Permission)
	unitPerms := getUnitPerms(ctx.Req.Form, newAccessMode)
	if newAccessMode < perm.AccessModeAdmin {
		// if newAccessMode is less than admin accessmode, then it should be general accessmode,
		// so we should calculate the minial accessmode from units accessmodes.
		newAccessMode = unit_model.MinUnitAccessMode(unitPerms)
	}
	isAuthChanged := false
	isIncludeAllChanged := false
	includesAllRepositories := form.RepoAccess == "all"

	usersForTeam, err := org_model.GetTeamUsersByTeamID(ctx, t.ID)
	if err != nil {
		log.Error("Error has occurred while getting team users: %v", err)
		ctx.JSON(http.StatusInternalServerError, fmt.Sprintf("Error has occurred while getting team users: %v", err))
	}

	userIDs := make([]int64, 0, len(usersForTeam))
	for idx := range usersForTeam {
		userIDs = append(userIDs, usersForTeam[idx].ID)
	}

	teamCustomPrivileges, err := customDb.GetCustomPrivilegesByTeam(t.Name)
	if err != nil {
		log.Error("Error has occurred while getting custom privileges by team: %v", err)
		ctx.JSON(http.StatusInternalServerError, fmt.Sprintf("Error has occurred while getting custom privileges by team: %v", err))
		return
	}

	customPrivileges := make([]forms.CustomPrivileges, 0, len(teamCustomPrivileges))
	for _, customPrivilege := range customPrivileges {
		customPrivileges = append(customPrivileges, forms.CustomPrivileges{
			AllRepositories: customPrivilege.AllRepositories,
			RepoID:          customPrivilege.RepoID,
			Privileges:      customPrivilege.Privileges,
		})
	}

	auditParams := map[string]string{
		"project":    ctx.Org.Organization.Name,
		"project_id": strconv.FormatInt(ctx.Org.Organization.ID, 10),
		"team":       t.Name,
		"team_id":    strconv.FormatInt(t.ID, 10),
	}

	auditParamsForEdit := map[string]string{
		"project":    ctx.Org.Organization.Name,
		"project_id": strconv.FormatInt(ctx.Org.Organization.ID, 10),
		"team":       t.Name,
		"team_id":    strconv.FormatInt(t.ID, 10),
	}

	type auditValue struct {
		AccessMode              string
		RepositoryIds           string
		IncludesAllRepositories bool
		CanCreateOrgRepo        bool
		AccessModeForTypes      map[string]string
		CustomPrivileges        []forms.CustomPrivileges
		UserIDs                 []int64
	}

	type auditValueForEdit struct {
		Name                    string
		Description             string
		AccessMode              string
		RepositoryIds           string
		IncludesAllRepositories bool
		CanCreateOrgRepo        bool
		AccessModeForTypes      map[string]string
		CustomPrivileges        []forms.CustomPrivileges
		UserIDs                 []int64
	}

	_ = t.LoadUnits(ctx)
	_ = t.LoadRepositories(ctx)

	oldAuditValue := auditValue{
		AccessMode:              t.AccessMode.String(),
		IncludesAllRepositories: t.IncludesAllRepositories,
		CanCreateOrgRepo:        t.CanCreateOrgRepo,
		AccessModeForTypes:      make(map[string]string),
		CustomPrivileges:        customPrivileges,
		UserIDs:                 userIDs,
	}

	oldAuditValueForEdit := auditValueForEdit{
		Name:                    t.Name,
		Description:             t.Description,
		AccessMode:              t.AccessMode.String(),
		IncludesAllRepositories: t.IncludesAllRepositories,
		CanCreateOrgRepo:        t.CanCreateOrgRepo,
		AccessModeForTypes:      make(map[string]string),
		CustomPrivileges:        form.CustomPrivileges,
		UserIDs:                 form.UserIDs,
	}

	if t.Repos != nil {
		var repoIds []string
		for _, repo := range t.Repos {
			repoIds = append(repoIds, strconv.FormatInt(repo.ID, 10))
		}
		oldAuditValue.RepositoryIds = strings.Join(repoIds, ",")
		oldAuditValueForEdit.RepositoryIds = strings.Join(repoIds, ",")
	}

	if t.Units != nil {
		for _, unit := range t.Units {
			oldAuditValue.AccessModeForTypes[unit.Type.String()] = unit.AccessMode.String()
			oldAuditValueForEdit.AccessModeForTypes[unit.Type.String()] = unit.AccessMode.String()
		}
	}
	oldAuditValueBytes, _ := json.Marshal(oldAuditValue)
	auditParams["old_value"] = string(oldAuditValueBytes)

	oldAuditValueForEditBytes, _ := json.Marshal(oldAuditValueForEdit)
	auditParamsForEdit["old_value"] = string(oldAuditValueForEditBytes)

	tenantId, err := tenant.GetTenantByOrgIdOrDefault(ctx, ctx.Org.Organization.ID)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, err.Error())
		return
	}
	ctx.Data["TenantID"] = tenantId
	ctx.Data["Title"] = ctx.Org.Organization.FullName
	ctx.Data["PageIsOrgTeams"] = true
	ctx.Data["Team"] = t
	ctx.Data["Units"] = unit_model.Units
	if !t.IsOwnerTeam() {
		t.Name = form.TeamName
		if t.AccessMode != newAccessMode {
			isAuthChanged = true
			t.AccessMode = newAccessMode
		}

		if t.IncludesAllRepositories != includesAllRepositories {
			isIncludeAllChanged = true
			t.IncludesAllRepositories = includesAllRepositories
		}
		t.CanCreateOrgRepo = form.CanCreateOrgRepo
	} else {
		t.CanCreateOrgRepo = true
	}

	t.Description = form.Description
	units := make([]*org_model.TeamUnit, 0, len(unitPerms))
	for tp, perm := range unitPerms {
		units = append(units, &org_model.TeamUnit{
			OrgID:      t.OrgID,
			TeamID:     t.ID,
			Type:       tp,
			AccessMode: perm,
		})
	}
	t.Units = units

	newAuditValue := auditValue{
		AccessMode:              t.AccessMode.String(),
		IncludesAllRepositories: t.IncludesAllRepositories,
		CanCreateOrgRepo:        t.CanCreateOrgRepo,
		AccessModeForTypes:      make(map[string]string),
	}

	newAuditValueForEdit := auditValueForEdit{
		Name:                    t.Name,
		Description:             t.Description,
		AccessMode:              t.AccessMode.String(),
		IncludesAllRepositories: t.IncludesAllRepositories,
		CanCreateOrgRepo:        t.CanCreateOrgRepo,
		AccessModeForTypes:      make(map[string]string),
	}

	if t.Repos != nil {
		var repoIds []string
		for _, repo := range t.Repos {
			repoIds = append(repoIds, strconv.FormatInt(repo.ID, 10))
		}
		newAuditValue.RepositoryIds = strings.Join(repoIds, ",")
		newAuditValueForEdit.RepositoryIds = strings.Join(repoIds, ",")
	}

	if t.Units != nil {
		for _, unit := range t.Units {
			newAuditValue.AccessModeForTypes[unit.Type.String()] = unit.AccessMode.String()
			newAuditValueForEdit.AccessModeForTypes[unit.Type.String()] = unit.AccessMode.String()
		}
	}
	newAuditValueBytes, _ := json.Marshal(newAuditValue)
	auditParams["new_value"] = string(newAuditValueBytes)

	newAuditValueForEditBytes, _ := json.Marshal(newAuditValueForEdit)
	auditParamsForEdit["new_value"] = string(newAuditValueForEditBytes)

	if ctx.HasError() {
		ctx.HTML(http.StatusOK, tplTeamNew)
		auditParams["error"] = "Error occurs in form validation"
		auditParamsForEdit["error"] = "Error occurs in form validation"
		audit.CreateAndSendEvent(audit.ProjectTeamRightsChangeEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		audit.CreateAndSendEvent(audit.TeamUpdateInProjectEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParamsForEdit)
		return
	}

	if t.AccessMode < perm.AccessModeAdmin && len(unitPerms) == 0 {
		ctx.RenderWithErr(ctx.Tr("form.team_no_units_error"), tplTeamNew, &form)
		auditParams["error"] = "No access to at least one repository section"
		auditParamsForEdit["error"] = "No access to at least one repository section"
		audit.CreateAndSendEvent(audit.ProjectTeamRightsChangeEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		audit.CreateAndSendEvent(audit.TeamUpdateInProjectEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParamsForEdit)
		return
	}
	teamExists, err := org_model.GetTeamByID(ctx, t.ID)
	if err != nil {
		log.Error("Error has occurred while getting team by id: %v", err)
		ctx.ServerError("Error has occurred while getting team by id: %v", err)
		return
	}

	if err := models.UpdateTeam(t, isAuthChanged, isIncludeAllChanged); err != nil {
		ctx.Data["Err_TeamName"] = true
		switch {
		case org_model.IsErrTeamAlreadyExist(err):
			ctx.JSON(http.StatusBadRequest, map[string]string{"message": ctx.Tr("org.teams.team_is_already_exists")})
			auditParams["error"] = "Team name been taken"
			auditParamsForEdit["error"] = "Team name been taken"
		default:
			ctx.ServerError("UpdateTeam", err)
			auditParams["error"] = "Error has occurred while updating team"
			auditParamsForEdit["error"] = "Error has occurred while updating team"
		}
		audit.CreateAndSendEvent(audit.ProjectTeamRightsChangeEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		audit.CreateAndSendEvent(audit.TeamUpdateInProjectEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParamsForEdit)
		return
	}
	audit.CreateAndSendEvent(audit.ProjectTeamRightsChangeEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)
	audit.CreateAndSendEvent(audit.TeamUpdateInProjectEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParamsForEdit)

	// удаляем все права кастомизации для этой команды
	if teamExists.Name != form.TeamName {
		if err := s.repoRequestAccessor.RemoveCustomPrivilegesByFieldIdxAndName(3, teamExists.Name); err != nil {
			log.Error("Error has occurred while removing custom privileges by params: %v", err)
			ctx.ServerError("Error has occurred while removing custom privileges by params: %v", err)
			return
		}

		if err = customDb.DeleteCustomPrivilegesByTeam(teamExists.Name); err != nil {
			log.Error("Error has occurred while removing custom privileges from team: %v", err)
			ctx.ServerError("Error has occurred while removing custom privileges from team: %v", err)
			return
		}
	}
	// если добавляем пользователя в персональную команду то, не переходим на страницу команды
	customActivate := false
	if ctx.Data["CustomActivate"] != nil {
		customActivate = true
	}

	if len(form.CustomPrivileges) > 0 {
		if err := models.RemoveAllRepositories(t); err != nil {
			log.Error("Error has occurred while updating custom privileges to team and user: %v", err)
			ctx.ServerError("Error has occurred while updating custom privileges to team and user: %v", err)
			auditParams["error"] = "Error has occurred while removing all repositories"
			audit.CreateAndSendEvent(audit.TeamUpdateInProjectEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}

		if err = s.customCreator.UpdateCustomPrivilegeToTeamUser(ctx, ctx.Org.Organization.ID, form.TeamName, form.CustomPrivileges); err != nil {
			log.Error("Error has occurred while updating custom privileges to team and user: %v", err)
			ctx.ServerError("Error has occurred while updating custom privileges to team and user: %v", err)
			auditParams["error"] = "Error has occurred while updating custom privileges"
			audit.CreateAndSendEvent(audit.UpdateCustomPrivilegesEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			audit.CreateAndSendEvent(audit.TeamUpdateInProjectEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}

		if err = customDb.DeleteCustomPrivilegesByTeam(form.TeamName); err != nil {
			log.Error("Error has occurred while removing custom privileges from team: %v", err)
			ctx.ServerError("Error has occurred while removing custom privileges from team: %v", err)
			auditParams["error"] = "Error has occurred while deleting custom privileges"
			audit.CreateAndSendEvent(audit.UpdateCustomPrivilegesEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			audit.CreateAndSendEvent(audit.TeamUpdateInProjectEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}

		if err = team.InsertCustomPrivilegeToTeamUser(form.TeamName, form.CustomPrivileges); err != nil {
			log.Error("Error has occurred while removing custom privileges from team: %v", err)
			ctx.ServerError("Error has occurred while removing custom privileges from team: %v", err)
			auditParams["error"] = "Error has occurred while inserting custom privileges"
			audit.CreateAndSendEvent(audit.UpdateCustomPrivilegesEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			audit.CreateAndSendEvent(audit.TeamUpdateInProjectEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}
		audit.CreateAndSendEvent(audit.UpdateCustomPrivilegesEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)
	}

	// добавляем пользователей в команду при редактировании шаблона
	if len(form.UserIDs) > 0 {
		if err := s.customCreator.AddUserToTeam(ctx, t.ID, ctx.Org.Organization.ID, tenantId, form.UserIDs); err != nil {
			log.Error("Error has occurred while adding user from team: %v", err)
			ctx.ServerError("Error has occurred while adding user from team: %v", err)
			auditParams["error"] = "Error has occurred while adding users to team"
			audit.CreateAndSendEvent(audit.TeamUpdateInProjectEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}
	}

	if !customActivate {
		ctx.Redirect(ctx.Org.OrgLink + "/teams/" + url.PathEscape(t.LowerName))
		return
	}
}
