package user

import (
	asymkey_model "code.gitea.io/gitea/models/asymkey"
	"code.gitea.io/gitea/modules/context"
	apiError "code.gitea.io/gitea/routers/sbt/apierror"
	"code.gitea.io/gitea/routers/sbt/logger"
	asymkey_service "code.gitea.io/gitea/services/asymkey"
	"net/http"
)

/*
DeleteUserSshKeyById метод удаления публичного ssh кчлюча
*/
func DeleteUserSshKeyById(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	keyId := ctx.ParamsInt64(":keyId")

	externallyManaged, err := asymkey_model.PublicKeyIsExternallyManaged(keyId)
	if err != nil {
		if asymkey_model.IsErrKeyNotExist(err) {
			log.Debug("Public SSH key for user with username: %s and keyId: %d not exist", ctx.Doer.Name, keyId)
			ctx.JSON(http.StatusBadRequest, apiError.SshKeyNotExist(keyId))
		} else {
			log.Error("Error has occurred while checking public SSH key with keyId: %d is externally managed error message: %v", keyId, err)
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		}
		return
	}

	if externallyManaged {
		log.Debug("Public SSH key with keyId: %d is externally managed for user with userName: %s", keyId, ctx.Doer.Name)
		ctx.JSON(http.StatusForbidden, apiError.SshKeyExternallyManaged())
		return
	}

	if err := asymkey_service.DeletePublicKey(ctx.Doer, keyId); err != nil {
		if asymkey_model.IsErrKeyAccessDenied(err) {
			log.Debug("User with userName: %s is not owner of SSH key with keyId: %d", ctx.Doer.Name, keyId)
			ctx.JSON(http.StatusForbidden, apiError.SshKeyUserIsNotOwner(keyId))
		} else {
			log.Error("Error has occurred while deleting public SSH key with keyId: %d error message: %v", keyId, err)
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		}
		return
	}

	ctx.Status(http.StatusOK)
}
