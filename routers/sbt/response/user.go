package response // User represents a user
import "time"

type User struct {
	// Идентификатор
	ID int64 `json:"id"`
	// Имя пользователя
	UserName string `json:"login"`
	// Логин
	LoginName string `json:"login_name"`
	// Полное имя
	FullName string `json:"full_name"`
	// Почта
	Email string `json:"email"`
	// Адрес аватарки
	AvatarURL string `json:"avatar_url"`
	// Локаль (язык)
	Language string `json:"language"`
	// Есть ли права админа
	IsAdmin bool `json:"is_admin"`
	// Последний вход в систему
	LastLogin time.Time `json:"last_login,omitempty"`
	// Дата создания учетной записи
	Created time.Time `json:"created,omitempty"`
	// Есть ли ограничения
	Restricted bool `json:"restricted"`
	// Активен ил
	IsActive bool `json:"active"`
	// Запрещен ли вход
	ProhibitLogin bool `json:"prohibit_login"`
	// Местоположение
	Location string `json:"location"`
	// вебсайт
	Website string `json:"website"`
	// О себе
	Description string `json:"description"`
	// Настройки приватности: public, limited, private
	Visibility string `json:"visibility"`

	// счетчики
	Followers    int `json:"followers_count"`
	Following    int `json:"following_count"`
	StarredRepos int `json:"starred_repos_count"`
}

// UserListResults результаты поиска пользователей
type UserListResults struct {
	Total int64   `json:"total"`
	Data  []*User `json:"data"`
}
