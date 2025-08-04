// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package org

import (
	"code.gitea.io/gitea/modules/sbt/audit"
	"fmt"
	"net/http"
	"strconv"

	"code.gitea.io/gitea/models/auth"
	"code.gitea.io/gitea/models/tenant"
	"code.gitea.io/gitea/modules/base"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/setting"
	user_setting "code.gitea.io/gitea/routers/web/user/setting"
)

const (
	tplSettingsApplications         base.TplName = "org/settings/applications"
	tplSettingsOAuthApplicationEdit base.TplName = "org/settings/applications_oauth2_edit"
)

func newOAuth2CommonHandlers(org *context.Organization) *user_setting.OAuth2CommonHandlers {
	return &user_setting.OAuth2CommonHandlers{
		OwnerID:            org.Organization.ID,
		BasePathList:       fmt.Sprintf("%s/org/%s/settings/applications", setting.AppSubURL, org.Organization.Name),
		BasePathEditPrefix: fmt.Sprintf("%s/org/%s/settings/applications/oauth2", setting.AppSubURL, org.Organization.Name),
		TplAppEdit:         tplSettingsOAuthApplicationEdit,
	}
}

// Applications render org applications page (for org, at the moment, there are only OAuth2 applications)
func Applications(ctx *context.Context) {
	tenantId, err := tenant.GetTenantByOrgIdOrDefault(ctx, ctx.Org.Organization.ID)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, err.Error())
		return
	}

	ctx.Data["TenantID"] = tenantId
	ctx.Data["Title"] = ctx.Tr("settings.applications")
	ctx.Data["PageIsOrgSettings"] = true
	ctx.Data["PageIsSettingsApplications"] = true

	apps, err := auth.GetOAuth2ApplicationsByUserID(ctx, ctx.Org.Organization.ID)
	if err != nil {
		ctx.ServerError("GetOAuth2ApplicationsByUserID", err)
		auditParams := map[string]string{
			"error": "Error has occurred while getting OAuth2 applications",
		}
		audit.CreateAndSendEvent(audit.ApplicationsSettingsOpen, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}
	ctx.Data["Applications"] = apps

	audit.CreateAndSendEvent(audit.ApplicationsSettingsOpen, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, nil)
	ctx.HTML(http.StatusOK, tplSettingsApplications)
}

// OAuthApplicationsPost response for adding an oauth2 application
func OAuthApplicationsPost(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("settings.applications")
	ctx.Data["PageIsOrgSettings"] = true
	ctx.Data["PageIsSettingsApplications"] = true

	oa := newOAuth2CommonHandlers(ctx.Org)
	oa.AddApp(ctx)
}

// OAuth2ApplicationShow displays the given application
func OAuth2ApplicationShow(ctx *context.Context) {
	ctx.Data["PageIsOrgSettings"] = true
	ctx.Data["PageIsSettingsApplications"] = true

	oa := newOAuth2CommonHandlers(ctx.Org)
	oa.EditShow(ctx)
}

// OAuth2ApplicationEdit response for editing oauth2 application
func OAuth2ApplicationEdit(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("settings.applications")
	ctx.Data["PageIsOrgSettings"] = true
	ctx.Data["PageIsSettingsApplications"] = true

	oa := newOAuth2CommonHandlers(ctx.Org)
	oa.EditSave(ctx)
}

// OAuthApplicationsRegenerateSecret handles the post request for regenerating the secret
func OAuthApplicationsRegenerateSecret(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("settings")
	ctx.Data["PageIsOrgSettings"] = true
	ctx.Data["PageIsSettingsApplications"] = true

	oa := newOAuth2CommonHandlers(ctx.Org)
	oa.RegenerateSecret(ctx)
}

// DeleteOAuth2Application deletes the given oauth2 application
func DeleteOAuth2Application(ctx *context.Context) {
	oa := newOAuth2CommonHandlers(ctx.Org)
	oa.DeleteApp(ctx)
}

// TODO: revokes the grant with the given id
