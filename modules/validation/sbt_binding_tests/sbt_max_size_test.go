//go:build !correct

package sbt_binding_tests

import (
	"fmt"
	"strings"
	"testing"

	"gitea.com/go-chi/binding"

	"code.gitea.io/gitea/modules/validation"
)

func Test_SbtMaxSize(t *testing.T) {
	validation.AddBindingRules()

	err := binding.Error{
		FieldNames:     []string{"MaxSize"},
		Classification: validation.ErrSbtMaxSize,
		Message:        fmt.Sprintf("Must be of type string and cannot be larger than %d characters.", maxSizeCount),
	}

	for _, testCases := range []validationTestCase{
		{
			description: "Empty string is valid",
			data: SbtTestForm{
				MaxSize: "",
			},
			expectedErrors: binding.Errors{},
		},
		{
			description: fmt.Sprintf("Less than max size (%d)", maxSizeCount),
			data: SbtTestForm{
				MaxSize: strings.Repeat("f", maxSizeCount-1),
			},
			expectedErrors: binding.Errors{},
		},
		{
			description: fmt.Sprintf("More than max size (%d)", maxSizeCount),
			data: SbtTestForm{
				MaxSize: strings.Repeat("f", maxSizeCount+1),
			},
			expectedErrors: binding.Errors{
				err,
			},
		},
		{
			description: fmt.Sprintf("Exactly %d characters", maxSizeCount),
			data: SbtTestForm{
				MaxSize: strings.Repeat("1", maxSizeCount),
			},
			expectedErrors: binding.Errors{},
		},
		{
			description: fmt.Sprintf("Cyrillic symbols less than max size (%d)", maxSizeCount),
			data: SbtTestForm{
				MaxSize: strings.Repeat("щ", maxSizeCount-1),
			},
			expectedErrors: binding.Errors{},
		},
	} {
		t.Run(testCases.description, func(t *testing.T) {
			validationTest(t, testCases)
		})
	}
}

func Test_SbtMaxSize_optional(t *testing.T) {
	validation.AddBindingRules()

	err := binding.Error{
		FieldNames:     []string{"MaxSize"},
		Classification: validation.ErrSbtMaxSize,
		Message:        fmt.Sprintf("Must be of type string and cannot be larger than %d characters.", maxSizeCount),
	}

	str := "3fdv76sd"
	emptyStr := ""
	longStr := "qwerty3678"
	cyrillicStr := "простоян"
	veryLongStr := "testJustTestQWERTYtestJustTestQWERTYtestJustTestQWERTYtestJustTestQWERTYtestJustTestQWERTYtestJustTestQWERTY"

	for _, testCases := range []validationTestCase{
		{
			description: "Less then max size",
			data: SbtTestFormOptional{
				MaxSize: &str,
			},
			expectedErrors: binding.Errors{},
		},
		{
			description: "Empty string is valid",
			data: SbtTestFormOptional{
				MaxSize: &emptyStr,
			},
			expectedErrors: binding.Errors{},
		},
		{
			description: "More then max size",
			data: SbtTestFormOptional{
				MaxSize: &longStr,
			},
			expectedErrors: binding.Errors{
				err,
			},
		},
		{
			description: fmt.Sprintf("Cyrillic symbols less then %d", maxSizeCount),
			data: SbtTestFormOptional{
				MaxSize: &cyrillicStr,
			},
			expectedErrors: binding.Errors{},
		},
		{
			description: fmt.Sprintf("Mach more then %d", maxSizeCount),
			data: SbtTestFormOptional{
				MaxSize: &veryLongStr,
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
