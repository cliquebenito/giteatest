//go:build !correct

package sbt_binding_tests

import (
	"code.gitea.io/gitea/modules/validation"
	"fmt"
	"gitea.com/go-chi/binding"
	"testing"
)

func Test_SbtMinSize(t *testing.T) {
	validation.AddBindingRules()

	err := binding.Error{
		FieldNames:     []string{"MinSize"},
		Classification: validation.ErrSbtMinSize,
		Message:        fmt.Sprintf("Must be of type string and cannot be less than %d characters.", minSizeCount),
	}

	for _, testCases := range []validationTestCase{
		{
			description: "Less then min size",
			data: SbtTestForm{
				MinSize: "sbt",
			},
			expectedErrors: binding.Errors{
				err,
			},
		},
		{
			description: "Same size like min size",
			data: SbtTestForm{
				MinSize: "Test",
			},
			expectedErrors: binding.Errors{},
		},
		/*
			"" - валидное значение, потому что Binding не проверяет 0 и "" то есть все значения,
			которые  возможно были не инициализированы.
			То есть если поставить валидацию MinSize(2) и придет пустрая строка - это значение будет валидно.
			Так же если поставить проверку поля на Required, а придет пустая строка, Binding вернет ошибку что поле обязательно,
			то есть ошибка не то что в строке символов меньше 2, а то что поле обязательно
			*string - проверяет это. Пустая строка не пройдет валидацию
		*/
		//{
		//	description: "Empty string must be not valid",
		//	data: SbtTestForm{
		//		MinSize: "",
		//	},
		//	expectedErrors: binding.Errors{
		//		err,
		//	},
		//},
		{
			description: "More then min size",
			data: SbtTestForm{
				MinSize: "SBTru",
			},
			expectedErrors: binding.Errors{},
		},
		{
			description: fmt.Sprintf("Cyrillic symbols less then %d", minSizeCount),
			data: SbtTestForm{
				MinSize: "СБТ",
			},
			expectedErrors: binding.Errors{
				err,
			},
		},
		{
			description: fmt.Sprintf("Cyrillic symbols more then %d", minSizeCount),
			data: SbtTestForm{
				MinSize: "-СБТ-",
			},
			expectedErrors: binding.Errors{},
		},
	} {
		t.Run(testCases.description, func(t *testing.T) {
			validationTest(t, testCases)
		})
	}
}

func Test_SbtMinSize_optional(t *testing.T) {
	validation.AddBindingRules()

	err := binding.Error{
		FieldNames:     []string{"MinSize"},
		Classification: validation.ErrSbtMinSize,
		Message:        fmt.Sprintf("Must be of type string and cannot be less than %d characters.", minSizeCount),
	}

	shortStr := "sbt"
	sameSizeStr := "Test"
	emptyStr := ""
	moreThenMinSize := "SBTru"
	cyrillicShortStr := "СБТ"
	cyrillicLongStr := "-СБТ-"

	for _, testCases := range []validationTestCase{
		{
			description: "Less then min size",
			data: SbtTestFormOptional{
				MinSize: &shortStr,
			},
			expectedErrors: binding.Errors{
				err,
			},
		},
		{
			description: "Same size like min size",
			data: SbtTestFormOptional{
				MinSize: &sameSizeStr,
			},
			expectedErrors: binding.Errors{},
		},
		{
			description: "Empty string must be not valid",
			data: SbtTestFormOptional{
				MinSize: &emptyStr,
			},
			expectedErrors: binding.Errors{
				err,
			},
		},
		{
			description: "More then min size",
			data: SbtTestFormOptional{
				MinSize: &moreThenMinSize,
			},
			expectedErrors: binding.Errors{},
		},
		{
			description: fmt.Sprintf("Cyrillic symbols less then %d", minSizeCount),
			data: SbtTestFormOptional{
				MinSize: &cyrillicShortStr,
			},
			expectedErrors: binding.Errors{
				err,
			},
		},
		{
			description: fmt.Sprintf("Cyrillic symbols more then %d", minSizeCount),
			data: SbtTestFormOptional{
				MinSize: &cyrillicLongStr,
			},
			expectedErrors: binding.Errors{},
		},
	} {
		t.Run(testCases.description, func(t *testing.T) {
			validationTest(t, testCases)
		})
	}
}
