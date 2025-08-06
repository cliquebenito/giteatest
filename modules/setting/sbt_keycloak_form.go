package setting

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	vault_model "code.gitea.io/gitea/models/vault_client"
	"code.gitea.io/gitea/modules/log"
)

var SbtKeycloakForm = struct {
	//Общие настройки Keycloak
	Enabled bool
	Url     string

	//Клиент мастер реалма Keycloak
	MasterClientId     string
	MasterClientSecret string
	AdminGrantType     string

	//Настройки не мастер реалма и клиента НЕ мастер реалма
	RealmName         string
	RealmClientId     string
	RealmClientSecret string
	UserGrantType     string

	//Url для работы с Keycloak
	GetOpenIdConfigUrl string
	GetAdminTokenUrl   string
	AdminUserUrl       string
	GetUserTokenUrl    string
	LogoutSessionUrl   string
	secManGetter       GetCredSecMan
}{
	Enabled:        false,
	AdminGrantType: "client_credentials",
	UserGrantType:  "password",
}

func loadSbtKeycloakForm(rootCfg ConfigProvider) {
	sec := rootCfg.Section("sbt.auth_keycloak")
	if !sec.Key("ENABLED").MustBool() {
		return
	}
	var resp *vault_model.SecretVaultResponse
	if CheckSettingsForIntegrationWithSecMan() {
		var err error
		configForKvGet := &vault_model.KeyValueConfigForGetSecrets{
			SecretPath:  strings.TrimSpace(SourceControlAuthKeycloakSecret.SecretPath),
			StoragePath: strings.TrimSpace(SourceControlAuthKeycloakSecret.StoragePath),
			VersionKey:  SourceControlAuthKeycloakSecret.VersionKey,
		}
		SbtKeycloakForm.secManGetter = NewGetterForSecMan()
		resp, err = SbtKeycloakForm.secManGetter.GetCredFromSecManByVersionKey(configForKvGet)
		if err != nil {
			log.Fatal("Error has occurred while trying to get cred from secret storage: %v", err)
		}

		if GetResponseNotNil(resp) {
			SbtKeycloakForm.MasterClientId = strings.TrimSpace(resp.Data[SourceControlAuthKeycloakSecret.AdminClientIDMasterRealm])
			CheckIfSecretIsEmptyAndReportToAudit("ADMIN_CLIENT_ID_MASTER_REALM", SbtKeycloakForm.MasterClientId, "ADMIN_CLIENT_ID_MASTER_REALM is empty in secret storage", log.Fatal)

			SbtKeycloakForm.MasterClientSecret = strings.TrimSpace(resp.Data[SourceControlAuthKeycloakSecret.AdminCliSecret])
			CheckIfSecretIsEmptyAndReportToAudit("ADMIN_CLI_SECRET", SbtKeycloakForm.MasterClientSecret, "ADMIN_CLI_SECRET is empty in secret storage", log.Fatal)

			SbtKeycloakForm.RealmClientId = strings.TrimSpace(resp.Data[SourceControlAuthKeycloakSecret.RealmClientID])
			CheckIfSecretIsEmptyAndReportToAudit("REALM_CLIENT_ID", SbtKeycloakForm.RealmClientId, "REALM_CLIENT_ID is empty in secret storage", log.Fatal)

			SbtKeycloakForm.RealmClientSecret = strings.TrimSpace(resp.Data[SourceControlAuthKeycloakSecret.RealmClientSecret])
			CheckIfSecretIsEmptyAndReportToAudit("REALM_CLIENT_SECRET", SbtKeycloakForm.RealmClientSecret, "REALM_CLIENT_SECRET is empty in secret storage", log.Fatal)
		} else {
			CheckIfSecretIsEmptyAndReportToAudit("ADMIN_CLIENT_ID_MASTER_REALM", "", "Response from secret storage is nil", log.Fatal)
		}
	} else {
		if SbtKeycloakForm.MasterClientId = sec.Key("ADMIN_CLIENT_ID_MASTER_REALM").MustString("admin-cli"); SbtKeycloakForm.MasterClientId == "" {
			log.Fatal("ADMIN_CLIENT_ID_MASTER_REALM is empty")
		}
		if SbtKeycloakForm.MasterClientSecret = sec.Key("ADMIN_CLI_SECRET").String(); SbtKeycloakForm.MasterClientSecret == "" {
			log.Fatal("ADMIN_CLI_SECRET is empty")
		}
		if SbtKeycloakForm.RealmClientId = sec.Key("REALM_CLIENT_ID").String(); SbtKeycloakForm.RealmClientId == "" {
			log.Fatal("REALM_CLIENT_ID is empty")
		}
		if SbtKeycloakForm.RealmClientSecret = sec.Key("REALM_CLIENT_SECRET").String(); SbtKeycloakForm.RealmClientSecret == "" {
			log.Fatal("REALM_CLIENT_SECRET is empty")
		}
	}

	SbtKeycloakForm.Enabled = sec.Key("ENABLED").MustBool()
	SbtKeycloakForm.Url = sec.Key("KEYCLOAK_URL").String()
	if len(SbtKeycloakForm.Url) == 0 {
		log.Fatal("KEYCLOAK_URL can not be empty in app.ini file")
	}
	SbtKeycloakForm.RealmName = sec.Key("REALM_NAME").MustString("gitverse")

	SbtKeycloakForm.GetOpenIdConfigUrl = sec.Key("OPENID_CONFIGURATION_URL").MustString(fmt.Sprintf("%s/realms/%s/.well-known/openid-configuration", SbtKeycloakForm.Url, SbtKeycloakForm.RealmName))
	if !strings.HasPrefix(SbtKeycloakForm.GetOpenIdConfigUrl, SbtKeycloakForm.Url) || !strings.Contains(SbtKeycloakForm.GetOpenIdConfigUrl, SbtKeycloakForm.RealmName) {
		log.Fatal("OPENID_CONFIGURATION_URL must contain same KEYCLOAK_URL and REALM_NAME")
	}

	SbtKeycloakForm.GetAdminTokenUrl = sec.Key("GET_ADMIN_TOKEN_URL").MustString(SbtKeycloakForm.Url + "/realms/master/protocol/openid-connect/token")
	SbtKeycloakForm.AdminUserUrl = sec.Key("ADMIN_USER_URL").MustString(fmt.Sprintf("%s/admin/realms/%s/users", SbtKeycloakForm.Url, SbtKeycloakForm.RealmName))

	openIdStruct, err := getOpenIdConfiguration(SbtKeycloakForm.GetOpenIdConfigUrl)

	if err != nil {
		log.Fatal("Error has occurred while getting openId configuration from: %s", SbtKeycloakForm.GetOpenIdConfigUrl)
	}

	if len(openIdStruct.TokenEndpoint) == 0 || len(openIdStruct.EndSessionEndpoint) == 0 {
		log.Fatal("Empty Keycloak open id configuration")
	} else {
		SbtKeycloakForm.GetUserTokenUrl = openIdStruct.TokenEndpoint
		SbtKeycloakForm.LogoutSessionUrl = openIdStruct.EndSessionEndpoint
	}
}

// OpenIdConfiguration структура ответа на запрос конфигураций openId (В теле ответа больше полей, это урезанная форма)
type OpenIdConfiguration struct {
	Issuer             string `json:"issuer"`
	AuthEndpoint       string `json:"authorization_endpoint"`
	TokenEndpoint      string `json:"token_endpoint"`
	IntrospectEndpoint string `json:"introspection_endpoint"`
	UserInfoEndpoint   string `json:"userinfo_endpoint"`
	EndSessionEndpoint string `json:"end_session_endpoint"`
}

// getOpenIdConfiguration метод в котором получаем эндпоинты для получения токена пользователя и завершения сессии в кейклоак
func getOpenIdConfiguration(confUrl string) (*OpenIdConfiguration, error) {
	httpClient := &http.Client{Timeout: 2 * time.Second}

	req, err := http.NewRequest(http.MethodGet, confUrl, nil)
	if err != nil {
		return nil, err
	}

	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		log.Fatal("Error has occurred while getting openId configuration from: %s", confUrl)
	}

	var config OpenIdConfiguration
	err = json.NewDecoder(res.Body).Decode(&config)

	if err != nil {
		log.Fatal("Error has occurred while getting openId configuration from: %s", confUrl)
	}

	return &config, nil
}
