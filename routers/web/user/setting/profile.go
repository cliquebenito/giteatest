// Copyright 2014 The Gogs Authors. All rights reserved.
// Copyright 2018 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package setting

import (
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strconv"
	"strings"

	"code.gitea.io/gitea/models/role_model"
	"code.gitea.io/gitea/modules/json"
	"code.gitea.io/gitea/modules/sbt/audit"
	"code.gitea.io/gitea/modules/trace"
	"code.gitea.io/gitea/routers/utils"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/organization"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/base"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/translation"
	"code.gitea.io/gitea/modules/typesniffer"
	"code.gitea.io/gitea/modules/util"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/modules/web/middleware"
	"code.gitea.io/gitea/services/forms"
	user_service "code.gitea.io/gitea/services/user"
)

const (
	tplSettingsProfile      base.TplName = "user/settings/profile"
	tplSettingsAppearance   base.TplName = "user/settings/appearance"
	tplSettingsOrganization base.TplName = "user/settings/organization"
	tplSettingsRepositories base.TplName = "user/settings/repos"
)

// Profile render user's profile page
func Profile(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("settings.profile")
	ctx.Data["PageIsSettingsProfile"] = true
	ctx.Data["AllowedUserVisibilityModes"] = setting.Service.AllowedUserVisibilityModesSlice.ToVisibleTypeSlice()

	ctx.HTML(http.StatusOK, tplSettingsProfile)
}

