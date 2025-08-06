// Copyright 2014 The Gogs Authors. All rights reserved.
// Copyright 2018 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package setting

import (
	"code.gitea.io/gitea/modules/sbt/audit"
	"net/http"
	"strconv"

	asymkey_model "code.gitea.io/gitea/models/asymkey"
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/modules/base"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/web"
	asymkey_service "code.gitea.io/gitea/services/asymkey"
	"code.gitea.io/gitea/services/forms"
)

const (
	tplSettingsKeys base.TplName = "user/settings/keys"
)

// Keys render user's SSH/GPG public keys page
func Keys(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("settings.ssh_gpg_keys")
	ctx.Data["PageIsSettingsKeys"] = true
	ctx.Data["DisableSSH"] = setting.SSH.Disabled
	ctx.Data["BuiltinSSH"] = setting.SSH.StartBuiltinServer
	ctx.Data["AllowPrincipals"] = setting.SSH.AuthorizedPrincipalsEnabled

	loadKeysData(ctx)

	ctx.HTML(http.StatusOK, tplSettingsKeys)
}

// KeysPost response for change user's SSH/GPG keys
func KeysPost(ctx *context.Context) {
	form := web.GetForm(ctx).(*forms.AddKeyForm)
	ctx.Data["Title"] = ctx.Tr("settings")
	ctx.Data["PageIsSettingsKeys"] = true
	ctx.Data["DisableSSH"] = setting.SSH.Disabled
	ctx.Data["BuiltinSSH"] = setting.SSH.StartBuiltinServer
	ctx.Data["AllowPrincipals"] = setting.SSH.AuthorizedPrincipalsEnabled

	auditParams := map[string]string{
		"email": ctx.Doer.Email,
	}

	if ctx.HasError() {
		loadKeysData(ctx)
		auditParams["error"] = "Error occurs in form validation"

		switch form.Type {
		case "gpg":
			audit.CreateAndSendEvent(audit.GPGKeyAddEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		case "ssh":
			audit.CreateAndSendEvent(audit.SSHKeyAddEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		}

		ctx.HTML(http.StatusOK, tplSettingsKeys)
		return
	}
	switch form.Type {
	case "principal":
		content, err := asymkey_model.CheckPrincipalKeyString(ctx.Doer, form.Content)
		if err != nil {
			if db.IsErrSSHDisabled(err) {
				ctx.Flash.Info(ctx.Tr("settings.ssh_disabled"))
			} else {
				ctx.Flash.Error(ctx.Tr("form.invalid_ssh_principal", err.Error()))
			}
			ctx.Redirect(setting.AppSubURL + "/user/settings/keys")
			return
		}
		if _, err = asymkey_model.AddPrincipalKey(ctx.Doer.ID, content, 0); err != nil {
			ctx.Data["HasPrincipalError"] = true
			switch {
			case asymkey_model.IsErrKeyAlreadyExist(err), asymkey_model.IsErrKeyNameAlreadyUsed(err):
				loadKeysData(ctx)

				ctx.Data["Err_Content"] = true
				ctx.RenderWithErr(ctx.Tr("settings.ssh_principal_been_used"), tplSettingsKeys, &form)
			default:
				ctx.ServerError("AddPrincipalKey", err)
			}
			return
		}
		ctx.Flash.Success(ctx.Tr("settings.add_principal_success", form.Content))
		ctx.Redirect(setting.AppSubURL + "/user/settings/keys")
	case "gpg":
		token := asymkey_model.VerificationToken(ctx.Doer, 1)
		lastToken := asymkey_model.VerificationToken(ctx.Doer, 0)

		keys, err := asymkey_model.AddGPGKey(ctx.Doer.ID, form.Content, token, form.Signature)
		if err != nil && asymkey_model.IsErrGPGInvalidTokenSignature(err) {
			keys, err = asymkey_model.AddGPGKey(ctx.Doer.ID, form.Content, lastToken, form.Signature)
		}
		if err != nil {
			ctx.Data["HasGPGError"] = true
			switch {
			case asymkey_model.IsErrGPGKeyParsing(err):
				ctx.Flash.Error(ctx.Tr("form.invalid_gpg_key", err.Error()))
				ctx.Redirect(setting.AppSubURL + "/user/settings/keys")
				auditParams["error"] = "Invalid GPG key"
			case asymkey_model.IsErrGPGKeyIDAlreadyUsed(err):
				loadKeysData(ctx)

				ctx.Data["Err_Content"] = true
				ctx.RenderWithErr(ctx.Tr("settings.gpg_key_id_used"), tplSettingsKeys, &form)
				auditParams["error"] = "GPG key id been used"
			case asymkey_model.IsErrGPGInvalidTokenSignature(err):
				loadKeysData(ctx)
				ctx.Data["Err_Content"] = true
				ctx.Data["Err_Signature"] = true
				keyID := err.(asymkey_model.ErrGPGInvalidTokenSignature).ID
				ctx.Data["KeyID"] = keyID
				ctx.Data["PaddedKeyID"] = asymkey_model.PaddedKeyID(keyID)
				ctx.RenderWithErr(ctx.Tr("settings.gpg_invalid_token_signature"), tplSettingsKeys, &form)
				auditParams["error"] = "GPG invalid token signature"
			case asymkey_model.IsErrGPGNoEmailFound(err):
				loadKeysData(ctx)

				ctx.Data["Err_Content"] = true
				ctx.Data["Err_Signature"] = true
				keyID := err.(asymkey_model.ErrGPGNoEmailFound).ID
				ctx.Data["KeyID"] = keyID
				ctx.Data["PaddedKeyID"] = asymkey_model.PaddedKeyID(keyID)
				ctx.RenderWithErr(ctx.Tr("settings.gpg_no_key_email_found"), tplSettingsKeys, &form)
				auditParams["error"] = "GPG no key email found"
			default:
				ctx.ServerError("AddPublicKey", err)
				auditParams["error"] = "Error has occurred while adding public key"
			}
			audit.CreateAndSendEvent(audit.GPGKeyAddEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}
		keyIDs := ""
		for _, key := range keys {
			keyIDs += key.KeyID
			keyIDs += ", "
		}
		if len(keyIDs) > 0 {
			keyIDs = keyIDs[:len(keyIDs)-2]
		}
		ctx.Flash.Success(ctx.Tr("settings.add_gpg_key_success", keyIDs))
		ctx.Redirect(setting.AppSubURL + "/user/settings/keys")
		audit.CreateAndSendEvent(audit.GPGKeyAddEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)
	case "verify_gpg":
		token := asymkey_model.VerificationToken(ctx.Doer, 1)
		lastToken := asymkey_model.VerificationToken(ctx.Doer, 0)

		keyID, err := asymkey_model.VerifyGPGKey(ctx.Doer.ID, form.KeyID, token, form.Signature)
		if err != nil && asymkey_model.IsErrGPGInvalidTokenSignature(err) {
			keyID, err = asymkey_model.VerifyGPGKey(ctx.Doer.ID, form.KeyID, lastToken, form.Signature)
		}
		if err != nil {
			ctx.Data["HasGPGVerifyError"] = true
			switch {
			case asymkey_model.IsErrGPGInvalidTokenSignature(err):
				loadKeysData(ctx)
				ctx.Data["VerifyingID"] = form.KeyID
				ctx.Data["Err_Signature"] = true
				keyID := err.(asymkey_model.ErrGPGInvalidTokenSignature).ID
				ctx.Data["KeyID"] = keyID
				ctx.Data["PaddedKeyID"] = asymkey_model.PaddedKeyID(keyID)
				ctx.RenderWithErr(ctx.Tr("settings.gpg_invalid_token_signature"), tplSettingsKeys, &form)
			default:
				ctx.ServerError("VerifyGPG", err)
			}
		}
		ctx.Flash.Success(ctx.Tr("settings.verify_gpg_key_success", keyID))
		ctx.Redirect(setting.AppSubURL + "/user/settings/keys")
	case "ssh":
		content, err := asymkey_model.CheckPublicKeyString(form.Content)
		if err != nil {
			if db.IsErrSSHDisabled(err) {
				ctx.Flash.Info(ctx.Tr("settings.ssh_disabled"))
				auditParams["error"] = "SSH disabled"
			} else if asymkey_model.IsErrKeyUnableVerify(err) {
				ctx.Flash.Info(ctx.Tr("form.unable_verify_ssh_key"))
				auditParams["error"] = "Unable verify SSH key"
			} else if err == asymkey_model.ErrKeyIsPrivate {
				ctx.Flash.Error(ctx.Tr("form.must_use_public_key"))
				auditParams["error"] = "Must use public key"
			} else {
				ctx.Flash.Error(ctx.Tr("form.invalid_ssh_key", err.Error()))
				auditParams["error"] = "Invalid SSH key"
			}
			ctx.Redirect(setting.AppSubURL + "/user/settings/keys")
			audit.CreateAndSendEvent(audit.SSHKeyAddEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}

		if _, err = asymkey_model.AddPublicKey(ctx.Doer.ID, form.Title, content, 0); err != nil {
			ctx.Data["HasSSHError"] = true
			switch {
			case asymkey_model.IsErrKeyAlreadyExist(err):
				loadKeysData(ctx)

				ctx.Data["Err_Content"] = true
				ctx.RenderWithErr(ctx.Tr("settings.ssh_key_been_used"), tplSettingsKeys, &form)
				auditParams["error"] = "SSH key been used"
			case asymkey_model.IsErrKeyNameAlreadyUsed(err):
				loadKeysData(ctx)

				ctx.Data["Err_Title"] = true
				ctx.RenderWithErr(ctx.Tr("settings.ssh_key_name_used"), tplSettingsKeys, &form)
				auditParams["error"] = "SSH key name been used"
			case asymkey_model.IsErrKeyUnableVerify(err):
				ctx.Flash.Info(ctx.Tr("form.unable_verify_ssh_key"))
				ctx.Redirect(setting.AppSubURL + "/user/settings/keys")
				auditParams["error"] = "Unable verify SSH key"
			default:
				ctx.ServerError("AddPublicKey", err)
				auditParams["error"] = "Error has occurred while adding public key"
			}
			audit.CreateAndSendEvent(audit.SSHKeyAddEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}
		ctx.Flash.Success(ctx.Tr("settings.add_key_success", form.Title))
		ctx.Redirect(setting.AppSubURL + "/user/settings/keys")
		audit.CreateAndSendEvent(audit.SSHKeyAddEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)
	case "verify_ssh":
		token := asymkey_model.VerificationToken(ctx.Doer, 1)
		lastToken := asymkey_model.VerificationToken(ctx.Doer, 0)

		fingerprint, err := asymkey_model.VerifySSHKey(ctx.Doer.ID, form.Fingerprint, token, form.Signature)
		if err != nil && asymkey_model.IsErrSSHInvalidTokenSignature(err) {
			fingerprint, err = asymkey_model.VerifySSHKey(ctx.Doer.ID, form.Fingerprint, lastToken, form.Signature)
		}
		if err != nil {
			ctx.Data["HasSSHVerifyError"] = true
			switch {
			case asymkey_model.IsErrSSHInvalidTokenSignature(err):
				loadKeysData(ctx)
				ctx.Data["Err_Signature"] = true
				ctx.Data["Fingerprint"] = err.(asymkey_model.ErrSSHInvalidTokenSignature).Fingerprint
				ctx.RenderWithErr(ctx.Tr("settings.ssh_invalid_token_signature"), tplSettingsKeys, &form)
			default:
				ctx.ServerError("VerifySSH", err)
			}
		}
		ctx.Flash.Success(ctx.Tr("settings.verify_ssh_key_success", fingerprint))
		ctx.Redirect(setting.AppSubURL + "/user/settings/keys")

	default:
		ctx.Flash.Warning("Function not implemented")
		ctx.Redirect(setting.AppSubURL + "/user/settings/keys")
	}
}

// DeleteKey response for delete user's SSH/GPG key
func DeleteKey(ctx *context.Context) {
	auditParams := map[string]string{
		"email": ctx.Doer.Email,
	}
	switch ctx.FormString("type") {
	case "gpg":
		if err := asymkey_model.DeleteGPGKey(ctx.Doer, ctx.FormInt64("id")); err != nil {
			ctx.Flash.Error("DeleteGPGKey: " + err.Error())
			auditParams["error"] = "Error has occurred while deleting GPG key"
			audit.CreateAndSendEvent(audit.GPGKeyRemoveEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		} else {
			ctx.Flash.Success(ctx.Tr("settings.gpg_key_deletion_success"))
			audit.CreateAndSendEvent(audit.GPGKeyRemoveEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)
		}
	case "ssh":
		keyID := ctx.FormInt64("id")
		external, err := asymkey_model.PublicKeyIsExternallyManaged(keyID)
		if err != nil {
			ctx.ServerError("sshKeysExternalManaged", err)
			auditParams["error"] = "Error has occurred while checking public key is externally managed"
			audit.CreateAndSendEvent(audit.SSHKeyRemoveEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}
		if external {
			ctx.Flash.Error(ctx.Tr("settings.ssh_externally_managed"))
			ctx.Redirect(setting.AppSubURL + "/user/settings/keys")
			auditParams["error"] = "SSH key externally managed"
			audit.CreateAndSendEvent(audit.SSHKeyRemoveEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}
		if err := asymkey_service.DeletePublicKey(ctx.Doer, keyID); err != nil {
			ctx.Flash.Error("DeletePublicKey: " + err.Error())
			auditParams["error"] = "Error has occurred while deleting SSH key"
			audit.CreateAndSendEvent(audit.SSHKeyRemoveEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		} else {
			ctx.Flash.Success(ctx.Tr("settings.ssh_key_deletion_success"))
			audit.CreateAndSendEvent(audit.SSHKeyRemoveEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)
		}
	case "principal":
		if err := asymkey_service.DeletePublicKey(ctx.Doer, ctx.FormInt64("id")); err != nil {
			ctx.Flash.Error("DeletePublicKey: " + err.Error())
		} else {
			ctx.Flash.Success(ctx.Tr("settings.ssh_principal_deletion_success"))
		}
	default:
		ctx.Flash.Warning("Function not implemented")
		ctx.Redirect(setting.AppSubURL + "/user/settings/keys")
	}
	ctx.JSON(http.StatusOK, map[string]interface{}{
		"redirect": setting.AppSubURL + "/user/settings/keys",
	})
}

func loadKeysData(ctx *context.Context) {
	keys, err := asymkey_model.ListPublicKeys(ctx.Doer.ID, db.ListOptions{})
	if err != nil {
		ctx.ServerError("ListPublicKeys", err)
		return
	}
	ctx.Data["Keys"] = keys

	externalKeys, err := asymkey_model.PublicKeysAreExternallyManaged(keys)
	if err != nil {
		ctx.ServerError("ListPublicKeys", err)
		return
	}
	ctx.Data["ExternalKeys"] = externalKeys

	gpgkeys, err := asymkey_model.ListGPGKeys(ctx, ctx.Doer.ID, db.ListOptions{})
	if err != nil {
		ctx.ServerError("ListGPGKeys", err)
		return
	}
	ctx.Data["GPGKeys"] = gpgkeys
	tokenToSign := asymkey_model.VerificationToken(ctx.Doer, 1)

	// generate a new aes cipher using the csrfToken
	ctx.Data["TokenToSign"] = tokenToSign

	principals, err := asymkey_model.ListPrincipalKeys(ctx.Doer.ID, db.ListOptions{})
	if err != nil {
		ctx.ServerError("ListPrincipalKeys", err)
		return
	}
	ctx.Data["Principals"] = principals

	ctx.Data["VerifyingID"] = ctx.FormString("verify_gpg")
	ctx.Data["VerifyingFingerprint"] = ctx.FormString("verify_ssh")
}
