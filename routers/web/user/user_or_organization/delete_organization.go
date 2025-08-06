package user_or_organization

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/google/uuid"

	"code.gitea.io/gitea/models/organization"
	repo_model "code.gitea.io/gitea/models/repo"
	tenat_model "code.gitea.io/gitea/models/tenant"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/sbt/audit"
	auditutils "code.gitea.io/gitea/modules/sbt/audit/utils"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/routers/web/user/accesser"
	org_service "code.gitea.io/gitea/services/org"
	"code.gitea.io/gitea/services/repository"
	tenant_service "code.gitea.io/gitea/services/tenant"
)

func (s Server) DeleteTenant(ctx *context.Context) {
	requiredAuditParams := auditutils.NewRequiredAuditParams(ctx)
	if !setting.SourceControl.MultiTenantEnabled {
		auditParams := map[string]string{
			"affected_user_id": strconv.FormatInt(ctx.Doer.ID, 10),
			"email":            ctx.Doer.Email,
		}
		auditParams["error"] = "Error has occurred while validate multi-tenant mode"
		audit.CreateAndSendEvent(audit.TenantDeleteEvent, requiredAuditParams.DoerName, requiredAuditParams.DoerID, audit.StatusFailure, requiredAuditParams.RemoteAddress, auditParams)
		log.Warn("DeleteTenant permission denied")
		ctx.Error(http.StatusForbidden, ctx.Tr("admin.permission_denied"))
		return
	}

	tenantID := ctx.Params("tenantid")
	auditParams := map[string]string{
		"tenant_id":        tenantID,
		"affected_user_id": strconv.FormatInt(ctx.Doer.ID, 10),
		"email":            ctx.Doer.Email,
	}

	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		auditParams["error"] = "Error has occurred while deleting tenant"
		audit.CreateAndSendEvent(audit.TenantDeleteEvent, requiredAuditParams.DoerName, requiredAuditParams.DoerID, audit.StatusFailure, requiredAuditParams.RemoteAddress, auditParams)
		log.Debug("DeleteTenant uuid.Parse failed: %v", err)
		ctx.Error(http.StatusBadRequest, fmt.Sprintf("DeleteTenant uuid.Parse failed: %v", err))
		return
	}
	tenant, err := tenant_service.TenantByID(ctx, tenantID)
	if err != nil {
		if tenat_model.IsErrorTenantNotExists(err) {
			auditParams["error"] = "Error has occurred while getting tenant"
			audit.CreateAndSendEvent(audit.TenantDeleteEvent, requiredAuditParams.DoerName, requiredAuditParams.DoerID, audit.StatusFailure, requiredAuditParams.RemoteAddress, auditParams)
			log.Debug("DeleteTenant tenant_service.TenantByID failed to get tenant %s: %v", tenantID, err)
			ctx.Error(http.StatusNotFound, fmt.Sprintf("DeleteTenant tenant_service.TenantByID: %v", err))
		} else {
			auditParams["error"] = "Error has occurred while getting tenant"
			audit.CreateAndSendEvent(audit.TenantDeleteEvent, requiredAuditParams.DoerName, requiredAuditParams.DoerID, audit.StatusFailure, requiredAuditParams.RemoteAddress, auditParams)
			log.Error("DeleteTenant tenant_service.TenantByID failed to get tenant %s: %v", tenantID, err)
			ctx.Error(http.StatusInternalServerError, fmt.Sprintf("DeleteTenant tenant_service.TenantByID: %v", err))
		}
		auditParams["error"] = "Error has occurred while getting tenant"
		audit.CreateAndSendEvent(audit.TenantDeleteEvent, requiredAuditParams.DoerName, requiredAuditParams.DoerID, audit.StatusFailure, requiredAuditParams.RemoteAddress, auditParams)
		return
	}
	type auditValue struct {
		ID         string
		Name       string
		OrgKey     string
		IsActive   bool
		TenantUUID string
	}
	oldValue := auditValue{
		ID:         tenant.ID,
		Name:       tenant.Name,
		OrgKey:     tenant.OrgKey,
		IsActive:   tenant.IsActive,
		TenantUUID: tenantUUID.String(),
	}
	oldValueBytes, _ := json.Marshal(oldValue)
	auditParams["old_value"] = string(oldValueBytes)

	tenantOrganizations, err := tenat_model.GetTenantOrganizations(ctx, tenant.ID)
	if err != nil {
		auditParams["error"] = "Error has occurred while getting tenant organizations"
		audit.CreateAndSendEvent(audit.TenantDeleteEvent, requiredAuditParams.DoerName, requiredAuditParams.DoerID, audit.StatusFailure, requiredAuditParams.RemoteAddress, auditParams)
		log.Error("DeleteTenant tenant_service.GetTenantOrganizations failed: %v", err)
		ctx.ServerError("DeleteTenant tenant_service.GetTenantOrganizations failed: %v", err)
		return
	}
	orgIDs := make([]int64, len(tenantOrganizations))
	for idx, tenOrg := range tenantOrganizations {
		orgIDs[idx] = tenOrg.OrganizationID
	}
	organizations, err := organization.GetOrganizationByIDs(ctx, orgIDs)
	if err != nil {
		auditParams["error"] = "Error has occurred while getting tenant organizations"
		audit.CreateAndSendEvent(audit.TenantDeleteEvent, requiredAuditParams.DoerName, requiredAuditParams.DoerID, audit.StatusFailure, requiredAuditParams.RemoteAddress, auditParams)
		log.Error("DeleteTenant organization.GetOrganizationByIDs tenantIDs: %v", err)
		ctx.ServerError("DeleteTenant organization.GetOrganizationByIDs failed: %v", err)
		return
	}
	repos, _, err := repo_model.GetUserRepositories(&repo_model.SearchRepoOptions{Actor: ctx.Doer, OwnerIDs: orgIDs})
	if err != nil {
		auditParams["error"] = "Error has occurred while getting tenant repositories"
		audit.CreateAndSendEvent(audit.TenantDeleteEvent, requiredAuditParams.DoerName, requiredAuditParams.DoerID, audit.StatusFailure, requiredAuditParams.RemoteAddress, auditParams)
		log.Error("DeleteTenant repo_model.GetUserRepositories failed: %v", err)
		ctx.ServerError("DeleteTenant repo_model.GetUserRepositories failed: %v", err)
		return
	}
	for _, rep := range repos {
		errDeleteRepository := repository.DeleteRepository(ctx, ctx.Doer, rep, true)
		if errDeleteRepository != nil {
			auditParams["error"] = "Error has occurred while deleting repository"
			audit.CreateAndSendEvent(audit.TenantDeleteEvent, requiredAuditParams.DoerName, requiredAuditParams.DoerID, audit.StatusFailure, requiredAuditParams.RemoteAddress, auditParams)
			log.Error("DeleteTenant repository.DeleteRepository failed: %v", errDeleteRepository)
			ctx.ServerError("DeleteTenant repository.DeleteRepository failed: %v", err)
			return
		}
	}
	for _, org := range organizations {
		errDeleteOrg := org_service.DeleteOrganization(org)
		if errDeleteOrg != nil {
			auditParams["error"] = "Error has occurred while deleting organization"
			audit.CreateAndSendEvent(audit.TenantDeleteEvent, requiredAuditParams.DoerName, requiredAuditParams.DoerID, audit.StatusFailure, requiredAuditParams.RemoteAddress, auditParams)
			log.Error("DeleteTenant org_service.DeleteOrganization failed: %v", errDeleteOrg)
			ctx.ServerError("DeleteTenant org_service.DeleteOrganization failed: %v", errDeleteOrg)
			return
		}
		if err := s.repoRequestAccessor.RemoveCustomPrivilegesByTenantAndOrgID(accesser.RepoAccessRequest{OrgID: org.ID, TargetTenantID: tenantID}); err != nil {
			auditParams["error"] = "Error has occurred while removing custom privileges"
			audit.CreateAndSendEvent(audit.TenantDeleteEvent, requiredAuditParams.DoerName, requiredAuditParams.DoerID, audit.StatusFailure, requiredAuditParams.RemoteAddress, auditParams)
			log.Error("Error has occurred while removing privileges for tenantID %s, organization id: %d : %v", tenantID, org.ID, err)
			ctx.ServerError("Remove custom privileges failed: %v", errDeleteOrg)
			return
		}
	}

	if err := tenant_service.RemoveTenantByID(ctx, tenantUUID.String(), orgIDs); err != nil {
		auditParams["error"] = "Error has occurred while deleting tenant"
		audit.CreateAndSendEvent(audit.TenantDeleteEvent, requiredAuditParams.DoerName, requiredAuditParams.DoerID, audit.StatusFailure, requiredAuditParams.RemoteAddress, auditParams)
		log.Error("DeleteTenant tenant_service.RemoveTenantByID failed: %v", err)
		ctx.ServerError("DeleteTenant tenant_service.RemoveTenantByID failed: %v", err)
		return
	}

	if err := s.repoRequestAccessor.RemoveCustomPrivilegesByTenantAndOrgID(accesser.RepoAccessRequest{TargetTenantID: tenantID}); err != nil {
		auditParams["error"] = "Error has occurred while removing privileges"
		audit.CreateAndSendEvent(audit.TenantDeleteEvent, requiredAuditParams.DoerName, requiredAuditParams.DoerID, audit.StatusFailure, requiredAuditParams.RemoteAddress, auditParams)
		log.Error("Error has occurred while removing privileges for tenantID %s : %v", tenantID, err)
		ctx.ServerError("Error has occurred while removing privileges: %v", err)
		return
	}

	audit.CreateAndSendEvent(audit.TenantDeleteEvent, requiredAuditParams.DoerName, requiredAuditParams.DoerID, audit.StatusSuccess, requiredAuditParams.RemoteAddress, auditParams)
	ctx.JSON(http.StatusNoContent, nil)
}
