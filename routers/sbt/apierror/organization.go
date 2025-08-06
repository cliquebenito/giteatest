package apiError

import (
	"fmt"
)

// 400 ошибки при работе с организациями

// OrgsNameAlreadyExistError ошибка в случае если организация (пользователь) с таким именем уже существует
func OrgsNameAlreadyExistError(name string) ApiError {
	return ApiError{Code: 3100, Message: fmt.Sprintf("Organization with name: %s already exist", name)}
}

// OrgsNameReservedError ошибка в случае создания организации с именем, которое зарезервировано
func OrgsNameReservedError(name string) ApiError {
	return ApiError{Code: 3101, Message: fmt.Sprintf("Organization name: %s is reserved", name)}
}

// OrgsNamePatternNotAllowedError ошибка в случае создания организации с именем, которое совпадает с запрещенным паттерном
func OrgsNamePatternNotAllowedError(name string) ApiError {
	return ApiError{Code: 3102, Message: fmt.Sprintf("Organization name: %s is not allowed by pattern: %s", name, "*.keys\", \"*.gpg\", \"*.rss\", \"*.atom\", \"*.png\"")}
}

// OrgsNameHasNotAllowedCharsError ошибка при создании организации: имя организации содержит не разрешенные символы
func OrgsNameHasNotAllowedCharsError(name string) ApiError {
	return ApiError{Code: 3103, Message: fmt.Sprintf("Organization name: %s is invalid, must be valid alpha or numeric or dash(-_) or dot characters", name)}
}

// OrganizationNotFoundByNameError ошибка в случае если не найдена организация с данным именем
func OrganizationNotFoundByNameError(name string) ApiError {
	return ApiError{Code: 3104, Message: fmt.Sprintf("Organization with name: %s is not found", name)}
}
