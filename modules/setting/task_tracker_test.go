//go:build !correct

package setting

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	vault_model "code.gitea.io/gitea/models/vault_client"
	"code.gitea.io/gitea/modules/setting/mocks"
	"code.gitea.io/gitea/services/vault_client"
)

func Test_loadTaskTracker(t *testing.T) {
	t.Run("BuggyKeyOverwritten", func(t *testing.T) {
		cfg, _ := NewConfigProviderFromData(`
[sourcecontrol]
ENABlED = true
[sourcecontrol.wrap.vault]
VAULT_ENABLED = true
[sourcecontrol.vault.task_tracker]
API_TOKEN = token
STORAGE_PATH = A/DEV/TEST/KV
SECRET_PATH = task_tracker
VERSION_KEY = 1
[sourcecontrol.tasktracker]
TASK_TRACKER_ENABLED = true
API_BASE_URL = https://dev.pd10.pvw.sbt/swtr/test/extension/plugin/v2/rest/api/swtr_task_tracker_plugin/v1
UNIT_BASE_URL = https://dev.pd10.pvw.sbt/swtr/test/units/all/unit
IAM_UNIT_BASE_URL = https://ift.pd10.pvw.sbt/swtr/test/units/all/unit
UNITS_VALIDATION_ENABLED = true
UNIT_LINKS_SENDER_INTERVAL_SECONDS = 10
API_TOKEN = eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ2.eyJncm23cHMiOlsiUk9MRV7TT1VSQ0VfQ09OVFJPTF9VU0VSIl0sInByZWZlcnJlZF91c2VybmFtZSI6IlNPVVJDRV9DT05UUk1MX4AA

`)
		sourceControl, err := cfg.Section("sourcecontrol").Key("ENABlED").Bool()
		assert.NoError(t, err)
		assert.Equal(t, true, sourceControl)
		SourceControl.Enabled = sourceControl

		vaultEnabled, err := cfg.Section("sourcecontrol.wrap.vault").Key("VAULT_ENABLED").Bool()
		assert.NoError(t, err)
		assert.Equal(t, true, vaultEnabled)
		SourceControlWrapVault.Enabled = vaultEnabled

		sec := cfg.Section("sourcecontrol.vault.task_tracker")
		apiToken := sec.Key("API_TOKEN").MustString("")
		assert.Equal(t, "token", apiToken)
		SourceControlTaskTracker.APIToken = apiToken

		storagePath := sec.Key("STORAGE_PATH").MustString("")
		assert.Equal(t, "A/DEV/TEST/KV", storagePath)
		SourceControlTaskTracker.StoragePath = storagePath

		secretPath := sec.Key("SECRET_PATH").MustString("")
		assert.Equal(t, "task_tracker", secretPath)
		SourceControlTaskTracker.SecretPath = secretPath

		versionKey := sec.Key("VERSION_KEY").MustInt(0)
		assert.Equal(t, 1, versionKey)
		SourceControlTaskTracker.VersionKey = versionKey

		sec = cfg.Section("sourcecontrol.tasktracker")
		sec.Key("TASK_TRACKER_ENABLED").MustBool(false)
		taskTrackerEnabled, err := sec.Key("TASK_TRACKER_ENABLED").Bool()
		assert.NoError(t, err)
		assert.Equal(t, true, taskTrackerEnabled)
		taskTrackerIAMUnitBaseURL := sec.Key("IAM_UNIT_BASE_URL").String()
		assert.Equal(t, "https://ift.pd10.pvw.sbt/swtr/test/units/all/unit", taskTrackerIAMUnitBaseURL)

		SourceControlVaultClient = &vault_client.VaultClient{
			URL:         "http://localhost",
			ClientToken: "token",
			Namespace:   "test",
		}
		configForKvGet := &vault_model.KeyValueConfigForGetSecrets{
			SecretPath:  strings.TrimSpace(SourceControlTaskTracker.SecretPath),
			StoragePath: strings.TrimSpace(SourceControlTaskTracker.StoragePath),
			VersionKey:  SourceControlTaskTracker.VersionKey,
		}
		getSecretMan := mocks.NewGetCredSecMan(t)
		getSecretMan.On("GetCredFromSecManByVersionKey", configForKvGet).
			Return(&vault_model.SecretVaultResponse{
				DefaultResponse: vault_model.DefaultResponse{},
				Data: map[string]string{
					"token": "eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ2.eyJncm23cHMiOlsiUk9MRV7TT1VSQ0VfQ09OVFJPTF9VU0VSIl0sInByZWZlcnJlZF91c2VybmFtZSI6IlNPVVJDRV9DT05UUk1MX4AA",
				},
			}, nil)
		TaskTracker.GetCredFor = getSecretMan
		loadTaskTracker(cfg)
	})
}
