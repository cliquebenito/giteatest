package user

import (
	"bytes"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/translation"
	"context"
	"encoding/json"
	"fmt"
	"github.com/Nerzal/gocloak/v13"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const RefTokenKey = "ref_token"

// KeycloakUser Структура JSON запроса на создание пользователя Keycloak
type KeycloakUser struct {
	Enabled       bool                  `json:"enabled"`
	EmailVerified bool                  `json:"emailVerified"`
	UserName      string                `json:"username"`
	Email         string                `json:"email"`
	Credentials   []KeycloakCredentials `json:"credentials"`
	LastName      string                `json:"lastName"`
	FirstName     string                `json:"firstName"`
}

// KeycloakCredentials Структура реквизитов для аутентификации пользователя в Keycloak
type KeycloakCredentials struct {
	Type      string `json:"type"`
	Value     string `json:"value"`
	Temporary bool   `json:"temporary"`
}

// KeycloakUserToken структура информации о токене пользователя, получаемая из Keycloak
type KeycloakUserToken struct {
	AccessToken      string `json:"access_token"`
	RefreshToken     string `json:"refresh_token"`
	ExpiresIn        int    `json:"expires_in"`
	RefreshExpiresIn int    `json:"refresh_expires_in"`
	TokenType        string `json:"token_type"`
	SessionState     string `json:"session_state"`
}

const timeSkew = 5 // дельта в секундах для определения валидности токена

// accessTokenParams jwt токен и время его последнего создания
type accessTokenParams struct {
	jwt     *gocloak.JWT
	created time.Time // время его последнего создания, для определения его актуальности
}

// keycloakClient клиент keycloak и его параметры
type keycloakClient struct {
	sync.Mutex
	ctx    context.Context
	client *gocloak.GoCloak
	token  accessTokenParams
	realm  string
}

// searchUsers Поиск пользователя в keycloak по имени пользователя или его имени и фамилии (не точное совпадение)
func (c *keycloakClient) searchUsers(q string) ([]*gocloak.User, error) {
	token, err := getAdminKeycloakToken()
	if err != nil {
		return nil, err
	}

	return c.client.GetUsers(
		c.ctx,
		token.AccessToken,
		c.realm,
		gocloak.GetUsersParams{
			Exact:  gocloak.BoolP(false),
			Search: gocloak.StringP(q),
		})
}

// getUserByUsernameOrEmail получение пользователя по полному совпадению имени или email
func (c *keycloakClient) getUserByUsernameOrEmail(name string) (*gocloak.User, error) {
	user, err := c.client.GetUsers(
		c.ctx,
		c.token.jwt.AccessToken,
		c.realm,
		gocloak.GetUsersParams{
			Exact:  gocloak.BoolP(true),
			Search: gocloak.StringP(name),
		},
	)

	if err != nil {
		return nil, err
	}

	if len(user) > 0 {
		return user[0], nil
	}
	return nil, ErrUserNotExist{Name: name}
}

// usersTotalCount получения общего количества пользователей в realm keycloak
func (c *keycloakClient) usersTotalCount() (int64, error) {
	token, err := getAdminKeycloakToken()
	if err != nil {
		return 0, err
	}

	count, err := c.client.GetUserCount(
		c.ctx,
		token.AccessToken,
		c.realm,
		gocloak.GetUsersParams{},
	)

	return int64(count), err
}

// keycloak admin_cli клиент для работы с admin REST API keycloak
var keycloakAdmin keycloakClient

// InitKeycloakAdminClient инициализация клиента keycloak
func InitKeycloakAdminClient() error {
	if setting.SbtKeycloakForm.Enabled {
		ctx := context.Background()

		client := gocloak.NewClient(setting.SbtKeycloakForm.Url)

		token, err := client.LoginClient(
			ctx,
			setting.SbtKeycloakForm.MasterClientId,
			setting.SbtKeycloakForm.MasterClientSecret,
			setting.SbtKeycloakForm.RealmName,
		)
		if err != nil {
			return err
		}

		keycloakAdmin = keycloakClient{
			ctx:    ctx,
			client: client,
			token: accessTokenParams{
				jwt:     token,
				created: time.Now(),
			},
			realm: setting.SbtKeycloakForm.RealmName,
		}
	}

	return nil
}

// CreateUserInKeycloak метод создания пользователя в реалме Keycloak
// todo Переделать на использование keycloakClient
func CreateUserInKeycloak(user *User, password string) error {
	keycloakForm := setting.SbtKeycloakForm

	token, err := getAdminKeycloakToken()
	if err != nil {
		return err
	}

	body := KeycloakUser{
		Enabled:  true,
		UserName: user.LowerName,
		Email:    user.Email,
		Credentials: []KeycloakCredentials{
			{
				Type:      "password",
				Value:     password,
				Temporary: false,
			},
		},
	}

	var b io.ReadWriter
	b = &bytes.Buffer{}

	if err := json.NewEncoder(b).Encode(body); err != nil {
		return ErrKeycloakWrongHttpRequest{err}
	}

	httpClient := &http.Client{Timeout: 2 * time.Second}

	req, err := http.NewRequest(http.MethodPost, keycloakForm.AdminUserUrl, b)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token.AccessToken))

	res, err := httpClient.Do(req)
	if err != nil {
		return ErrKeycloakWrongHttpRequest{err}
	}

	log.Debug("Http response status code: %d was received on request at url: %s for username: %s", res.StatusCode, keycloakForm.AdminUserUrl, user.LowerName)

	if res.StatusCode != http.StatusCreated {
		return ErrKeycloakWrongHttpStatus{res.StatusCode, req}
	}

	return nil
}

