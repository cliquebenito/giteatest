// Copyright 2014 The Gogs Authors. All rights reserved.
// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package org

import (
	"net/http"
	"strconv"

	"code.gitea.io/gitea/models/role_model"
	"code.gitea.io/gitea/models/tenant"
	"code.gitea.io/gitea/modules/json"
	"code.gitea.io/gitea/modules/sbt/audit"

	"code.gitea.io/gitea/models"
	"code.gitea.io/gitea/models/db"
	repo_model "code.gitea.io/gitea/models/repo"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/models/webhook"
	"code.gitea.io/gitea/modules/base"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/log"
	repo_module "code.gitea.io/gitea/modules/repository"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/web"
	user_setting "code.gitea.io/gitea/routers/web/user/setting"
	"code.gitea.io/gitea/services/forms"
	org_service "code.gitea.io/gitea/services/org"
	repo_service "code.gitea.io/gitea/services/repository"
	user_service "code.gitea.io/gitea/services/user"
)

const (
	// tplSettingsOptions template path for render settings
	tplSettingsOptions base.TplName = "org/settings/options"
	// tplSettingsDelete template path for render delete repository
	tplSettingsDelete base.TplName = "org/settings/delete"
	// tplSettingsHooks template path for render hook settings
	tplSettingsHooks base.TplName = "org/settings/hooks"
	// tplSettingsLabels template path for render labels settings
	tplSettingsLabels base.TplName = "org/settings/labels"
)

// Settings render the main settings page
func Settings(ctx *context.Context) {
	tenantId, err := tenant.GetTenantByOrgIdOrDefault(ctx, ctx.Org.Organization.ID)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, err.Error())
		return
	}

	ctx.Data["TenantID"] = tenantId
	ctx.Data["Title"] = ctx.Tr("org.settings")
	ctx.Data["PageIsOrgSettings"] = true
	ctx.Data["PageIsSettingsOptions"] = true
	ctx.Data["CurrentVisibility"] = ctx.Org.Organization.Visibility
	ctx.Data["RepoAdminChangeTeamAccess"] = ctx.Org.Organization.RepoAdminChangeTeamAccess
	ctx.Data["ContextUser"] = ctx.ContextUser
	ctx.HTML(http.StatusOK, tplSettingsOptions)
}

