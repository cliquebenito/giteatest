package apiError

import (
	"fmt"
)

// 400 ошибки при работе с пользователями

// WrongEmailError ошибка в случае если в настройках приложения указаны whitelist и/или blocklist для почтовых доменов
// и полученный в запросе email не содержится в whitelist и/или содержится в blocklist
func WrongEmailError(email string) ApiError {
	return ApiError{Code: 3000, Message: fmt.Sprintf("Wrong email: %s", email)}
}

// WrongPasswordError ошибка в случае если полученный в запросе пароль не соответствует требованиям [password.IsComplexEnough]
func WrongPasswordError(passwordComplexity string) ApiError {
	return ApiError{Code: 3001, Message: fmt.Sprintf("Password must contain: %s", passwordComplexity)}
}

// UserNameAlreadyExistError ошибка в случае если пользователь с таким именем уже существует
func UserNameAlreadyExistError(name string) ApiError {
	return ApiError{Code: 3002, Message: fmt.Sprintf("User with name: %s already exist", name)}
}

// EmailAlreadyExistError ошибка в случае если почтовый адрес уже зарегистрирован
func EmailAlreadyExistError(email string) ApiError {
	return ApiError{Code: 3003, Message: fmt.Sprintf("Email: %s already exist", email)}
}

// EmailContainsUnsupportedCharsError ошибка в случае если почтовый адрес содержит не поддерживаемые символы
func EmailContainsUnsupportedCharsError(email string) ApiError {
	return ApiError{Code: 3004, Message: fmt.Sprintf("Email: %s is invalid", email)}
}

// EmailInvalidError ошибка в случае если почтовый адрес не соответствует RFC 5322 или имеет символ '-' в начале
func EmailInvalidError(email string) ApiError {
	return ApiError{Code: 3005, Message: fmt.Sprintf("Email: %s is invalid", email)}
}

// UserNameReservedError ошибка в случае если полученное в запросе имя пользователя зарезервировано
func UserNameReservedError(name string) ApiError {
	return ApiError{Code: 3006, Message: fmt.Sprintf("Username: %s is reserved", name)}
}

// UserNamePatternNotAllowedError ошибка в случае если полученное в запросе имя пользователя совпадает с запрещенным паттерном
func UserNamePatternNotAllowedError(name string) ApiError {
	return ApiError{Code: 3007, Message: fmt.Sprintf("Username: %s is not allowed by pattern: %s", name, "*.keys\", \"*.gpg\", \"*.rss\", \"*.atom\", \"*.png\"")}
}

// UserNameHasNotAllowedCharsError ошибка в случае если полученное в запросе имя пользователя содержит не разрешенные символы
func UserNameHasNotAllowedCharsError(name string) ApiError {
	return ApiError{Code: 3008, Message: fmt.Sprintf("User name: %s is invalid, must be valid alpha or numeric or dash(-_) or dot characters", name)}
}

// LoginOrPasswordNotValidError ошибка аутентификации пользователя
func LoginOrPasswordNotValidError() ApiError {
	return ApiError{Code: 3009, Message: "Login or password not valid"}
}

// UserProhibitedLoginError ошибка в случае если вход для пользователя запрещен
func UserProhibitedLoginError() ApiError {
	return ApiError{Code: 3010, Message: "User is prohibited to login"}
}

// UserEmailNotConfirmedError ошибка в случае если включен режим обязательного подтверждения почты и пользователь ее не подтвердил
func UserEmailNotConfirmedError() ApiError {
	return ApiError{Code: 3011, Message: "User email is not confirmed"}
}

// UserNotFoundByNameError ошибка в случае если не найден пользователь с указанным именем
func UserNotFoundByNameError(name string) ApiError {
	return ApiError{Code: 3012, Message: fmt.Sprintf("User with name: %s is not found", name)}
}

// UserNotOrganization ошибка в случае ожидаемый пользователь должен быть организацией
func UserNotOrganization() ApiError {
	return ApiError{Code: 3013, Message: "User is not an organization"}
}

// UserIsNotOwner ошибка в случае если пользователь не является владельцем репозитория
func UserIsNotOwner() ApiError {
	return ApiError{Code: 3014, Message: "User is not owner of the repository or organization"}
}

// UserInsufficientPermission ошибка в случае если у пользователя недостаточно прав для выполнения операции
func UserInsufficientPermission(name string, action string) ApiError {
	return ApiError{Code: 3015, Message: fmt.Sprintf("User: %s should have a permission to perform action: %s", name, action)}
}

// UserNotActivatedError ошибка в случае если аккаунт не активирован
func UserNotActivatedError() ApiError {
	return ApiError{Code: 3028, Message: "User is not activated"}
}

