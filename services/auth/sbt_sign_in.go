package auth

import (
	"code.gitea.io/gitea/models/auth"
	"code.gitea.io/gitea/models/db"
	userModel "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/services/auth/source/oauth2"
	"code.gitea.io/gitea/services/auth/source/smtp"
	"net/http"
	"strings"
)

// SbtSignInUser метод аутентификации пользователя по БД или keycloak. Метод написан по аналогии UserSignIn
func SbtSignInUser(username string, password string, ctx *context.Context) (*userModel.User, error) {
	var user *userModel.User
	isEmail := false

	//Проверяем пользователя на наличие в БД сначала по адресу электронной почты, потом по юзернейму
	if strings.Contains(username, "@") {
		isEmail = true
		emailAddress := userModel.EmailAddress{LowerEmail: strings.ToLower(strings.TrimSpace(username))}
		// check same email
		has, err := db.GetEngine(db.DefaultContext).Get(&emailAddress)
		if err != nil {
			return nil, err
		}
		if has {
			if !emailAddress.IsActivated {
				return nil, userModel.ErrEmailAddressNotExist{Email: username}
			}
			user = &userModel.User{ID: emailAddress.UID}
		}
	} else {
		trimmedUsername := strings.TrimSpace(username)
		if len(trimmedUsername) == 0 {
			return nil, userModel.ErrUserNotExist{Name: username}
		}

		user = &userModel.User{LowerName: strings.ToLower(trimmedUsername)}
	}

	if user != nil {
		hasUser, err := userModel.GetUser(user)
		if err != nil {
			return nil, err
		}

		if hasUser {
			var userFromDb *userModel.User

			if setting.SbtKeycloakForm.Enabled {
				userFromDb, err = authenticateByKeycloak(ctx, isEmail, user, username, password)
			} else {
				userFromDb, err = authenticateBySource(user, password)
			}
			if err != nil {
				return nil, err
			}

			if userFromDb.ProhibitLogin {
				return nil, userModel.ErrUserProhibitLogin{UID: userFromDb.ID, Name: userFromDb.Name}
			}

			return userFromDb, nil
		}
	}

	if isEmail {
		return nil, userModel.ErrEmailAddressNotExist{Email: username}
	}

	return nil, userModel.ErrUserNotExist{Name: username}
}

// authenticateByKeycloak метод аутентификации пользователя в Keycloak
func authenticateByKeycloak(ctx *context.Context, isEmail bool, user *userModel.User, username string, password string) (userFromDb *userModel.User, err error) {
	//Получение токена из Keycloak
	token, err := userModel.GetUserTokenFromKeycloak(username, password)

	if err != nil {
		// Если Keycloak возвращает статус Unauthorized, пробуем аутентифицировать пользователя по базе данных
		// Если пользователь есть в базе данных, то мы его регистрируем в Keycloak, после чего получаем токен из Keycloak
		if userModel.IsErrKeycloakWrongHttpStatus(err) && err.(userModel.ErrKeycloakWrongHttpStatus).StatusCode == http.StatusUnauthorized {
			u, err2 := authenticateBySource(user, password)
			if err2 == nil {
				if err := userModel.CreateUserInKeycloak(u, password); err != nil {
					return nil, err
				}
				token, err = userModel.GetUserTokenFromKeycloak(username, password)
			} else {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	// Если пользователь найден в Keycloak тогда получаем токен.
	// Сохраняем Refresh токен в сессии для завершения Keycloak сессии (используем в методе LogoutUser())
	err = ctx.Session.Set(userModel.RefTokenKey, token.RefreshToken)
	if err != nil {
		return nil, err
	}

	// Получаем пользователя из базы данных
	if isEmail == true && user.Email != "" {
		userFromDb, err = userModel.GetUserByEmail(ctx, user.Email)
	} else {
		userFromDb, err = userModel.GetUserByName(ctx, user.LowerName)
	}
	if err != nil {
		return nil, err
	}
	return userFromDb, nil
}

// authenticateBySource метод аутентификации пользователя по типу LoginSource (по умолчанию это БД)
func authenticateBySource(user *userModel.User, password string) (userFromDb *userModel.User, err error) {
	source, err := auth.GetSourceByID(user.LoginSource)
	if err != nil {
		return nil, err
	}

	if !source.IsActive {
		return nil, oauth2.ErrAuthSourceNotActived
	}

	authenticator, ok := source.Cfg.(PasswordAuthenticator)
	if !ok {
		return nil, smtp.ErrUnsupportedLoginType
	}

	userFromDb, err = authenticator.Authenticate(user, user.LoginName, password)
	if err != nil {
		return nil, err
	}

	return
}
