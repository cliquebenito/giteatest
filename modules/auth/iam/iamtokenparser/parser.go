package iamtokenparser

import (
	"errors"
	"fmt"

	"github.com/golang-jwt/jwt/v4"

	"code.gitea.io/gitea/modules/auth/iam/iamtoken"
	"code.gitea.io/gitea/modules/log"
)

const (
	userGlobalIDKey = "sub"
	emailKey        = "email"
	userNameKey     = "preferred_username"
	familyNameKey   = "family_name"
	givenNameKey    = "given_name"
	tenantKey       = "organization"
	groupsKey       = "groups"
)

type IAMJWTTokenParser struct {
	WhiteListRoles        WhiteListRoles
	isWSPrivilegesEnabled bool
}

type WhiteListRoles struct {
	User  map[string]struct{}
	Admin map[string]struct{}
}

func NewWithKeyfunc(
	isWSPrivilegesEnabled bool,
	whiteListRolesUser []string,
	whiteListRolesAdmin []string,
) (IAMJWTTokenParser, error) {
	whiteListRolesUserMap := make(map[string]struct{})
	for _, role := range whiteListRolesUser {
		whiteListRolesUserMap[role] = struct{}{}
	}

	whiteListRolesAdminMap := make(map[string]struct{})
	for _, role := range whiteListRolesAdmin {
		whiteListRolesAdminMap[role] = struct{}{}
	}

	return IAMJWTTokenParser{
		WhiteListRoles: WhiteListRoles{
			User:  whiteListRolesUserMap,
			Admin: whiteListRolesAdminMap,
		},
		isWSPrivilegesEnabled: isWSPrivilegesEnabled,
	}, nil
}

func (t IAMJWTTokenParser) OpenFromString(content string) (iamtoken.IAMJWT, bool, error) {
	iamTokenSigningAlg := []string{"RS256"}

	opts := []jwt.ParserOption{jwt.WithValidMethods(iamTokenSigningAlg)}
	parser := jwt.NewParser(opts...)

	var (
		token *jwt.Token
		err   error
	)

	token, _, err = parser.ParseUnverified(content, jwt.MapClaims{})
	if err != nil {
		log.Error("Error has occurred while parsing iam token without validation %v", err)
		return iamtoken.IAMJWT{}, false, fmt.Errorf("parse iam token without validation: %w", err)
	}

	claims, err := t.getClaims(token)
	if err != nil {
		log.Error("Error has occurred while getting claims %v", err)
		return iamtoken.IAMJWT{}, false, fmt.Errorf("get claims: %w", err)
	}

	userGlobalID, err := getString(userGlobalIDKey, claims)
	if err != nil {
		log.Error("Error has occurred while getting globalID %v", err)
		return iamtoken.IAMJWT{}, false, fmt.Errorf("get global id: %w", err)
	}

	email, err := getString(emailKey, claims)
	if err != nil {
		log.Error("Error has occurred while getting email %v", err)
		return iamtoken.IAMJWT{}, false, fmt.Errorf("get email: %w", err)
	}

	userName, err := getString(userNameKey, claims)
	if err != nil {
		log.Error("Error has occurred while getting preferred_username %v", err)
		return iamtoken.IAMJWT{}, false, fmt.Errorf("get preferred_username: %w", err)
	}

	familyName, err := getString(familyNameKey, claims)
	if err != nil {
		log.Error("Error has occurred while getting family name %v", err)
		return iamtoken.IAMJWT{}, false, fmt.Errorf("get family name: %w", err)
	}

	givenName, err := getString(givenNameKey, claims)
	if err != nil {
		log.Error("Error has occurred while getting given name %v", err)
		return iamtoken.IAMJWT{}, false, fmt.Errorf("get given name: %w", err)
	}

	tenantName, err := getString(tenantKey, claims)
	if err != nil {
		log.Debug("Error has occurred while getting tenant %v", err)
	}

	role, isGroupsInToken, err := t.getRole(claims)
	if err != nil {
		log.Error("Error has occurred while getting role %v", err)
		return iamtoken.IAMJWT{}, false, fmt.Errorf("get role: %w", err)
	}

	return iamtoken.IAMJWT{
		JWTToken:   token,
		GlobalID:   userGlobalID,
		Name:       userName,
		FullName:   fmt.Sprintf("%s %s", familyName, givenName),
		Email:      email,
		Role:       role,
		TenantName: tenantName,
	}, isGroupsInToken, nil
}

