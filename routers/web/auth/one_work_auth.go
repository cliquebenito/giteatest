package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	vault_model "code.gitea.io/gitea/models/vault_client"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
)

// tokenValidityDuration время валидности токена
var tokenValidityDuration = 60 * time.Second

// AuthForOneWork функция для проверки доступа к urls по one_work_token
func AuthForOneWork(ctx *context.Context) {
	tokenFromAuth := ctx.Req.Header.Get("Authorization")
	if len(tokenFromAuth) == 0 {
		log.Error("AuthForOneWork failed: because tokenFromAuth is empty")
		ctx.Error(http.StatusForbidden, "Header Authorization is empty")
		return
	}
	tokenBearer := strings.Split(strings.TrimPrefix(tokenFromAuth, "Bearer "), ".")
	if len(tokenBearer) < 2 {
		log.Error("AuthForOneWork failed: because len(tokenBearer) < 2")
		ctx.Error(404, "Incorrect auth token")
		return
	}
	// message время в unix формате
	message := tokenBearer[1]
	if !checkTimeToLifeFromToken(message) {
		log.Error("AuthForOneWork checkTimeToLifeFromToken failed: token is expired")
		ctx.Error(http.StatusForbidden, "Token is expired")
		return
	}

	// configForKvGet создаем переменную для запроса в sec man, для получения token
	configForKvGet := &vault_model.KeyValueConfigForGetSecrets{
		SecretPath:  strings.TrimSpace(setting.SourceControlOneWorkToken.SecretPath),
		StoragePath: strings.TrimSpace(setting.SourceControlOneWorkToken.StoragePath),
		VersionKey:  setting.SourceControlOneWorkToken.VersionKey,
	}
	tokenValidity := setting.NewGetterForSecMan()
	resp, err := tokenValidity.GetCredFromSecManByVersionKey(configForKvGet)
	if err != nil {
		log.Debug("AuthForOneWork setting.GetCredFromSecManByVersionKey failed: %v", err)
		ctx.ServerError("AuthForOneWork GetCredFromSecManByVersionKey", err)
		return
	}

	var tokenSecMan string
	if setting.GetResponseNotNil(resp) {
		tokenSecMan = resp.Data[setting.SourceControlOneWorkToken.OneWorkToken]
		setting.CheckIfSecretIsEmptyAndReportToAudit("ONE_WORK_TOKEN", tokenSecMan, "ONE_WORK_TOKEN is empty in secret storage", log.Fatal)
	} else {
		setting.CheckIfSecretIsEmptyAndReportToAudit("ONE_WORK_TOKEN", "", "Response from secret storage is nil", log.Fatal)
	}

	generateToken := generateTokenByParams(message, tokenSecMan)
	if tokenFromAuth != generateToken {
		log.Debug("AuthForOneWork failed: generateToken hasn't matched with tokenFromAuth")
		ctx.ServerError("AuthForOneWork failed", fmt.Errorf("AuthForOneWork failed: generateToken hasn't matched with tokenFromAuth"))
		return
	}
}

// generateTokenByParams хэшируемый token из sec man и unix time из header
func generateTokenByParams(message, token string) string {
	signature := hmacSign([]byte(token), message)
	return "Bearer " + fmt.Sprintf("%x.%s", signature, message)
}

// checkTimeToLifeFromToken проверяем время валидности token
func checkTimeToLifeFromToken(message string) bool {
	timestamp, err := strconv.ParseInt(message, 10, 64)
	if err != nil {
		log.Error("AuthForOneWork checkTimeToLifeFromToken failed: %v", err)
		return false
	}
	issuedAt := time.Unix(timestamp, 0)
	targetTime := time.Now()
	lowerBound := targetTime.Add(-tokenValidityDuration)
	upperBound := targetTime.Add(tokenValidityDuration)

	if issuedAt.Before(lowerBound) {
		log.Error("AuthForOneWork checkTimeToLifeFromToken failed: time token is expired")
		return false
	}
	if issuedAt.After(upperBound) {
		log.Error("AuthForOneWork checkTimeToLifeFromToken failed: time token is exceeded")
		return false
	}
	return true
}

// hmacSign генерируем token в slice byte
func hmacSign(secret []byte, message string) []byte {
	mac := hmac.New(sha256.New, secret)
	// hash.Hash never returns an error.
	_, _ = mac.Write([]byte(message))
	return mac.Sum(nil)
}
