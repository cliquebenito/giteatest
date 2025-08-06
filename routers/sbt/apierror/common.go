package apiError

/*
ApiError стандартный ответ при возникновени ошибки обработки REST запроса
https://dzo.sw.sbc.space/wiki/pages/viewpage.action?pageId=160430062
*/
type ApiError struct {
	Code            int16             `json:"code"`
	Message         string            `json:"message"`
	ValidationError []ValidationError `json:"validationError,omitempty"`
}

type ValidationError struct {
	FieldName    string `json:"field"`
	ErrorMessage string `json:"error"`
}

// HTTP 500

// InternalServerError Внутренняя ошибка сервиса
func InternalServerError() ApiError {
	return ApiError{Code: 1000, Message: "Internal Server Error"}
}

// BranchWasNotDeletedInternalServerError Ошибка в случае если пр был смерджен, а ветка не была удалена из-за внутренней ошибки сервиса
func BranchWasNotDeletedInternalServerError() ApiError {
	return ApiError{Code: 1001, Message: "Branch was not deleted because Internal Server Error"}
}

// CommentWasNotAdded Ошибка в случае если статус пулл реквеста был изменен, но сопутствующий комментарий не был добавлен
func CommentWasNotAdded() ApiError {
	return ApiError{Code: 1002, Message: "Comment was not added because Internal Server Error"}
}

// Ошибки авторизации HTTP 401

// UserUnauthorized не авторизован
func UserUnauthorized() ApiError {
	return ApiError{Code: 5000, Message: "Unauthorized"}
}

// PullRequestIsLocked пулл реквест заблокирован для редактирования
func PullRequestIsLocked() ApiError {
	return ApiError{Code: 5001, Message: "Pull request is locked"}
}

//HTTP 400

//Request validation error

// RequestFieldValidationError ошибка валидации полей запроса
func RequestFieldValidationError(message string, validationError []ValidationError) ApiError {
	return ApiError{Code: 2000, Message: message, ValidationError: validationError}
}

// ProofOfWorkValidation ошибка в случае если запрос не прошел Proof-Of-Work валидацию
func ProofOfWorkValidation() ApiError {
	return ApiError{Code: 2500, Message: "Request validation failed"}
}
