package protected_branch

import (
	"encoding/json"
	"net/http"
	"strconv"

	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/sbt/audit"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/routers/api/v3/models"
	protected_brancher "code.gitea.io/gitea/services/protected_branch"
)

// Get all branch protection rules for repository
func (s Server) GetBranchProtections(ctx *context.APIContext) {
	repo := ctx.Repo.Repository
	rules, err := s.protectedBranchManager.FindRepoProtectedBranchRules(ctx, repo.ID)
	if err != nil {
		log.Error("Error occurred while getting branch protections in repository: %s/%s", repo.Name, err)
		ctx.Error(http.StatusInternalServerError, "Internal server error", err)
		return
	}
	ctx.JSON(http.StatusOK, s.branchProtectionConverter.ToBranchProtectionRulesBody(rules))
}

// Get branch protection rule by name and repository
func (s Server) GetBranchProtection(ctx *context.APIContext) {
	repo := ctx.Repo.Repository
	name := ctx.Params(":branch_name")
	rule, err := s.protectedBranchManager.GetProtectedBranchRuleByName(ctx, repo.ID, name)
	if err != nil {
		switch {
		case protected_brancher.IsProtectedBranchNotFoundError(err):
			log.Warn("Not found protected branch: %s", name)
			ctx.Error(http.StatusNotFound, "Protected Branch rule no found", err)
		default:
			log.Error("Error occurred while getting branch protection: %s/%s", name, err)
			ctx.Error(http.StatusInternalServerError, "Internal server error", err)
		}
		return
	}
	ctx.JSON(http.StatusOK, s.branchProtectionConverter.ToBranchProtectionBody(*rule))
}

