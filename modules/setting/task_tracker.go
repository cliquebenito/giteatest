package setting

import (
	"strings"

	vault_model "code.gitea.io/gitea/models/vault_client"
	"code.gitea.io/gitea/modules/log"
)

// TaskTracker настройки интеграции
var TaskTracker struct {
	Enabled bool

	APIBaseURL     string
	APIToken       string
	UnitBaseURL    string
	IAMUnitBaseURL string

	UnitsValidationEnabled         bool
	UnitLinksSenderIntervalSeconds int64
	GetCredFor                     GetCredSecMan
}

// //go:generate mockery --name=GetCredSecMan --exported
type GetCredSecMan interface {
	GetCredFromSecManByVersionKey(configForKvGet *vault_model.KeyValueConfigForGetSecrets) (*vault_model.SecretVaultResponse, error)
}

// loadTaskTracker подтягивает настройки из конфигурационного файла
func loadTaskTracker(rootCfg ConfigProvider) {
	sec := rootCfg.Section("sourcecontrol.tasktracker")

	TaskTracker.Enabled = sec.Key("TASK_TRACKER_ENABLED").MustBool(false)

	if TaskTracker.Enabled && SourceControl.Enabled {
		TaskTracker.UnitsValidationEnabled = sec.Key("UNITS_VALIDATION_ENABLED").MustBool(true)
		TaskTracker.UnitLinksSenderIntervalSeconds = sec.Key("UNIT_LINKS_SENDER_INTERVAL_SECONDS").MustInt64(300)

		TaskTracker.APIBaseURL = sec.Key("API_BASE_URL").MustString("")
		if len(TaskTracker.APIBaseURL) == 0 {
			log.Fatal("API_BASE_URL can't be blank")
		}

		TaskTracker.UnitBaseURL = sec.Key("UNIT_BASE_URL").MustString("")
		if len(TaskTracker.UnitBaseURL) == 0 {
			log.Fatal("UNIT_BASE_URL can't be blank")
		}

		TaskTracker.IAMUnitBaseURL = sec.Key("IAM_UNIT_BASE_URL").MustString("")
		if IAM.Enabled && OneWork.Enabled && len(TaskTracker.UnitBaseURL) == 0 {
			log.Fatal("IAM_UNIT_BASE_URL can't be blank")
		}

		tokenString := sec.Key("API_TOKEN").String()
		if CheckSettingsForIntegrationWithSecMan() {
			configForKvGet := &vault_model.KeyValueConfigForGetSecrets{
				SecretPath:  strings.TrimSpace(SourceControlTaskTracker.SecretPath),
				StoragePath: strings.TrimSpace(SourceControlTaskTracker.StoragePath),
				VersionKey:  SourceControlTaskTracker.VersionKey,
			}
			if TaskTracker.GetCredFor == nil {
				TaskTracker.GetCredFor = NewGetterForSecMan()
			}
			resp, err := TaskTracker.GetCredFor.GetCredFromSecManByVersionKey(configForKvGet)
			if err != nil {
				log.Fatal("loadTaskTracker failed when we tried to get cred from sec man: %v", err)
			}
			if GetResponseNotNil(resp) && resp.Data[SourceControlTaskTracker.APIToken] != "" {
				tokenString = strings.TrimSpace(resp.Data[SourceControlTaskTracker.APIToken])
			} else {
				log.Fatal("API_TOKEN not found in Vault storage")
			}
		}
		if len(tokenString) == 0 {
			log.Fatal("API_TOKEN can't be blank")
		}
		TaskTracker.APIToken = tokenString
	}
}
