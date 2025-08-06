package apiError

// AttachmentsNotAllowed запрещено загружать файлы
func AttachmentsNotAllowed() ApiError {
	return ApiError{Code: 4900, Message: "Attachments are not allowed"}
}

// FileTypeNotAllowed тип файла запрещен к загрузке
func FileTypeNotAllowed() ApiError {
	return ApiError{Code: 4901, Message: "File type is not allowed"}
}

// AttachmentNotFound файл не найден
func AttachmentNotFound() ApiError {
	return ApiError{Code: 4902, Message: "Attachment is not found"}
}
