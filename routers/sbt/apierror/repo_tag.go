package apiError

import "fmt"

//400 ошибки при работе с тегами

// TagAlreadyExist ошибка в случае если тег с таким именем уже существует в репозитории
func TagAlreadyExist(name string) ApiError {
	return ApiError{Code: 4500, Message: fmt.Sprintf("Tag %s already exist", name)}
}
