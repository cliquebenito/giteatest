package apiError

// KeycloakUserWasNotRegistered ошибка в случае ошибки регистрации пользователя в Keycloak
func KeycloakUserWasNotRegistered() ApiError {
	return ApiError{Code: 4800, Message: "User was not created in Keycloak"}
}

// KeycloakUserAlreadyExist ошибка в случае ошибки регистрации пользователя в Keycloak,
// потому что пользователь с таким именем или почтой уже существует в Keycloak
func KeycloakUserAlreadyExist() ApiError {
	return ApiError{Code: 4801, Message: "User already exist in Keycloak"}
}

// KeycloakUserWasNotAuthenticate ошибка в случае ошибки аутентификации пользователя в Keycloak
func KeycloakUserWasNotAuthenticate() ApiError {
	return ApiError{Code: 4802, Message: "User was not authenticate in Keycloak"}
}

// KeycloakUserWasNotLogout ошибка в случае ошибки закрытия сессии пользователя в Keycloak
func KeycloakUserWasNotLogout() ApiError {
	return ApiError{Code: 4803, Message: "User was not logout from Keycloak"}
}
