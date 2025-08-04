package user

import (
	asymkey_model "code.gitea.io/gitea/models/asymkey"
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/modules/context"
	apiError "code.gitea.io/gitea/routers/sbt/apierror"
	"code.gitea.io/gitea/routers/sbt/logger"
	"code.gitea.io/gitea/routers/sbt/response"
	"net/http"
)

/*
GetListSshKey метод получения списка публичных SSH ключей для авторизованного пользователя
db.ListOptions{} - пустая струткура, на данный момент мы выводим список всех ключей пользователя без пагинации
*/
func GetListSshKey(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	keys, err := asymkey_model.ListPublicKeys(ctx.Doer.ID, db.ListOptions{})
	if err != nil {
		log.Error("Error has occurred while getting list of public SSH key for user with username: %s and error message: %v", ctx.Doer.Name, err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())

		return
	}

	list := make([]response.PublicSshKey, 0, len(keys))
	for _, key := range keys {
		list = append(list, response.PublicSshKeyMapper(key))
	}

	ctx.JSON(http.StatusOK, list)

	return
}
