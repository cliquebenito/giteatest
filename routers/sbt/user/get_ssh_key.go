package user

import (
	asymkey_model "code.gitea.io/gitea/models/asymkey"
	"code.gitea.io/gitea/modules/context"
	apiError "code.gitea.io/gitea/routers/sbt/apierror"
	"code.gitea.io/gitea/routers/sbt/logger"
	"code.gitea.io/gitea/routers/sbt/response"
	"net/http"
)

/*
GetUserSshKeyById метод, который возвращает информацию о публичном SSH ключе пользователя по идентификатору ключа
*/
func GetUserSshKeyById(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	keyId := ctx.ParamsInt64(":keyId")

	key, err := asymkey_model.GetPublicKeyByIDAndOwnerId(keyId, ctx.Doer.ID)
	if err != nil {
		if asymkey_model.IsErrKeyNotExist(err) {
			log.Debug("Public SSH key for user with username: %s and keyId: %d not exist", ctx.Doer.Name, keyId)
			ctx.JSON(http.StatusBadRequest, apiError.SshKeyNotExist(keyId))

		} else {
			log.Error("Error has occurred while getting public SSH key for user with username: %s and keyId: %d. Error message: %v", ctx.Doer.Name, keyId, err)
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		}

		return
	}

	ctx.JSON(http.StatusOK, response.PublicSshKeyMapper(key))
}
