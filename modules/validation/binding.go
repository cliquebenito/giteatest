// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package validation

import (
	"fmt"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"gitea.com/go-chi/binding"
	"github.com/gobwas/glob"
	"github.com/unknwon/com"

	"code.gitea.io/gitea/modules/auth"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/routers/web/utils"
)

const (
	// ErrGitRefName is git reference name error
	ErrGitRefName = "GitRefNameError"
	// ErrGlobPattern is returned when glob pattern is invalid
	ErrGlobPattern = "GlobPattern"
	// ErrRegexPattern is returned when a regex pattern is invalid
	ErrRegexPattern = "RegexPattern"
	// ErrUsername is username error
	ErrUsername = "UsernameError"
	// ErrInvalidGroupTeamMap is returned when a group team mapping is invalid
	ErrInvalidGroupTeamMap = "InvalidGroupTeamMap"
	// ErrSbtMaxSize в случае если размер строки больше разрешенного значения
	ErrSbtMaxSize = "ErrSbtMaxSize"
	//ErrSbtMinSize в случае если размер строки меньше разрешенного значения
	ErrSbtMinSize = "ErrSbtMinSize"
	//ErrSbtRange в случае если значение не входит в диапазон
	ErrSbtRange = "ErrSbtRange"
	//ErrSbtUrl в случае если строка не является ссылкой
	ErrSbtUrl = "ErrSbtUrl"
	//ErrSbtIn в случае если строка не входит в перечень строк
	ErrSbtIn = "ErrSbtIn"
	//ErrSbtGitRefName в случае если в строке используются недопустимые символы в гите
	ErrSbtGitRefName = "ErrSbtGitRefName"
	//ErrSbtNotEmpty в случае если строка пуста
	ErrSbtNotEmpty = "ErrSbtNotEmpty"
	//ErrSbtAlphaDashDot в случае если строка содержит что-то кроме цифр, букв, тире и нижнего подчеркивания
	ErrSbtAlphaDashDot = "ErrSbtAlphaDashDot"
	//ErrSbtEmail в случае если строка не соответствует паттерну электронной почты
	ErrSbtEmail = "ErrSbtEmail"
)

// AddBindingRules adds additional binding rules
func AddBindingRules() {
	addGitRefNameBindingRule()
	addValidURLBindingRule()
	addValidSiteURLBindingRule()
	addGlobPatternRule()
	addRegexPatternRule()
	addGlobOrRegexPatternRule()
	addUsernamePatternRule()
	addValidGroupTeamMapRule()
	addSbtMaxSizeRule()
	addSbtMinSizeRule()
	addSbtRangeRule()
	addSbtUrlRule()
	addSbtInRule()
	addSbtGitRefNameRule()
	addSbtNotEmptyRule()
	addSbtAlphaDashDotRule()
	addSbtEmailRule()
}

// addSbtMaxSizeValidator Проверка строкового поля на максимально допустимое количество символов.
// Данная проверка может быть использована для типов string и *string
// Пример использования: binding:"SbtMaxSize(100)"
func addSbtMaxSizeRule() {
	binding.AddParamRule(&binding.ParamRule{
		IsMatch: func(rule string) bool {
			return strings.HasPrefix(rule, "SbtMaxSize")
		},
		IsValid: func(errs binding.Errors, rule string, name string, val interface{}) (bool, binding.Errors) {
			maxLength, err := strconv.Atoi(rule[len("SbtMaxSize(") : len(rule)-1])
			if err != nil {
				errs.Add([]string{name}, ErrSbtMaxSize, "Could not parse SbtMaxSize value.")

				return false, errs
			}
			if utils.IsBranchNameTooLong(getStringValue(val), maxLength) {
				errs.Add([]string{name}, ErrSbtMaxSize, fmt.Sprintf("Must be of type string and cannot be larger than %d characters.", maxLength))

				return false, errs
			}

			return true, errs
		},
	})
}

// addSbtMinSizeRule Проверка строкового поля на минимально допустимое количество символов.
// Данная проверка может быть использована для типов string и *string
// Пример использования: binding:"SbtMinSize(100)"
func addSbtMinSizeRule() {
	binding.AddParamRule(&binding.ParamRule{
		IsMatch: func(rule string) bool {
			return strings.HasPrefix(rule, "SbtMinSize")
		},
		IsValid: func(errs binding.Errors, rule string, name string, val interface{}) (bool, binding.Errors) {
			min, _ := strconv.Atoi(rule[11 : len(rule)-1])

			if utf8.RuneCountInString(getStringValue(val)) < min {
				errs.Add([]string{name}, ErrSbtMinSize, fmt.Sprintf("Must be of type string and cannot be less than %d characters.", min))

				return false, errs
			}

			return true, errs
		},
	})
}

