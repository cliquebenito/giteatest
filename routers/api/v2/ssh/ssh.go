package ssh

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"unicode/utf8"

	asymkey_model "code.gitea.io/gitea/models/asymkey"
	"code.gitea.io/gitea/models/db"
	user2 "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/sbt/audit"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/routers/api/v2/models"
	sshmodels "code.gitea.io/gitea/routers/api/v2/models/ssh"
)

// Create handler to create ssh key for user
func Create(ctx *context.APIContext) {
	//swagger:operation POST /admin/users/keys admin createSSHKey
	//---
	//summary: Create ssh key for user
	//consumes:
	//- application/json
	//produces:
	//- application/json
	//parameters:
	//- name: user_key
	//  in: query
	//  type: string
	//- name: body
	//  in: body
	//  description: Details of the ssh key to be created
	//  required: true
	//  schema:
	//    type: object
	//    required:
	//      - title
	//      - key
	//    properties:
	//      title:
	//        type: string
	//        description: Title of ssh key
	//      key:
	//        type: string
	//        description: Public key of ssh key
	//responses:
	//  "201":
	//    description: OK
	//    schema:
	//      type: object
	//      properties:
	//         id:
	//           type: integer
	//           format: int64
	//           description: Unique identifier of the ssh key
	//         fingerprint:
	//           type: string
	//           description: Fingerprint of the ssh key
	//         key_type:
	//           type: string
	//           description: Type of the ssh key
	//  "400":
	//    description: Bad request
	//  "401":
	//    description: Not authenticated
	//  "404":
	//    description: Not found
	//  "500":
	//    description: Internal server error

	auditParams := map[string]string{}
	userKey := models.ParseUserKeyGetOpt(ctx)

	req := web.GetForm(ctx).(*sshmodels.CreateSSHKeyRequest)
	if req == nil {
		log.Error("Error has occurred while getting form")
		ctx.Error(http.StatusBadRequest, "", "Err: Request incorrect")
		return
	}

	auditParams = map[string]string{
		"title": req.Title,
		"key":   req.Key,
	}

	if statusCode, err := validateCreateSSHKeyRequest(ctx, userKey, req, auditParams); err != nil {
		ctx.Error(statusCode, "", fmt.Sprintf("Err: %v", err))
		return
	}

	userInfo, err := user2.GetUserByLoginName(ctx, userKey)
	if err != nil {
		log.Error("Error has occurred while getting user: %v", err)
		if auditParams != nil {
			auditParams["error"] = "Error has occurred while getting user by user key"
			audit.CreateAndSendEvent(audit.SSHKeyAddEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		}
		ctx.Error(http.StatusNotFound, "", fmt.Sprintf("Err: user does not exist [userKey: %s]", userKey))
	}

	content, err := asymkey_model.CheckPublicKeyString(req.Key)
	if err != nil {
		handlerErrorCheckPublicKey(ctx, err, auditParams)
		return
	}

	key, err := asymkey_model.AddPublicKey(userInfo.ID, req.Title, content, 0)
	if err != nil {
		handlerErrorAddKey(ctx, err, req.Title, content, auditParams)
		return
	}

	audit.CreateAndSendEvent(audit.SSHKeyAddEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)
	ctx.JSON(http.StatusCreated, sshmodels.CreateSSHKeyResponse{
		ID:          key.ID,
		Fingerprint: key.Fingerprint,
		KeyType:     asymkey_model.KeyType_name[int(key.Type)],
	})
}

func validateCreateSSHKeyRequest(ctx *context.APIContext, userKey string, req *sshmodels.CreateSSHKeyRequest, auditParams map[string]string) (statusCode int, err error) {
	if userKey == "" {
		log.Error("Error has occurred while getting user key: user key is empty")
		if auditParams != nil {
			auditParams["error"] = "Error has occurred while getting user key: user key is empty"
			audit.CreateAndSendEvent(audit.SSHKeyAddEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		}
		return http.StatusBadRequest, fmt.Errorf("user key is required")
	}

	if req.Title == "" {
		log.Error("Error has occurred while getting ssh key title: title is empty")
		if auditParams != nil {
			auditParams["error"] = "Error has occurred while getting ssh key title: title is empty"
			audit.CreateAndSendEvent(audit.SSHKeyAddEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		}
		return http.StatusBadRequest, fmt.Errorf("ssh key title is required")
	}

	if utf8.RuneCountInString(req.Title) > 50 {
		log.Error("Error has occurred while getting ssh key title: title is too long")
		if auditParams != nil {
			auditParams["error"] = "Error has occurred while getting ssh key title: title is too long"
			audit.CreateAndSendEvent(audit.SSHKeyAddEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		}
		return http.StatusBadRequest, fmt.Errorf("ssh key title is too long")
	}

	if req.Key == "" {
		log.Error("Error has occurred while getting ssh key: key is empty")
		if auditParams != nil {
			auditParams["error"] = "Error has occurred while getting ssh key: key is empty"
			audit.CreateAndSendEvent(audit.SSHKeyAddEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		}
		return http.StatusBadRequest, fmt.Errorf("ssh key is required")
	}

	if ctx != nil && ctx.Doer != nil {
		if !ctx.Doer.IsAdmin {
			if auditParams != nil {
				auditParams["error"] = "Error has occurred while adding ssh key: user must be admin"
				audit.CreateAndSendEvent(audit.SSHKeyAddEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			}
			return http.StatusBadRequest, fmt.Errorf("user must be admin")
		}
	}

	return 0, nil
}

func handlerErrorCheckPublicKey(ctx *context.APIContext, err error, auditParams map[string]string) {
	var errorMessage string
	switch {
	case db.IsErrSSHDisabled(err):
		errorMessage = "SSH key is disabled."
		if auditParams != nil {
			auditParams["error"] = "Error has occurred while getting ssh key: ssh disabled"
		}
	case asymkey_model.IsErrInvalidKeyFormat(err) || asymkey_model.IsErrInvalidFormatOrLength(err) ||
		asymkey_model.IsErrInvalidKeyLine(err):
		errorMessage = "Invalid format of ssh key"
		if auditParams != nil {
			auditParams["error"] = "Error has occurred while getting ssh key: invalid format of ssh key"
		}
	case asymkey_model.IsErrInvalidKeyType(err) || asymkey_model.IsErrNotAllowedKeyType(err):
		errorMessage = "Unsupported key type."
		if auditParams != nil {
			auditParams["error"] = "Error has occurred while getting ssh key: unsupported key type"
		}
	case asymkey_model.IsErrInvalidKeyLength(err):
		errorMessage = "Invalid length of ssh key"
		if auditParams != nil {
			auditParams["error"] = "Error has occurred while getting ssh key: invalid length of ssh key"
		}
	case asymkey_model.IsErrInvalidPEMBlock(err):
		errorMessage = "Invalid PEM block"
		if auditParams != nil {
			auditParams["error"] = "Error has occurred while getting ssh key: invalid PEM block"
		}
	case asymkey_model.IsErrKeyUnableVerify(err):
		errorMessage = "Unable to verify key content."
		if auditParams != nil {
			auditParams["error"] = "Error has occurred while getting ssh key: unable to verify key content"
		}
	case errors.Is(err, asymkey_model.ErrKeyIsPrivate):
		errorMessage = "Use public SSH key instead private."
		if auditParams != nil {
			auditParams["error"] = "Error has occurred while getting ssh key: use public SSH key instead private"
		}
	default:
		errorMessage = "Invalid public SSH key."
		if auditParams != nil {
			auditParams["error"] = "Error has occurred while getting ssh key: invalid public SSH key"
		}
	}

	log.Debug("Error has occurred while checking new public SSH key by username: %s, error: %v", ctx.Doer.Name, err)
	audit.CreateAndSendEvent(audit.SSHKeyAddEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
	ctx.Error(http.StatusBadRequest, "", fmt.Sprintf("Err: %s", errorMessage))
}

func handlerErrorAddKey(ctx *context.APIContext, err error, title string, content string, auditParams map[string]string) {
	switch {
	case asymkey_model.IsErrKeyAlreadyExist(err):
		log.Debug("SSH key has already been added to the server: %s", content)
		if auditParams != nil {
			auditParams["error"] = "Error has occurred while adding ssh key: ssh key already exist"
		}
		ctx.Error(http.StatusBadRequest, "", "Err: SSH key already exist")
	case asymkey_model.IsErrKeyNameAlreadyUsed(err):
		log.Debug("Key title has been used: %s", title)
		if auditParams != nil {
			auditParams["error"] = "Error has occurred while adding ssh key: ssh key title already used"
		}
		ctx.Error(http.StatusBadRequest, "", "Err: SSH key title already exist")
	case asymkey_model.IsErrKeyUnableVerify(err):
		log.Debug("Cannot verify the SSH key, double-check it for mistakes: %v", err)
		if auditParams != nil {
			auditParams["error"] = "Error has occurred while adding ssh key: cannot verify the SSH key, double-check it for mistakes"
		}
		ctx.Error(http.StatusBadRequest, "", "Err: Can not verify the SSH key, double-check it for mistake")
	default:
		log.Error("Unknown error type has occurred: %v", err)
		if auditParams != nil {
			auditParams["error"] = "Error has occurred while adding ssh key"
		}
		ctx.Error(http.StatusInternalServerError, "", "Err: Internal Server Error")
	}

	audit.CreateAndSendEvent(audit.SSHKeyAddEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
}

// Delete handler for delete ssh key
func Delete(ctx *context.APIContext) {
	// swagger:operation DELETE /admin/users/keys admin deleteSSHKey
	// ---
	// summary: Delete SSH key for user
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: user_key
	//   in: query
	//   type: string
	// - name: title
	//   in: query
	//   type: string
	// responses:
	//   "200":
	//     description: OK
	//   "400":
	//     description: Bad request
	//   "401":
	//     description: Not authenticated
	//   "404":
	//     description: Not found
	//   "500":
	//     description: Internal server error

	sshKey := models.ParseSSHKeyGetOpts(ctx)
	auditParams := map[string]string{
		"user_key": sshKey.UserKey,
		"title":    sshKey.Title,
	}

	if statusCode, err := validateDeleteSSHKeyRequest(ctx, sshKey, auditParams); err != nil {
		ctx.Error(statusCode, "", fmt.Sprintf("Err: %v", err))
		return
	}

	userInfo, err := user2.GetUserByLoginName(ctx, sshKey.UserKey)
	if err != nil {
		log.Error("Error has occurred while getting user: %v", err)
		if auditParams != nil {
			auditParams["error"] = "Error has occurred while getting user by user key"
			audit.CreateAndSendEvent(audit.SSHKeyRemoveEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		}
		ctx.Error(http.StatusNotFound, "", fmt.Sprintf("Err: user does not exist [userKey: %s]", sshKey.UserKey))
		return
	}

	key, err := asymkey_model.GetSSHKeyByUserAndTitle(userInfo.ID, sshKey.Title)
	if err != nil {
		log.Error("Error has occurred while getting ssh key by user id and title: %v", err)
		if asymkey_model.IsErrKeyNotExist(err) {
			if auditParams != nil {
				auditParams["error"] = "Error has occurred while getting ssh key by user id and title"
				audit.CreateAndSendEvent(audit.SSHKeyRemoveEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			}
			ctx.Error(http.StatusNotFound, "", fmt.Sprintf("Err: ssh key does not exist [userID: %d], [sshKeyTitle: %s]", userInfo.ID, sshKey.Title))
			return
		}

		auditParams["error"] = "Error has occurred while getting ssh key by user id and title"
		audit.CreateAndSendEvent(audit.SSHKeyRemoveEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		ctx.Error(http.StatusInternalServerError, "", "Err: Internal Server Error")
		return
	}

	if err := deleteSSHKey(key); err != nil {
		auditParams["error"] = "Error has occurred while deleting ssh key"
		audit.CreateAndSendEvent(audit.SSHKeyRemoveEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		log.Error("Error has occurred while deleting ssh key: %v", err)
		ctx.Error(http.StatusInternalServerError, "", "Err: Internal Server Error")
		return
	}

	audit.CreateAndSendEvent(audit.SSHKeyRemoveEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)
	ctx.Status(http.StatusOK)
}

func validateDeleteSSHKeyRequest(ctx *context.APIContext, opts sshmodels.SSHKeyOptions, auditParams map[string]string) (statusCode int, err error) {
	if opts.UserKey == "" {
		log.Error("Error has occurred while getting user key: user key is empty")
		if auditParams != nil {
			auditParams["error"] = "Error has occurred while getting user key: user key is empty"
			audit.CreateAndSendEvent(audit.SSHKeyRemoveEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		}
		return http.StatusBadRequest, fmt.Errorf("user key is required")
	}

	if opts.Title == "" {
		log.Error("Error has occurred while getting ssh key title: title is empty")
		if auditParams != nil {
			auditParams["error"] = "Error has occurred while getting ssh key title: title is empty"
			audit.CreateAndSendEvent(audit.SSHKeyRemoveEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		}
		return http.StatusBadRequest, fmt.Errorf("ssh key title is required")
	}

	if ctx != nil && ctx.Doer != nil {
		if !ctx.Doer.IsAdmin {
			if auditParams != nil {
				auditParams["error"] = "Error has occurred while deleting ssh key: user must be admin"
				audit.CreateAndSendEvent(audit.SSHKeyRemoveEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			}
			return http.StatusBadRequest, fmt.Errorf("user must be admin")
		}
	}

	return 0, nil
}

// deleteSSHKey deletes SSH key information both in database and authorized_keys file.
func deleteSSHKey(key *asymkey_model.PublicKey) (err error) {
	ctx, committer, err := db.TxContext(db.DefaultContext)
	if err != nil {
		return err
	}
	defer committer.Close()

	if err = asymkey_model.DeletePublicKeys(ctx, key.ID); err != nil {
		return err
	}

	if err = committer.Commit(); err != nil {
		return err
	}

	if key.Type == asymkey_model.KeyTypePrincipal {
		return asymkey_model.RewriteAllPrincipalKeys(db.DefaultContext)
	}

	return asymkey_model.RewriteAllPublicKeys()
}
