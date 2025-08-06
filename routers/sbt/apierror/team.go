package apiError

// 400 ошибки при работе с командами

// TeamDoesNotExist ошибка в случае если команда не найдена
func TeamDoesNotExist() ApiError {
	return ApiError{Code: 3200, Message: "Team does not exist"}
}
