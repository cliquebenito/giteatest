package user

import (
	"code.gitea.io/gitea/models/db"
	userModel "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/sbt/auth/password"
	"code.gitea.io/gitea/modules/session"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/web"
	apiError "code.gitea.io/gitea/routers/sbt/apierror"
	"code.gitea.io/gitea/routers/sbt/logger"
	"code.gitea.io/gitea/routers/sbt/request"
	"fmt"
	"net/http"
	"strings"
)

/*
PostCreateUser метод регистрации пользователя.
В случае ошибки возвращается BadRequest (400) и стандартной API ошибкой,
в случае успешного создания пользователя Created (201) и JSON с именем и почтой пользователя
*/
func PostCreateUser(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	req := web.GetForm(ctx).(*request.RegisterUser)

	if !req.IsEmailDomainAllowed() {
		log.Debug("Email: %s is wrong "+
			"(contains in domain blocklist or not contained in domain whitelist, only if such list is not empty in app.ini)", req.Email)

		ctx.JSON(http.StatusBadRequest, apiError.WrongEmailError(req.Email))

		return
	}

	if len(req.Password) < setting.MinPasswordLength {
		log.Debug("Short password in request, required password length: %d", setting.MinPasswordLength)

		ctx.JSON(http.StatusBadRequest, apiError.WrongPasswordError(fmt.Sprintf("%d or more symbols", setting.MinPasswordLength)))

		return
	}

	if ok, errorMessage := password.IsComplexEnough(req.Password); !ok {
		log.Debug("Wrong password in request, required %s", errorMessage)

		ctx.JSON(http.StatusBadRequest, apiError.WrongPasswordError(errorMessage))

		return
	}

	u := &userModel.User{
		Name:   req.UserName,
		Email:  strings.ToLower(req.Email),
		Passwd: req.Password,
	}

	if !createAndHandleCreatedUser(ctx, u, log) {
		return
	}

	handleSignIn(ctx, u, log)

	ctx.JSON(http.StatusCreated, map[string]string{
		"username": req.UserName,
		"email":    req.Email,
	})
}

/*
createAndHandleCreatedUser метод создания пользователя
*/
func createAndHandleCreatedUser(ctx *context.Context, u *userModel.User, log logger.Logger) bool {
	if !createUserInContext(ctx, u) {
		return false
	}
	return handleUserCreated(ctx, u, log)
}

