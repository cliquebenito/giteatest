package vault_client

import (
	"net/http"
	"time"
)

// LoginPayload roleID и secretID для логина в sec man через AppRole
type LoginPayload struct {
	RoleID   string `json:"role_id"`
	SecretID string `json:"secret_id"`
}

// ConfigForWrapToken содержит всю необходимую информацию для подключения к sec man
type ConfigForWrapToken struct {
	RoleID           string `json:"role_id"`
	URL              string `json:"url"`
	Namespace        string `json:"namespace"`
	WrappedTokenFile string `json:"wrapped_token_file"`
	PeriodTime       int    `json:"period_time"`
	TtlWrapToken     int    `json:"ttl_wrap_token"`
	HttpClient       http.Client
}

// WrappedConfig wrappedSecretID поле содержащее token для получения cred из sec man
type WrappedConfig struct {
	WrappedSecretID string `json:"wrapped_secret_id"`
}

// UnwrapResponse ответ при unwrap token в sec man
type UnwrapResponse struct {
	DefaultResponse
	Data map[string]string `json:"data"`
}

// DefaultResponse ответ от sec man
type DefaultResponse struct {
	RequestId     string            `json:"request_id"`
	LeaseId       string            `json:"lease_id"`
	LeaseDuration int               `json:"lease_duration"`
	Renewable     bool              `json:"renewable"`
	Data          map[string]string `json:"data"`
	Warnings      interface{}       `json:"warnings"`
	WrapInfo      *WrapInfo         `json:"wrap_info"`
	Auth          *Auth             `json:"auth"`
}

// Auth информация об аунтефикации в sec man
type Auth struct {
	ClientToken   string   `json:"client_token"`
	Accessor      string   `json:"accessor"`
	Policies      []string `json:"policies"`
	TokenPolicies []string `json:"token_policies"`
	Metadata      struct {
		RoleName string `json:"role_name"`
		Tag1     string `json:"tag1"`
	} `json:"metadata"`
	LeaseDuration  int         `json:"lease_duration"`
	Renewable      bool        `json:"renewable"`
	EntityId       string      `json:"entity_id"`
	TokenType      string      `json:"token_type"`
	Orphan         bool        `json:"orphan"`
	MfaRequirement interface{} `json:"mfa_requirement"`
	NumUses        int         `json:"num_uses"`
}

// WrapInfo информация при wrap token в sec man
type WrapInfo struct {
	Token        string    `json:"token"`
	Ttl          int       `json:"ttl"`
	CreationTime time.Time `json:"creation_time"`
	CreationPath string    `json:"creation_path"`
}

// SecretVaultResponse ответ при получении cred из KV хранилища в sec man
type SecretVaultResponse struct {
	DefaultResponse
	Data map[string]string `json:"data"`
}

// KeyValueConfigForGetSecrets пути для подкючения к директории в sec man
type KeyValueConfigForGetSecrets struct {
	// StoragePath часть url  для директории с keyValue хранилищем
	StoragePath string
	// SecretPath конкретная директория
	SecretPath string
	// VersionKey версия для получения cred из sec man
	VersionKey int
}
