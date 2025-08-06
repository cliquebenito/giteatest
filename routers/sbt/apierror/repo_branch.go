package apiError

import "fmt"

// 400 ошибки при работе с ветками репозитория

// BranchAlreadyExist ошибка в случае если ветка с таким именем уже существует в репозитории
func BranchAlreadyExist(name string) ApiError {
	return ApiError{Code: 4200, Message: fmt.Sprintf("Branch %s already exist", name)}
}

// BranchNotExist ошибка в случае если ветка не существует в репозитории
func BranchNotExist(name string) ApiError {
	return ApiError{Code: 4201, Message: fmt.Sprintf("Branch %s not exist", name)}
}

// BranchIsDefault ошибка в случае если ветка дефолтовая
func BranchIsDefault(name string) ApiError {
	return ApiError{Code: 4202, Message: fmt.Sprintf("Branch %s is default", name)}
}

// BranchIsProtected ошибка в случае если ветка защищена
func BranchIsProtected(name string) ApiError {
	return ApiError{Code: 4203, Message: fmt.Sprintf("Branch %s is protected", name)}
}

// BranchesAreIdentical ошибка в случае если ветки содержат одинаковый контент
func BranchesAreIdentical() ApiError {
	return ApiError{Code: 4204, Message: "There is no difference between the head and the base"}
}
