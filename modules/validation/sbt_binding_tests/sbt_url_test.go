//go:build !correct

package sbt_binding_tests

import (
	"code.gitea.io/gitea/modules/validation"
	"gitea.com/go-chi/binding"
	"testing"
)

func Test_SbtUrlRule(t *testing.T) {
	validation.AddBindingRules()

	err := binding.Error{
		FieldNames:     []string{"Url"},
		Classification: validation.ErrSbtUrl,
		Message:        "Must be of type string and must be url.",
	}

	for _, testCases := range []validationTestCase{
		{
			description: "Empty string is valid",
			data: SbtTestForm{
				Url: "",
			},
			expectedErrors: binding.Errors{},
		},
		{
			description: "Valid url with protocol, domain, path and anchor",
			data: SbtTestForm{
				Url: "https://www.jetbrains.com/help/go/local-history.html",
			},
			expectedErrors: binding.Errors{},
		},
		{
			description: "Valid url without protocol and path",
			data: SbtTestForm{
				Url: "ya.ru",
			},
			expectedErrors: binding.Errors{},
		},
		{
			description: "Valid url with path",
			data: SbtTestForm{
				Url: "https://yandex.ru/pogoda/?via=hl",
			},
			expectedErrors: binding.Errors{},
		},
		{
			description: "Valid url without protocol and with path",
			data: SbtTestForm{
				Url: "yandex.ru/pogoda?file=qwerty&dhhj=q",
			},
			expectedErrors: binding.Errors{},
		},
		{
			description: "Not valid url, wrong protocol",
			data: SbtTestForm{
				Url: "htt://yandex.ru/pogoda/?via=hl",
			},
			expectedErrors: binding.Errors{
				err,
			},
		},
		{
			description: "Not valid url, wrong protocol",
			data: SbtTestForm{
				Url: "https:///yandex.ru/pogoda/?via=hl",
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

func Test_SbtUrlRule_optional(t *testing.T) {
	validation.AddBindingRules()

	err := binding.Error{
		FieldNames:     []string{"Url"},
		Classification: validation.ErrSbtUrl,
		Message:        "Must be of type string and must be url.",
	}

	emptyUrl := ""
	validUrl := "https://www.jetbrains.com/help/go/local-history.html"
	validShortUrl := "ya.ru"
	validUrlWithPath := "https://yandex.ru/pogoda/?via=hl"
	validUrlWithoutProtocol := "yandex.ru/pogoda?file=qwerty&dhhj=q"
	notValidProtocolUrl := "htt://yandex.ru/pogoda/?via=hl"
	notValidProtocolUrl2 := "https:///yandex.ru/pogoda/?via=hl"

	for _, testCases := range []validationTestCase{
		{
			description: "Empty string is valid",
			data: SbtTestFormOptional{
				Url: &emptyUrl,
			},
			expectedErrors: binding.Errors{},
		},
		{
			description: "Valid url with protocol, domain, path and anchor",
			data: SbtTestFormOptional{
				Url: &validUrl,
			},
			expectedErrors: binding.Errors{},
		},
		{
			description: "Valid url without protocol and path",
			data: SbtTestFormOptional{
				Url: &validShortUrl,
			},
			expectedErrors: binding.Errors{},
		},
		{
			description: "Valid url with path",
			data: SbtTestFormOptional{
				Url: &validUrlWithPath,
			},
			expectedErrors: binding.Errors{},
		},
		{
			description: "Valid url without protocol and with path",
			data: SbtTestFormOptional{
				Url: &validUrlWithoutProtocol,
			},
			expectedErrors: binding.Errors{},
		},
		{
			description: "Not valid url, wrong protocol",
			data: SbtTestFormOptional{
				Url: &notValidProtocolUrl,
			},
			expectedErrors: binding.Errors{
				err,
			},
		},
		{
			description: "Not valid url, wrong protocol",
			data: SbtTestFormOptional{
				Url: &notValidProtocolUrl2,
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
