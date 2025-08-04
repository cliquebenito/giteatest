// Copyright 2014 The Gogs Authors. All rights reserved.
// Copyright 2020 The Gitea Authors.
// SPDX-License-Identifier: MIT

package admin

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"code.gitea.io/gitea/models"
	"code.gitea.io/gitea/models/auth"
	"code.gitea.io/gitea/models/db"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/auth/password"
	"code.gitea.io/gitea/modules/base"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/json"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/sbt/audit"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/util"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/routers/web/explore"
	user_setting "code.gitea.io/gitea/routers/web/user/setting"
	"code.gitea.io/gitea/services/forms"
	"code.gitea.io/gitea/services/mailer"
	user_service "code.gitea.io/gitea/services/user"
)

const (
	tplUsers    base.TplName = "admin/user/list"
	tplUserNew  base.TplName = "admin/user/new"
	tplUserEdit base.TplName = "admin/user/edit"
)

// Users show all the users
func Users(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("admin.users")
	ctx.Data["PageIsAdminUsers"] = true

	extraParamStrings := map[string]string{}
	statusFilterKeys := []string{"is_active", "is_admin", "is_restricted", "is_2fa_enabled", "is_prohibit_login"}
	statusFilterMap := map[string]string{}
	for _, filterKey := range statusFilterKeys {
		paramKey := "status_filter[" + filterKey + "]"
		paramVal := ctx.FormString(paramKey)
		statusFilterMap[filterKey] = paramVal
		if paramVal != "" {
			extraParamStrings[paramKey] = paramVal
		}
	}

	sortType := ctx.FormString("sort")
	if sortType == "" {
		sortType = explore.UserSearchDefaultAdminSort
		ctx.SetFormString("sort", sortType)
	}
	ctx.PageData["adminUserListSearchForm"] = map[string]interface{}{
		"StatusFilterMap": statusFilterMap,
		"SortType":        sortType,
	}

	explore.RenderUserSearch(ctx, &user_model.SearchUserOptions{
		Actor: ctx.Doer,
		Type:  user_model.UserTypeIndividual,
		ListOptions: db.ListOptions{
			PageSize: setting.UI.Admin.UserPagingNum,
		},
		SearchByEmail:      true,
		IsActive:           util.OptionalBoolParse(statusFilterMap["is_active"]),
		IsAdmin:            util.OptionalBoolParse(statusFilterMap["is_admin"]),
		IsRestricted:       util.OptionalBoolParse(statusFilterMap["is_restricted"]),
		IsTwoFactorEnabled: util.OptionalBoolParse(statusFilterMap["is_2fa_enabled"]),
		IsProhibitLogin:    util.OptionalBoolParse(statusFilterMap["is_prohibit_login"]),
		SearchWithTuz:      false,
		ExtraParamStrings:  extraParamStrings,
	}, tplUsers)
}

// NewUser render adding a new user page
func NewUser(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("admin.users.new_account")
	ctx.Data["PageIsAdminUsers"] = true
	ctx.Data["DefaultUserVisibilityMode"] = setting.Service.DefaultUserVisibilityMode
	ctx.Data["AllowedUserVisibilityModes"] = setting.Service.AllowedUserVisibilityModesSlice.ToVisibleTypeSlice()

	ctx.Data["login_type"] = "0-0"

	sources, err := auth.Sources()
	if err != nil {
		ctx.ServerError("auth.Sources", err)
		return
	}
	ctx.Data["Sources"] = sources

	ctx.Data["CanSendEmail"] = setting.MailService != nil
	ctx.HTML(http.StatusOK, tplUserNew)
}

