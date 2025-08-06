package apiError

import (
	ctx "code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/validation"
	"code.gitea.io/gitea/routers/sbt/logger"
	"gitea.com/go-chi/binding"
	"net/http"
	"strings"
)

/*
HandleValidationErrors метод обработки ошибок Bind.binding
Ошибки валидации по прописанным тегам для валидации в разделе `binding` и десериализации
В результате выводится ошибка RequestFieldValidationError
*/
func HandleValidationErrors(ctx *ctx.Context, errs binding.Errors, method string) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	var validationError []ValidationError
	var fieldName string
	var message string

	for i := 0; i < len(errs); i++ {
		fieldName = ""
		message = ""

		if len(errs[i].FieldNames) != 0 {
			fieldName = errs[i].FieldNames[0]
		}
		if strings.HasPrefix(errs[i].Message, "json: ") {
			message = "Deserialization error"
		} else {
			message = errs[i].Message
		}

		switch errs[i].Classification {
		case binding.ERR_ALPHA_DASH:
			validationError = append(validationError, ValidationError{
				ErrorMessage: "Should contain only alphanumeric, dash ('-') and underscore ('_') characters.",
				FieldName:    fieldName,
			})
		case binding.ERR_SIZE:
			validationError = append(validationError, ValidationError{
				ErrorMessage: "Must be size.",
				FieldName:    fieldName,
			})
		case binding.ERR_INCLUDE:
			validationError = append(validationError, ValidationError{
				ErrorMessage: "Must contain substring.",
				FieldName:    fieldName,
			})
		case validation.ErrGlobPattern:
			validationError = append(validationError, ValidationError{
				ErrorMessage: "Glob pattern is invalid.",
				FieldName:    fieldName,
			})
		case validation.ErrRegexPattern:
			validationError = append(validationError, ValidationError{
				ErrorMessage: "Regex pattern is invalid.",
				FieldName:    fieldName,
			})
		case validation.ErrUsername:
			validationError = append(validationError, ValidationError{
				ErrorMessage: "Can only contain alphanumeric chars ('0-9','a-z','A-Z'), dash ('-'), underscore ('_') and dot ('.'). It cannot begin or end with non-alphanumeric chars, and consecutive non-alphanumeric chars are also forbidden.",
				FieldName:    fieldName,
			})
		case validation.ErrInvalidGroupTeamMap:
			validationError = append(validationError, ValidationError{
				ErrorMessage: "Mapping is invalid.",
				FieldName:    fieldName,
			})
		default:
			validationError = append(validationError, ValidationError{
				ErrorMessage: message,
				FieldName:    fieldName,
			})
		}
	}

	if ctx.Doer != nil {
		log.Debug("Error has occurred while validate request %s for username: %s with error message: %s", method, ctx.Doer.Name, validationError)
	} else {
		log.Debug("Error has occurred while validate request %s with error message: %s", method, validationError)
	}

	ctx.JSON(http.StatusBadRequest, RequestFieldValidationError("Validation error has occurred", validationError))
}