// addSbtRangeRule Проверка числового значения, которе должно находиться в диапазоне
// Данная проверка может быть использована для типов int и *int
// Пример использования: binding:"SbtRange(0,5)"
func addSbtRangeRule() {
	binding.AddParamRule(&binding.ParamRule{
		IsMatch: func(rule string) bool {
			return strings.HasPrefix(rule, "SbtRange")
		},
		IsValid: func(errs binding.Errors, rule string, name string, val interface{}) (bool, binding.Errors) {
			value := getIntValue(val)

			nums := strings.Split(rule[9:len(rule)-1], ",")
			if len(nums) != 2 || nums[0] > nums[1] {
				errs.Add([]string{name}, ErrSbtRange, "Range must have min and max params.")

				return false, errs
			}
			if value < com.StrTo(nums[0]).MustInt() || value > com.StrTo(nums[1]).MustInt() {
				errs.Add([]string{name}, ErrSbtRange, fmt.Sprintf("Must be between: %s - %s", nums[0], nums[1]))

				return false, errs
			}

			return true, errs
		},
	})
}

// addSbtUrlRule Проверка является ли строковое значение url
// В случае если строка пустая "", проверка возвращает true
// Данная проверка может быть использована для типов string и *string
// Пример использования: binding:"SbtUrl"
func addSbtUrlRule() {
	binding.AddParamRule(&binding.ParamRule{
		IsMatch: func(rule string) bool {
			return strings.HasPrefix(rule, "SbtUrl")
		},
		IsValid: func(errs binding.Errors, rule string, name string, val interface{}) (bool, binding.Errors) {
			if !sbtUrlValidator(getStringValue(val)) {
				errs.Add([]string{name}, ErrSbtUrl, "Must be of type string and must be url.")

				return false, errs
			}

			return true, errs
		},
	})
}

func sbtUrlValidator(str string) bool {
	if str == "" {
		return true
	}

	u, err := url.Parse(str)
	if err != nil {
		return false
	}
	if strings.HasPrefix(u.Host, ".") {
		return false
	}
	if u.Host == "" && (u.Path != "" && !strings.Contains(u.Path, ".")) {
		return false
	}
	return binding.URLPattern.MatchString(str)
}

/*
addSbtInRule Проверка входит ли строка в список допустимых строк
Данная проверка может быть использована для типов string и *string
Пример использования: binding:"SbtIn(public,limited,private)"
*/
func addSbtInRule() {
	binding.AddParamRule(&binding.ParamRule{
		IsMatch: func(rule string) bool {
			return strings.HasPrefix(rule, "SbtIn")
		},
		IsValid: func(errs binding.Errors, rule string, name string, val interface{}) (bool, binding.Errors) {
			str := getStringValue(val)

			arr := rule[6 : len(rule)-1]
			vals := strings.Split(arr, ",")
			isIn := false
			for _, v := range vals {
				if v == str {
					isIn = true
					break
				}
			}

			if !isIn {
				errs.Add([]string{name}, ErrSbtIn, fmt.Sprintf("Must be one of %v", vals))
				return false, errs
			}

			return true, errs
		},
	})
}

/*
addSbtGitRefNameRule Проверка содержит ли строка недопустимые знаки для имен в гите.
Сначала происходит проверка IsValidRefPattern, затем дополнительно проверка от SC, соответствующая СТ.
Данная проверка может быть использована для типов string и *string.
Пример использования: binding:"SbtGitRefName".
Допускаются латинские буквы, цифры, тире, точки, знаки подчеркивания, символ косой черты.
Имя ветки не должно быть пустым.
Имя ветки не должно начинаться на слеш / или точку .
*/
func addSbtGitRefNameRule() {
	binding.AddParamRule(&binding.ParamRule{
		IsMatch: func(rule string) bool {
			return strings.HasPrefix(rule, "SbtGitRefName")
		},
		IsValid: func(errs binding.Errors, rule string, name string, val interface{}) (bool, binding.Errors) {
			str := getStringValue(val)

			if !git.IsValidRefPattern(str) || !utils.ValidateBranchNameIsRefName(str) {
				errs.Add([]string{name}, ErrSbtGitRefName, "Wrong git reference name.")
				return false, errs
			}
			return true, errs
		},
	})
}

