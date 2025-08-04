package protected_branch

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"code.gitea.io/gitea/models/tenant"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/sbt/audit"
	protected_brancher "code.gitea.io/gitea/services/protected_branch"
)

// DeleteProtectedBranchRulePost delete protected branch rule by id
func (s Server) DeleteProtectedBranchRulePost(ctx *context.Context) {
	ruleID := ctx.ParamsInt64("id")
	orgID := ctx.Repo.Repository.OwnerID
	auditParams := map[string]string{
		"repository":    ctx.Repo.Repository.Name,
		"repository_id": strconv.FormatInt(ctx.Repo.Repository.ID, 10),
		"project_id":    strconv.FormatInt(orgID, 10),
		"project_key":   ctx.Repo.Repository.OwnerName,
	}

	tenantOrganization, err := tenant.GetTenantOrganizationsByOrgId(ctx, orgID)
	if err != nil {
		auditParams["error"] = "Error has occurred while getting tenant organizations by org id"
		if tenant.IsErrorTenantNotExists(err) {
			ctx.NotFound("Tenant organizations not found", err)
		} else {
			ctx.ServerError("internal server error when get tenant organizations", err)
		}

		audit.CreateAndSendEvent(audit.BranchProtectionDeleteFromRepositoryEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}
	auditParams["tenant_id"] = tenantOrganization.TenantID
	auditParams["tenant_key"] = tenantOrganization.OrgKey

	if ruleID <= 0 {
		ctx.Flash.Error(ctx.Tr("repo.settings.remove_protected_branch_failed", fmt.Sprintf("%d", ruleID)))
		ctx.JSON(http.StatusOK, map[string]interface{}{
			"redirect": fmt.Sprintf("%s/settings/branches", ctx.Repo.RepoLink),
		})
		auditParams["error"] = "Protected branch rule id is empty"
		audit.CreateAndSendEvent(audit.BranchProtectionDeleteFromRepositoryEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}

	rule, err := s.protectedBranchManager.GetProtectedBranchRuleByID(ctx, ctx.Repo.Repository.ID, ruleID)
	if err != nil {
		if protected_brancher.IsProtectedBranchNotFoundError(err) {
			auditParams["error"] = "Protected branch rule not found by id"
		} else {
			auditParams["error"] = "Error has occurred while getting protected branch rule by id"
		}

		ctx.Flash.Error(ctx.Tr("repo.settings.remove_protected_branch_failed", fmt.Sprintf("%d", ruleID)))
		ctx.JSON(http.StatusOK, map[string]interface{}{
			"redirect": fmt.Sprintf("%s/settings/branches", ctx.Repo.RepoLink),
		})
		audit.CreateAndSendEvent(audit.BranchProtectionDeleteFromRepositoryEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}

	oldValue := s.auditConverter.Convert(*rule)
	oldValueBytes, err := json.Marshal(oldValue)

	if err != nil {
		auditParams["error"] = "Error has occured while serialize protected branch for audit"
		audit.CreateAndSendEvent(audit.BranchProtectionDeleteFromRepositoryEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		ctx.ServerError("Internal server error", err)
		return
	}
	auditParams["old_value"] = string(oldValueBytes)

	if err := s.protectedBranchManager.DeleteProtectedBranch(ctx, ctx.Repo.Repository.ID, ruleID); err != nil {
		ctx.Flash.Error(ctx.Tr("repo.settings.remove_protected_branch_failed", rule.RuleName))
		ctx.JSON(http.StatusOK, map[string]interface{}{
			"redirect": fmt.Sprintf("%s/settings/branches", ctx.Repo.RepoLink),
		})
		auditParams["error"] = "Error has occurred while deleting protected branch rule by id"
		audit.CreateAndSendEvent(audit.BranchProtectionDeleteFromRepositoryEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}

	audit.CreateAndSendEvent(audit.BranchProtectionDeleteFromRepositoryEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)
	ctx.Flash.Success(ctx.Tr("repo.settings.remove_protected_branch_success", rule.RuleName))
	ctx.JSON(http.StatusOK, map[string]interface{}{
		"redirect": fmt.Sprintf("%s/settings/branches", ctx.Repo.RepoLink),
	})
}