// Create branch protection rule for repository
// first initialize audit, then validate, then create protected branch
func (s Server) CreateBranchProtection(ctx *context.APIContext) {
	repo := ctx.Repo.Repository
	opt := web.GetForm(ctx).(*models.BranchProtectionBody)
	auditParams := map[string]string{
		"repository":    ctx.Repo.Repository.Name,
		"repository_id": strconv.FormatInt(ctx.Repo.Repository.ID, 10),
		"project_id":    strconv.FormatInt(ctx.Repo.Repository.OwnerID, 10),
		"tenant_id":     ctx.Tenant.TenantID,
		"tenant_key":    ctx.Tenant.OrgKey,
		"project_key":   ctx.Repo.Repository.OwnerName,
	}
	protectedBranch := s.branchProtectionConverter.ToProtectedBranch(ctx, *opt)

	auditValues := s.auditConverter.Convert(*protectedBranch)
	newValueBytes, err := json.Marshal(auditValues)
	if err != nil {
		log.Error("Error serialize protected branch audit: %v", err)
		auditParams["error"] = "Error has occured while serialize protected branch audit"
		audit.CreateAndSendEvent(audit.BranchProtectionAddToRepositoryEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		ctx.Error(http.StatusInternalServerError, "Internal server error", err)
		return
	}
	auditParams["new_value"] = string(newValueBytes)

	if err := opt.Validate(); err != nil {
		log.Warn("Bad request when validate request body: %s", err)
		auditParams["error"] = "Error has occurred while validating form"
		audit.CreateAndSendEvent(audit.BranchProtectionAddToRepositoryEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		ctx.Error(http.StatusBadRequest, "Invalid request", err)
		return
	}
	ruleName := protectedBranch.RuleName

	protectedBranch, err = s.protectedBranchManager.CreateProtectedBranch(ctx, repo, protectedBranch)
	if err != nil {
		switch {
		case protected_brancher.IsProtectedBranchAlreadyExistError(err):
			log.Warn("Branch protection with name: %s, aldeay exist", ruleName)
			ctx.Error(http.StatusConflict, "Protected Branch rule already exist", err)
		default:
			log.Error("Error occurred while creating protected branch: %s", err)
			ctx.Error(http.StatusInternalServerError, "Internal server error", err)
		}
		auditParams["error"] = "Error has occurred while creating protected branch"
		audit.CreateAndSendEvent(audit.BranchProtectionAddToRepositoryEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}

	audit.CreateAndSendEvent(audit.BranchProtectionAddToRepositoryEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)
	ctx.JSON(http.StatusCreated, s.branchProtectionConverter.ToBranchProtectionBody(*protectedBranch))
}

// Update branch protection rule by name and repository
// first initialize audit, then validate, then check exisiting protected_brach,
// then update protected branch
func (s Server) UpdateBranchProtection(ctx *context.APIContext) {
	repo := ctx.Repo.Repository
	opt := web.GetForm(ctx).(*models.BranchProtectionBody)
	name := ctx.Params(":branch_name")

	auditParams := map[string]string{
		"repository":    ctx.Repo.Repository.Name,
		"repository_id": strconv.FormatInt(ctx.Repo.Repository.ID, 10),
		"project_id":    strconv.FormatInt(ctx.Repo.Repository.OwnerID, 10),
		"tenant_id":     ctx.Tenant.TenantID,
		"tenant_key":    ctx.Tenant.OrgKey,
		"project_key":   ctx.Repo.Repository.OwnerName,
	}

	protectedBranch, err := s.protectedBranchManager.GetProtectedBranchRuleByName(ctx, repo.ID, name)
	if err != nil {
		switch {
		case protected_brancher.IsProtectedBranchNotFoundError(err):
			log.Warn("Not found protected branch: %s", name)
			ctx.Error(http.StatusNotFound, "Protected Branch rule no found", err)
		default:
			log.Error("Error occurred while getting branch protection: %s/%s", name, err)
			ctx.Error(http.StatusInternalServerError, "Internal server error", err)
		}
		auditParams["error"] = "Error has occurred while getting protected branch"
		audit.CreateAndSendEvent(audit.BranchProtectionUpdateInRepositoryEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}

	auditValues := s.auditConverter.Convert(*protectedBranch)
	oldValueBytes, err := json.Marshal(auditValues)
	if err != nil {
		log.Error("Error serialize protected branch audit: %v", err)
		auditParams["error"] = "Error has occured while serialize protected branch audit"
		audit.CreateAndSendEvent(audit.BranchProtectionAddToRepositoryEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		ctx.Error(http.StatusInternalServerError, "Internal server error", err)
		return
	}

	auditParams["old_value"] = string(oldValueBytes)

	if err := opt.Validate(); err != nil {
		log.Warn("Bad request when validate request body: %s", err)
		auditParams["error"] = "Error has occurred while validating form"
		audit.CreateAndSendEvent(audit.BranchProtectionUpdateInRepositoryEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		ctx.Error(http.StatusBadRequest, "Invalid request", err)
		return
	}

	protectedBranch = s.branchProtectionConverter.ToProtectedBranch(ctx, *opt)
	protectedBranch, err = s.protectedBranchManager.UpdateProtectedBranch(ctx, repo, protectedBranch, name)
	if err != nil {
		switch {
		case protected_brancher.IsProtectedBranchNotFoundError(err):
			log.Warn("Not found protected branch: %s", name)
			ctx.Error(http.StatusNotFound, "Protected Branch rule no found", err)
		default:
			log.Error("Error occurred while creating branch protection: %s/%s", name, err)
			ctx.Error(http.StatusInternalServerError, "Internal server error", err)
		}
		auditParams["error"] = "Error has occurred while updating protected branch"
		audit.CreateAndSendEvent(audit.BranchProtectionUpdateInRepositoryEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}

	auditValues = s.auditConverter.Convert(*protectedBranch)
	newValueBytes, err := json.Marshal(auditValues)
	if err != nil {
		log.Error("Error serialize protected branch audit: %v", err)
		auditParams["error"] = "Error has occured while serialize protected branch audit"
		audit.CreateAndSendEvent(audit.BranchProtectionAddToRepositoryEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		ctx.Error(http.StatusInternalServerError, "Internal server error", err)
		return
	}

	auditParams["new_value"] = string(newValueBytes)

	audit.CreateAndSendEvent(audit.BranchProtectionUpdateInRepositoryEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)
	ctx.JSON(http.StatusOK, s.branchProtectionConverter.ToBranchProtectionBody(*protectedBranch))
}

// Delete branch protection rule by name and repository
// initialize audit, then check on existing protected_branch,
// delete protected branch
func (s Server) DeleteBranchProtection(ctx *context.APIContext) {
	repo := ctx.Repo.Repository
	name := ctx.Params(":branch_name")

	auditParams := map[string]string{
		"repository":    ctx.Repo.Repository.Name,
		"repository_id": strconv.FormatInt(ctx.Repo.Repository.ID, 10),
		"project_id":    strconv.FormatInt(ctx.Repo.Repository.OwnerID, 10),
		"tenant_id":     ctx.Tenant.TenantID,
		"tenant_key":    ctx.Tenant.OrgKey,
		"project_key":   ctx.Repo.Repository.OwnerName,
	}

	protectedBranch, err := s.protectedBranchManager.GetProtectedBranchRuleByName(ctx, repo.ID, name)
	if err != nil {
		switch {
		case protected_brancher.IsProtectedBranchNotFoundError(err):
			log.Warn("Not found protected branch: %s", name)
			ctx.Error(http.StatusNotFound, "Protected Branch rule no found", err)
		default:
			log.Error("Error occurred while getting branch protection: %s/%s", name, err)
			ctx.Error(http.StatusInternalServerError, "Internal server error", err)
		}

		auditParams["error"] = "Error has occurred while getting protected branch"
		audit.CreateAndSendEvent(audit.BranchProtectionDeleteFromRepositoryEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}

	auditValues := s.auditConverter.Convert(*protectedBranch)
	oldValueBytes, err := json.Marshal(auditValues)
	if err != nil {
		log.Error("Error serialize protected branch audit: %v", err)
		auditParams["error"] = "Error has occured while serialize protected branch audit"
		audit.CreateAndSendEvent(audit.BranchProtectionDeleteFromRepositoryEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		ctx.Error(http.StatusInternalServerError, "Internal server error", err)
		return
	}

	auditParams["old_value"] = string(oldValueBytes)

	err = s.protectedBranchManager.DeleteProtectedBranchByRuleName(ctx, repo, name)
	if err != nil {
		switch {
		case protected_brancher.IsProtectedBranchNotFoundError(err):
			log.Warn("Not found protected branch: %s", name)
			ctx.Error(http.StatusNotFound, "Protected Branch rule no found", err)
		default:
			log.Error("Error occurred while deleting branch protection: %s/%s", name, err)
			ctx.Error(http.StatusInternalServerError, "Internal server error", err)
		}

		auditParams["error"] = "Error has occurred while deleting protected branch"
		audit.CreateAndSendEvent(audit.BranchProtectionDeleteFromRepositoryEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}

	audit.CreateAndSendEvent(audit.BranchProtectionDeleteFromRepositoryEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)
	ctx.Status(http.StatusNoContent)
}
