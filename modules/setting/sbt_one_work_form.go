package setting

import (
	"log"
	"net/url"
	"time"

	"gitea.com/go-chi/binding"
)

const defaultFallbackTimeout = 10 * time.Second

var OneWork = struct {
	Enabled         bool
	ServiceURL      string
	ToolKey         string
	ContextURI      string
	MenuKey         string
	LoginLink       string
	LogoutLink      string
	FallbackTimeout time.Duration
	// ExtendedAdminPanel Признак того, надо ди отображать в панели админа такие разделы как "Авторизация", "Тенанты" и "Адреса эл. почты пользователей"
	ExtendedAdminPanel bool
}{}

// loadSbtOneWorkForm метод загрузки переменных, необходимых для работы с OneWork, из app.ini
func loadSbtOneWorkForm(rootCfg ConfigProvider) {
	sec := rootCfg.Section("sbt.one_work")
	if !sec.Key("ENABLED").MustBool() {
		return
	}
	OneWork.Enabled = sec.Key("ENABLED").MustBool()

	if !OneWork.Enabled {
		return
	}

	OneWork.ServiceURL = sec.Key("SERVICE_URL").String()
	if len(OneWork.ServiceURL) == 0 {
		log.Fatal("SERVICE_URL can not be empty for ONEWORK menu type in app.ini file")
	}
	_, err := url.Parse(OneWork.ServiceURL)
	if err != nil || !binding.URLPattern.MatchString(OneWork.ServiceURL) {
		log.Fatal("SERVICE_URL must be url type")
	}

	OneWork.ToolKey = sec.Key("TOOL_KEY").String()
	if len(OneWork.ToolKey) == 0 {
		log.Fatal("TOOL_KEY can not be empty for ONEWORK menu type in app.ini file")
	}

	OneWork.ContextURI = sec.Key("CONTEXT_URI").String()
	if len(OneWork.ContextURI) == 0 {
		log.Fatal("CONTEXT_URI can not be empty for ONEWORK menu type in app.ini file")
	}

	OneWork.FallbackTimeout = sec.Key("FALLBACK_TIMEOUT").MustDuration(defaultFallbackTimeout)

	OneWork.MenuKey = sec.Key("MENU_KEY").String()
	if len(OneWork.MenuKey) == 0 {
		log.Fatal("MENU_KEY can not be empty for ONEWORK menu type in app.ini file")
	}

	OneWork.LoginLink = sec.Key("LOGIN_LINK").String()
	if len(OneWork.LoginLink) == 0 {
		log.Fatal("LOGIN_LINK can not be empty in app.ini file")
	}

	OneWork.LogoutLink = sec.Key("LOGOUT_LINK").String()

	OneWork.ExtendedAdminPanel = sec.Key("EXTENDED_ADMIN_PANEL").MustBool(false)
}
