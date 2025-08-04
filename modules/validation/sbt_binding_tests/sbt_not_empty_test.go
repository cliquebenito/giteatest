//go:build !correct

package sbt_binding_tests

import (
	"code.gitea.io/gitea/modules/validation"
	"gitea.com/go-chi/binding"
	"testing"
)

func Test_SbtNotEmpty(t *testing.T) {
	validation.AddBindingRules()

	err := binding.Error{
		FieldNames:     []string{"NotEmpty"},
		Classification: validation.ErrSbtNotEmpty,
		Message:        "Must be not empty string.",
	}

	str := "test"
	emptyStr := ""
	blankStr := " "

	for _, testCases := range []validationTestCase{
		/*
			"" - валидное значение, потому что Binding не проверяет 0 и "" то есть все значения,
			которые  возможно были не инициализированы.
			То есть если поставить валидацию NotEmpty и придет пустрая строка - это значение будет валидно.
			Так же если поставить проверку поля на Required, а придет пустая строка, Binding вернет ошибку что поле обязательно,
			то есть ошибка не то что не должна быть пустая строка, а то что поле обязательно
		*/
		//{
		//	description: "Empty string is not valid",
		//	data: SbtTestForm{
		//		NotEmpty: "",
		//	},
		//	expectedErrors: binding.Errors{
		//		err,
		//	},
		//},
		{
			description: "Blank string is valid",
			data: SbtTestForm{
				NotEmpty: blankStr,
			},
			expectedErrors: binding.Errors{},
		},
		{
			description: "Not empty string is valid",
			data: SbtTestForm{
				NotEmpty: str,
			},
			expectedErrors: binding.Errors{},
		},
		{
			description: "Blank string is valid",
			data: SbtTestFormOptional{
				NotEmpty: &blankStr,
			},
			expectedErrors: binding.Errors{},
		},
		{
			description: "Empty string is not valid",
			data: SbtTestFormOptional{
				NotEmpty: &emptyStr,
			},
			expectedErrors: binding.Errors{
				err,
			},
		},
		{
			description: "Not empty string is valid",
			data: SbtTestFormOptional{
				NotEmpty: &str,
			},
			expectedErrors: binding.Errors{},
		},
	} {
		t.Run(testCases.description, func(t *testing.T) {
			validationTest(t, testCases)
		})
	}
}