// HandleUsernameChange handle username changes from user settings and admin interface
func HandleUsernameChange(ctx *context.Context, user *user_model.User, newName string) error {
	oldName := user.Name
	auditParams := map[string]string{
		"email":     user.Email,
		"old_value": oldName,
		"new_value": newName,
	}
	// rename user
	if err := user_service.RenameUser(ctx, user, newName); err != nil {
		switch {
		// Noop as username is not changed
		case user_model.IsErrUsernameNotChanged(err):
			ctx.Flash.Error(ctx.Tr("form.username_has_not_been_changed"))
			auditParams["error"] = "Username has not changed"
		// Non-local users are not allowed to change their username.
		case user_model.IsErrUserIsNotLocal(err):
			ctx.Flash.Error(ctx.Tr("form.username_change_not_local_user"))
			auditParams["error"] = "Non-local users are not allowed to change their username"
		case user_model.IsErrUserAlreadyExist(err):
			ctx.Flash.Error(ctx.Tr("form.username_been_taken"))
			auditParams["error"] = "Username been take"
		case user_model.IsErrEmailAlreadyUsed(err):
			ctx.Flash.Error(ctx.Tr("form.email_been_used"))
			auditParams["error"] = "Email been used"
		case db.IsErrNameReserved(err):
			ctx.Flash.Error(ctx.Tr("user.form.name_reserved", newName))
			auditParams["error"] = "Username reserved"
		case db.IsErrNamePatternNotAllowed(err):
			ctx.Flash.Error(ctx.Tr("user.form.name_pattern_not_allowed", newName))
			auditParams["error"] = "Username pattern not allowed"
		case db.IsErrNameCharsNotAllowed(err):
			ctx.Flash.Error(ctx.Tr("user.form.name_chars_not_allowed", newName))
			auditParams["error"] = "Username chars not allowed"
		default:
			ctx.ServerError("ChangeUserName", err)
			auditParams["error"] = "Error has occurred while renaming user"
		}
		audit.CreateAndSendEvent(audit.UserNameChangeEvent, user.Name, strconv.FormatInt(user.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return err
	}
	log.Trace("User name changed: %s -> %s", oldName, newName)
	audit.CreateAndSendEvent(audit.UserNameChangeEvent, user.Name, strconv.FormatInt(user.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)
	return nil
}

// ProfilePost response for change user's profile
func ProfilePost(ctx *context.Context) {
	form := web.GetForm(ctx).(*forms.UpdateProfileForm)
	ctx.Data["Title"] = ctx.Tr("settings")
	ctx.Data["PageIsSettingsProfile"] = true
	auditParams := map[string]string{
		"email": ctx.Doer.Email,
	}

	type auditValue struct {
		Name                string
		LowerName           string
		FullName            string
		KeepEmailPrivate    bool
		Website             string
		Location            string
		Description         string
		KeepActivityPrivate bool
		Visibility          string
	}

	oldAuditValue := auditValue{
		Name:                ctx.Doer.Name,
		LowerName:           ctx.Doer.LowerName,
		FullName:            ctx.Doer.FullName,
		KeepEmailPrivate:    ctx.Doer.KeepEmailPrivate,
		Website:             ctx.Doer.Website,
		Location:            ctx.Doer.Location,
		Description:         ctx.Doer.Description,
		KeepActivityPrivate: ctx.Doer.KeepActivityPrivate,
		Visibility:          ctx.Doer.Visibility.String(),
	}

	newAuditValue := auditValue{
		Name:                form.Name,
		LowerName:           strings.ToLower(form.Name),
		FullName:            form.FullName,
		KeepEmailPrivate:    form.KeepEmailPrivate,
		Website:             form.Website,
		Location:            form.Location,
		Description:         form.Description,
		KeepActivityPrivate: form.KeepActivityPrivate,
		Visibility:          form.Visibility.String(),
	}

	oldAuditValueBytes, _ := json.Marshal(oldAuditValue)
	auditParams["old_value"] = string(oldAuditValueBytes)

	newAuditValueBytes, _ := json.Marshal(newAuditValue)
	auditParams["new_value"] = string(newAuditValueBytes)

	if ctx.HasError() {
		ctx.HTML(http.StatusOK, tplSettingsProfile)
		auditParams["error"] = "Error occurs in form validation"
		audit.CreateAndSendEvent(audit.UserProfileEditEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}

	if len(form.Name) != 0 && ctx.Doer.Name != form.Name {
		log.Debug("Changing name for %s to %s", ctx.Doer.Name, form.Name)
		if err := HandleUsernameChange(ctx, ctx.Doer, form.Name); err != nil {
			ctx.Redirect(setting.AppSubURL + "/user/settings")
			auditParams["error"] = "Error has occurred while changing username"
			audit.CreateAndSendEvent(audit.UserProfileEditEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}
		ctx.Doer.Name = form.Name
		ctx.Doer.LowerName = strings.ToLower(form.Name)
	}

	ctx.Doer.FullName = form.FullName
	ctx.Doer.KeepEmailPrivate = form.KeepEmailPrivate
	ctx.Doer.Website = form.Website
	ctx.Doer.Location = form.Location
	ctx.Doer.Description = form.Description
	ctx.Doer.KeepActivityPrivate = form.KeepActivityPrivate
	ctx.Doer.Visibility = form.Visibility
	if err := user_model.UpdateUserSetting(ctx.Doer); err != nil {
		if _, ok := err.(user_model.ErrEmailAlreadyUsed); ok {
			ctx.Flash.Error(ctx.Tr("form.email_been_used"))
			ctx.Redirect(setting.AppSubURL + "/user/settings")
			auditParams["error"] = "Email been used"
			audit.CreateAndSendEvent(audit.UserProfileEditEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}
		ctx.ServerError("UpdateUser", err)
		auditParams["error"] = "Error has occurred while updating username"
		audit.CreateAndSendEvent(audit.UserProfileEditEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}

	audit.CreateAndSendEvent(audit.UserProfileEditEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)
	log.Trace("User settings updated: %s", ctx.Doer.Name)
	ctx.Flash.Success(ctx.Tr("settings.update_profile_success"))
	ctx.Redirect(setting.AppSubURL + "/user/settings")
}

// UpdateAvatarSetting update user's avatar
// FIXME: limit size.
func UpdateAvatarSetting(ctx *context.Context, form *forms.AvatarForm, ctxUser *user_model.User) error {
	ctxUser.UseCustomAvatar = form.Source == forms.AvatarLocal
	if len(form.Gravatar) > 0 {
		if form.Avatar != nil {
			ctxUser.Avatar = base.EncodeMD5(form.Gravatar)
		} else {
			ctxUser.Avatar = ""
		}
		ctxUser.AvatarEmail = form.Gravatar
	}

	if form.Avatar != nil && form.Avatar.Filename != "" {
		fr, err := form.Avatar.Open()
		if err != nil {
			return fmt.Errorf("Avatar.Open: %w", err)
		}
		defer fr.Close()

		if form.Avatar.Size > setting.Avatar.MaxFileSize {
			return errors.New(ctx.Tr("settings.uploaded_avatar_is_too_big"))
		}

		data, err := io.ReadAll(fr)
		if err != nil {
			return fmt.Errorf("io.ReadAll: %w", err)
		}

		st := typesniffer.DetectContentType(data)
		if !(st.IsImage() && !st.IsSvgImage()) {
			return errors.New(ctx.Tr("settings.uploaded_avatar_not_a_image"))
		}
		if err = user_service.UploadAvatar(ctxUser, data); err != nil {
			return fmt.Errorf("UploadAvatar: %w", err)
		}
	} else if ctxUser.UseCustomAvatar && ctxUser.Avatar == "" {
		// No avatar is uploaded but setting has been changed to enable,
		// generate a random one when needed.
		if err := user_model.GenerateRandomAvatar(ctx, ctxUser); err != nil {
			log.Error("GenerateRandomAvatar[%d]: %v", ctxUser.ID, err)
		}
	}

	if err := user_model.UpdateUserCols(ctx, ctxUser, "avatar", "avatar_email", "use_custom_avatar"); err != nil {
		return fmt.Errorf("UpdateUser: %w", err)
	}

	return nil
}

// AvatarPost response for change user's avatar request
func AvatarPost(ctx *context.Context) {
	form := web.GetForm(ctx).(*forms.AvatarForm)
	if err := UpdateAvatarSetting(ctx, form, ctx.Doer); err != nil {
		ctx.Flash.Error(err.Error())
		auditParams := map[string]string{
			"error": "Error has occurred while updating avatar settings",
		}
		audit.CreateAndSendEvent(audit.UserAvatarChange, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
	} else {
		ctx.Flash.Success(ctx.Tr("settings.update_avatar_success"))
		audit.CreateAndSendEvent(audit.UserAvatarChange, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, nil)
	}

	ctx.Redirect(setting.AppSubURL + "/user/settings")
}

// DeleteAvatar render delete avatar page
func DeleteAvatar(ctx *context.Context) {
	if err := user_service.DeleteAvatar(ctx.Doer); err != nil {
		auditParams := map[string]string{
			"error": "Error has occurred while deleting avatar",
		}
		audit.CreateAndSendEvent(audit.UserAvatarDelete, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		ctx.Flash.Error(err.Error())
		ctx.Redirect(setting.AppSubURL + "/user/settings")
		return
	}

	ctx.Flash.Success(ctx.Tr("settings.delete_user_avatar_success"))
	audit.CreateAndSendEvent(audit.UserAvatarDelete, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, nil)
	ctx.Redirect(setting.AppSubURL + "/user/settings")
}

// Organization render all the organization of the user
func Organization(ctx *context.Context) {
	logTracer := trace.NewLogTracer()
	message := logTracer.CreateTraceMessage(ctx)
	err := logTracer.Trace(message)
	if err != nil {
		log.Error("Error has occurred while creating trace message: %v", err)
	}
	defer func() {
		err = logTracer.TraceTime(message)
		if err != nil {
			log.Error("Error has occurred while creating trace time message: %v", err)
		}
	}()

	ctx.Data["Title"] = ctx.Tr("settings.organization")
	ctx.Data["PageIsSettingsOrganization"] = true

	opts := organization.FindOrgOptions{
		ListOptions: db.ListOptions{
			PageSize: setting.UI.Admin.UserPagingNum,
			Page:     ctx.FormInt("page"),
		},
		UserID:         ctx.Doer.ID,
		IncludePrivate: ctx.IsSigned,
	}

	if opts.Page <= 0 {
		opts.Page = 1
	}

	orgs := make([]*organization.Organization, 0)
	var total int64
	if setting.SourceControl.TenantWithRoleModeEnabled {
		tenantID, errGetTenantIdByUserId := role_model.GetUserTenantId(ctx, ctx.Doer.ID)
		if errGetTenantIdByUserId != nil {
			log.Error("Organization role_model.GetUserTenantId failed: %v", errGetTenantIdByUserId)
			ctx.Error(http.StatusNotFound, fmt.Sprintf("Organization role_model.GetUserTenantId failed: %v", errGetTenantIdByUserId))
			return
		}
		privileges, errGetTenantPrivilege := utils.GetTenantsPrivilegesByUserID(ctx, ctx.Doer.ID)
		if errGetTenantPrivilege != nil {
			log.Error("Organization utils.GetTenantsPrivilegesByUserID failed: %v", errGetTenantIdByUserId)
			ctx.Error(http.StatusNotFound, fmt.Sprintf("Organization utils.GetUsersPrivilegesByUserID failed: %v", errGetTenantIdByUserId))
			return
		}
		organizationsPrivilege := utils.ConvertTenantPrivilegesInOrganizations(privileges)
		for _, orgPrivilege := range organizationsPrivilege {
			allowed, errCheckPermission := role_model.CheckUserPermissionToOrganization(ctx, ctx.Doer, tenantID, orgPrivilege, role_model.READ)
			if errCheckPermission != nil {
				log.Error("Organization role_model.CheckUserPermissionToOrganization failed: %v", errCheckPermission)
				return
			}
			if allowed {
				orgs = append(orgs, orgPrivilege)
			}
		}
		total = int64(len(orgs))
	} else {
		orgs, err = organization.FindOrgs(opts)
		if err != nil {
			ctx.ServerError("FindOrgs", err)
			return
		}
		total, err = organization.CountOrgs(opts)
		if err != nil {
			ctx.ServerError("CountOrgs", err)
			return
		}
	}
	ctx.Data["Orgs"] = orgs
	pager := context.NewPagination(int(total), opts.PageSize, opts.Page, 5)
	pager.SetDefaultParams(ctx)
	ctx.Data["Page"] = pager
	ctx.HTML(http.StatusOK, tplSettingsOrganization)
}

// Appearance render user's appearance settings
func Appearance(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("settings.appearance")
	ctx.Data["PageIsSettingsAppearance"] = true

	var hiddenCommentTypes *big.Int
	val, err := user_model.GetUserSetting(ctx.Doer.ID, user_model.SettingsKeyHiddenCommentTypes)
	if err != nil {
		ctx.ServerError("GetUserSetting", err)
		return
	}
	hiddenCommentTypes, _ = new(big.Int).SetString(val, 10) // we can safely ignore the failed conversion here

	ctx.Data["IsCommentTypeGroupChecked"] = func(commentTypeGroup string) bool {
		return forms.IsUserHiddenCommentTypeGroupChecked(commentTypeGroup, hiddenCommentTypes)
	}

	ctx.HTML(http.StatusOK, tplSettingsAppearance)
}

// UpdateUIThemePost is used to update users' specific theme
func UpdateUIThemePost(ctx *context.Context) {
	form := web.GetForm(ctx).(*forms.UpdateThemeForm)
	ctx.Data["Title"] = ctx.Tr("settings")
	ctx.Data["PageIsSettingsAppearance"] = true

	if ctx.HasError() {
		ctx.Redirect(setting.AppSubURL + "/user/settings/appearance")
		return
	}

	if !form.IsThemeExists() {
		ctx.Flash.Error(ctx.Tr("settings.theme_update_error"))
		ctx.Redirect(setting.AppSubURL + "/user/settings/appearance")
		return
	}

	if err := user_model.UpdateUserTheme(ctx.Doer, form.Theme); err != nil {
		ctx.Flash.Error(ctx.Tr("settings.theme_update_error"))
		ctx.Redirect(setting.AppSubURL + "/user/settings/appearance")
		return
	}

	log.Trace("Update user theme: %s", ctx.Doer.Name)
	ctx.Flash.Success(ctx.Tr("settings.theme_update_success"))
	ctx.Redirect(setting.AppSubURL + "/user/settings/appearance")
}

// UpdateUserLang update a user's language
func UpdateUserLang(ctx *context.Context) {
	form := web.GetForm(ctx).(*forms.UpdateLanguageForm)
	ctx.Data["Title"] = ctx.Tr("settings")
	ctx.Data["PageIsSettingsAppearance"] = true

	if len(form.Language) != 0 {
		if !util.SliceContainsString(setting.Langs, form.Language) {
			ctx.Flash.Error(ctx.Tr("settings.update_language_not_found", form.Language))
			ctx.Redirect(setting.AppSubURL + "/user/settings/appearance")
			return
		}
		ctx.Doer.Language = form.Language
	}

	if err := user_model.UpdateUserSetting(ctx.Doer); err != nil {
		ctx.ServerError("UpdateUserSetting", err)
		return
	}

	// Update the language to the one we just set
	middleware.SetLocaleCookie(ctx.Resp, ctx.Doer.Language, 0)

	log.Trace("User settings updated: %s", ctx.Doer.Name)
	ctx.Flash.Success(translation.NewLocale(ctx.Doer.Language).Tr("settings.update_language_success"))
	ctx.Redirect(setting.AppSubURL + "/user/settings/appearance")
}

// UpdateUserHiddenComments update a user's shown comment types
func UpdateUserHiddenComments(ctx *context.Context) {
	err := user_model.SetUserSetting(ctx.Doer.ID, user_model.SettingsKeyHiddenCommentTypes, forms.UserHiddenCommentTypesFromRequest(ctx).String())
	if err != nil {
		ctx.ServerError("SetUserSetting", err)
		return
	}

	log.Trace("User settings updated: %s", ctx.Doer.Name)
	ctx.Flash.Success(ctx.Tr("settings.saved_successfully"))
	ctx.Redirect(setting.AppSubURL + "/user/settings/appearance")
}
