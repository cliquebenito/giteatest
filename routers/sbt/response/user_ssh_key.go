package response

import (
	"code.gitea.io/gitea/models/asymkey"
	"time"
)

/*
PublicSshKey структура ответа публичного ssh-ключа пользователя
*/
type PublicSshKey struct {
	ID          int64     `json:"id"`
	Key         string    `json:"key"`
	Title       string    `json:"title"`
	Fingerprint string    `json:"fingerprint"`
	Created     time.Time `json:"created_at"`
	LastUsageAt time.Time `json:"last_usage_at"`
}

/*
PublicSshKeyMapper маппер для PublicSshKey из asymkey.PublicKey
*/
func PublicSshKeyMapper(key *asymkey.PublicKey) PublicSshKey {
	return PublicSshKey{
		ID:          key.ID,
		Key:         key.Content,
		Fingerprint: key.Fingerprint,
		Title:       key.Name,
		Created:     key.CreatedUnix.AsTime(),
		LastUsageAt: key.UpdatedUnix.AsTime(),
	}
}
