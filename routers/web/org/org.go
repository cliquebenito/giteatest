// Copyright 2014 The Gogs Authors. All rights reserved.
// Copyright 2018 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package org

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/organization"
	"code.gitea.io/gitea/models/role_model"
	"code.gitea.io/gitea/models/tenant"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/base"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/json"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/sbt/audit"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/services/forms"
	tenant_service "code.gitea.io/gitea/services/tenant"

	"github.com/google/uuid"
)

const (
	// tplCreateOrg template path for create organization
	tplCreateOrg base.TplName = "org/create"
)

// Create render the page for create organization
func Create(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("new_org")
	ctx.Data["DefaultOrgVisibilityMode"] = setting.Service.DefaultOrgVisibilityMode
	if !ctx.Doer.CanCreateOrganization() {
		ctx.ServerError("Not allowed", errors.New(ctx.Tr("org.form.create_org_not_allowed")))
		return
	}
	ctx.HTML(http.StatusOK, tplCreateOrg)
}

// CreatePost response for create organization
func CreatePost(ctx *context.Context) {
	form := *web.GetForm(ctx).(*forms.CreateOrgForm)
	ctx.Data["Title"] = ctx.Tr("new_org")
	auditParams := map[string]string{
		"project": form.OrgName,
	}
	type auditValue struct {
		Name                      string
		IsActive                  bool
		Visibility                string
		RepoAdminChangeTeamAccess bool
	}

	newValue := auditValue{
		Name:                      form.OrgName,
		IsActive:                  true,
		Visibility:                form.Visibility.String(),
		RepoAdminChangeTeamAccess: form.RepoAdminChangeTeamAccess,
	}

	newValueBytes, _ := json.Marshal(newValue)
	auditParams["new_value"] = string(newValueBytes)

	if !ctx.Doer.CanCreateOrganization() {
		ctx.ServerError("Not allowed", errors.New(ctx.Tr("org.form.create_org_not_allowed")))
		auditParams["error"] = "User is not allowed to create an organization"
		audit.CreateAndSendEvent(audit.ProjectCreateEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}

	if ctx.HasError() {
		ctx.HTML(http.StatusOK, tplCreateOrg)
		auditParams["error"] = "Error occurs in form validation"
		audit.CreateAndSendEvent(audit.ProjectCreateEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}

	if setting.SourceControl.Enabled && setting.SourceControl.TenantWithRoleModeEnabled && form.Visibility == 0 {
		log.Debug("Cannot create public organization")
		ctx.RenderWithErr(ctx.Tr("org.form.create_public_org_not_allowed"), tplCreateOrg, &form)
		auditParams["error"] = "Error occurs in form validation"
		return
	}

	org := &organization.Organization{
		Name:                      form.OrgName,
		IsActive:                  true,
		Type:                      user_model.UserTypeOrganization,
		Visibility:                form.Visibility,
		RepoAdminChangeTeamAccess: form.RepoAdminChangeTeamAccess,
	}

	if err := organization.CreateOrganization(org, ctx.Doer); err != nil {
		ctx.Data["Err_OrgName"] = true
		switch {
		case user_model.IsErrUserAlreadyExist(err):
			ctx.RenderWithErr(ctx.Tr("form.org_name_been_taken"), tplCreateOrg, &form)
			auditParams["error"] = "Organization name been taken"
		case db.IsErrNameReserved(err):
			ctx.RenderWithErr(ctx.Tr("org.form.name_reserved", err.(db.ErrNameReserved).Name), tplCreateOrg, &form)
			auditParams["error"] = "Organization name reserved"
		case db.IsErrNamePatternNotAllowed(err):
			ctx.RenderWithErr(ctx.Tr("org.form.name_pattern_not_allowed", err.(db.ErrNamePatternNotAllowed).Pattern), tplCreateOrg, &form)
			auditParams["error"] = "Name pattern not allowed"
		case organization.IsErrUserNotAllowedCreateOrg(err):
			ctx.RenderWithErr(ctx.Tr("org.form.create_org_not_allowed"), tplCreateOrg, &form)
			auditParams["error"] = "User is not allowed to create an organization"
		default:
			ctx.ServerError("CreateOrganization", err)
			auditParams["error"] = "Error has occurred while creating organization"
		}
		audit.CreateAndSendEvent(audit.ProjectCreateEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}
	audit.CreateAndSendEvent(audit.ProjectCreateEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)
	log.Trace("Organization created: %s", org.Name)

	if setting.SourceControl.Enabled && setting.SourceControl.TenantWithRoleModeEnabled && form.Visibility == 1 {
		err := role_model.AddProjectToInnerSource(org)
		if err != nil {
			ctx.RenderWithErr(ctx.Tr("form.privileges_not_granted"), tplCreateOrg, &form)
			return
		}
	}

	if setting.SourceControl.Enabled && setting.SourceControl.TenantWithRoleModeEnabled {
		tenantId, err := role_model.GetUserTenantId(ctx, ctx.Doer.ID)
		if err != nil {
			ctx.RenderWithErr(ctx.Tr("form.privileges_not_granted"), tplCreateOrg, &form)
			return
		}

		tenantOrganization := &tenant.ScTenantOrganizations{
			ID:             uuid.NewString(),
			TenantID:       tenantId,
			OrganizationID: org.ID,
		}
		err = tenant_service.CreateRelationTenantOrganization(ctx, tenantOrganization)
		if err != nil {
			ctx.RenderWithErr(ctx.Tr("form.privileges_not_granted"), tplCreateOrg, &form)
			return
		}

		if err := role_model.GrantUserPermissionToOrganization(ctx.Doer, tenantId, org, role_model.OWNER); err != nil {
			ctx.RenderWithErr(ctx.Tr("form.privileges_not_granted "), tplCreateOrg, &form)
			return
		}
	}

	ctx.Redirect(org.AsUser().DashboardLink())
}

// GetRepositoriesFromOrg получаем все репозитории, которые находятся в проекте
func GetRepositoriesFromOrg(ctx *context.Context) {
	if ctx.Org == nil {
		log.Warn("No organization found in context")
		ctx.JSON(http.StatusNotFound, "No organization found in context")
		return
	}

	repositories, err := organization.GetOrgRepositories(ctx, ctx.Org.Organization.ID)
	if err != nil {
		log.Error("Error has occurred while getting repositories by org_id: %v", err)
		ctx.JSON(http.StatusInternalServerError, fmt.Sprintf("Error has occurred while getting repositories by org_id: %v", err))
		return
	}

	ctx.JSON(http.StatusOK, repositories)
}
