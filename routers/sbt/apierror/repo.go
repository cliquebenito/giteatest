package apiError

import (
	repo_module "code.gitea.io/gitea/modules/repository"
	"fmt"
)

//400 ошибки при работе с репо

func ReadmeTemplateNotExists() ApiError {
	return ApiError{Code: 4000, Message: fmt.Sprintf("readme template does not exist, available templates: %v", repo_module.Readmes)}
}

func RepoAlreadyExists() ApiError {
	return ApiError{Code: 4001, Message: "The repository with the same name already exists."}
}

func RepoWrongName() ApiError {
	return ApiError{Code: 4002, Message: "Provided name is reserved name or name pattern is not allowed."}
}

func RepoWrongLabels() ApiError {
	return ApiError{Code: 4003, Message: "Wrong label template."}
}

func RepoDoesNotExist(userName string, repoName string) ApiError {
	return ApiError{Code: 4004, Message: fmt.Sprintf("User: %s doesn't have repository with name: %s", userName, repoName)}
}

// UnprocessableRemoteRepoAddress адресс удаленного репозитория содержит ошибки или не разрешен к клонированию
func UnprocessableRemoteRepoAddress() ApiError {
	return ApiError{Code: 4005, Message: "Unprocessable remote repository address is provided."}
}

// RepoMigrationProhibited миграция репозитория запрещена настройками
func RepoMigrationProhibited(msg string) ApiError {
	return ApiError{Code: 4006, Message: fmt.Sprintf("Migration is prohibited by settings. %s", msg)}
}

// RepoNotEmpty репозиторий уже содержит файлы
func RepoNotEmpty() ApiError {
	return ApiError{Code: 4007, Message: fmt.Sprintf("Files already exist for this repository. Adopt them or delete them.")}
}

// RemoteRepoAuthFailed данные авторизации для подключения к удаленному репозиторию неверны
func RemoteRepoAuthFailed(msg string) ApiError {
	return ApiError{Code: 4008, Message: fmt.Sprintf("Authentication failed: %v.", msg)}
}

// RepoIsEmpty ошибка в случае если пустой репозиторий
func RepoIsEmpty() ApiError {
	return ApiError{Code: 4009, Message: fmt.Sprintf("Repository is empty.")}
}

// RepoIsArchived ошибка в случае если репозиторий архивирован
func RepoIsArchived() ApiError {
	return ApiError{Code: 4010, Message: fmt.Sprintf("Repository is archived.")}
}

// GitReferenceNotExist ошибка в случае если в репозитории не найдена сссылка (ветка/тег/коммит/..)
func GitReferenceNotExist(ref string) ApiError {
	return ApiError{Code: 4011, Message: fmt.Sprintf("No such reference: %s in git repo", ref)}
}

// GitRepoDoesNotExist ошибка в случае если не найден git репозиторий
func GitRepoDoesNotExist(repoPath string) ApiError {
	return ApiError{Code: 4012, Message: fmt.Sprintf("Git repository: %s does not exist", repoPath)}
}

// RepoCountLimitIsReached ошибка в случае если достигнут лимит количества репозиториев
func RepoCountLimitIsReached() ApiError {
	return ApiError{Code: 4013, Message: "Repositories count limit is reached"}
}

// RepoUnknownActionType ошибка в случае если к репозиторию применяется неизвестное действие
func RepoUnknownActionType(action string) ApiError {
	return ApiError{Code: 4014, Message: fmt.Sprintf("Unknown type of repository action: %s", action)}
}

// RepoIsMirror ошибка в случае если репозиторий является зеркалом
func RepoIsMirror() ApiError {
	return ApiError{Code: 4015, Message: "Repository is mirror."}
}

// RepoUnknownSearchMode ошибка в случае если к репозиторию применяется неизвестное действие
func RepoUnknownSearchMode(mode string) ApiError {
	return ApiError{Code: 4016, Message: fmt.Sprintf("Unknown repository search mode: %s", mode)}
}

// RepoTransferInProgress ошибка в случае если репозиторий находится в процессе передачи прав на него
func RepoTransferInProgress() ApiError {
	return ApiError{Code: 4017, Message: "Repository transfer in progress"}
}

// RepoTransferNotInProgress ошибка в случае если репозиторий не в процессе передачи прав на него
func RepoTransferNotInProgress() ApiError {
	return ApiError{Code: 4018, Message: "Repository transfer not in progress"}
}

// ForkAlreadyExist ошибка в случае если форк репозитория уже есть у данного пользователя
func ForkAlreadyExist(repoName string) ApiError {
	return ApiError{Code: 4019, Message: fmt.Sprintf("Fork of repository already exist with name: %s", repoName)}
}

// RepoInMigrationProcess ошибка в случае если репозиторий находится в процессе миграции
func RepoInMigrationProcess() ApiError {
	return ApiError{Code: 4020, Message: "Repository in migration process"}
}

// RepoNotInMigrationProcess ошибка в случае если репозиторий не находится в процессе миграции
func RepoNotInMigrationProcess() ApiError {
	return ApiError{Code: 4021, Message: "Repository not in migration process"}
}

// MigrationRateLimit ошибка в случае если при миграции репозитория с github вернулся ответ 403 с ограничением скорости
func MigrationRateLimit() ApiError {
	return ApiError{Code: 4022, Message: "Repository migration rate limit"}
}

// MigrationTaskDoesNotExist ошибка в случае если задача миграции не найдена
func MigrationTaskDoesNotExist() ApiError {
	return ApiError{Code: 4023, Message: "Repository migration task does not exist"}
}

// UserRepoCreate ошибка в случае если репозиторий создаётся под пользователем
func UserRepoCreate() ApiError {
	return ApiError{Code: 4024, Message: "Creating a repository outside the project is prohibited"}
}