// NewUserPost response for adding a new user
func NewUserPost(ctx *context.Context) {
	form := web.GetForm(ctx).(*forms.AdminCreateUserForm)
	if form.Email == "" && !setting.SourceControl.EmptyEmailEnabled {
		ctx.RenderWithErr(ctx.Tr("user.empty_email_enabled"), tplUserNew, &form)
		auditParams := map[string]string{
			"error": "Email is required or you didn't active the source control without email'",
		}
		audit.CreateAndSendEvent(audit.UserCreateEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}
	ctx.Data["Title"] = ctx.Tr("admin.users.new_account")
	ctx.Data["PageIsAdminUsers"] = true
	ctx.Data["DefaultUserVisibilityMode"] = setting.Service.DefaultUserVisibilityMode
	ctx.Data["AllowedUserVisibilityModes"] = setting.Service.AllowedUserVisibilityModesSlice.ToVisibleTypeSlice()

	sources, err := auth.Sources()
	if err != nil {
		ctx.ServerError("auth.Sources", err)
		auditParams := map[string]string{
			"error": "Error has occurred while getting sources",
		}
		audit.CreateAndSendEvent(audit.UserCreateEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}
	ctx.Data["Sources"] = sources

	ctx.Data["CanSendEmail"] = setting.MailService != nil

	auditParams := map[string]string{
		"email": form.Email,
	}

	if ctx.HasError() {
		ctx.HTML(http.StatusOK, tplUserNew)
		auditParams["error"] = "Error occurs in form validation"
		audit.CreateAndSendEvent(audit.UserCreateEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}

	u := &user_model.User{
		Name:      form.UserName,
		Email:     form.Email,
		Passwd:    form.Password,
		LoginType: auth.Plain,
	}

	overwriteDefault := &user_model.CreateUserOverwriteOptions{
		IsActive:   util.OptionalBoolTrue,
		Visibility: &form.Visibility,
	}

	if len(form.LoginType) > 0 {
		fields := strings.Split(form.LoginType, "-")
		if len(fields) == 2 {
			lType, _ := strconv.ParseInt(fields[0], 10, 0)
			u.LoginType = auth.Type(lType)
			u.LoginSource, _ = strconv.ParseInt(fields[1], 10, 64)
			u.LoginName = form.LoginName
		}
	}
	if u.LoginType == auth.NoType || u.LoginType == auth.Plain {
		if len(form.Password) < setting.MinPasswordLength {
			ctx.Data["Err_Password"] = true
			ctx.RenderWithErr(ctx.Tr("auth.password_too_short", setting.MinPasswordLength), tplUserNew, &form)
			auditParams["error"] = "Incorrect password"
			audit.CreateAndSendEvent(audit.UserCreateEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}
		if !password.IsComplexEnough(form.Password) {
			ctx.Data["Err_Password"] = true
			ctx.RenderWithErr(password.BuildComplexityError(ctx.Locale), tplUserNew, &form)
			auditParams["error"] = "Incorrect password"
			audit.CreateAndSendEvent(audit.UserCreateEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}
		pwned, err := password.IsPwned(ctx, form.Password)
		if pwned {
			ctx.Data["Err_Password"] = true
			errMsg := ctx.Tr("auth.password_pwned")
			if err != nil {
				log.Error(err.Error())
				errMsg = ctx.Tr("auth.password_pwned_err")
			}
			ctx.RenderWithErr(errMsg, tplUserNew, &form)
			auditParams["error"] = "Incorrect password"
			audit.CreateAndSendEvent(audit.UserCreateEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}
		u.MustChangePassword = form.MustChangePassword
	}

	if err := user_model.CreateUser(u, overwriteDefault); err != nil {
		switch {
		case user_model.IsErrUserAlreadyExist(err):
			ctx.Data["Err_UserName"] = true
			ctx.RenderWithErr(ctx.Tr("form.username_been_taken"), tplUserNew, &form)
			auditParams["error"] = "Incorrect username"
		case user_model.IsErrEmailAlreadyUsed(err):
			ctx.Data["Err_Email"] = true
			ctx.RenderWithErr(ctx.Tr("form.email_been_used"), tplUserNew, &form)
			auditParams["error"] = "Incorrect email"
		case user_model.IsErrEmailCharIsNotSupported(err):
			ctx.Data["Err_Email"] = true
			ctx.RenderWithErr(ctx.Tr("form.email_invalid"), tplUserNew, &form)
			auditParams["error"] = "Incorrect email"
		case user_model.IsErrEmailInvalid(err):
			ctx.Data["Err_Email"] = true
			ctx.RenderWithErr(ctx.Tr("form.email_invalid"), tplUserNew, &form)
			auditParams["error"] = "Incorrect email"
		case db.IsErrNameReserved(err):
			ctx.Data["Err_UserName"] = true
			ctx.RenderWithErr(ctx.Tr("user.form.name_reserved", err.(db.ErrNameReserved).Name), tplUserNew, &form)
			auditParams["error"] = "Incorrect username"
		case db.IsErrNamePatternNotAllowed(err):
			ctx.Data["Err_UserName"] = true
			ctx.RenderWithErr(ctx.Tr("user.form.name_pattern_not_allowed", err.(db.ErrNamePatternNotAllowed).Pattern), tplUserNew, &form)
			auditParams["error"] = "Incorrect username"
		case db.IsErrNameCharsNotAllowed(err):
			ctx.Data["Err_UserName"] = true
			ctx.RenderWithErr(ctx.Tr("user.form.name_chars_not_allowed", err.(db.ErrNameCharsNotAllowed).Name), tplUserNew, &form)
			auditParams["error"] = "Incorrect username"
		default:
			ctx.ServerError("CreateUser", err)
			auditParams["error"] = "Error has occurred while creating user"
		}
		audit.CreateAndSendEvent(audit.UserCreateEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}
	log.Trace("Account created by admin (%s): %s", ctx.Doer.Name, u.Name)

	// Send email notification.
	if form.SendNotify {
		mailer.SendRegisterNotifyMail(u)
	}

	ctx.Flash.Success(ctx.Tr("admin.users.new_success", u.Name))
	ctx.Redirect(setting.AppSubURL + "/admin/users/" + strconv.FormatInt(u.ID, 10))

	audit.CreateAndSendEvent(audit.UserCreateEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)
}

func prepareUserInfo(ctx *context.Context) *user_model.User {
	u, err := user_model.GetUserByID(ctx, ctx.ParamsInt64(":userid"))
	if err != nil {
		if user_model.IsErrUserNotExist(err) {
			ctx.Redirect(setting.AppSubURL + "/admin/users")
		} else {
			ctx.ServerError("GetUserByID", err)
		}
		return nil
	}
	ctx.Data["User"] = u

	if u.LoginSource > 0 {
		ctx.Data["LoginSource"], err = auth.GetSourceByID(u.LoginSource)
		if err != nil {
			ctx.ServerError("auth.GetSourceByID", err)
			return nil
		}
	} else {
		ctx.Data["LoginSource"] = &auth.Source{}
	}

	sources, err := auth.Sources()
	if err != nil {
		ctx.ServerError("auth.Sources", err)
		return nil
	}
	ctx.Data["Sources"] = sources

	hasTOTP, err := auth.HasTwoFactorByUID(u.ID)
	if err != nil {
		ctx.ServerError("auth.HasTwoFactorByUID", err)
		return nil
	}
	hasWebAuthn, err := auth.HasWebAuthnRegistrationsByUID(u.ID)
	if err != nil {
		ctx.ServerError("auth.HasWebAuthnRegistrationsByUID", err)
		return nil
	}
	ctx.Data["TwoFactorEnabled"] = hasTOTP || hasWebAuthn

	return u
}

// EditUser show editing user page
func EditUser(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("admin.users.edit_account")
	ctx.Data["PageIsAdminUsers"] = true
	ctx.Data["DisableRegularOrgCreation"] = setting.Admin.DisableRegularOrgCreation
	ctx.Data["DisableMigrations"] = setting.Repository.DisableMigrations
	ctx.Data["AllowedUserVisibilityModes"] = setting.Service.AllowedUserVisibilityModesSlice.ToVisibleTypeSlice()

	prepareUserInfo(ctx)
	if ctx.Written() {
		return
	}

	ctx.HTML(http.StatusOK, tplUserEdit)
}

// EditUserPost response for editing user
func EditUserPost(ctx *context.Context) {
	form := web.GetForm(ctx).(*forms.AdminEditUserForm)
	if form.Email == "" && !setting.SourceControl.EmptyEmailEnabled {
		ctx.RenderWithErr(ctx.Tr("user.empty_email_enabled"), tplUserEdit, &form)
		auditParams := map[string]string{
			"error": "Email is required or you didn't active the source control without email",
		}
		audit.CreateAndSendEvent(audit.UserCreateEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}
	ctx.Data["Title"] = ctx.Tr("admin.users.edit_account")
	ctx.Data["PageIsAdminUsers"] = true
	ctx.Data["DisableMigrations"] = setting.Repository.DisableMigrations
	ctx.Data["AllowedUserVisibilityModes"] = setting.Service.AllowedUserVisibilityModesSlice.ToVisibleTypeSlice()

	auditParams := map[string]string{
		"affected_user_id": strconv.FormatInt(ctx.ParamsInt64(":userid"), 10),
	}

	auditParamsForEdit := map[string]string{
		"affected_user_id": strconv.FormatInt(ctx.ParamsInt64(":userid"), 10),
	}

	u := prepareUserInfo(ctx)
	if ctx.Written() {
		auditParamsForEdit["error"] = "Error has occurred while preparing user info"
		audit.CreateAndSendEvent(audit.UserProfileEditEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParamsForEdit)
		return
	}

	auditParams["affected_user"] = u.Name
	auditParamsForEdit["affected_user"] = u.Name

	type auditValue struct {
		LoginSource             string
		LoginType               string
		Name                    string
		LowerName               string
		LoginName               string
		FullName                string
		Email                   string
		Website                 string
		Location                string
		MaxRepoCreation         string
		IsActive                bool
		IsAdmin                 bool
		IsRestricted            bool
		AllowGitHook            bool
		AllowImportLocal        bool
		AllowCreateOrganization bool
		Visibility              string
		ProhibitLogin           bool
	}

	oldValue := auditValue{
		LoginSource:             strconv.FormatInt(u.LoginSource, 10),
		LoginType:               u.LoginType.String(),
		Name:                    u.Name,
		LowerName:               u.LowerName,
		LoginName:               u.LoginName,
		FullName:                u.FullName,
		Email:                   u.Email,
		Website:                 u.Website,
		Location:                u.Location,
		MaxRepoCreation:         strconv.Itoa(u.MaxRepoCreation),
		IsActive:                u.IsActive,
		IsAdmin:                 u.IsAdmin,
		IsRestricted:            u.IsRestricted,
		AllowGitHook:            u.AllowGitHook,
		AllowImportLocal:        u.AllowImportLocal,
		AllowCreateOrganization: u.AllowCreateOrganization,
		Visibility:              u.Visibility.String(),
		ProhibitLogin:           u.ProhibitLogin,
	}

	newValue := auditValue{
		Name:                    form.UserName,
		LowerName:               strings.ToLower(form.UserName),
		LoginName:               form.LoginName,
		FullName:                form.FullName,
		Email:                   form.Email,
		Website:                 form.Website,
		Location:                form.Location,
		MaxRepoCreation:         strconv.Itoa(form.MaxRepoCreation),
		IsActive:                form.Active,
		IsAdmin:                 form.Admin,
		IsRestricted:            form.Restricted,
		AllowGitHook:            form.AllowGitHook,
		AllowImportLocal:        form.AllowImportLocal,
		AllowCreateOrganization: form.AllowCreateOrganization,
		Visibility:              form.Visibility.String(),
		ProhibitLogin:           form.ProhibitLogin,
	}

	oldValueBytes, _ := json.Marshal(oldValue)
	auditParamsForEdit["old_value"] = string(oldValueBytes)

	newValueBytes, _ := json.Marshal(newValue)
	auditParamsForEdit["new_value"] = string(newValueBytes)

	if ctx.HasError() {
		ctx.HTML(http.StatusOK, tplUserEdit)
		auditParamsForEdit["error"] = "Error occurs in form validation"
		audit.CreateAndSendEvent(audit.UserProfileEditEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParamsForEdit)
		return
	}

	fields := strings.Split(form.LoginType, "-")
	if len(fields) == 2 {
		loginType, _ := strconv.ParseInt(fields[0], 10, 0)
		authSource, _ := strconv.ParseInt(fields[1], 10, 64)

		if u.LoginSource != authSource {
			u.LoginSource = authSource
			u.LoginType = auth.Type(loginType)
		}
	}
	newValue.LoginSource = strconv.FormatInt(u.LoginSource, 10)
	newValue.LoginType = u.LoginType.String()

	newValueBytes, _ = json.Marshal(newValue)
	auditParamsForEdit["new_value"] = string(newValueBytes)

	if len(form.Password) > 0 && (u.IsLocal() || u.IsOAuth2() || u.IsIAM()) {
		var err error
		if len(form.Password) < setting.MinPasswordLength {
			ctx.Data["Err_Password"] = true
			ctx.RenderWithErr(ctx.Tr("auth.password_too_short", setting.MinPasswordLength), tplUserEdit, &form)
			auditParams["error"] = "Password is too short"
			audit.CreateAndSendEvent(audit.UserPasswordChangeEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}
		if !password.IsComplexEnough(form.Password) {
			ctx.RenderWithErr(password.BuildComplexityError(ctx.Locale), tplUserEdit, &form)
			auditParams["error"] = "Incorrect password"
			audit.CreateAndSendEvent(audit.UserPasswordChangeEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}
		pwned, err := password.IsPwned(ctx, form.Password)
		if pwned {
			ctx.Data["Err_Password"] = true
			errMsg := ctx.Tr("auth.password_pwned")
			if err != nil {
				log.Error(err.Error())
				errMsg = ctx.Tr("auth.password_pwned_err")
			}
			ctx.RenderWithErr(errMsg, tplUserEdit, &form)
			auditParams["error"] = "Incorrect password"
			audit.CreateAndSendEvent(audit.UserPasswordChangeEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}

		if err := user_model.ValidateEmail(form.Email); err != nil {
			ctx.Data["Err_Email"] = true
			ctx.RenderWithErr(ctx.Tr("form.email_error"), tplUserEdit, &form)
			auditParams["error"] = "Incorrect email"
			audit.CreateAndSendEvent(audit.UserPasswordChangeEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}

		if u.Salt, err = user_model.GetUserSalt(); err != nil {
			ctx.ServerError("UpdateUser", err)
			auditParams["error"] = "Error has occurred while getting user salt"
			audit.CreateAndSendEvent(audit.UserPasswordChangeEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}
		if err = u.SetPassword(form.Password); err != nil {
			ctx.ServerError("SetPassword", err)
			auditParams["error"] = "Error has occurred while setting password"
			audit.CreateAndSendEvent(audit.UserPasswordChangeEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}
		audit.CreateAndSendEvent(audit.UserPasswordChangeEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)
	}

	if len(form.UserName) != 0 && u.Name != form.UserName {
		if err := user_setting.HandleUsernameChange(ctx, u, form.UserName); err != nil {
			if ctx.Written() {
				auditParamsForEdit["error"] = "Error has occurred while changing username"
				audit.CreateAndSendEvent(audit.UserProfileEditEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParamsForEdit)
				return
			}
			auditParamsForEdit["error"] = "Error has occurred while changing username"
			audit.CreateAndSendEvent(audit.UserProfileEditEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParamsForEdit)
			ctx.RenderWithErr(ctx.Flash.ErrorMsg, tplUserEdit, &form)
			return
		}
		u.Name = form.UserName
		u.LowerName = strings.ToLower(form.UserName)
	}

	if form.Reset2FA {
		tf, err := auth.GetTwoFactorByUID(u.ID)
		if err != nil && !auth.IsErrTwoFactorNotEnrolled(err) {
			ctx.ServerError("auth.GetTwoFactorByUID", err)
			auditParamsForEdit["error"] = "Error has occurred while getting two factor by user id"
			audit.CreateAndSendEvent(audit.UserProfileEditEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParamsForEdit)
			return
		} else if tf != nil {
			if err := auth.DeleteTwoFactorByID(tf.ID, u.ID); err != nil {
				ctx.ServerError("auth.DeleteTwoFactorByID", err)
				auditParamsForEdit["error"] = "Error has occurred while deleting two factor by id"
				audit.CreateAndSendEvent(audit.UserProfileEditEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParamsForEdit)
				return
			}
		}

		wn, err := auth.GetWebAuthnCredentialsByUID(u.ID)
		if err != nil {
			ctx.ServerError("auth.GetTwoFactorByUID", err)
			auditParamsForEdit["error"] = "Error has occurred while getting web auth credentials by user id"
			audit.CreateAndSendEvent(audit.UserProfileEditEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParamsForEdit)
			return
		}
		for _, cred := range wn {
			if _, err := auth.DeleteCredential(cred.ID, u.ID); err != nil {
				ctx.ServerError("auth.DeleteCredential", err)
				auditParamsForEdit["error"] = "Error has occurred while deleting credential"
				audit.CreateAndSendEvent(audit.UserProfileEditEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParamsForEdit)
				return
			}
		}

	}

	var event audit.Event
	if u.IsAdmin != form.Admin {
		if form.Admin {
			event = audit.GlobalRightsGrantedEvent
		} else {
			event = audit.GlobalRightsRemoveEvent
		}
	}

	u.LoginName = form.LoginName
	u.FullName = form.FullName
	emailChanged := !strings.EqualFold(u.Email, form.Email)
	u.Email = form.Email
	u.Website = form.Website
	u.Location = form.Location
	u.MaxRepoCreation = form.MaxRepoCreation
	u.IsActive = form.Active
	u.IsAdmin = form.Admin
	u.IsRestricted = form.Restricted
	u.AllowGitHook = form.AllowGitHook
	u.AllowImportLocal = form.AllowImportLocal
	u.AllowCreateOrganization = form.AllowCreateOrganization

	u.Visibility = form.Visibility

	// skip self Prohibit Login
	if ctx.Doer.ID == u.ID {
		u.ProhibitLogin = false
	} else {
		u.ProhibitLogin = form.ProhibitLogin
	}

	if err := user_model.UpdateUser(ctx, u, emailChanged); err != nil {
		if user_model.IsErrEmailAlreadyUsed(err) {
			ctx.Data["Err_Email"] = true
			ctx.RenderWithErr(ctx.Tr("form.email_been_used"), tplUserEdit, &form)
			auditParams["error"] = "Email been used"
			auditParamsForEdit["error"] = "Email been used"
		} else if user_model.IsErrEmailCharIsNotSupported(err) ||
			user_model.IsErrEmailInvalid(err) {
			ctx.Data["Err_Email"] = true
			ctx.RenderWithErr(ctx.Tr("form.email_invalid"), tplUserEdit, &form)
			auditParams["error"] = "Incorrect email"
			auditParamsForEdit["error"] = "Incorrect email"
		} else {
			ctx.ServerError("UpdateUser", err)
			auditParams["error"] = "Error has occurred while updating user"
			auditParamsForEdit["error"] = "Error has occurred while updating user"
		}
		if event != 0 {
			audit.CreateAndSendEvent(event, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		}
		audit.CreateAndSendEvent(audit.UserProfileEditEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParamsForEdit)
		return
	}
	log.Trace("Account profile updated by admin (%s): %s", ctx.Doer.Name, u.Name)

	if event != 0 {
		audit.CreateAndSendEvent(event, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)
	}
	audit.CreateAndSendEvent(audit.UserProfileEditEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParamsForEdit)

	ctx.Flash.Success(ctx.Tr("admin.users.update_profile_success"))
	ctx.Redirect(setting.AppSubURL + "/admin/users/" + url.PathEscape(ctx.Params(":userid")))
}

// DeleteUser response for deleting a user
func DeleteUser(ctx *context.Context) {
	u, err := user_model.GetUserByID(ctx, ctx.ParamsInt64(":userid"))
	if err != nil {
		ctx.ServerError("GetUserByID", err)
		auditParams := map[string]string{
			"affected_user_id": strconv.FormatInt(ctx.ParamsInt64(":userid"), 10),
			"error":            "Failed to get user",
		}
		audit.CreateAndSendEvent(audit.UserDeleteEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}

	auditParams := map[string]string{
		"email":            u.Email,
		"affected_user_id": strconv.FormatInt(u.ID, 10),
		"affected_user":    u.Name,
	}

	// admin should not delete themself
	if u.ID == ctx.Doer.ID {
		ctx.Flash.Error(ctx.Tr("admin.users.cannot_delete_self"))
		ctx.Redirect(setting.AppSubURL + "/admin/users/" + url.PathEscape(ctx.Params(":userid")))
		auditParams["error"] = "Cannot delete self"
		audit.CreateAndSendEvent(audit.UserDeleteEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}

	if err = user_service.DeleteUser(ctx, u, ctx.FormBool("purge")); err != nil {
		switch {
		case models.IsErrUserOwnRepos(err):
			ctx.Flash.Error(ctx.Tr("admin.users.still_own_repo"))
			auditParams["error"] = "User still is owner repository"
			ctx.Redirect(setting.AppSubURL + "/admin/users/" + url.PathEscape(ctx.Params(":userid")))
		case models.IsErrUserHasOrgs(err):
			ctx.Flash.Error(ctx.Tr("admin.users.still_has_org"))
			auditParams["error"] = "User still has organization"
			ctx.Redirect(setting.AppSubURL + "/admin/users/" + url.PathEscape(ctx.Params(":userid")))
		case models.IsErrUserOwnPackages(err):
			ctx.Flash.Error(ctx.Tr("admin.users.still_own_packages"))
			auditParams["error"] = "User still is owner packages"
			ctx.Redirect(setting.AppSubURL + "/admin/users/" + ctx.Params(":userid"))
		default:
			ctx.ServerError("DeleteUser", err)
			auditParams["error"] = "Error has occurred while deleting user"
		}
		audit.CreateAndSendEvent(audit.UserDeleteEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}
	log.Trace("Account deleted by admin (%s): %s", ctx.Doer.Name, u.Name)

	ctx.Flash.Success(ctx.Tr("admin.users.deletion_success"))
	audit.CreateAndSendEvent(audit.UserDeleteEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)
	ctx.Redirect(setting.AppSubURL + "/admin/users")
}

// AvatarPost response for change user's avatar request
func AvatarPost(ctx *context.Context) {
	u := prepareUserInfo(ctx)
	auditParams := map[string]string{
		"affected_user_id": strconv.FormatInt(u.ID, 10),
		"affected_user":    u.Name,
	}
	if ctx.Written() {
		auditParams["error"] = "Error occurs in form validation"
		audit.CreateAndSendEvent(audit.UserAvatarChange, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}

	form := web.GetForm(ctx).(*forms.AvatarForm)
	if err := user_setting.UpdateAvatarSetting(ctx, form, u); err != nil {
		ctx.Flash.Error(err.Error())
		auditParams["error"] = "Error has occurred while updating avatar settings"
		audit.CreateAndSendEvent(audit.UserAvatarChange, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
	} else {
		ctx.Flash.Success(ctx.Tr("settings.update_user_avatar_success"))
		audit.CreateAndSendEvent(audit.UserAvatarChange, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)
	}

	ctx.Redirect(setting.AppSubURL + "/admin/users/" + strconv.FormatInt(u.ID, 10))
}

// DeleteAvatar render delete avatar page
func DeleteAvatar(ctx *context.Context) {
	u := prepareUserInfo(ctx)
	auditParams := map[string]string{
		"affected_user_id": strconv.FormatInt(u.ID, 10),
		"affected_user":    u.Name,
	}
	if ctx.Written() {
		auditParams["error"] = "Error occurs in form validation"
		audit.CreateAndSendEvent(audit.UserAvatarDelete, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}

	if err := user_service.DeleteAvatar(u); err != nil {
		ctx.Flash.Error(err.Error())
		auditParams["error"] = "Error has occurred while deleting avatar"
		audit.CreateAndSendEvent(audit.UserAvatarDelete, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		ctx.Redirect(setting.AppSubURL + "/admin/users/" + strconv.FormatInt(u.ID, 10))
		return
	}

	ctx.Flash.Success(ctx.Tr("settings.delete_user_avatar_success"))
	audit.CreateAndSendEvent(audit.UserAvatarDelete, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)
	ctx.Redirect(setting.AppSubURL + "/admin/users/" + strconv.FormatInt(u.ID, 10))
}
