// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package auth

import (
	"code.gitea.io/gitea/modules/sbt/audit"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"code.gitea.io/gitea/models/auth"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/base"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/web"
	auth_service "code.gitea.io/gitea/services/auth"
	"code.gitea.io/gitea/services/auth/source/oauth2"
	"code.gitea.io/gitea/services/externalaccount"
	"code.gitea.io/gitea/services/forms"

	"github.com/markbates/goth"
)

var tplLinkAccount base.TplName = "user/auth/link_account"

// LinkAccount shows the page where the user can decide to login or create a new account
func LinkAccount(ctx *context.Context) {
	ctx.Data["DisablePassword"] = !setting.Service.RequireExternalRegistrationPassword || setting.Service.AllowOnlyExternalRegistration
	ctx.Data["Title"] = ctx.Tr("link_account")
	ctx.Data["LinkAccountMode"] = true
	ctx.Data["EnableCaptcha"] = setting.Service.EnableCaptcha && setting.Service.RequireExternalRegistrationCaptcha
	ctx.Data["Captcha"] = context.GetImageCaptcha()
	ctx.Data["CaptchaType"] = setting.Service.CaptchaType
	ctx.Data["RecaptchaURL"] = setting.Service.RecaptchaURL
	ctx.Data["RecaptchaSitekey"] = setting.Service.RecaptchaSitekey
	ctx.Data["HcaptchaSitekey"] = setting.Service.HcaptchaSitekey
	ctx.Data["McaptchaSitekey"] = setting.Service.McaptchaSitekey
	ctx.Data["McaptchaURL"] = setting.Service.McaptchaURL
	ctx.Data["DisableRegistration"] = setting.Service.DisableRegistration
	ctx.Data["AllowOnlyInternalRegistration"] = setting.Service.AllowOnlyInternalRegistration
	ctx.Data["ShowRegistrationButton"] = false

	// use this to set the right link into the signIn and signUp templates in the link_account template
	ctx.Data["SignInLink"] = setting.AppSubURL + "/user/link_account_signin"
	ctx.Data["SignUpLink"] = setting.AppSubURL + "/user/link_account_signup"

	gothUser := ctx.Session.Get("linkAccountGothUser")
	if gothUser == nil {
		ctx.ServerError("UserSignIn", errors.New("not in LinkAccount session"))
		return
	}

	gu, _ := gothUser.(goth.User)
	uname := getUserName(&gu)
	email := gu.Email
	ctx.Data["user_name"] = uname
	ctx.Data["email"] = email

	if len(email) != 0 {
		u, err := user_model.GetUserByEmail(ctx, email)
		if err != nil && !user_model.IsErrUserNotExist(err) {
			ctx.ServerError("UserSignIn", err)
			return
		}
		if u != nil {
			ctx.Data["user_exists"] = true
		}
	} else if len(uname) != 0 {
		u, err := user_model.GetUserByName(ctx, uname)
		if err != nil && !user_model.IsErrUserNotExist(err) {
			ctx.ServerError("UserSignIn", err)
			return
		}
		if u != nil {
			ctx.Data["user_exists"] = true
		} else {
			u, err = user_model.GetUserByID(ctx, u.ID)
			if err != nil && !user_model.IsErrUserNotExist(err) {
				ctx.ServerError("UserSignIn", err)
				return
			}
			if u != nil {
				ctx.Data["user_exists"] = true
			}
		}
	}

	ctx.HTML(http.StatusOK, tplLinkAccount)
}

