package apiError

import "fmt"

//400 ошибки при работе с аватарами

// DecodeBase64Error Ошибка в случае если не удалось декодировать формат Base64
func DecodeBase64Error() ApiError {
	return ApiError{Code: 4700, Message: "Can not decode base64"}
}

// DecodeImageConfigError Ошибка в случае если не удалось декодировать конфигурацию картинки
func DecodeImageConfigError() ApiError {
	return ApiError{Code: 4701, Message: "Can not decode image's config"}
}

// NotValidImageSize Ошибка в случае если размеры аватара не валидного размера
func NotValidImageSize(height int, width int) ApiError {
	return ApiError{Code: 4702, Message: fmt.Sprintf("Image must be less than height: %d and width: %d", height, width)}
}
