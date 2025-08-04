package ssh

// CreateSSHKeyRequest структура запроса создания ssh ключа
type CreateSSHKeyRequest struct {
	Title string `json:"title" binding:"Required"`
	Key   string `json:"key" binding:"Required"`
}

// CreateSSHKeyResponse структура ответа создания ssh ключа
type CreateSSHKeyResponse struct {
	ID          int64  `json:"id"`
	Fingerprint string `json:"fingerprint"`
	KeyType     string `json:"key_type"`
}

// SSHKeyOptions структура опций ssh ключа
type SSHKeyOptions struct {
	UserKey string `json:"user_key" binding:"Required"`
	Title   string `json:"title" binding:"Required"`
}
