// Copyright 2014 The Gogs Authors. All rights reserved.
// Copyright 2018 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package setting

import (
	"code.gitea.io/gitea/modules/sbt/audit"
	"net/http"
	"strconv"

	auth_model "code.gitea.io/gitea/models/auth"
	"code.gitea.io/gitea/modules/base"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/services/forms"
)

const (
	tplSettingsApplications base.TplName = "user/settings/applications"
)

// Applications render manage access token page
func Applications(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("settings.applications")
	ctx.Data["PageIsSettingsApplications"] = true

	loadApplicationsData(ctx)

	ctx.HTML(http.StatusOK, tplSettingsApplications)
}

// ApplicationsPost response for add user's access token
func ApplicationsPost(ctx *context.Context) {
	form := web.GetForm(ctx).(*forms.NewAccessTokenForm)
	ctx.Data["Title"] = ctx.Tr("settings")
	ctx.Data["PageIsSettingsApplications"] = true

	auditParams := map[string]string{
		"email":     ctx.Doer.Email,
		"new_value": form.Name,
	}
	if ctx.HasError() {
		loadApplicationsData(ctx)

		auditParams["error"] = "Error occurs in form validation"
		audit.CreateAndSendEvent(audit.UserTokenCreateEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)

		ctx.HTML(http.StatusOK, tplSettingsApplications)
		return
	}

	scope, err := form.GetScope()
	if err != nil {
		ctx.ServerError("GetScope", err)
		auditParams["error"] = "Error has occurred while getting scope"
		audit.CreateAndSendEvent(audit.UserTokenCreateEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}
	t := &auth_model.AccessToken{
		UID:   ctx.Doer.ID,
		Name:  form.Name,
		Scope: scope,
	}

	exist, err := auth_model.AccessTokenByNameExists(t)
	if err != nil {
		ctx.ServerError("AccessTokenByNameExists", err)
		auditParams["error"] = "Error has occurred while checking access token by name exists"
		audit.CreateAndSendEvent(audit.UserTokenCreateEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}
	if exist {
		ctx.Flash.Error(ctx.Tr("settings.generate_token_name_duplicate", t.Name))
		ctx.Redirect(setting.AppSubURL + "/user/settings/applications")
		auditParams["error"] = "Token name is duplicated"
		audit.CreateAndSendEvent(audit.UserTokenCreateEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}

	if err := auth_model.NewAccessToken(t); err != nil {
		ctx.ServerError("NewAccessToken", err)
		auditParams["error"] = "Error has occurred while creating new access token"
		audit.CreateAndSendEvent(audit.UserTokenCreateEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}

	ctx.Flash.Success(ctx.Tr("settings.generate_token_success"))
	ctx.Flash.Info(t.Token)

	audit.CreateAndSendEvent(audit.UserTokenCreateEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)

	ctx.Redirect(setting.AppSubURL + "/user/settings/applications")
}

// DeleteApplication response for delete user access token
func DeleteApplication(ctx *context.Context) {
	auditParams := map[string]string{
		"email":     ctx.Doer.Email,
		"old_value": strconv.FormatInt(ctx.FormInt64("id"), 10),
	}
	if err := auth_model.DeleteAccessTokenByID(ctx.FormInt64("id"), ctx.Doer.ID); err != nil {
		ctx.Flash.Error("DeleteAccessTokenByID: " + err.Error())
		auditParams["error"] = "Error has occurred while deleting access token by id"
		audit.CreateAndSendEvent(audit.UserTokenDeleteEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
	} else {
		audit.CreateAndSendEvent(audit.UserTokenDeleteEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)
		ctx.Flash.Success(ctx.Tr("settings.delete_token_success"))
	}

	ctx.JSON(http.StatusOK, map[string]interface{}{
		"redirect": setting.AppSubURL + "/user/settings/applications",
	})
}

func loadApplicationsData(ctx *context.Context) {
	tokens, err := auth_model.ListAccessTokens(auth_model.ListAccessTokensOptions{UserID: ctx.Doer.ID})
	if err != nil {
		ctx.ServerError("ListAccessTokens", err)
		return
	}
	ctx.Data["Tokens"] = tokens
	ctx.Data["EnableOAuth2"] = setting.OAuth2.Enable
	if setting.OAuth2.Enable {
		ctx.Data["Applications"], err = auth_model.GetOAuth2ApplicationsByUserID(ctx, ctx.Doer.ID)
		if err != nil {
			ctx.ServerError("GetOAuth2ApplicationsByUserID", err)
			return
		}
		ctx.Data["Grants"], err = auth_model.GetOAuth2GrantsByUserID(ctx, ctx.Doer.ID)
		if err != nil {
			ctx.ServerError("GetOAuth2GrantsByUserID", err)
			return
		}
	}
}
