package setting

import "github.com/google/uuid"

// SourceControl настройки специфичные только для SourceControl
var SourceControl struct {
	//Enabled Активирован ли SourceControl
	Enabled bool
	//TenantWithRoleModeEnabled Параметр для включения функционала тенанта с ролевой модели
	TenantWithRoleModeEnabled bool
	// MultiTenantEnabled Параметр для включения мультитенантности
	MultiTenantEnabled bool
	//InternalPrivilegeManagement Параметр для включения функционала управления группами привилегий в проекте
	InternalPrivilegeManagement bool
	//InternalProjectCreate Параметр для включения функционала создания проекта
	InternalProjectCreate bool
	//ExternalPreReceiveHookEnabled Параметр для включения функционала запуска скриптов перед заливкой кода в репозиторий
	ExternalPreReceiveHookEnabled bool
	// EmptyEmailEnabled Параметр для включения функционала без регистрации пользователей по email
	EmptyEmailEnabled bool
	//IMToolName Имя стенда для onework
	IAMToolName string
	//DefaultTenantName Имя тенанта по умолчанию
	DefaultTenantName string
	//DefaultTenantID Идентификатор тенанта по умолчанию
	DefaultTenantID string
	//DefaultTenantID Идентификатор организации	по умолчанию
	DefaultOrgKey string
}

var WidgetSW struct {
	Enabled     bool
	SystemName  string
	ApiHostUrl  string
	DepsHostUrl string
}

// loadSourceControl подтягивает настройки из конфигурационного файла
func loadSourceControl(rootCfg ConfigProvider) {
	sec := rootCfg.Section("sourcecontrol")

	SourceControl.Enabled = sec.Key("ENABLED").MustBool(false)
	if SourceControl.Enabled {
		SourceControl.TenantWithRoleModeEnabled = sec.Key("TENANT_WITH_ROLE_MODEL_ENABLED").MustBool(false)
		SourceControl.MultiTenantEnabled = sec.Key("MULTI_TENANT_ENABLED").MustBool(false)
		SourceControl.InternalPrivilegeManagement = sec.Key("INTERNAL_PRIVILEGE_MANAGEMENT").MustBool(false)
		SourceControl.InternalProjectCreate = sec.Key("INTERNAL_PROJECT_CREATE").MustBool(false)
		SourceControl.ExternalPreReceiveHookEnabled = sec.Key("EXTERNAL_PRE_RECEIVE_HOOK_ENABLED").MustBool(false)
		SourceControl.EmptyEmailEnabled = sec.Key("EMPTY_EMAIL_ENABLED").MustBool(false)
		SourceControl.IAMToolName = sec.Key("IAM_TOOL_NAME").MustString("sc")
		SourceControl.DefaultTenantName = sec.Key("DEFAULT_TENANT_NAME").MustString("tenant")
		SourceControl.DefaultTenantID = sec.Key("DEFAULT_TENANT_ID").MustString(uuid.NewString())
		SourceControl.DefaultOrgKey = sec.Key("DEFAULT_ORG_KEY").MustString("")

		widgetSec := rootCfg.Section("sourcecontrol.widget-sw")
		WidgetSW.Enabled = widgetSec.Key("ENABLED").MustBool(false)
		if WidgetSW.Enabled {
			WidgetSW.SystemName = widgetSec.Key("SYSTEM_NAME").MustString("")
			WidgetSW.ApiHostUrl = widgetSec.Key("API_HOST_URL").MustString("")
			WidgetSW.DepsHostUrl = widgetSec.Key("DEPS_HOST_URL").MustString("")
		}
	}
}
