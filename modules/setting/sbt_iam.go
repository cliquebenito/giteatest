package setting

import (
	"strings"
)

// IAM настройки
var IAM struct {
	// Enabled Активирован ли IAM
	Enabled bool
	// CasbinCacheUpdateTTLInSeconds интервал обновления кеша Casbin
	CasbinCacheUpdateTTLInSeconds int
	// WhiteListRolesUser содержит белый список ролей пользователей SC
	WhiteListRolesUser []string
	// WhiteListRolesAdmin содержит белый список ролей админов SC
	WhiteListRolesAdmin []string
	// WsPrivilegesEnabled активирован ли парсинг и применение header Ws-Privileges
	WsPrivilegesEnabled bool
	// BaseURL - ссылка, которая возвращается при отображении клонирования репозитория
	BaseURL string
	// SSHDomain - домен, который возвращается при отображении клонирования репозитория ssh
	SSHDomain string
	// EnableRepositoryDelete - флаг для переключения отоборажения кнпоки удаления репозитория
	EnableRepositoryDelete bool
	// AuthorizationFormEnabled отображаем ли форму авторизации SC
	AuthorizationFormEnabled bool
}

// loadIAM загрузить настройки IAM
func loadIAM(rootCfg ConfigProvider) {
	sec := rootCfg.Section("iam")

	IAM.Enabled = sec.Key("ENABLED").MustBool(false)
	if IAM.Enabled {
		IAM.CasbinCacheUpdateTTLInSeconds = sec.Key("CASBIN_CACHE_UPDATE_TTL_IN_SECONDS").MustInt(30)
		IAM.WhiteListRolesUser = parseWhiteListRolesUser(sec)
		IAM.WhiteListRolesAdmin = parseWhiteListRolesAdmin(sec)
		IAM.WsPrivilegesEnabled = sec.Key("WS_PRIVILEGES_ENABLED").MustBool(true)
		IAM.BaseURL = sec.Key("BASE_URL").MustString("")
		IAM.SSHDomain = sec.Key("SSH_DOMAIN").MustString("")
		IAM.EnableRepositoryDelete = sec.Key("ENABLE_REPOSITORY_DELETE").MustBool(false)
		IAM.AuthorizationFormEnabled = sec.Key("AUTHORIZATION_FORM_ENABLED").MustBool(false)
	}
}

func parseWhiteListRolesUser(iamSection ConfigSection) []string {
	whiteListRolesUser := iamSection.
		Key("WHITE_LIST_ROLES_USER").String()

	if whiteListRolesUser == "" {
		return nil
	}

	return strings.Split(whiteListRolesUser, ",")
}

func parseWhiteListRolesAdmin(iamSection ConfigSection) []string {
	whiteListRolesAdmin := iamSection.
		Key("WHITE_LIST_ROLES_ADMIN").String()

	if whiteListRolesAdmin == "" {
		return nil
	}

	return strings.Split(whiteListRolesAdmin, ",")
}
