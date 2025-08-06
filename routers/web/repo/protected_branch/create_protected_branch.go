package protected_branch

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"code.gitea.io/gitea/models/git/protected_branch"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/tenant"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/sbt/audit"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/services/forms"
	protected_brancher "code.gitea.io/gitea/services/protected_branch"
	pull_service "code.gitea.io/gitea/services/pull"
)

// SetDefaultBranchPost set default branch
func (s Server) SetDefaultBranchPost(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("repo.settings.branches.update_default_branch")
	ctx.Data["PageIsSettingsBranches"] = true

	repo := ctx.Repo.Repository

	switch ctx.FormString("action") {
	case "default_branch":
		if ctx.HasError() {
			ctx.HTML(http.StatusOK, tplBranches)
			return
		}

		branch := ctx.FormString("branch")
		if !ctx.Repo.GitRepo.IsBranchExist(branch) {
			ctx.Status(http.StatusNotFound)
			return
		} else if repo.DefaultBranch != branch {
			repo.DefaultBranch = branch
			if err := ctx.Repo.GitRepo.SetDefaultBranch(branch); err != nil {
				if !git.IsErrUnsupportedVersion(err) {
					ctx.ServerError("SetDefaultBranch", err)
					return
				}
			}
			if err := repo_model.UpdateDefaultBranch(repo); err != nil {
				ctx.ServerError("SetDefaultBranch", err)
				return
			}
		}

		log.Trace("Repository basic settings updated: %s/%s", ctx.Repo.Owner.Name, repo.Name)

		ctx.Flash.Success(ctx.Tr("repo.settings.update_settings_success"))
		ctx.Redirect(setting.AppSubURL + ctx.Req.URL.EscapedPath())
	default:
		ctx.NotFound("", nil)
	}
}