// Получение токена администратора мастер реалма для дальнейшей работы с admin REST API keycloak.
// Выполняется проверка, что токен не протух и в случае протухания происходит обновление токена
func getAdminKeycloakToken() (*gocloak.JWT, error) {
	keycloakAdmin.Lock()
	defer keycloakAdmin.Unlock()

	// Если токен не протух, то вернем его
	if !accessTokenExpired(keycloakAdmin.token) {
		return keycloakAdmin.token.jwt, nil
	}

	// Если токен протух, то получим новый, обновим его и вернем новый
	token, err := keycloakAdmin.client.LoginClient(
		keycloakAdmin.ctx,
		setting.SbtKeycloakForm.MasterClientId,
		setting.SbtKeycloakForm.MasterClientSecret,
		setting.SbtKeycloakForm.RealmName)
	if err != nil {
		return nil, err
	}

	keycloakAdmin.token.jwt = token
	keycloakAdmin.token.created = time.Now()

	return token, nil
}

// accessTokenExpired проверка, что токен не протух
// (время последнего получения токена в секундах unix формата) + (время жизни токена в секундах) - (дельта в секундах) - (текущее время в секундах):
// если отрицательное значение то токен протух
func accessTokenExpired(token accessTokenParams) bool {
	if token.created.Unix()+int64(token.jwt.ExpiresIn)-int64(timeSkew)-time.Now().Unix() <= 0 {
		return true
	}

	return false
}

// GetUserTokenFromKeycloak Получение токена пользователя Keycloak по логину и паролю
// todo Переделать на использование keycloakClient
func GetUserTokenFromKeycloak(login string, password string) (*KeycloakUserToken, error) {
	keycloakForm := setting.SbtKeycloakForm

	data := url.Values{}
	data.Set("client_id", keycloakForm.RealmClientId)
	data.Set("client_secret", keycloakForm.RealmClientSecret)
	data.Set("grant_type", keycloakForm.UserGrantType)
	data.Set("username", login)
	data.Set("password", password)

	httpClient := &http.Client{Timeout: 2 * time.Second}

	req, err := http.NewRequest(http.MethodPost, keycloakForm.GetUserTokenUrl, strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	if err != nil {
		return nil, ErrKeycloakWrongHttpRequest{err}
	}

	res, err := httpClient.Do(req)

	if err != nil {
		return nil, ErrKeycloakWrongHttpRequest{err}
	}

	log.Debug("Http response with status code: %d was received on request at url: %s for login: %s", res.StatusCode, keycloakForm.GetUserTokenUrl, login)

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, ErrKeycloakWrongHttpStatus{StatusCode: res.StatusCode, Request: req}
	}

	var token KeycloakUserToken
	err = json.NewDecoder(res.Body).Decode(&token)

	if err != nil {
		return nil, ErrKeycloakWrongHttpRequest{err}
	}

	return &token, nil
}

