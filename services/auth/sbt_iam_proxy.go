package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	authmodel "code.gitea.io/gitea/models/auth"
	"code.gitea.io/gitea/models/db"
	usermodel "code.gitea.io/gitea/models/user"
	iampriveleges "code.gitea.io/gitea/modules/auth/iam/iamprivileges"
	"code.gitea.io/gitea/modules/auth/iam/iamtoken"
	"code.gitea.io/gitea/modules/auth/iam/iamtokenparser"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/timeutil"
	"code.gitea.io/gitea/routers/sbt/user/createuser"
	"code.gitea.io/gitea/services/auth/iamprivileger"
)

var (
	_ Method = &IAMProxy{}
	_ Named  = &IAMProxy{}
)

const IAMProxyMethodName = "iam_proxy"

func (i *IAMProxy) Name() string {
	return IAMProxyMethodName
}

type IAMProxy struct {
	privileger iamprivileger.Privileger
	engine     db.Engine
}

func NewIAMProxy(privileger iamprivileger.Privileger, engine db.Engine) *IAMProxy {
	return &IAMProxy{privileger: privileger, engine: engine}
}

func (i *IAMProxy) Verify(req *http.Request, _ http.ResponseWriter, _ DataStore, _ SessionStore) (user *usermodel.User, err error) {
	iamToken, whiteListRolesAdmin, isGroupsInToken, err := parseIAMToken(req)
	if err != nil {
		if errIAM := new(ErrorIncorrectTokenType); errors.As(err, &errIAM) {
			log.Warn("wrong token type")
			return nil, nil
		}

		log.Error("iam:verify token parse: %v. Try to authenticate another way", err)
		return nil, NewErrorParseIAMJWT(err)
	}

	log.Debug("iam:verify parse token: %s", iamToken.GlobalID)

	wsPrivilegesEnabled := shouldUseWsPrivileges(req.Header.Get("Git-Protocol"))
	// В токене не пришло поле organization (тенант) и включен флаг wsPrivileges
	if iamToken.TenantName == "" && wsPrivilegesEnabled {
		log.Error("Error has occurred while receiving tenant field in token")
		return nil, NewErrorParseIAMJWT(fmt.Errorf("couldn't receive tenant field in token"))
	}

	ctx := req.Context()

	_, isAdmin := whiteListRolesAdmin[string(iamToken.Role)]

	user, err = usermodel.GetIAMUserByLoginName(ctx, i.engine, iamToken.GlobalID)
	if err != nil {
		log.Debug("GetIAMUserByLoginName is failed next step is find user handleUserNotFoundError")
		user, err = i.handleUserNotFoundError(ctx, iamToken, isAdmin, err)
		if err != nil {
			return nil, err
		}
	}

	// проверяем требуется ли обновить роль пользователя
	if isGroupsInToken && user.IsAdmin != isAdmin {
		if err = i.updateUserRole(ctx, user, isAdmin); err != nil {
			log.Error("Error has occurred while updating user role: %v", err)
			return nil, fmt.Errorf("updating user role: %w", err)
		}
	}

	log.Debug("Verify user by login name from token: %d", user.ID)
	log.Debug("Verify User: %v\n", user)

	// Парсим привилегии из JWT токена и headers и применяем их (если включен флаг)
	if wsPrivilegesEnabled {
		iamPrivileges, err := parseWsPrivileges(req)
		if err != nil {
			return nil, NewErrorParsePrivileges(err)
		}
		log.Debug("Parsed privileges from token: %v", iamPrivileges)

		if !shouldCachePrivileges(user.LastLoginUnix) {
			log.Info("Privileges applied successfully without caching")
			return user, nil
		}

		if err = i.applyPrivileges(ctx, user, iamToken, iamPrivileges); err != nil {
			return nil, NewErrorApplyPrivileges(err)
		}

		if err = i.updateUserLastLoginTs(ctx, user); err != nil {
			return nil, fmt.Errorf("update user: %w", err)
		}

		log.Info("Privileges applied successfully with caching")
	}

	return user, nil
}

// shouldUseWsPrivileges проверяет, содержит ли заголовок протокола Git
// определенный префикс и установлен ли флаг использования привилегий.
func shouldUseWsPrivileges(gitProtocolHeader string) bool {
	wsPrivilegesEnabled := setting.IAM.WsPrivilegesEnabled
	if gitProtocolHeader != "" {
		log.Debug("WS-Privileges disabled due to request type: git")
		wsPrivilegesEnabled = false
	}

	return wsPrivilegesEnabled
}

func (i *IAMProxy) updateUserLastLoginTs(ctx context.Context, user *usermodel.User) error {
	user.SetLastLogin()

	if err := usermodel.UpdateUser(ctx, user, false, "last_login_unix"); err != nil {
		return fmt.Errorf("update user: %v", err)
	}

	return nil
}