// LinkAccountPostSignIn handle the coupling of external account with another account using signIn
func LinkAccountPostSignIn(ctx *context.Context) {
	signInForm := web.GetForm(ctx).(*forms.SignInForm)
	ctx.Data["DisablePassword"] = !setting.Service.RequireExternalRegistrationPassword || setting.Service.AllowOnlyExternalRegistration
	ctx.Data["Title"] = ctx.Tr("link_account")
	ctx.Data["LinkAccountMode"] = true
	ctx.Data["LinkAccountModeSignIn"] = true
	ctx.Data["EnableCaptcha"] = setting.Service.EnableCaptcha && setting.Service.RequireExternalRegistrationCaptcha
	ctx.Data["RecaptchaURL"] = setting.Service.RecaptchaURL
	ctx.Data["Captcha"] = context.GetImageCaptcha()
	ctx.Data["CaptchaType"] = setting.Service.CaptchaType
	ctx.Data["RecaptchaSitekey"] = setting.Service.RecaptchaSitekey
	ctx.Data["HcaptchaSitekey"] = setting.Service.HcaptchaSitekey
	ctx.Data["McaptchaSitekey"] = setting.Service.McaptchaSitekey
	ctx.Data["McaptchaURL"] = setting.Service.McaptchaURL
	ctx.Data["DisableRegistration"] = setting.Service.DisableRegistration
	ctx.Data["ShowRegistrationButton"] = false

	// use this to set the right link into the signIn and signUp templates in the link_account template
	ctx.Data["SignInLink"] = setting.AppSubURL + "/user/link_account_signin"
	ctx.Data["SignUpLink"] = setting.AppSubURL + "/user/link_account_signup"

	gothUser := ctx.Session.Get("linkAccountGothUser")
	if gothUser == nil {
		ctx.ServerError("UserSignIn", errors.New("not in LinkAccount session"))
		auditParams := map[string]string{
			"error": "Error has occurred while linking account goth user",
		}
		audit.CreateAndSendEvent(audit.UserLoginEvent, signInForm.UserName, audit.EmptyRequiredField, audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}

	if ctx.HasError() {
		ctx.HTML(http.StatusOK, tplLinkAccount)
		auditParams := map[string]string{
			"error": "Error occurs in form validation",
		}
		audit.CreateAndSendEvent(audit.UserLoginEvent, signInForm.UserName, audit.EmptyRequiredField, audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}

	u, _, err := auth_service.UserSignIn(signInForm.UserName, signInForm.Password)
	if err != nil {
		auditParams := make(map[string]string)
		if user_model.IsErrUserNotExist(err) {
			ctx.Data["user_exists"] = true
			ctx.RenderWithErr(ctx.Tr("form.username_password_incorrect"), tplLinkAccount, &signInForm)
			auditParams["error"] = "Username or password is incorrect"
		} else {
			ctx.ServerError("UserLinkAccount", err)
			auditParams["error"] = "Error has occurred while user signing in"
		}
		audit.CreateAndSendEvent(audit.UserLoginEvent, signInForm.UserName, audit.EmptyRequiredField, audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}

	linkAccount(ctx, u, gothUser.(goth.User), signInForm.Remember)
}

func linkAccount(ctx *context.Context, u *user_model.User, gothUser goth.User, remember bool) {
	updateAvatarIfNeed(gothUser.AvatarURL, u)
	auditParams := map[string]string{
		"email": u.Email,
	}

	// If this user is enrolled in 2FA, we can't sign the user in just yet.
	// Instead, redirect them to the 2FA authentication page.
	// We deliberately ignore the skip local 2fa setting here because we are linking to a previous user here
	_, err := auth.GetTwoFactorByUID(u.ID)
	if err != nil {
		if !auth.IsErrTwoFactorNotEnrolled(err) {
			ctx.ServerError("UserLinkAccount", err)
			auditParams["error"] = "Two factor not enrolled"
			audit.CreateAndSendEvent(audit.UserLoginEvent, u.Name, strconv.FormatInt(u.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}

		err = externalaccount.LinkAccountToUser(u, gothUser)
		if err != nil {
			ctx.ServerError("UserLinkAccount", err)
			auditParams["error"] = "Error has occurred while getting link account"
			audit.CreateAndSendEvent(audit.UserLoginEvent, u.Name, strconv.FormatInt(u.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}

		handleSignIn(ctx, u, remember)
		return
	}

	if err := updateSession(ctx, nil, map[string]interface{}{
		// User needs to use 2FA, save data and redirect to 2FA page.
		"twofaUid":      u.ID,
		"twofaRemember": remember,
		"linkAccount":   true,
	}); err != nil {
		ctx.ServerError("RegenerateSession", err)
		auditParams["error"] = "Error has occurred while updating session"
		audit.CreateAndSendEvent(audit.UserLoginEvent, u.Name, strconv.FormatInt(u.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}

	// If WebAuthn is enrolled -> Redirect to WebAuthn instead
	regs, err := auth.GetWebAuthnCredentialsByUID(u.ID)
	if err == nil && len(regs) > 0 {
		ctx.Redirect(setting.AppSubURL + "/user/webauthn")
		return
	}

	ctx.Redirect(setting.AppSubURL + "/user/two_factor")
}

// LinkAccountPostRegister handle the creation of a new account for an external account using signUp
func LinkAccountPostRegister(ctx *context.Context) {
	form := web.GetForm(ctx).(*forms.RegisterForm)
	// TODO Make insecure passwords optional for local accounts also,
	//      once email-based Second-Factor Auth is available
	ctx.Data["DisablePassword"] = !setting.Service.RequireExternalRegistrationPassword || setting.Service.AllowOnlyExternalRegistration
	ctx.Data["Title"] = ctx.Tr("link_account")
	ctx.Data["LinkAccountMode"] = true
	ctx.Data["LinkAccountModeRegister"] = true
	ctx.Data["EnableCaptcha"] = setting.Service.EnableCaptcha && setting.Service.RequireExternalRegistrationCaptcha
	ctx.Data["RecaptchaURL"] = setting.Service.RecaptchaURL
	ctx.Data["Captcha"] = context.GetImageCaptcha()
	ctx.Data["CaptchaType"] = setting.Service.CaptchaType
	ctx.Data["RecaptchaSitekey"] = setting.Service.RecaptchaSitekey
	ctx.Data["HcaptchaSitekey"] = setting.Service.HcaptchaSitekey
	ctx.Data["McaptchaSitekey"] = setting.Service.McaptchaSitekey
	ctx.Data["McaptchaURL"] = setting.Service.McaptchaURL
	ctx.Data["DisableRegistration"] = setting.Service.DisableRegistration
	ctx.Data["ShowRegistrationButton"] = false

	// use this to set the right link into the signIn and signUp templates in the link_account template
	ctx.Data["SignInLink"] = setting.AppSubURL + "/user/link_account_signin"
	ctx.Data["SignUpLink"] = setting.AppSubURL + "/user/link_account_signup"

	gothUserInterface := ctx.Session.Get("linkAccountGothUser")
	if gothUserInterface == nil {
		ctx.ServerError("UserSignUp", errors.New("not in LinkAccount session"))
		auditParams := map[string]string{
			"error": "Not in LinkAccount session",
		}
		audit.CreateAndSendEvent(audit.UserCreateEvent, form.UserName, audit.EmptyRequiredField, audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}
	gothUser, ok := gothUserInterface.(goth.User)
	if !ok {
		ctx.ServerError("UserSignUp", fmt.Errorf("session linkAccountGothUser type is %t but not goth.User", gothUserInterface))
		auditParams := map[string]string{
			"error": "Session linkAccountGothUser type isn't goth.User",
		}
		audit.CreateAndSendEvent(audit.UserCreateEvent, form.UserName, audit.EmptyRequiredField, audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}

	auditParams := map[string]string{
		"email": gothUser.Email,
	}

	if ctx.HasError() {
		ctx.HTML(http.StatusOK, tplLinkAccount)
		auditParams["error"] = "Error occurs in form validation"
		audit.CreateAndSendEvent(audit.UserCreateEvent, form.UserName, audit.EmptyRequiredField, audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}

	if setting.Service.DisableRegistration || setting.Service.AllowOnlyInternalRegistration {
		ctx.Error(http.StatusForbidden)
		auditParams["error"] = "Registration disabled or allow only external registration"
		audit.CreateAndSendEvent(audit.UserCreateEvent, form.UserName, audit.EmptyRequiredField, audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}

	if setting.Service.EnableCaptcha && setting.Service.RequireExternalRegistrationCaptcha {
		context.VerifyCaptcha(ctx, tplLinkAccount, form)
		if ctx.Written() {
			auditParams["error"] = "Failed to verify captcha"
			audit.CreateAndSendEvent(audit.UserCreateEvent, form.UserName, audit.EmptyRequiredField, audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}
	}

	if !form.IsEmailDomainAllowed() {
		ctx.RenderWithErr(ctx.Tr("auth.email_domain_blacklisted"), tplLinkAccount, &form)
		auditParams["error"] = "Email domain blacklisted"
		audit.CreateAndSendEvent(audit.UserCreateEvent, form.UserName, audit.EmptyRequiredField, audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}

	if setting.Service.AllowOnlyExternalRegistration || !setting.Service.RequireExternalRegistrationPassword {
		// In user_model.User an empty password is classed as not set, so we set form.Password to empty.
		// Eventually the database should be changed to indicate "Second Factor"-enabled accounts
		// (accounts that do not introduce the security vulnerabilities of a password).
		// If a user decides to circumvent second-factor security, and purposefully create a password,
		// they can still do so using the "Recover Account" option.
		form.Password = ""
	} else {
		if (len(strings.TrimSpace(form.Password)) > 0 || len(strings.TrimSpace(form.Retype)) > 0) && form.Password != form.Retype {
			ctx.Data["Err_Password"] = true
			ctx.RenderWithErr(ctx.Tr("form.password_not_match"), tplLinkAccount, &form)
			auditParams["error"] = "Password is not match"
			audit.CreateAndSendEvent(audit.UserCreateEvent, form.UserName, audit.EmptyRequiredField, audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}
		if len(strings.TrimSpace(form.Password)) > 0 && len(form.Password) < setting.MinPasswordLength {
			ctx.Data["Err_Password"] = true
			ctx.RenderWithErr(ctx.Tr("auth.password_too_short", setting.MinPasswordLength), tplLinkAccount, &form)
			auditParams["error"] = "Password is too short"
			audit.CreateAndSendEvent(audit.UserCreateEvent, form.UserName, audit.EmptyRequiredField, audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}
	}

	authSource, err := auth.GetActiveOAuth2SourceByName(gothUser.Provider)
	if err != nil {
		ctx.ServerError("CreateUser", err)
		auditParams["error"] = "Error has occurred while getting active oauth2 source by name"
		audit.CreateAndSendEvent(audit.UserCreateEvent, form.UserName, audit.EmptyRequiredField, audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}

	u := &user_model.User{
		Name:        form.UserName,
		Email:       form.Email,
		Passwd:      form.Password,
		LoginType:   auth.OAuth2,
		LoginSource: authSource.ID,
		LoginName:   gothUser.UserID,
	}

	if !createAndHandleCreatedUser(ctx, tplLinkAccount, form, u, nil, &gothUser, false) {
		// error already handled
		return
	}

	audit.CreateAndSendEvent(audit.UserCreateEvent, u.Name, strconv.FormatInt(u.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)

	source := authSource.Cfg.(*oauth2.Source)
	if err := syncGroupsToTeams(ctx, source, &gothUser, u); err != nil {
		ctx.ServerError("SyncGroupsToTeams", err)
		auditParams["error"] = "Error has occurred while synchronizing groups to teams"
		audit.CreateAndSendEvent(audit.UserLoginEvent, u.Name, strconv.FormatInt(u.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}

	handleSignIn(ctx, u, false)
}
