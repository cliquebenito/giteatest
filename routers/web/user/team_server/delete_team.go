package team_server

import (
	"fmt"
	"net/http"
	"strconv"

	"code.gitea.io/gitea/models"
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/organization/custom"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/sbt/audit"
)

// DeleteTeam response for the delete team request
func (s *Server) DeleteTeam(ctx *context.Context) {
	auditParams := map[string]string{
		"project":    ctx.Org.Organization.Name,
		"project_id": strconv.FormatInt(ctx.Org.Organization.ID, 10),
		"team":       ctx.Org.Team.Name,
		"team_id":    strconv.FormatInt(ctx.Org.Team.ID, 10),
	}

	dbEngine := db.GetEngine(ctx)
	customDb := custom.NewCustomDB(dbEngine)

	teamCustomPrivileges, err := customDb.GetCustomPrivilegesByTeam(ctx.Org.Team.Name)
	if err != nil {
		log.Error("Error has occurred while getting custom privileges by team: %v", err)
		ctx.ServerError("Error has occurred while getting custom privileges by team: %v", err)
		auditParams["error"] = "Error has occurred while getting custom privileges for team"
		audit.CreateAndSendEvent(audit.TeamUpdateInProjectEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}

	if len(teamCustomPrivileges) > 0 {
		if err := s.repoRequestAccessor.RemoveCustomPrivilegesByFieldIdxAndName(3, ctx.Org.Team.Name); err != nil {
			log.Error("Error has occurred while removing custom privileges by params: %v", err)
			ctx.ServerError("Error has occurred while removing custom privileges by params: %v", err)
			auditParams["error"] = "Error has occurred while removing custom privileges"
			audit.CreateAndSendEvent(audit.RemoveCustomPrivilegesEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			audit.CreateAndSendEvent(audit.TeamUpdateInProjectEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}

		if err := s.customCreator.RemoveCustomPrivilegesByTeam(ctx, ctx.Org.Team.Name); err != nil {
			log.Error("Error has occurred while creating or removing custom privileges for user: %v", err)
			ctx.ServerError("Error has occurred while creating or removing custom privileges for user: %v", err)
			auditParams["error"] = "Error has occurred while deleting team with custom privileges"
			audit.CreateAndSendEvent(audit.RemoveCustomPrivilegesEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			audit.CreateAndSendEvent(audit.TeamUpdateInProjectEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}

		if err = customDb.DeleteCustomPrivilegesByTeam(ctx.Org.Team.Name); err != nil {
			log.Error("Error has occurred while deleting custom privileges: %v", err)
			ctx.ServerError("Error has occurred while deleting custom privileges: %v", err)
			auditParams["error"] = "Error has occurred while deleting team with custom privileges"
			audit.CreateAndSendEvent(audit.RemoveCustomPrivilegesEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			audit.CreateAndSendEvent(audit.TeamUpdateInProjectEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}
		audit.CreateAndSendEvent(audit.RemoveCustomPrivilegesEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)
	}

	if err := models.DeleteTeam(ctx.Org.Team); err != nil {
		log.Error("Error has occurred while deleting team with custom privileges: %v", err)
		ctx.Flash.Error(fmt.Sprintf("Error has occurred while deleting team with custom privileges: %v", err))
		auditParams["error"] = "Error has occurred while deleting team"
		audit.CreateAndSendEvent(audit.TeamRemoveFromProjectEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	} else {
		ctx.Flash.Success(ctx.Tr("org.teams.delete_team_success"))
		audit.CreateAndSendEvent(audit.TeamRemoveFromProjectEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)
	}

	ctx.JSON(http.StatusOK, map[string]interface{}{
		"redirect": ctx.Org.OrgLink + "/teams",
	})
}
