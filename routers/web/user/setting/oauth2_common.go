// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package setting

import (
	"fmt"
	"net/http"
	"strconv"

	"code.gitea.io/gitea/models/auth"
	"code.gitea.io/gitea/modules/base"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/json"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/sbt/audit"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/services/forms"
)

type OAuth2CommonHandlers struct {
	OwnerID            int64        // 0 for instance-wide, otherwise OrgID or UserID
	BasePathList       string       // the base URL for the application list page, eg: "/user/setting/applications"
	BasePathEditPrefix string       // the base URL for the application edit page, will be appended with app id, eg: "/user/setting/applications/oauth2"
	TplAppEdit         base.TplName // the template for the application edit page
}

func (oa *OAuth2CommonHandlers) renderEditPage(ctx *context.Context) {
	app := ctx.Data["App"].(*auth.OAuth2Application)
	ctx.Data["FormActionPath"] = fmt.Sprintf("%s/%d", oa.BasePathEditPrefix, app.ID)
	ctx.HTML(http.StatusOK, oa.TplAppEdit)
}

// AddApp adds an oauth2 application
func (oa *OAuth2CommonHandlers) AddApp(ctx *context.Context) {
	form := web.GetForm(ctx).(*forms.EditOAuth2ApplicationForm)
	auditParams := map[string]string{
		"app_name":                form.Name,
		"app_redirect_uri":        form.RedirectURI,
		"app_confidential_client": strconv.FormatBool(form.ConfidentialClient),
	}
	if ctx.HasError() {
		// go to the application list page
		ctx.Redirect(oa.BasePathList)
		auditParams["error"] = "Error occurs in form validation"
		audit.CreateAndSendEvent(audit.ApplicationsSettingsAdd, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}

	// TODO validate redirect URI
	app, err := auth.CreateOAuth2Application(ctx, auth.CreateOAuth2ApplicationOptions{
		Name:               form.Name,
		RedirectURIs:       []string{form.RedirectURI},
		UserID:             oa.OwnerID,
		ConfidentialClient: form.ConfidentialClient,
	})
	if err != nil {
		ctx.ServerError("CreateOAuth2Application", err)
		auditParams["error"] = "Error has occurred while creating OAuth2 application"
		audit.CreateAndSendEvent(audit.ApplicationsSettingsAdd, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}

	// render the edit page with secret
	audit.CreateAndSendEvent(audit.ApplicationsSettingsAdd, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)
	ctx.Flash.Success(ctx.Tr("settings.create_oauth2_application_success"), true)
	ctx.Data["App"] = app
	ctx.Data["ClientSecret"], err = app.GenerateClientSecret()
	if err != nil {
		ctx.ServerError("GenerateClientSecret", err)
		return
	}
	oa.renderEditPage(ctx)
}

// EditShow displays the given application
func (oa *OAuth2CommonHandlers) EditShow(ctx *context.Context) {
	auditParams := map[string]string{
		"app_id": strconv.FormatInt(ctx.ParamsInt64("id"), 10),
	}
	app, err := auth.GetOAuth2ApplicationByID(ctx, ctx.ParamsInt64("id"))
	if err != nil {
		if auth.IsErrOAuthApplicationNotFound(err) {
			ctx.NotFound("Application not found", err)
			auditParams["error"] = "Application not found"
			audit.CreateAndSendEvent(audit.ApplicationsSettingsEdit, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}
		ctx.ServerError("GetOAuth2ApplicationByID", err)
		auditParams["error"] = "Error has occurred while getting OAuth2 application by id"
		audit.CreateAndSendEvent(audit.ApplicationsSettingsEdit, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}
	if app.UID != oa.OwnerID {
		ctx.NotFound("Application not found", nil)
		auditParams["error"] = "Application not found"
		audit.CreateAndSendEvent(audit.ApplicationsSettingsEdit, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}
	ctx.Data["App"] = app
	oa.renderEditPage(ctx)
}

// EditSave saves the oauth2 application
func (oa *OAuth2CommonHandlers) EditSave(ctx *context.Context) {
	form := web.GetForm(ctx).(*forms.EditOAuth2ApplicationForm)
	auditParams := map[string]string{
		"app_id":                  strconv.FormatInt(ctx.ParamsInt64("id"), 10),
		"app_name":                form.Name,
		"app_redirect_uri":        form.RedirectURI,
		"app_confidential_client": strconv.FormatBool(form.ConfidentialClient),
	}

	type auditValue struct {
		AppName               string
		AppRedirectURIs       []string
		AppConfidentialClient bool
	}

	old, appGetErr := auth.GetOAuth2ApplicationByID(ctx, ctx.ParamsInt64("id"))
	if appGetErr != nil {
		log.Error("error has occurred while trying to retrieve old information for application with id %v, error: %v", ctx.ParamsInt64("id"), appGetErr)
		ctx.Error(http.StatusNotFound, "Не существует приложения с данным id")
		auditParams["error"] = "Error has occurred while trying to retrieve old application info"
		audit.CreateAndSendEvent(audit.ApplicationsSettingsEdit, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
	}

	oldValue := auditValue{
		AppName:               old.Name,
		AppRedirectURIs:       old.RedirectURIs,
		AppConfidentialClient: old.ConfidentialClient,
	}

	oldValueBytes, jsonMarshalErr := json.Marshal(oldValue)
	if jsonMarshalErr != nil {
		log.Error("error has occurred while converting data to bytes: %v, error: %v", oldValue, jsonMarshalErr)
		ctx.Error(http.StatusBadRequest, "Не удалось преобразовать данные")
		auditParams["error"] = "Error has occurred while converting data to bytes"
		audit.CreateAndSendEvent(audit.ApplicationsSettingsEdit, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
	}
	auditParams["old_value"] = string(oldValueBytes)

	if ctx.HasError() {
		oa.renderEditPage(ctx)
		auditParams["error"] = "Error occurs in form validation"
		audit.CreateAndSendEvent(audit.ApplicationsSettingsEdit, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}

	// TODO validate redirect URI
	var err error
	if ctx.Data["App"], err = auth.UpdateOAuth2Application(auth.UpdateOAuth2ApplicationOptions{
		ID:                 ctx.ParamsInt64("id"),
		Name:               form.Name,
		RedirectURIs:       []string{form.RedirectURI},
		UserID:             oa.OwnerID,
		ConfidentialClient: form.ConfidentialClient,
	}); err != nil {
		ctx.ServerError("UpdateOAuth2Application", err)
		auditParams["error"] = "Error has occurred while updating OAuth2 application"
		audit.CreateAndSendEvent(audit.ApplicationsSettingsEdit, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}

	newValue := auditValue{
		AppName:               form.Name,
		AppRedirectURIs:       []string{form.RedirectURI},
		AppConfidentialClient: form.ConfidentialClient,
	}

	newValueBytes, jsonMarshalErr := json.Marshal(newValue)
	if jsonMarshalErr != nil {
		log.Error("error has occurred while converting data to bytes: %v, error: %v", newValue, jsonMarshalErr)
		ctx.Error(http.StatusBadRequest, "Не удалось преобразовать данные")
		auditParams["error"] = "Error has occurred while converting data to bytes"
		audit.CreateAndSendEvent(audit.ApplicationsSettingsEdit, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
	}
	auditParams["new_value"] = string(newValueBytes)

	audit.CreateAndSendEvent(audit.ApplicationsSettingsEdit, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)
	ctx.Flash.Success(ctx.Tr("settings.update_oauth2_application_success"))
	ctx.Redirect(oa.BasePathList)
}

// RegenerateSecret regenerates the secret
func (oa *OAuth2CommonHandlers) RegenerateSecret(ctx *context.Context) {
	auditParams := map[string]string{
		"app_id": strconv.FormatInt(ctx.ParamsInt64("id"), 10),
	}
	app, err := auth.GetOAuth2ApplicationByID(ctx, ctx.ParamsInt64("id"))
	if err != nil {
		if auth.IsErrOAuthApplicationNotFound(err) {
			ctx.NotFound("Application not found", err)
			auditParams["error"] = "Application not found"
			audit.CreateAndSendEvent(audit.ApplicationsSettingsGenerateSecret, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}
		ctx.ServerError("GetOAuth2ApplicationByID", err)
		auditParams["error"] = "Error has occurred while getting OAuth2 application by id"
		audit.CreateAndSendEvent(audit.ApplicationsSettingsGenerateSecret, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}
	if app.UID != oa.OwnerID {
		ctx.NotFound("Application not found", nil)
		auditParams["error"] = "Application not found"
		audit.CreateAndSendEvent(audit.ApplicationsSettingsGenerateSecret, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}
	ctx.Data["App"] = app

	auditParams["app_name"] = app.Name
	auditParams["app_redirect_uri"] = app.RedirectURIs[0]
	auditParams["app_confidential_client"] = strconv.FormatBool(app.ConfidentialClient)

	ctx.Data["ClientSecret"], err = app.GenerateClientSecret()
	if err != nil {
		ctx.ServerError("GenerateClientSecret", err)
		auditParams["error"] = "Error has occurred while generating client secret"
		audit.CreateAndSendEvent(audit.ApplicationsSettingsGenerateSecret, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}
	audit.CreateAndSendEvent(audit.ApplicationsSettingsGenerateSecret, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)
	ctx.Flash.Success(ctx.Tr("settings.update_oauth2_application_success"), true)
	oa.renderEditPage(ctx)
}

// DeleteApp deletes the given oauth2 application
func (oa *OAuth2CommonHandlers) DeleteApp(ctx *context.Context) {
	auditParams := map[string]string{
		"app_id": strconv.FormatInt(ctx.ParamsInt64("id"), 10),
	}
	if err := auth.DeleteOAuth2Application(ctx.ParamsInt64("id"), oa.OwnerID); err != nil {
		ctx.ServerError("DeleteOAuth2Application", err)
		auditParams["error"] = "Error has occurred while deleting OAuth2 application"
		audit.CreateAndSendEvent(audit.ApplicationsSettingsDelete, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}

	audit.CreateAndSendEvent(audit.ApplicationsSettingsDelete, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)
	ctx.Flash.Success(ctx.Tr("settings.remove_oauth2_application_success"))
	ctx.JSON(http.StatusOK, map[string]interface{}{"redirect": oa.BasePathList})
}

// RevokeGrant revokes the grant
func (oa *OAuth2CommonHandlers) RevokeGrant(ctx *context.Context) {
	if err := auth.RevokeOAuth2Grant(ctx, ctx.ParamsInt64("grantId"), oa.OwnerID); err != nil {
		ctx.ServerError("RevokeOAuth2Grant", err)
		return
	}

	ctx.Flash.Success(ctx.Tr("settings.revoke_oauth2_grant_success"))
	ctx.JSON(http.StatusOK, map[string]interface{}{"redirect": oa.BasePathList})
}
