package asymkey

import "code.gitea.io/gitea/models/db"

// GetPublicKeyByIDAndOwnerId возвращает публичный ssh ключ по идентификатору пользователя и ключа
func GetPublicKeyByIDAndOwnerId(keyID int64, ownerId int64) (*PublicKey, error) {
	key := new(PublicKey)
	has, err := db.GetEngine(db.DefaultContext).
		Where("id = ? AND owner_id = ?", keyID, ownerId).
		Get(key)
	if err != nil {
		return nil, err
	} else if !has {
		return nil, ErrKeyNotExist{ID: keyID}
	}
	return key, nil
}
