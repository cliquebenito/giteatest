package user

import (
	asymkey_model "code.gitea.io/gitea/models/asymkey"
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/web"
	apiError "code.gitea.io/gitea/routers/sbt/apierror"
	"code.gitea.io/gitea/routers/sbt/logger"
	"code.gitea.io/gitea/routers/sbt/request"
	"code.gitea.io/gitea/routers/sbt/response"
	"net/http"
)

/*
CreateSshKey метод создания публичного ssh-ключа в настройках пользователя
*/
func CreateSshKey(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	req := web.GetForm(ctx).(*request.CreateUserSshKey)

	content, err := asymkey_model.CheckPublicKeyString(req.Key)
	if err != nil {
		handlerErrorCheckPublicKey(ctx, err, log)

		return
	}

	key, err := asymkey_model.AddPublicKey(ctx.Doer.ID, req.Title, content, 0)
	if err != nil {
		handlerErrorAddKey(ctx, err, req.Title, content, log)

		return
	}

	ctx.JSON(http.StatusCreated, response.PublicSshKeyMapper(key))
}

func handlerErrorCheckPublicKey(ctx *context.Context, err error, log logger.Logger) {
	var errorMessage string

	if db.IsErrSSHDisabled(err) {
		errorMessage = "SSH key is disabled."
	} else if asymkey_model.IsErrKeyUnableVerify(err) {
		errorMessage = "Unable to verify key content."
	} else if err == asymkey_model.ErrKeyIsPrivate {
		errorMessage = "Use public SSH key instead private."
	} else {
		errorMessage = "Invalid public SSH key."
	}

	log.Debug("Error has occurred while checking new public SSH key by username: %s, error: %v", ctx.Doer.Name, err)
	ctx.JSON(http.StatusBadRequest, apiError.InvalidSshKey(errorMessage))
}

func handlerErrorAddKey(ctx *context.Context, err error, title string, content string, log logger.Logger) {
	switch {
	case asymkey_model.IsErrKeyAlreadyExist(err):
		log.Debug("SSH key has already been added to the server: %s", content)
		ctx.JSON(http.StatusBadRequest, apiError.SshKeyAlreadyExist())
	case asymkey_model.IsErrKeyNameAlreadyUsed(err):
		log.Debug("Key title has been used: %s", title)
		ctx.JSON(http.StatusBadRequest, apiError.SshKeyNameAlreadyExist())
	case asymkey_model.IsErrKeyUnableVerify(err):
		log.Debug("Cannot verify the SSH key, double-check it for mistakes: %v", err)
		ctx.JSON(http.StatusBadRequest, apiError.SshKeyUnableVerify())
	default:
		log.Error("Unknown error type has occurred: %v", err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
	}
}
