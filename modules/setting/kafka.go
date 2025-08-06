package setting

import (
	"strconv"
	"strings"
	"time"

	vault_model "code.gitea.io/gitea/models/vault_client"
	"code.gitea.io/gitea/modules/log"
)

const KafkaRepositoryTopic = "kafka.repository"

// Kafka настройки
var Kafka struct {
	// Enabled Активирована ли Kafka
	Enabled bool
	// Address адрес Kafka сервера
	Address string
	// Port порт Kafka сервера
	Port int
	//URL Kafka сервера
	URL string
	//Issuer Название системы отправителя сообщений
	Issuer string
	//AuthEnabled Активирована ли авторизация
	AuthEnabled bool
	//Certificate сертификат
	Certificate string
	//PrivateKey приватный ключ
	PrivateKey string
	//CARootCertificate ca root сертификат
	CARootCertificate string
	//ConnectRetries количество попыток подключения к серверу
	ConnectRetries int
	//ConnectBackoff время задержки перед повторной попыткой подключения к серверу
	ConnectBackoff time.Duration
	//Topics список топиков
	Topics       map[string]TopicConfig
	secManGetter GetCredSecMan
}

// SourceControlKafkaAuth настройки авторизации
var SourceControlKafkaAuth struct {
	// Certificate сертификат
	Certificate string
	//PrivateKey приватный ключ
	PrivateKey string
	//CARootCertificate ca root сертификат
	CARootCertificate string
	// StoragePath путь к директории с secrets в vault хранилище
	StoragePath string
	// SecretPaths пути к директориям в vault хранилище
	SecretPath string
	// VersionKey версия получения cred
	VersionKey int
}

// TopicConfig настройки топика
type TopicConfig struct {
	// Enabled Активировано ли откидлывание событий в топик
	Enabled bool
	// Name название топика для событий
	Name string
	// Type тип топика
	Type string
}

// loadKafka загрузить настройки Kafka
func loadKafka(rootCfg ConfigProvider) {
	sec := rootCfg.Section("kafka")

	Kafka.Enabled = sec.Key("ENABLED").MustBool(false)
	if Kafka.Enabled {
		Kafka.Address = sec.Key("ADDRESS").MustString("")
		Kafka.Port = sec.Key("PORT").MustInt(0)
		if Kafka.Address == "" {
			log.Fatal("Kafka address is empty")
		} else if Kafka.Port == 0 {
			log.Fatal("Kafka port is empty")
		}
		Kafka.URL = Kafka.Address + ":" + strconv.Itoa(Kafka.Port)
		Kafka.ConnectRetries = sec.Key("CONNECT_RETRIES").MustInt(10)
		Kafka.ConnectBackoff = sec.Key("RETRY_BACKOFF").MustDuration(3 * time.Second)
		Kafka.AuthEnabled = sec.Key("AUTH_ENABLED").MustBool(false)
		Kafka.Issuer = sec.Key("ISSUER").MustString("sc")

		Kafka.Topics = make(map[string]TopicConfig)
		newRepositoryTopic(rootCfg.Section(KafkaRepositoryTopic))
	}
}

// loadKafkaAuthFromVault загрузить настройки Kafka авторизации из vault
func loadKafkaAuthFromVault(rootCfg ConfigProvider) {
	if SourceControl.Enabled && SourceControlWrapVault.Enabled && Kafka.AuthEnabled {
		vaultSec := rootCfg.Section("sourcecontrol.vault.kafka")
		SourceControlKafkaAuth.Certificate = vaultSec.Key("CERTIFICATE").MustString("")
		SourceControlKafkaAuth.PrivateKey = vaultSec.Key("PRIVATE_KEY").MustString("")
		SourceControlKafkaAuth.CARootCertificate = vaultSec.Key("CA_ROOT_CERTIFICATE").MustString("")
		SourceControlKafkaAuth.StoragePath = vaultSec.Key("STORAGE_PATH").MustString("")
		SourceControlKafkaAuth.SecretPath = vaultSec.Key("SECRET_PATH").MustString("")
		SourceControlKafkaAuth.VersionKey = vaultSec.Key("VERSION_KEY").MustInt(0)

		loadKafkaAuth()
	}
}

// newRepositoryTopic создать топик для репозитория
func newRepositoryTopic(kafkaRepositorySec ConfigSection) {
	if !kafkaRepositorySec.HasKey("TOPIC_ENABLED") {
		log.Fatal("Kafka repository not configured")
	}

	Kafka.Topics[KafkaRepositoryTopic] = TopicConfig{
		kafkaRepositorySec.Key("TOPIC_ENABLED").MustBool(false),
		kafkaRepositorySec.Key("TOPIC").MustString(""),
		kafkaRepositorySec.Key("TYPE").MustString("multiple"),
	}
}

// loadKafkaAuth загрузить настройки Kafka авторизации
func loadKafkaAuth() {
	if CheckSettingsForIntegrationWithSecMan() {
		configForKvGet := &vault_model.KeyValueConfigForGetSecrets{
			SecretPath:  strings.TrimSpace(SourceControlKafkaAuth.SecretPath),
			StoragePath: strings.TrimSpace(SourceControlKafkaAuth.StoragePath),
			VersionKey:  SourceControlKafkaAuth.VersionKey,
		}
		Kafka.secManGetter = NewGetterForSecMan()
		resp, err := Kafka.secManGetter.GetCredFromSecManByVersionKey(configForKvGet)
		if err != nil {
			log.Fatal("Error has occurred while trying to get cred from secret storage: %v", err)
		}

		if GetResponseNotNil(resp) {
			Kafka.Certificate = resp.Data[SourceControlKafkaAuth.Certificate]
			CheckIfSecretIsEmptyAndReportToAudit("CERTIFICATE", Kafka.Certificate, "CERTIFICATE to kafka is empty in secret storage", log.Fatal)

			Kafka.PrivateKey = resp.Data[SourceControlKafkaAuth.PrivateKey]
			CheckIfSecretIsEmptyAndReportToAudit("PRIVATE_KEY", Kafka.PrivateKey, "PRIVATE_KEY to kafka is empty in secret storage", log.Fatal)

			Kafka.CARootCertificate = resp.Data[SourceControlKafkaAuth.CARootCertificate]
			CheckIfSecretIsEmptyAndReportToAudit("CA_ROOT_CERTIFICATE", Kafka.CARootCertificate, "CA_ROOT_CERTIFICATE to kafka is empty in secret storage", log.Warn)
		} else {
			CheckIfSecretIsEmptyAndReportToAudit("CERTIFICATE", "", "Response from secret storage is nil", log.Fatal)
		}
	} else {
		log.Fatal("Kafka auth not configured, but enabled")
	}
}
