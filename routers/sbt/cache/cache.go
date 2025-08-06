package cache

const GroupSeparator = ":"

// GenerateUserKey генерирует ключ кеширования пользователя (проекта), под которым находятся репозитории
func GenerateUserKey(owner string) string {
	return "users" + GroupSeparator + owner
}

// GenerateRepoKey генерирует ключ кеширования репозитория пользователя (проекта)
func GenerateRepoKey(owner string, repoName string) string {
	return "users" + GroupSeparator + owner + GroupSeparator + "repos" + GroupSeparator + repoName
}

// GenerateRepoListKey генерирует ключ кеширования списка репозиториев пользователя (проекта)
func GenerateRepoListKey(owner string) string {
	return "users" + GroupSeparator + owner + GroupSeparator + "repos" + GroupSeparator + "list"
}