/*
*
addSbtAlphaDashDotRule Проверка строки в случае если строка должна содержать только буквы, цифры, тире и нижнее подчеркивание
Данная проверка может быть использована для типов string и *string
Пример использования: binding:"SbtAlphaDashDot"
*/
func addSbtAlphaDashDotRule() {
	AlphaDashDotPattern := regexp.MustCompile(`[^\d\w-_\.]`)

	binding.AddParamRule(&binding.ParamRule{
		IsMatch: func(rule string) bool {
			return strings.HasPrefix(rule, "SbtAlphaDashDot")
		},
		IsValid: func(errs binding.Errors, rule string, name string, val interface{}) (bool, binding.Errors) {
			str := getStringValue(val)

			if AlphaDashDotPattern.MatchString(str) && str != "" {
				errs.Add([]string{name}, ErrSbtAlphaDashDot, "Should contain only alphanumeric, dash ('-'), underscore ('_') and dot ('.') characters.")
				return false, errs
			}
			return true, errs
		},
	})
}

// addSbtEmailRule Проверка является ли строковое значение адресом электронной почты
// Данная проверка может быть использована для типов string и *string
// Пример использования: binding:"SbtEmail"
func addSbtEmailRule() {
	EmailPattern := regexp.MustCompile(`\A[\w!#$%&'*+/=?^_` + "`" + `{|}~-]+(?:\.[\w!#$%&'*+/=?^_` + "`" + `{|}~-]+)*@(?:[\w](?:[\w-]*[\w])?\.)+[a-zA-Z0-9](?:[\w-]*[\w])?\z`)

	binding.AddParamRule(&binding.ParamRule{
		IsMatch: func(rule string) bool {
			return strings.HasPrefix(rule, "SbtEmail")
		},
		IsValid: func(errs binding.Errors, rule string, name string, val interface{}) (bool, binding.Errors) {
			str := getStringValue(val)

			if !EmailPattern.MatchString(str) && str != "" {
				errs.Add([]string{name}, ErrSbtEmail, "Is not a valid email address.")
				return false, errs
			}
			return true, errs
		},
	})
}

/*
getIntValue - метод, который возвращает число из интерфейса если это указатель,
если это число, то возвращает число
*/
func getIntValue(val interface{}) int {
	var value int
	switch v := reflect.ValueOf(val); v.Kind() {
	case reflect.Int:
		value = val.(int)
	case reflect.Pointer:
		value = *val.(*int)
	}
	return value
}

/*
getStringValue - метод, который возвращает строку из интерфейса если это указатель,
если это строка, то возвращает строку
*/
func getStringValue(val interface{}) string {
	var str string
	switch v := reflect.ValueOf(val); v.Kind() {
	case reflect.String:
		str = val.(string)
	case reflect.Pointer:
		str = *val.(*string)
	}
	return str
}

func addGitRefNameBindingRule() {
	// Git refname validation rule
	binding.AddRule(&binding.Rule{
		IsMatch: func(rule string) bool {
			return strings.HasPrefix(rule, "GitRefName")
		},
		IsValid: func(errs binding.Errors, name string, val interface{}) (bool, binding.Errors) {
			str := fmt.Sprintf("%v", val)

			if !git.IsValidRefPattern(str) {
				errs.Add([]string{name}, ErrGitRefName, "GitRefName")
				return false, errs
			}
			return true, errs
		},
	})
}

func addValidURLBindingRule() {
	// URL validation rule
	binding.AddRule(&binding.Rule{
		IsMatch: func(rule string) bool {
			return strings.HasPrefix(rule, "ValidUrl")
		},
		IsValid: func(errs binding.Errors, name string, val interface{}) (bool, binding.Errors) {
			str := fmt.Sprintf("%v", val)
			if len(str) != 0 && !IsValidURL(str) {
				errs.Add([]string{name}, binding.ERR_URL, "Url")
				return false, errs
			}

			return true, errs
		},
	})
}