// UserMustChangePasswordError ошибка в случае если пользователь должен сменить пароль
func UserMustChangePasswordError() ApiError {
	return ApiError{Code: 3029, Message: "User must change password"}
}

// UserNotAdminError ошибка в случае если нет прав администратора
func UserNotAdminError() ApiError {
	return ApiError{Code: 3030, Message: "User does not have admin privileges"}
}

// UserIsAlreadyOwner ошибка в случае если пользователь владелец репозитория
func UserIsAlreadyOwner() ApiError {
	return ApiError{Code: 3031, Message: "User is already owner"}
}

// UserIsOrganization ошибка в случае если ожидаемый пользователь является организацией
func UserIsOrganization() ApiError {
	return ApiError{Code: 3032, Message: "User is an organization"}
}

// UserIsAlreadyCollaborator ошибка в случае если пользователь уже совладелец
func UserIsAlreadyCollaborator() ApiError {
	return ApiError{Code: 3033, Message: "User is already collaborator"}
}

// UserIsNotRepoAdmin ошибка в случае если пользователь не является администратором репозитория
func UserIsNotRepoAdmin() ApiError {
	return ApiError{Code: 3034, Message: "User is not admin of the repository"}
}

//todo переработать последовательность кодов - получилась каша
//-------------------- Ошибки при работе с ssh ключами пользователя ---------------------------------

// InvalidSshKey ошибка в случае если не валидный ssh ключ
func InvalidSshKey(message string) ApiError {
	return ApiError{Code: 3016, Message: message}
}

// SshKeyAlreadyExist ошибка в случае если такой ssh ключ уже существует
func SshKeyAlreadyExist() ApiError {
	return ApiError{Code: 3017, Message: "SSH key already exist"}
}

// SshKeyNameAlreadyExist ошибка в случае если ssh ключ с таким именем уже существует
func SshKeyNameAlreadyExist() ApiError {
	return ApiError{Code: 3018, Message: "SSH key title already exist"}
}

// SshKeyUnableVerify ошибка в случае когда ssh ключ не удается проверить
func SshKeyUnableVerify() ApiError {
	return ApiError{Code: 3019, Message: "Can not verify the SSH key, double-check it for mistakes"}
}

// SshKeyNotExist ошибка в случае если ssh ключ не доступен этому пользователю
func SshKeyNotExist(keyId int64) ApiError {
	return ApiError{Code: 3020, Message: fmt.Sprintf("SSH key with id: %d is not exist", keyId)}
}

// SshKeyExternallyManaged ошибка в случае если SSH ключ управляется извне для этого пользователя
func SshKeyExternallyManaged() ApiError {
	return ApiError{Code: 3021, Message: "This SSH key is externally managed for user"}
}

// SshKeyUserIsNotOwner ошибка в случае если ssh ключ принадлежит другому пользователю
func SshKeyUserIsNotOwner(keyId int64) ApiError {
	return ApiError{Code: 3022, Message: fmt.Sprintf("User is not owner of SSH key with id: %d", keyId)}
}

//-------------------- Ошибки при попытке удалить аккаунт пользователя ---------------------------------

// UserHasReposError ошибка в случае попытки удалить аккаунт, но у пользователя все еще есть репозитории
func UserHasReposError() ApiError {
	return ApiError{Code: 3023, Message: "Account owns one or more repositories, delete or transfer them first."}
}

// UserHasOrgsError ошибка в случае попытки удалить аккаунт, но пользователь состоит в организации
func UserHasOrgsError() ApiError {
	return ApiError{Code: 3024, Message: "User is a member of an organization. Remove the user from any organizations first."}
}

// UserHasPackagesError ошибка в случае попытки удалить аккаунт, но у пользователя все еще есть пакеты
func UserHasPackagesError() ApiError {
	return ApiError{Code: 3025, Message: "User still owns one or more packages, delete these packages first."}
}

// UserIsNotPartOfOrganizationError ошибка в случае если пользователь не входит в организацию
func UserIsNotPartOfOrganizationError(userName string, orgName string) ApiError {
	return ApiError{Code: 3026, Message: fmt.Sprintf("User: %s is not part of organization: %s ", userName, orgName)}
}

// UserNotAllowedCreateOrgsError ошибка в случае если у пользователя нет прав на создание организации
func UserNotAllowedCreateOrgsError() ApiError {
	return ApiError{Code: 3027, Message: "User not allowed create organization."}
}

// UserDoesNotExist ошибка в случае если пользователь не найден
func UserDoesNotExist() ApiError {
	return ApiError{Code: 3035, Message: "User does not exist"}
}
