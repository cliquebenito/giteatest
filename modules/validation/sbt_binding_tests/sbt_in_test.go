//go:build !correct

package sbt_binding_tests

import (
	"code.gitea.io/gitea/modules/validation"
	"gitea.com/go-chi/binding"
	"testing"
)

/*
SbtInTest написан на примере VisibilityType
*/
func Test_SbtIn(t *testing.T) {
	validation.AddBindingRules()

	err := binding.Error{
		FieldNames:     []string{"In"},
		Classification: validation.ErrSbtIn,
		Message:        "Must be one of [public limited private]",
	}

	for _, testCases := range []validationTestCase{
		/*
			Пустая строка не может быть валидной. В случае если приходит пустая строка.
			Bind метод не валидирует кастомные теги, если поле быыло не required
		*/
		//{
		//	description: "Empty string is invalid",
		//	data: SbtTestForm{
		//		In: "",
		//	},
		//	expectedErrors: binding.Errors{
		//		err,
		//	},
		//},
		{
			description: "Public visibility",
			data: SbtTestForm{
				In: "public",
			},
			expectedErrors: binding.Errors{},
		},
		{
			description: "Public visibility",
			data: SbtTestForm{
				In: "private",
			},
			expectedErrors: binding.Errors{},
		},
		{
			description: "Limited visibility",
			data: SbtTestForm{
				In: "LIMITED",
			},
			expectedErrors: binding.Errors{
				err,
			},
		},
		{
			description: "Wrong visibility",
			data: SbtTestForm{
				In: "Privat",
			},
			expectedErrors: binding.Errors{
				err,
			},
		},
		{
			description: "Wrong visibility absolutely",
			data: SbtTestForm{
				In: "something",
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

func Test_SbtIn_optional(t *testing.T) {
	validation.AddBindingRules()

	err := binding.Error{
		FieldNames:     []string{"In"},
		Classification: validation.ErrSbtIn,
		Message:        "Must be one of [public limited private]",
	}

	emptyVis := ""
	notLowerCaseVis := "Public"
	lowerCaseVis := "limited"
	wrongVis := "privat"
	WrongVis2 := "testing"

	for _, testCases := range []validationTestCase{
		{
			description: "Empty string is not valid",
			data: SbtTestFormOptional{
				In: &emptyVis,
			},
			expectedErrors: binding.Errors{
				err,
			},
		},
		{
			description: "Public visibility camel case",
			data: SbtTestFormOptional{
				In: &notLowerCaseVis,
			},
			expectedErrors: binding.Errors{
				err,
			},
		},
		{
			description: "Public visibility lower case",
			data: SbtTestFormOptional{
				In: &lowerCaseVis,
			},
			expectedErrors: binding.Errors{},
		},
		{
			description: "Wrong visibility",
			data: SbtTestFormOptional{
				In: &wrongVis,
			},
			expectedErrors: binding.Errors{
				err,
			},
		},
		{
			description: "Wrong visibility 2",
			data: SbtTestFormOptional{
				In: &WrongVis2,
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