// SettingsPost response for settings change submitted
func SettingsPost(ctx *context.Context) {
	tenantId, err := tenant.GetTenantByOrgIdOrDefault(ctx, ctx.Org.Organization.ID)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, err.Error())
		return
	}

	ctx.Data["TenantID"] = tenantId
	form := web.GetForm(ctx).(*forms.UpdateOrgSettingForm)
	ctx.Data["Title"] = ctx.Tr("org.settings")
	ctx.Data["PageIsOrgSettings"] = true
	ctx.Data["PageIsSettingsOptions"] = true
	ctx.Data["CurrentVisibility"] = ctx.Org.Organization.Visibility

	org := ctx.Org.Organization

	auditParams := map[string]string{
		"project":    ctx.Org.Organization.Name,
		"project_id": strconv.FormatInt(ctx.Org.Organization.ID, 10),
	}

	type auditValue struct {
		Name                      string
		LowerName                 string
		FullName                  string
		Description               string
		Website                   string
		Location                  string
		Visibility                string
		RepoAdminChangeTeamAccess bool
		MaxRepoCreation           string
	}

	oldValue := auditValue{
		Name:                      org.Name,
		LowerName:                 org.LowerName,
		FullName:                  org.FullName,
		Description:               org.Description,
		Website:                   org.Website,
		Location:                  org.Location,
		Visibility:                org.Visibility.String(),
		RepoAdminChangeTeamAccess: org.RepoAdminChangeTeamAccess,
		MaxRepoCreation:           strconv.Itoa(org.MaxRepoCreation),
	}
	oldValueBytes, _ := json.Marshal(oldValue)
	auditParams["old_value"] = string(oldValueBytes)

	newValue := auditValue{
		Name:                      org.Name,
		LowerName:                 org.LowerName,
		FullName:                  form.FullName,
		Description:               form.Description,
		Website:                   form.Website,
		Location:                  form.Location,
		Visibility:                form.Visibility.String(),
		RepoAdminChangeTeamAccess: form.RepoAdminChangeTeamAccess,
		MaxRepoCreation:           strconv.Itoa(form.MaxRepoCreation),
	}

	newValueBytes, _ := json.Marshal(newValue)
	auditParams["new_value"] = string(newValueBytes)

	if ctx.HasError() {
		ctx.HTML(http.StatusOK, tplSettingsOptions)
		auditParams["error"] = "Error occurs in form validation"
		audit.CreateAndSendEvent(audit.ProjectSettingsChangeEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}

	if setting.SourceControl.Enabled && setting.SourceControl.TenantWithRoleModeEnabled && form.Visibility == 0 {
		log.Debug("Cannot change to public organization")
		ctx.RenderWithErr(ctx.Tr("org.form.have_public_org_not_allowed"), tplSettingsOptions, &form)
		auditParams["error"] = "Error occurs in form validation"
		audit.CreateAndSendEvent(audit.ProjectSettingsChangeEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}

	nameChanged := org.Name != form.Name

	// Check if organization name has been changed.
	if nameChanged {
		log.Warn("Project name cannot change")
	}

	if ctx.Doer.IsAdmin {
		org.MaxRepoCreation = form.MaxRepoCreation
	}

	org.FullName = form.FullName
	org.Description = form.Description
	org.Website = form.Website
	org.Location = form.Location
	org.RepoAdminChangeTeamAccess = form.RepoAdminChangeTeamAccess

	visibilityChanged := form.Visibility != org.Visibility
	org.Visibility = form.Visibility

	if err := user_model.UpdateUser(ctx, org.AsUser(), false); err != nil {
		ctx.ServerError("UpdateUser", err)
		auditParams["error"] = "Error has occurred while updating organization"
		audit.CreateAndSendEvent(audit.ProjectSettingsChangeEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}

	// update forks visibility
	if visibilityChanged {
		if setting.SourceControl.Enabled && setting.SourceControl.TenantWithRoleModeEnabled {
			if form.Visibility == 1 {
				err := role_model.AddProjectToInnerSource(org)
				if err != nil {
					auditParams["error"] = "Error has occurred while updating organization"
					audit.CreateAndSendEvent(audit.ProjectSettingsChangeEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
					ctx.RenderWithErr(ctx.Tr("form.privileges_not_granted"), tplCreateOrg, &form)
					return
				}
			} else if form.Visibility == 2 {
				err := role_model.RemoveProjectToInnerSource(org)
				if err != nil {
					auditParams["error"] = "Error has occurred while updating organization"
					audit.CreateAndSendEvent(audit.ProjectSettingsChangeEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
					ctx.RenderWithErr(ctx.Tr("form.privileges_not_granted"), tplCreateOrg, &form)
					return
				}
			}
		}
		repos, _, err := repo_model.GetUserRepositories(&repo_model.SearchRepoOptions{
			Actor: org.AsUser(), Private: true, ListOptions: db.ListOptions{Page: 1, PageSize: org.NumRepos},
		})
		if err != nil {
			ctx.ServerError("GetRepositories", err)
			auditParams["error"] = "Error has occurred while getting organization repositories"
			audit.CreateAndSendEvent(audit.ProjectSettingsChangeEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}
		for _, repo := range repos {
			repo.OwnerName = org.Name
			if err := repo_service.UpdateRepository(ctx, repo, true); err != nil {
				ctx.ServerError("UpdateRepository", err)
				auditParams["error"] = "Error has occurred while updating organization repositories"
				audit.CreateAndSendEvent(audit.ProjectSettingsChangeEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
				return
			}
		}
	} else if nameChanged {
		if err := repo_model.UpdateRepositoryOwnerNames(org.ID, org.Name); err != nil {
			ctx.ServerError("UpdateRepository", err)
			auditParams["error"] = "Error has occurred while updating organization repositories"
			audit.CreateAndSendEvent(audit.ProjectSettingsChangeEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}
	}

	audit.CreateAndSendEvent(audit.ProjectSettingsChangeEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)
	log.Trace("Organization setting updated: %s", org.Name)
	ctx.Flash.Success(ctx.Tr("org.settings.update_setting_success"))
	ctx.Redirect(ctx.Org.OrgLink + "/settings")
}

// SettingsAvatar response for change avatar on settings page
func SettingsAvatar(ctx *context.Context) {
	form := web.GetForm(ctx).(*forms.AvatarForm)
	form.Source = forms.AvatarLocal
	auditParams := map[string]string{
		"project_id": strconv.FormatInt(ctx.Org.Organization.ID, 10),
		"project":    ctx.Org.Organization.Name,
	}
	if err := user_setting.UpdateAvatarSetting(ctx, form, ctx.Org.Organization.AsUser()); err != nil {
		ctx.Flash.Error(err.Error())
		auditParams["error"] = "Error has occurred while updating avatar settings"
		audit.CreateAndSendEvent(audit.ProjectAvatarChange, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
	} else {
		ctx.Flash.Success(ctx.Tr("org.settings.update_avatar_success"))
		audit.CreateAndSendEvent(audit.ProjectAvatarChange, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)
	}

	ctx.Redirect(ctx.Org.OrgLink + "/settings")
}

// SettingsDeleteAvatar response for delete avatar on settings page
func SettingsDeleteAvatar(ctx *context.Context) {
	auditParams := map[string]string{
		"project_id": strconv.FormatInt(ctx.Org.Organization.ID, 10),
		"project":    ctx.Org.Organization.Name,
	}
	if err := user_service.DeleteAvatar(ctx.Org.Organization.AsUser()); err != nil {
		auditParams["error"] = "Error has occurred while deleting avatar settings"
		audit.CreateAndSendEvent(audit.ProjectAvatarDelete, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		ctx.Flash.Error(err.Error())
		return
	}

	ctx.Flash.Success(ctx.Tr("settings.delete_user_avatar_success"))
	audit.CreateAndSendEvent(audit.ProjectAvatarDelete, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)
	ctx.Redirect(ctx.Org.OrgLink + "/settings")
}

// SettingsDelete response for deleting an organization
func SettingsDelete(ctx *context.Context) {
	tenantId, err := tenant.GetTenantByOrgIdOrDefault(ctx, ctx.Org.Organization.ID)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, err.Error())
		return
	}

	ctx.Data["TenantID"] = tenantId
	ctx.Data["Title"] = ctx.Tr("org.settings")
	ctx.Data["PageIsOrgSettings"] = true
	ctx.Data["PageIsSettingsDelete"] = true

	if ctx.Req.Method == "POST" {
		auditParams := map[string]string{
			"project":    ctx.Org.Organization.Name,
			"project_id": strconv.FormatInt(ctx.Org.Organization.ID, 10),
		}
		if ctx.Org.Organization.Name != ctx.FormString("org_name") {
			ctx.Data["Err_OrgName"] = true
			ctx.RenderWithErr(ctx.Tr("form.enterred_invalid_org_name"), tplSettingsDelete, nil)
			auditParams["error"] = "Entered invalid organization name"
			audit.CreateAndSendEvent(audit.ProjectDeleteEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}

		if err := org_service.DeleteOrganization(ctx.Org.Organization); err != nil {
			if models.IsErrUserOwnRepos(err) {
				ctx.Flash.Error(ctx.Tr("form.org_still_own_repo"))
				ctx.Redirect(ctx.Org.OrgLink + "/settings/delete")
				auditParams["error"] = "Organization still owns one or more repositories"
			} else if models.IsErrUserOwnPackages(err) {
				ctx.Flash.Error(ctx.Tr("form.org_still_own_packages"))
				ctx.Redirect(ctx.Org.OrgLink + "/settings/delete")
				auditParams["error"] = "Organization still owns one or more packages"
			} else {
				ctx.ServerError("DeleteOrganization", err)
				auditParams["error"] = "Error has occurred while deleting organization"
			}
			audit.CreateAndSendEvent(audit.ProjectDeleteEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		} else {
			audit.CreateAndSendEvent(audit.ProjectDeleteEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)
			log.Trace("Organization deleted: %s", ctx.Org.Organization.Name)
			ctx.Redirect(setting.AppSubURL + "/")
		}
		return
	}

	ctx.HTML(http.StatusOK, tplSettingsDelete)
}

// Webhooks render webhook list page
func Webhooks(ctx *context.Context) {
	tenantId, err := tenant.GetTenantByOrgIdOrDefault(ctx, ctx.Org.Organization.ID)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, err.Error())
		return
	}

	ctx.Data["TenantID"] = tenantId
	ctx.Data["Title"] = ctx.Tr("org.settings")
	ctx.Data["PageIsOrgSettings"] = true
	ctx.Data["PageIsSettingsHooks"] = true
	ctx.Data["BaseLink"] = ctx.Org.OrgLink + "/settings/hooks"
	ctx.Data["BaseLinkNew"] = ctx.Org.OrgLink + "/settings/hooks"
	ctx.Data["Description"] = ctx.Tr("org.settings.hooks_desc")

	ws, err := webhook.ListWebhooksByOpts(ctx, &webhook.ListWebhookOptions{OwnerID: ctx.Org.Organization.ID})
	if err != nil {
		ctx.ServerError("ListWebhooksByOpts", err)
		return
	}

	ctx.Data["Webhooks"] = ws
	ctx.HTML(http.StatusOK, tplSettingsHooks)
}

// DeleteWebhook response for delete webhook
func DeleteWebhook(ctx *context.Context) {
	auditParams := map[string]string{
		"hook_id": strconv.FormatInt(ctx.FormInt64("id"), 10),
	}
	if err := webhook.DeleteWebhookByOwnerID(ctx.Org.Organization.ID, ctx.FormInt64("id")); err != nil {
		ctx.Flash.Error("DeleteWebhookByOwnerID: " + err.Error())
		auditParams["error"] = "Error has occurred while deleting organization hook"
		audit.CreateAndSendEvent(audit.HookInProjectRemoveEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
	} else {
		ctx.Flash.Success(ctx.Tr("repo.settings.webhook_deletion_success"))
		audit.CreateAndSendEvent(audit.HookInProjectRemoveEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)
	}

	ctx.JSON(http.StatusOK, map[string]interface{}{
		"redirect": ctx.Org.OrgLink + "/settings/hooks",
	})
}

// Labels render organization labels page
func Labels(ctx *context.Context) {
	tenantId, err := tenant.GetTenantByOrgIdOrDefault(ctx, ctx.Org.Organization.ID)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, err.Error())
		return
	}

	ctx.Data["TenantID"] = tenantId
	ctx.Data["Title"] = ctx.Tr("repo.labels")
	ctx.Data["PageIsOrgSettings"] = true
	ctx.Data["PageIsOrgSettingsLabels"] = true
	ctx.Data["LabelTemplateFiles"] = repo_module.LabelTemplateFiles
	ctx.HTML(http.StatusOK, tplSettingsLabels)
}
