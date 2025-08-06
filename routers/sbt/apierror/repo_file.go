package apiError

import "fmt"

// 400 ошибки при работе с файлами репозитория

// InvalidFilename ошибка в случае если имя файло не валидно
func InvalidFilename(name string) ApiError {
	return ApiError{Code: 4300, Message: fmt.Sprintf("Filename %s is invalid", name)}
}

// FileSHANotMatch ошибка в случае если SHA файла не совпадают
func FileSHANotMatch(name string) ApiError {
	return ApiError{Code: 4301, Message: fmt.Sprintf("SHA of file %s does not match", name)}
}

// CorruptedFileContent ошибка в случае если содержимое файла не верного формата, должна быть строка в base64
func CorruptedFileContent() ApiError {
	return ApiError{Code: 4302, Message: fmt.Sprintf("File content is corrupted, must be a valid string in base64")}
}

// InvalidFilePath  ошибка в случае если путь файла не валидный
func InvalidFilePath(path string) ApiError {
	return ApiError{Code: 4303, Message: fmt.Sprintf("File path %s is invalid", path)}
}

// FileAlreadyExist ошибка в случае если файл уже существует
func FileAlreadyExist(name string) ApiError {
	return ApiError{Code: 4304, Message: fmt.Sprintf("File %s already exist", name)}
}

// FileNotFound ошибка в случае если файл не найден
func FileNotFound(filepath string) ApiError {
	return ApiError{Code: 4305, Message: fmt.Sprintf("File: %s not found", filepath)}
}