func (i *IAMProxy) updateUserRole(ctx context.Context, user *usermodel.User, isAdmin bool) error {
	user.IsAdmin = isAdmin
	if err := usermodel.UpdateUser(ctx, user, false, "is_admin"); err != nil {
		log.Error("Error has occurred while updating user: %v", err)
		return fmt.Errorf("update user: %v", err)
	}

	return nil
}

func (i *IAMProxy) handleUserNotFoundError(
	ctx context.Context,
	iamToken iamtoken.IAMJWT,
	isAdmin bool,
	err error,
) (*usermodel.User, error) {
	if userNotFoundErr := new(usermodel.ErrUserNotExist); !errors.As(err, &userNotFoundErr) {
		return nil, fmt.Errorf("iam common err: %w", err)
	}

	userToCreate := &usermodel.User{
		Name:      iamToken.Name,
		FullName:  iamToken.FullName,
		Email:     iamToken.Email,
		LoginName: iamToken.GlobalID,
		LoginType: authmodel.IAM,
		IsAdmin:   isAdmin,
	}

	if err = createuser.Create(ctx, i.engine, userToCreate); err != nil {
		return userToCreate, fmt.Errorf("create and handle user: %w", err)
	}

	return userToCreate, nil
}

func shouldCachePrivileges(lastCacheUpdateTs timeutil.TimeStamp) bool {
	currentTime := timeutil.TimeStamp(time.Now().Unix())
	return int(currentTime-lastCacheUpdateTs) > setting.IAM.CasbinCacheUpdateTTLInSeconds
}

func (i *IAMProxy) applyPrivileges(
	ctx context.Context,
	user *usermodel.User,
	iamToken iamtoken.IAMJWT,
	iamPrivileges iampriveleges.SourceControlPrivilegesByTenant,
) error {
	if err := i.privileger.ApplyPrivileges(ctx, user, iamToken, iamPrivileges); err != nil {
		return fmt.Errorf("apply privileges: %w", err)
	}

	return nil
}

func parseIAMToken(req *http.Request) (iamtoken.IAMJWT, map[string]struct{}, bool, error) {
	iamTokenRaw, err := GetJWTFromHeader(req)
	if err != nil {
		return iamtoken.IAMJWT{}, nil, false, fmt.Errorf("get jwt token from header: %w", err)
	}

	iamTokenParser, err := iamtokenparser.NewWithKeyfunc(
		setting.IAM.WsPrivilegesEnabled,
		setting.IAM.WhiteListRolesUser,
		setting.IAM.WhiteListRolesAdmin,
	)
	if err != nil {
		return iamtoken.IAMJWT{}, nil, false, fmt.Errorf("create iam token parser: %w", err)
	}

	iamToken, isGroupsInToken, err := iamTokenParser.OpenFromString(iamTokenRaw)
	if err != nil {
		return iamtoken.IAMJWT{}, nil, false, fmt.Errorf("parse iam token: %w", err)
	}

	return iamToken, iamTokenParser.WhiteListRoles.Admin, isGroupsInToken, nil
}

func parseWsPrivileges(req *http.Request) (iampriveleges.SourceControlPrivilegesByTenant, error) {
	// В заголовках приходят привилегии OW
	wsPrivileges, err := GetWsPrivilegesFromHeader(req)
	if err != nil {
		return nil, fmt.Errorf("get ws iam privileges from header: %w", err)
	}

	log.Debug("Found raw Ws-Privileges %v", wsPrivileges)

	// форматируем привилегии в мапу привилегий
	iamPrivileges, err := iampriveleges.OpenFromString(wsPrivileges)
	if err != nil {
		return nil, fmt.Errorf("open iam privileges: %w", err)
	}

	jsonIamPrivileges, _ := iamPrivileges.JSON()
	log.Debug("Got json Ws-Privileges %s", jsonIamPrivileges)

	return iamPrivileges, nil
}

// GetJWTFromHeader парсит JWT из заголовка Authorization
func GetJWTFromHeader(req *http.Request) (string, error) {
	const bearerToken = "Bearer"
	reqToken := req.Header.Get("Authorization")
	if len(reqToken) == 0 {
		log.Error("Error has occurred while getting authorization header")
		return "", NewErrorIncorrectTokenType()
	}

	splitToken := strings.Split(reqToken, " ")
	if len(splitToken) != 2 { // "Bearer {token}"
		log.Error("Error has occurred while getting two parts of token")
		return "", fmt.Errorf("bad authorization header")
	}

	if splitToken[0] != bearerToken {
		log.Error("Error has occurred while getting bearer token")
		return "", NewErrorIncorrectTokenType()
	}

	return splitToken[1], nil
}

// GetWsPrivilegesFromHeader парсит Ws-Privileges из заголовка Ws-Privileges
func GetWsPrivilegesFromHeader(req *http.Request) (string, error) {
	privileges := req.Header.Get("Ws-Privileges")
	if len(privileges) == 0 {
		return "", fmt.Errorf("no privileges header")
	}

	return privileges, nil
}
