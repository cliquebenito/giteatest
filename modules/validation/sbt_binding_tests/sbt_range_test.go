//go:build !correct

package sbt_binding_tests

import (
	"code.gitea.io/gitea/modules/validation"
	"fmt"
	"gitea.com/go-chi/binding"
	"testing"
)

func Test_SbtRange(t *testing.T) {
	validation.AddBindingRules()

	err := binding.Error{
		FieldNames:     []string{"Range"},
		Classification: validation.ErrSbtRange,
		Message:        fmt.Sprintf("Must be between: %d - %d", minSizeCount, maxSizeCount),
	}

	for _, testCases := range []validationTestCase{
		/*
			0 - валидное значение, потому что Binding не проверяет 0 и "" то есть все значения,
			которые  возможно были не инициализированы.
			То есть если поставить валидацию SbtRange(4,9) и придет поле равное 0 - это значение будет валидно.
			Так же если поставить проверку поля на Required, а придет 0 значение, Binding вернет ошибку что поле обязательно
		*/
		//{
		//	description: "Zero is invalid",
		//	data: SbtTestForm{
		//		Range: 0,
		//	},
		//	expectedErrors: binding.Errors{ //TODO c required - 0 не допустимо
		//		err,
		//	},
		//},
		{
			description: "Zero is invalid",
			data: SbtTestForm{
				Range: 1,
			},
			expectedErrors: binding.Errors{
				err,
			},
		},
		{
			description: "Valid",
			data: SbtTestForm{
				Range: 6,
			},
			expectedErrors: binding.Errors{},
		},
		{
			description: "Minus value",
			data: SbtTestForm{
				Range: -100,
			},
			expectedErrors: binding.Errors{
				err,
			},
		},
		{
			description: fmt.Sprintf("Equal max value %d", maxSizeCount),
			data: SbtTestForm{
				Range: maxSizeCount,
			},
			expectedErrors: binding.Errors{},
		},
		{
			description: fmt.Sprintf("Equal min value %d", minSizeCount),
			data: SbtTestForm{
				Range: minSizeCount,
			},
			expectedErrors: binding.Errors{},
		},
		{
			description: fmt.Sprintf("More than max value %d", maxSizeCount),
			data: SbtTestForm{
				Range: 111,
			},
			expectedErrors: binding.Errors{
				err,
			},
		},
	} {
		t.Run(testCases.description, func(t *testing.T) {
			validationTest(t, testCases)
		})
	}
}

func Test_SbtRange_optional(t *testing.T) {
	validation.AddBindingRules()

	err := binding.Error{
		FieldNames:     []string{"Range"},
		Classification: validation.ErrSbtRange,
		Message:        fmt.Sprintf("Must be between: %d - %d", minSizeCount, maxSizeCount),
	}

	var zeroNum int
	six := 6
	minusNum := -9
	bigNum := 123456

	for _, testCases := range []validationTestCase{
		{
			description: "Zero is invalid",
			data: SbtTestFormOptional{
				Range: &zeroNum,
			},
			expectedErrors: binding.Errors{
				err,
			},
		},
		{
			description: "Valid",
			data: SbtTestFormOptional{
				Range: &six,
			},
			expectedErrors: binding.Errors{},
		},
		{
			description: "Minus value",
			data: SbtTestFormOptional{
				Range: &minusNum,
			},
			expectedErrors: binding.Errors{
				err,
			},
		},
		{
			description: fmt.Sprintf("More than max value %d", maxSizeCount),
			data: SbtTestFormOptional{
				Range: &bigNum,
			},
			expectedErrors: binding.Errors{
				err,
			},
		},
	} {
		t.Run(testCases.description, func(t *testing.T) {
			validationTest(t, testCases)
		})
	}
}
