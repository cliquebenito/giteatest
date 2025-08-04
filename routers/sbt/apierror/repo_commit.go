package apiError

import "fmt"

// 400 ошибки при работе с файлами репозитория

// CommitNotExist ошибка в случае если коммит не найден в репозитории
func CommitNotExist(relPath string) ApiError {
	return ApiError{Code: 4400, Message: fmt.Sprintf("Commit %s not exist", relPath)}
}

// CommitIDDoesNotMatch ошибка в случае если id коммита не совпадает
func CommitIDDoesNotMatch(id string) ApiError {
	return ApiError{Code: 4401, Message: fmt.Sprintf("Commit id %s does not match", id)}
}

func SHAOrCommitIDNotProvided() ApiError {
	return ApiError{Code: 4402, Message: "SHA or commit ID must be proved"}
}