// KeycloakLogoutSession Завершение сессии пользователя в Keycloak
// todo Переделать на использование keycloakClient
func KeycloakLogoutSession(token string, username string) error {
	keycloakForm := setting.SbtKeycloakForm

	data := url.Values{}
	data.Set("client_id", keycloakForm.RealmClientId)
	data.Set("client_secret", keycloakForm.RealmClientSecret)
	data.Set("refresh_token", token)

	httpClient := &http.Client{Timeout: 2 * time.Second}

	req, err := http.NewRequest(http.MethodPost, keycloakForm.LogoutSessionUrl, strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	if err != nil {
		return ErrKeycloakWrongHttpRequest{err}
	}

	res, err := httpClient.Do(req)

	if err != nil {
		return ErrKeycloakWrongHttpRequest{err}
	}

	log.Debug("Http response with status code: %d was received on request at url: %s for username: %s", res.StatusCode, keycloakForm.LogoutSessionUrl, username)

	if res.StatusCode != http.StatusNoContent {
		return ErrKeycloakWrongHttpStatus{StatusCode: res.StatusCode, Request: req}
	}

	return nil
}

// SearchUsersInKeycloak Поиск пользователей в keycloak по именам пользователей или именам и фамилиям (не точное совпадение)
func SearchUsersInKeycloak(query string) ([]*User, error) {

	keycloakUsers, err := keycloakAdmin.searchUsers(query)
	if err != nil {
		return nil, err
	}

	var users []*User
	for _, keycloakUser := range keycloakUsers {
		if *keycloakUser.EmailVerified {
			users = append(users, &User{
				Name:     *keycloakUser.Username,
				FullName: fmt.Sprintf("%s %s", *keycloakUser.LastName, *keycloakUser.FirstName),
				Email:    *keycloakUser.Email,
			})
		}
	}

	return users, nil
}

// GetUsersTotalCountFromKeycloak получения общего количества пользователей в realm keycloak
func GetUsersTotalCountFromKeycloak() (int64, error) {
	return keycloakAdmin.usersTotalCount()
}

// getUserByNameOrEmailFromKeycloak Получение пользователя по имени или email
func getUserByNameOrEmailFromKeycloak(name string, locale translation.Locale) (*User, error) {
	keycloakUser, err := keycloakAdmin.getUserByUsernameOrEmail(name)

	if err != nil {
		return nil, err
	}

	user := User{
		Name:      *keycloakUser.Username,
		IsActive:  *keycloakUser.Enabled,
		FullName:  fmt.Sprintf("%s %s", *keycloakUser.LastName, *keycloakUser.FirstName),
		LowerName: strings.ToLower(*keycloakUser.Username),
		Language:  locale.Language(),
	}

	if keycloakUser.Email != nil {
		user.Email = *keycloakUser.Email
	}

	return &user, nil
}

// GetAndCreateUserByNameOrEmailFromKeycloak Получение пользователя из keycloak и его добавление в БД для последующей привязки к команде или коллаборантам, если включена работа с keycloak
func GetAndCreateUserByNameOrEmailFromKeycloak(searchString string, locale translation.Locale, ctx context.Context) (*User, error) {
	var u *User
	if setting.SbtKeycloakForm.Enabled { // если включена работа с keycloak
		keycloakUser, err := getUserByNameOrEmailFromKeycloak(searchString, locale) // то попробуем найти в keycloak
		if IsErrUserNotExist(err) {
			return nil, err
		}
		// если пользователь найден в БД, возвращаем его и не создаем нового
		u, _ = GetUserByName(ctx, keycloakUser.Name)
		if u != nil {
			return u, nil // если нашли, то вернем его
		}
		if len(keycloakUser.Email) == 0 { // если у пользователя нет почты (wtf???), то это какая-то ошибка во внешнем хранилище пользователей, не будем с таким пользователем работать
			return nil, ErrEmailAddressNotExist{keycloakUser.Email}
		}
		err = CreateUser(keycloakUser) // если нашли, то добавим в БД
		if err != nil {
			log.Error("Error has occurred while try create user from keycloak with name: %s, err: %v", keycloakUser.LowerName, err)
			return nil, err
		}
		u, err = GetUserByName(ctx, keycloakUser.Name) // и получим его с ИД и тд
		if err != nil {
			log.Error("Error has occurred while try get user from db with name or email: %s, err: %v", searchString, err)
			return nil, err
		}
	} else { // если работа с keycloak выключена, то вернем ошибку
		return nil, ErrUserNotExist{Name: searchString}
	}

	return u, nil
}