func addValidSiteURLBindingRule() {
	// URL validation rule
	binding.AddRule(&binding.Rule{
		IsMatch: func(rule string) bool {
			return strings.HasPrefix(rule, "ValidSiteUrl")
		},
		IsValid: func(errs binding.Errors, name string, val interface{}) (bool, binding.Errors) {
			str := fmt.Sprintf("%v", val)
			if len(str) != 0 && !IsValidSiteURL(str) {
				errs.Add([]string{name}, binding.ERR_URL, "Url")
				return false, errs
			}

			return true, errs
		},
	})
}

func addGlobPatternRule() {
	binding.AddRule(&binding.Rule{
		IsMatch: func(rule string) bool {
			return rule == "GlobPattern"
		},
		IsValid: globPatternValidator,
	})
}

func globPatternValidator(errs binding.Errors, name string, val interface{}) (bool, binding.Errors) {
	str := fmt.Sprintf("%v", val)

	if len(str) != 0 {
		if _, err := glob.Compile(str); err != nil {
			errs.Add([]string{name}, ErrGlobPattern, err.Error())
			return false, errs
		}
	}

	return true, errs
}

func addRegexPatternRule() {
	binding.AddRule(&binding.Rule{
		IsMatch: func(rule string) bool {
			return rule == "RegexPattern"
		},
		IsValid: regexPatternValidator,
	})
}

func regexPatternValidator(errs binding.Errors, name string, val interface{}) (bool, binding.Errors) {
	str := fmt.Sprintf("%v", val)

	if _, err := regexp.Compile(str); err != nil {
		errs.Add([]string{name}, ErrRegexPattern, err.Error())
		return false, errs
	}

	return true, errs
}

func addGlobOrRegexPatternRule() {
	binding.AddRule(&binding.Rule{
		IsMatch: func(rule string) bool {
			return rule == "GlobOrRegexPattern"
		},
		IsValid: func(errs binding.Errors, name string, val interface{}) (bool, binding.Errors) {
			str := strings.TrimSpace(fmt.Sprintf("%v", val))

			if len(str) >= 2 && strings.HasPrefix(str, "/") && strings.HasSuffix(str, "/") {
				return regexPatternValidator(errs, name, str[1:len(str)-1])
			}
			return globPatternValidator(errs, name, val)
		},
	})
}

func addUsernamePatternRule() {
	binding.AddRule(&binding.Rule{
		IsMatch: func(rule string) bool {
			return rule == "Username"
		},
		IsValid: func(errs binding.Errors, name string, val interface{}) (bool, binding.Errors) {
			str := fmt.Sprintf("%v", val)
			if !IsValidUsername(str) {
				errs.Add([]string{name}, ErrUsername, "invalid username")
				return false, errs
			}
			return true, errs
		},
	})
}

func addValidGroupTeamMapRule() {
	binding.AddRule(&binding.Rule{
		IsMatch: func(rule string) bool {
			return strings.HasPrefix(rule, "ValidGroupTeamMap")
		},
		IsValid: func(errs binding.Errors, name string, val interface{}) (bool, binding.Errors) {
			_, err := auth.UnmarshalGroupTeamMapping(fmt.Sprintf("%v", val))
			if err != nil {
				errs.Add([]string{name}, ErrInvalidGroupTeamMap, err.Error())
				return false, errs
			}

			return true, errs
		},
	})
}

func portOnly(hostport string) string {
	colon := strings.IndexByte(hostport, ':')
	if colon == -1 {
		return ""
	}
	if i := strings.Index(hostport, "]:"); i != -1 {
		return hostport[i+len("]:"):]
	}
	if strings.Contains(hostport, "]") {
		return ""
	}
	return hostport[colon+len(":"):]
}

func validPort(p string) bool {
	for _, r := range []byte(p) {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// addSbtNotEmptyRule Проверка не пусто ли строковое значение
// Данная проверка может быть использована для типов string и *string
// Пример использования: binding:"SbtNotEmpty"
func addSbtNotEmptyRule() {
	binding.AddParamRule(&binding.ParamRule{
		IsMatch: func(rule string) bool {
			return strings.HasPrefix(rule, "SbtNotEmpty")
		},
		IsValid: func(errs binding.Errors, rule string, name string, val interface{}) (bool, binding.Errors) {
			if getStringValue(val) == "" {
				errs.Add([]string{name}, ErrSbtNotEmpty, "Must be not empty string.")

				return false, errs
			}

			return true, errs
		},
	})
}