// SettingsProtectedBranchPost updates the protected branch settings
func (s Server) SettingsProtectedBranchPost(ctx *context.Context) {
	var err error
	f := web.GetForm(ctx).(*forms.ProtectBranchForm)
	orgID := ctx.Repo.Repository.OwnerID

	auditParams := map[string]string{
		"repository":    ctx.Repo.Repository.Name,
		"repository_id": strconv.FormatInt(ctx.Repo.Repository.ID, 10),
		"project_id":    strconv.FormatInt(orgID, 10),
		"project_key":   ctx.Repo.Repository.OwnerName,
	}
	var auditEvent audit.Event
	if f.RuleID > 0 {
		auditEvent = audit.BranchProtectionUpdateInRepositoryEvent
	} else {
		auditEvent = audit.BranchProtectionAddToRepositoryEvent
	}

	tenantOrganization, err := tenant.GetTenantOrganizationsByOrgId(ctx, orgID)
	if err != nil {
		auditParams["error"] = "Error has occurred while getting tenant organizations by org id"
		if tenant.IsErrorTenantNotExists(err) {
			ctx.NotFound("Tenant organizations not found", err)
		} else {
			ctx.ServerError("internal server error when get tenant organizations", err)
		}

		audit.CreateAndSendEvent(auditEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}
	auditParams["tenant_id"] = tenantOrganization.TenantID
	auditParams["tenant_key"] = tenantOrganization.OrgKey

	protectBranch := &protected_branch.ProtectedBranch{}
	protectBranch, err = s.formConverter.ConvertProtectBranchFormToProtectedBranchRule(f, protectBranch)
	if err != nil {
		auditParams["error"] = "Error has occured while convert form to protected branch"
		audit.CreateAndSendEvent(auditEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		ctx.ServerError("Internal server error", err)
		return
	}
	auditValues := s.auditConverter.Convert(*protectBranch)
	newValueBytes, err := json.Marshal(auditValues)
	if err != nil {
		auditParams["error"] = "Error has occured while serialize protected branch for audit"
		audit.CreateAndSendEvent(auditEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		ctx.ServerError("Internal server error", err)
		return
	}
	auditParams["new_value"] = string(newValueBytes)

	if f.RuleName == "" {
		ctx.Flash.Error(ctx.Tr("repo.settings.protected_branch_required_rule_name"))
		ctx.Redirect(fmt.Sprintf("%s/settings/branches/edit", ctx.Repo.RepoLink))
		auditParams["error"] = "Protected branch required rule name"
		audit.CreateAndSendEvent(auditEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}

	if f.RuleID > 0 {
		// If the RuleID isn't 0, it must be an edit operation. So we get rule by id.
		protectBranch, err = s.protectedBranchManager.GetProtectedBranchRuleByID(ctx, ctx.Repo.Repository.ID, f.RuleID)
		if err != nil && !protected_brancher.IsProtectedBranchNotFoundError(err) {
			ctx.ServerError("GetProtectBranchOfRepoByID", err)
			auditParams["error"] = "Error has occurred while getting protected branch rule by id"
			audit.CreateAndSendEvent(auditEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}
		if protectBranch != nil && protectBranch.RuleName != f.RuleName {
			// RuleName changed. We need to check if there is a rule with the same name.
			// If a rule with the same name exists, an error should be returned.
			oldValue := s.auditConverter.Convert(*protectBranch)
			oldValueBytes, err := json.Marshal(oldValue)
			if err != nil {
				auditParams["error"] = "Error has occured while serialize protected branch for audit"
				audit.CreateAndSendEvent(auditEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
				ctx.ServerError("Internal server error", err)
				return
			}
			auditParams["old_value"] = string(oldValueBytes)

			sameNameProtectBranch, err := s.protectedBranchManager.GetProtectedBranchRuleByName(ctx, ctx.Repo.Repository.ID, f.RuleName)
			if err != nil && !protected_brancher.IsProtectedBranchNotFoundError(err) {
				ctx.ServerError("GetProtectBranchOfRepoByName", err)
				auditParams["error"] = "Error has occurred while getting protected branch rule by name"
				audit.CreateAndSendEvent(auditEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
				return
			}
			if sameNameProtectBranch != nil {
				protectBranchMessage := fmt.Sprintf("<a href=\"%s\">%s</a>", fmt.Sprintf("%s/settings/branches/edit?rule_name=%s", ctx.Repo.RepoLink, protectBranch.RuleName), f.RuleName)
				ctx.Flash.Error(ctx.Tr("repo.settings.protected_branch_duplicate_rule_name", protectBranchMessage))
				ctx.Redirect(fmt.Sprintf("%s/settings/branches/edit?rule_name=%s", ctx.Repo.RepoLink, protectBranch.RuleName))
				auditParams["error"] = "Protected branch duplicate rule name"
				audit.CreateAndSendEvent(auditEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
				return
			}
		}
	} else {
		protectBranch, err = s.protectedBranchManager.GetProtectedBranchRuleByName(ctx, ctx.Repo.Repository.ID, f.RuleName)
		if err != nil && !protected_brancher.IsProtectedBranchNotFoundError(err) {
			ctx.ServerError("GetProtectBranchOfRepoByName", err)
			auditParams["error"] = "Error has occurred while getting protected branch rule by name"
			audit.CreateAndSendEvent(auditEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}
		// обработка кейса когда пользователь пытается создать правила для уже существующего правила
		if protectBranch != nil {
			protectBranchMessage := fmt.Sprintf("<a href=\"%s\">%s</a>", fmt.Sprintf("%s/settings/branches/edit?rule_name=%s", ctx.Repo.RepoLink, protectBranch.RuleName), f.RuleName)
			ctx.Flash.Error(ctx.Tr("repo.settings.protected_branch_duplicate_rule_name", protectBranchMessage))
			ctx.Redirect(fmt.Sprintf("%s/settings/branches/edit", ctx.Repo.RepoLink))
			auditParams["error"] = "Protected branch duplicate rule name"
			audit.CreateAndSendEvent(auditEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}
	}
	if protectBranch == nil {
		// No options found, create defaults.
		protectBranch = &protected_branch.ProtectedBranch{
			RepoID:   ctx.Repo.Repository.ID,
			RuleName: f.RuleName,
		}
		auditEvent = audit.BranchProtectionAddToRepositoryEvent
	} else {
		auditEvent = audit.BranchProtectionUpdateInRepositoryEvent

		oldValue := s.auditConverter.Convert(*protectBranch)
		oldValueBytes, err := json.Marshal(oldValue)

		if err != nil {
			auditParams["error"] = "Error has occured while serialize protected branch for audit"
			audit.CreateAndSendEvent(auditEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			ctx.ServerError("Internal server error", err)
			return
		}
		auditParams["old_value"] = string(oldValueBytes)
	}

	// после похода в бд нужно обновить параметры
	protectBranch, err = s.formConverter.ConvertProtectBranchFormToProtectedBranchRule(f, protectBranch)
	if err != nil {
		auditParams["error"] = "Error has occured while convert form to protected branch"
		audit.CreateAndSendEvent(auditEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		ctx.ServerError("Internal server error", err)
		return
	}

	if f.RequiredApprovals < 0 {
		ctx.Flash.Error(ctx.Tr("repo.settings.protected_branch_required_approvals_min"))
		ctx.Redirect(fmt.Sprintf("%s/settings/branches/edit?rule_name=%s", ctx.Repo.RepoLink, f.RuleName))
		auditParams["error"] = "The number of approvals required for protected branch cannot be negative"
		audit.CreateAndSendEvent(auditEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}

	err = s.protectedBranchManager.UpsertProtectBranch(ctx, ctx.Repo.Repository, protectBranch, protected_branch.WhitelistOptions{
		UserIDs:          protectBranch.WhitelistUserIDs,
		MergeUserIDs:     protectBranch.MergeWhitelistUserIDs,
		ApprovalsUserIDs: protectBranch.ApprovalsWhitelistUserIDs,
		DeleteUserIDs:    protectBranch.DeleterWhitelistUserIDs,
		ForcePushUserIDs: protectBranch.ForcePushWhitelistUserIDs,
	})
	if err != nil {
		ctx.ServerError("UpdateProtectBranch", err)
		auditParams["error"] = "Error has occurred while saving protect branch"
		audit.CreateAndSendEvent(auditEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}

	// FIXME: since we only need to recheck files protected rules, we could improve this
	matchedBranches, err := s.protectedBranchManager.FindAllMatchedBranches(ctx, ctx.Repo.GitRepo, protectBranch.RuleName)
	if err != nil {
		ctx.ServerError("FindAllMatchedBranches", err)
		auditParams["error"] = "Error has occurred while finding all matched branches"
		audit.CreateAndSendEvent(auditEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}
	for _, branchName := range matchedBranches {
		if err = pull_service.CheckPRsForBaseBranch(ctx.Repo.Repository, branchName); err != nil {
			ctx.ServerError("CheckPRsForBaseBranch", err)
			auditParams["error"] = "Error has occurred while checking pull requests for base branch"
			audit.CreateAndSendEvent(auditEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}
	}

	audit.CreateAndSendEvent(auditEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)

	ctx.Flash.Success(ctx.Tr("repo.settings.update_protect_branch_success", protectBranch.RuleName))
	ctx.Redirect(fmt.Sprintf("%s/settings/branches?rule_name=%s", ctx.Repo.RepoLink, protectBranch.RuleName))
}