func createUserInContext(ctx *context.Context, u *userModel.User) (ok bool) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	if err := userModel.SbtCreateUser(u); err != nil {
		if userModel.IsErrUserAlreadyExist(err) || userModel.IsErrEmailAlreadyUsed(err) {
			var user *userModel.User
			user = &userModel.User{Name: u.Name}
			hasUser, err := userModel.GetUser(user)
			if !hasUser || err != nil {
				log.Debug("User not found by name: %s", u.Name)

				user = &userModel.User{Email: u.Email}
				hasUser, err = userModel.GetUser(user)
				if !hasUser || err != nil {
					log.Debug("User not found by email: %s", u.Email)

					log.Error("Internal server error has occurred while getting user from database by name: %s and email: %s. Error: %v", u.Name, u.Email, err)

					ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())

					return
				}
			}
		}

		switch {
		case userModel.IsErrUserAlreadyExist(err):
			log.Debug("User with name: %s already exist", u.Name)
			ctx.JSON(http.StatusBadRequest, apiError.UserNameAlreadyExistError(u.Name))

		case userModel.IsErrEmailAlreadyUsed(err):
			log.Debug("Email: %s already exist", u.Email)
			ctx.JSON(http.StatusBadRequest, apiError.EmailAlreadyExistError(u.Email))

		case userModel.IsErrEmailCharIsNotSupported(err):
			log.Debug("Email: %s contains unsupported characters", u.Email)
			ctx.JSON(http.StatusBadRequest, apiError.EmailContainsUnsupportedCharsError(u.Email))

		case userModel.IsErrEmailInvalid(err):
			log.Debug("Email: %s is invalid", u.Email)
			ctx.JSON(http.StatusBadRequest, apiError.EmailInvalidError(u.Email))

		case db.IsErrNameReserved(err):
			log.Debug("Username: %s is reserved", u.Name)
			ctx.JSON(http.StatusBadRequest, apiError.UserNameReservedError(u.Name))

		case db.IsErrNamePatternNotAllowed(err):
			log.Debug("Username: %s pattern is not allowed", u.Name)
			ctx.JSON(http.StatusBadRequest, apiError.UserNamePatternNotAllowedError(u.Name))

		case db.IsErrNameCharsNotAllowed(err):
			log.Debug("Username: %s has not allowed characters", u.Name)
			ctx.JSON(http.StatusBadRequest, apiError.UserNameHasNotAllowedCharsError(u.Name))

		case userModel.IsErrKeycloakWrongHttpRequest(err):
			log.Debug("User with username: %s was not registered in Keycloak because of wrong http request. Error: %s", u.Name, err.(userModel.ErrKeycloakWrongHttpRequest).ReasonErr)
			ctx.JSON(http.StatusBadRequest, apiError.KeycloakUserWasNotRegistered)

		case userModel.IsErrKeycloakWrongHttpStatus(err):
			if err.(userModel.ErrKeycloakWrongHttpStatus).StatusCode == http.StatusConflict {
				log.Debug("User with username: %s was not registered in Keycloak because username or email already exist. Error: %s", u.Name, err.(userModel.ErrKeycloakWrongHttpRequest).ReasonErr)
				ctx.JSON(http.StatusBadRequest, apiError.KeycloakUserAlreadyExist)
			} else {
				log.Debug("User with username: %s was not registered in Keycloak. Error: %v", u.Name, err)
				ctx.JSON(http.StatusBadRequest, apiError.KeycloakUserWasNotRegistered)
			}
		default:
			log.Error("Unknown error type has occurred: %v", err)
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		}

		return
	}

	log.Debug("Account created: %s", u.Name)

	return true
}

func handleUserCreated(ctx *context.Context, u *userModel.User, log logger.Logger) (ok bool) {
	// Auto-set admin for the only user.
	if userModel.CountUsers(nil) == 1 {
		u.IsAdmin = true
		u.IsActive = true
		u.SetLastLogin()
		if err := userModel.UpdateUserCols(ctx, u, "is_admin", "is_active", "last_login_unix"); err != nil {
			log.Error("Error has occurred: %v", err)
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())

			return
		}
	}

	return true
}

/*
handleSignIn метод авторизации пользователя:
- обновление сессии
- обновление времени последнего логина
*/
func handleSignIn(ctx *context.Context, u *userModel.User, log logger.Logger) {
	if err := updateSession(ctx, []string{
		// Delete the openid, 2fa and linkaccount data
		"openid_verified_uri",
		"openid_signin_remember",
		"openid_determined_email",
		"openid_determined_username",
		"twofaUid",
		"twofaRemember",
		"linkAccount",
	}, map[string]interface{}{
		"uid":   u.ID,
		"uname": u.Name,
	}); err != nil {
		log.Error("Error has occurred: %v", err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
	}

	u.SetLastLogin()
	if err := userModel.UpdateUserCols(ctx, u, "last_login_unix"); err != nil {
		log.Error("Error has occurred: %v", err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
	}
}

func updateSession(ctx *context.Context, deletes []string, updates map[string]interface{}) error {
	if _, err := session.RegenerateSession(ctx.Resp, ctx.Req); err != nil {
		return fmt.Errorf("regenerate session: %w", err)
	}
	sess := ctx.Session
	sessID := sess.ID()
	for _, k := range deletes {
		if err := sess.Delete(k); err != nil {

			return fmt.Errorf("delete %v in session[%s]: %w", k, sessID, err)
		}
	}
	for k, v := range updates {
		if err := sess.Set(k, v); err != nil {

			return fmt.Errorf("set %v in session[%s]: %w", k, sessID, err)
		}
	}
	if err := sess.Release(); err != nil {

		return fmt.Errorf("store session[%s]: %w", sessID, err)
	}

	return nil
}
