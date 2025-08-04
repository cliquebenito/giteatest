package request

/*
CreateUserSshKey - структура запроса на добавление публичного ssh-ключа в настройках пользователя
*/
type CreateUserSshKey struct {
	// Title of the key to add
	Title string `json:"title" binding:"Required;SbtMaxSize(255)"`
	// SSH key to add
	Key string `json:"key" binding:"Required"`
}
