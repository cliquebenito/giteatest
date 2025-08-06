//go:build !correct

package sbt_binding_tests

import (
	"code.gitea.io/gitea/modules/validation"
	"gitea.com/go-chi/binding"
	"testing"
)

func Test_SbtEmail(t *testing.T) {
	validation.AddBindingRules()

	err := binding.Error{
		FieldNames:     []string{"Email"},
		Classification: validation.ErrSbtEmail,
		Message:        "Is not a valid email address.",
	}

	for _, testCases := range []validationTestCase{
		{
			description: "Valid email",
			data: SbtTestForm{
				Email: "mail@email.sbt",
			},
			expectedErrors: binding.Errors{},
		},
		{
			description: "Empty email is valid",
			data: SbtTestForm{
				Email: "",
			},
			expectedErrors: binding.Errors{},
		},
		{
			description: "Invalid email format. Without domain zone",
			data: SbtTestForm{
				Email: "test@test",
			},
			expectedErrors: binding.Errors{
				err,
			},
		},
		{
			description: "Invalid email format. Only alphabet symbols",
			data: SbtTestForm{
				Email: "test",
			},
			expectedErrors: binding.Errors{
				err,
			},
		},
		{
			description: "Valid email. Long domain zone",
			data: SbtTestForm{
				Email: "test@test.qwerty",
			},
			expectedErrors: binding.Errors{},
		},
		{
			description: "Invalid email format. Cyrillic symbols in email",
			data: SbtTestForm{
				Email: "почта@почта.ру",
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

func Test_SbtEmail_optional(t *testing.T) {
	validation.AddBindingRules()

	err := binding.Error{
		FieldNames:     []string{"Email"},
		Classification: validation.ErrSbtEmail,
		Message:        "Is not a valid email address.",
	}

	email := "test@tets.tu"
	emailEmpty := ""
	emailWithoutDomainZone := "test@test"
	emailOnlyAlphabetSym := "testtestru"
	emailLongDomainZone := "test@testtest.testtesttesttesttest"
	emailCyrillicSym := "почта@почта.ру"

	for _, testCases := range []validationTestCase{
		{
			description: "Valid email",
			data: SbtTestFormOptional{
				Email: &email,
			},
			expectedErrors: binding.Errors{},
		},
		{
			description: "Empty email is valid",
			data: SbtTestFormOptional{
				Email: &emailEmpty,
			},
			expectedErrors: binding.Errors{},
		},
		{
			description: "Invalid email format. Without domain zone",
			data: SbtTestFormOptional{
				Email: &emailWithoutDomainZone,
			},
			expectedErrors: binding.Errors{
				err,
			},
		},
		{
			description: "Invalid email format. Only alphabet symbols",
			data: SbtTestFormOptional{
				Email: &emailOnlyAlphabetSym,
			},
			expectedErrors: binding.Errors{
				err,
			},
		},
		{
			description: "Valid email. Long domain zone",
			data: SbtTestFormOptional{
				Email: &emailLongDomainZone,
			},
			expectedErrors: binding.Errors{},
		},
		{
			description: "Invalid email format. Cyrillic symbols in email",
			data: SbtTestFormOptional{
				Email: &emailCyrillicSym,
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
