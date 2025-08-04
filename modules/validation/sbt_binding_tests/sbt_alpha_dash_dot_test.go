//go:build !correct

package sbt_binding_tests

import (
	"code.gitea.io/gitea/modules/validation"
	"gitea.com/go-chi/binding"
	"testing"
)

func Test_SbtAlphaDashDot(t *testing.T) {
	validation.AddBindingRules()

	err := binding.Error{
		FieldNames:     []string{"AlphaDashDot"},
		Classification: validation.ErrSbtAlphaDashDot,
		Message:        "Should contain only alphanumeric, dash ('-'), underscore ('_') and dot ('.') characters.",
	}

	for _, testCases := range []validationTestCase{
		{
			description: "Empty string is valid",
			data: SbtTestForm{
				AlphaDashDot: "",
			},
			expectedErrors: binding.Errors{},
		},
		{
			description: "Alphanumeric symbols is valid",
			data: SbtTestForm{
				AlphaDashDot: "sbtTest1",
			},
			expectedErrors: binding.Errors{},
		},
		{
			description: "Only numeric symbols is valid",
			data: SbtTestForm{
				AlphaDashDot: "9876543212345678",
			},
			expectedErrors: binding.Errors{},
		},
		{
			description: "Only alphabet symbols is valid",
			data: SbtTestForm{
				AlphaDashDot: "testAlphaDashDot",
			},
			expectedErrors: binding.Errors{},
		},
		{
			description: "Valid string with dots",
			data: SbtTestForm{
				AlphaDashDot: "...test...",
			},
			expectedErrors: binding.Errors{},
		},
		{
			description: "Valid string with dash",
			data: SbtTestForm{
				AlphaDashDot: "-_-",
			},
			expectedErrors: binding.Errors{},
		},
		{
			description: "Invalid string with space",
			data: SbtTestForm{
				AlphaDashDot: "test 1",
			},
			expectedErrors: binding.Errors{
				err,
			},
		},
		{
			description: "Invalid string with comma",
			data: SbtTestForm{
				AlphaDashDot: "one,second",
			},
			expectedErrors: binding.Errors{
				err,
			},
		},
		{
			description: "Invalid string with bracket",
			data: SbtTestForm{
				AlphaDashDot: "test)",
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

func Test_SbtAlphaDashDot_optional(t *testing.T) {
	validation.AddBindingRules()

	err := binding.Error{
		FieldNames:     []string{"AlphaDashDot"},
		Classification: validation.ErrSbtAlphaDashDot,
		Message:        "Should contain only alphanumeric, dash ('-'), underscore ('_') and dot ('.') characters.",
	}

	emptyStr := ""
	alphaNum := "sbtTest1"
	alphabetStr := "TestSbtAlphaDashDotOptional"
	numericStr := "98765432123456789"
	dotsStr := "..."
	underscore := "Test_SbtAlphaDashDot_optional"
	semicolonStr := "test;"

	for _, testCases := range []validationTestCase{
		{
			description: "Empty string is valid",
			data: SbtTestFormOptional{
				AlphaDashDot: &emptyStr,
			},
			expectedErrors: binding.Errors{},
		},
		{
			description: "Alphanumeric symbols is valid",
			data: SbtTestFormOptional{
				AlphaDashDot: &alphaNum,
			},
			expectedErrors: binding.Errors{},
		},
		{
			description: "Only numeric symbols is valid",
			data: SbtTestFormOptional{
				AlphaDashDot: &numericStr,
			},
			expectedErrors: binding.Errors{},
		},
		{
			description: "Only alphabet symbols is valid",
			data: SbtTestFormOptional{
				AlphaDashDot: &alphabetStr,
			},
			expectedErrors: binding.Errors{},
		},
		{
			description: "Valid string with dots",
			data: SbtTestFormOptional{
				AlphaDashDot: &dotsStr,
			},
			expectedErrors: binding.Errors{},
		},
		{
			description: "Valid string with underscore",
			data: SbtTestFormOptional{
				AlphaDashDot: &underscore,
			},
			expectedErrors: binding.Errors{},
		},
		{
			description: "Invalid string with semicolon",
			data: SbtTestFormOptional{
				AlphaDashDot: &semicolonStr,
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