func (t IAMJWTTokenParser) getRole(claims jwt.MapClaims) (role iamtoken.SourceControlGlobalRole, isGroupsInToken bool, err error) {
	roles, err := getStringSlice(groupsKey, claims)
	if err != nil {
		if errNotExists := new(ErrorIAMClaimNotExists); !errors.As(err, &errNotExists) {
			log.Error("Error has occurred while getting groups from claims: %v", err)
			return "", false, fmt.Errorf("error has occurred while getting groups from claims: %w", err)
		}

		log.Debug("field [groups] does not exist in token")
	}

	// Если роли в токене не пришли, возвращаем sc_user
	if len(roles) == 0 || roles[0] == "" {
		if t.isWSPrivilegesEnabled {
			log.Error("Error has occurred while required role into token for ws-privileges enable")
			return "", false, fmt.Errorf("required role into token for ws-privileges enable")
		}
		return iamtoken.UserRole, false, nil
	}

	scRolesUser := make([]string, 0)
	scRolesAdmin := make([]string, 0)
	for _, role := range roles {
		if _, ok := t.WhiteListRoles.User[role]; ok {
			scRolesUser = append(scRolesUser, role)
		}
		if _, ok := t.WhiteListRoles.Admin[role]; ok {
			scRolesAdmin = append(scRolesAdmin, role)
		}
	}

	// Если роли в токене не соответствуют белому списку из app.ini
	// и белый список задан, то возвращаем ошибку
	// Если белый список не задан, возвращаем sc_user
	// Роль admin в приоритете
	if len(scRolesUser) == 0 && len(scRolesAdmin) == 0 {
		if len(t.WhiteListRoles.User) > 0 || len(t.WhiteListRoles.Admin) > 0 {
			log.Error("Error has occurred while matching roles with whitelist of config file")
			return "", false, fmt.Errorf("roles from groups don't match roles from whitelist of config file")
		}

		return iamtoken.UserRole, false, nil
	}

	if len(scRolesAdmin) > 0 {
		return iamtoken.AdminRole, true, nil
	}

	return iamtoken.UserRole, true, nil
}

func (t IAMJWTTokenParser) getClaims(token *jwt.Token) (jwt.MapClaims, error) {
	if token == nil || token.Claims == nil {
		return nil, fmt.Errorf("jwt token claims are empty")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("claims is not map: %v", claims)
	}

	return claims, nil
}

func getString(name string, claims jwt.MapClaims) (string, error) {
	if claims == nil {
		return "", fmt.Errorf("jwt token claims are empty")
	}

	value, exists := claims[name]
	if !exists {
		return "", NewErrorIAMClaimNotExists(fmt.Errorf("claim %s does not exists", name))
	}

	castedValue, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("claim value is not string: %v", value)
	}

	return castedValue, nil
}

func getStringSlice(name string, claims jwt.MapClaims) ([]string, error) {
	if claims == nil {
		return nil, fmt.Errorf("jwt token claims are empty")
	}

	value, exists := claims[name]
	if !exists {
		return nil, NewErrorIAMClaimNotExists(fmt.Errorf("claim %s does not exists", name))
	}

	castedValue, ok := value.([]interface{})
	if !ok {
		return nil, fmt.Errorf("claim value is not []interface{}: %v", value)
	}

	var stringValues []string
	for _, rawValue := range castedValue {
		if stringValue, ok := rawValue.(string); ok {
			stringValues = append(stringValues, stringValue)
		}
	}

	return stringValues, nil
}
