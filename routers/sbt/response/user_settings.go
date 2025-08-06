package response

/*
UserSettings - структура настроек профиля пользователя
*/
type UserSettings struct {
	Name                string `json:"username"`
	FullName            string `json:"full_name"`
	Website             string `json:"website"`
	Location            string `json:"location"`
	Description         string `json:"description"`
	Visibility          string `json:"visibility"`
	KeepEmailPrivate    bool   `json:"hide_email"`
	KeepActivityPrivate bool   `json:"hide_activity"`
}
