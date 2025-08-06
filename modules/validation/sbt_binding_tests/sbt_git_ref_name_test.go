//go:build !correct

package sbt_binding_tests

import (
	"testing"

	"gitea.com/go-chi/binding"

	"code.gitea.io/gitea/modules/validation"
)

func Test_SbtRefName(t *testing.T) {
	validation.AddBindingRules()

	err := binding.Error{
		FieldNames:     []string{"GitRefName"},
		Classification: validation.ErrSbtGitRefName,
		Message:        "Wrong git reference name.",
	}

	for _, testCases := range []validationTestCase{
		{
			description: "Valid reference name",
			data: SbtTestForm{
				GitRefName: "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ-._/1234567890",
			},
			expectedErrors: binding.Errors{},
		},
		{
			description: "Invalid beginning character in ref name: /",
			data: SbtTestForm{
				GitRefName: "/branch",
			},
			expectedErrors: binding.Errors{
				err,
			},
		},
		{
			description: "Invalid beginning character in ref name: .",
			data: SbtTestForm{
				GitRefName: ".branch",
			},
			expectedErrors: binding.Errors{
				err,
			},
		},
		{
			description: "Invalid sequence of characters in the end of ref name: .lock",
			data: SbtTestForm{
				GitRefName: "branch.lock",
			},
			expectedErrors: binding.Errors{
				err,
			},
		},
		{
			description: "Valid reference name: .lock is not in the end",
			data: SbtTestForm{
				GitRefName: "ref.lock_branch",
			},
			expectedErrors: binding.Errors{},
		},
		{
			description: "Invalid sequence of characters in ref name: ..",
			data: SbtTestForm{
				GitRefName: "tag1..0",
			},
			expectedErrors: binding.Errors{
				err,
			},
		},
		{
			description: "Invalid sequence of characters in ref name: /.",
			data: SbtTestForm{
				GitRefName: "tag1/.0",
			},
			expectedErrors: binding.Errors{
				err,
			},
		},
		{
			description: "Invalid sequence of characters in ref name: //",
			data: SbtTestForm{
				GitRefName: "tag1//0",
			},
			expectedErrors: binding.Errors{
				err,
			},
		},
		{
			description: "Invalid sequence of characters in ref name: @{",
			data: SbtTestForm{
				GitRefName: "tag1@{0",
			},
			expectedErrors: binding.Errors{
				err,
			},
		},
		{
			description: "Invalid character in ref name: [",
			data: SbtTestForm{
				GitRefName: "branch[name",
			},
			expectedErrors: binding.Errors{
				err,
			},
		},
		{
			description: "Invalid character in ref name: space",
			data: SbtTestForm{
				GitRefName: "branch name",
			},
			expectedErrors: binding.Errors{
				err,
			},
		},
		{
			description: "Invalid character in ref name: ~",
			data: SbtTestForm{
				GitRefName: "branch~name",
			},
			expectedErrors: binding.Errors{
				err,
			},
		},
		{
			description: "Invalid character in ref name: ^",
			data: SbtTestForm{
				GitRefName: "branch^name",
			},
			expectedErrors: binding.Errors{
				err,
			},
		},
		{
			description: "Invalid character in ref name: :",
			data: SbtTestForm{
				GitRefName: "branch:name",
			},
			expectedErrors: binding.Errors{
				err,
			},
		},
		{
			description: "Invalid character in ref name: ?",
			data: SbtTestForm{
				GitRefName: "branch?name",
			},
			expectedErrors: binding.Errors{
				err,
			},
		},
		{
			description: "Invalid character in ref name: *",
			data: SbtTestForm{
				GitRefName: "branch*name",
			},
			expectedErrors: binding.Errors{
				err,
			},
		},
		{
			description: "Invalid ref name: @",
			data: SbtTestForm{
				GitRefName: "@",
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

func Test_SbtRefName_optional(t *testing.T) {
	validation.AddBindingRules()

	err := binding.Error{
		FieldNames:     []string{"GitRefName"},
		Classification: validation.ErrGitRefName,
		Message:        "Wrong git reference name.",
	}

	emptyStr := ""
	validName := "new@branch"
	twoDots := "tag1..0"
	colon := "tag:1.0.1"
	asterisk := "qwerty2*2qwerty"

	for _, testCases := range []validationTestCase{
		{
			description: "Empty string is valid",
			data: SbtTestFormOptional{
				GitRefName: &emptyStr,
			},
			expectedErrors: binding.Errors{},
		},
		{
			description: "Valid reference name",
			data: SbtTestFormOptional{
				GitRefName: &validName,
			},
			expectedErrors: binding.Errors{},
		},
		{
			description: "Invalid ref name. Two dots near",
			data: SbtTestFormOptional{
				GitRefName: &twoDots,
			},
			expectedErrors: binding.Errors{
				err,
			},
		},
		{
			description: "Invalid ref name. : - not exist",
			data: SbtTestFormOptional{
				GitRefName: &colon,
			},
			expectedErrors: binding.Errors{
				err,
			},
		},
		{
			description: "Invalid ref name. * - not exist",
			data: SbtTestFormOptional{
				GitRefName: &asterisk,
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
