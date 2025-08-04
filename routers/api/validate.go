package api

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"unicode/utf8"
)

type ValidationErrors struct {
	Errors []string
}

func (v ValidationErrors) Error() string {
	if len(v.Errors) > 1 {
		return "Ошибки валидации"
	}
	return "Ошибка валидации"
}

// RequestValidator парсит структуру и проверяет на соответствие тегам "validate".
func RequestValidator(form interface{}) error {
	val := reflect.ValueOf(form)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	typ := val.Type()

	var errors []string

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)
		jsonTag := fieldType.Tag.Get("json")
		validateTag := fieldType.Tag.Get("validate")
		fieldName := strings.Split(jsonTag, ",")[0]

		if field.Kind() == reflect.String {
			str := field.String()
			rules := strings.Split(validateTag, ",")

			for _, rule := range rules {
				switch {
				case rule == "required":
					if strings.TrimSpace(str) == "" {
						errors = append(errors, fmt.Sprintf("поле '%s' обязательно для заполнения", fieldName))
						goto NextField
					}

				case strings.HasPrefix(rule, "max="):
					maxLenStr := strings.TrimPrefix(rule, "max=")
					maxLen, err := strconv.Atoi(maxLenStr)
					if err == nil && utf8.RuneCountInString(str) > maxLen {
						errors = append(errors, fmt.Sprintf("длина поля '%s' превышает %d символов", fieldName, maxLen))
						goto NextField
					}

				case strings.HasPrefix(rule, "min="):
					minLenStr := strings.TrimPrefix(rule, "min=")
					minLen, err := strconv.Atoi(minLenStr)
					if err == nil && utf8.RuneCountInString(str) < minLen {
						errors = append(errors, fmt.Sprintf("длина поля '%s' должна быть не менее %d символов", fieldName, minLen))
						goto NextField
					}

				case strings.HasPrefix(rule, "prefix="):
					prefix := strings.TrimPrefix(rule, "prefix=")
					if !strings.HasPrefix(str, prefix) {
						errors = append(errors, fmt.Sprintf("поле '%s' должно начинаться с %s", fieldName, prefix))
						goto NextField
					}

				case strings.HasPrefix(rule, "in="):
					allowed := strings.Split(strings.TrimPrefix(rule, "in="), "|")
					match := false
					for _, val := range allowed {
						if str == val {
							match = true
							break
						}
					}
					if !match {
						errors = append(errors, fmt.Sprintf("поле '%s' должно быть одним из: %v", fieldName, allowed))
						goto NextField
					}
				}
			}
		}
	NextField:
	}

	if len(errors) > 0 {
		return ValidationErrors{Errors: errors}
	}
	return nil
}
