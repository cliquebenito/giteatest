package utils

import (
	"regexp"
)

var containsOnly = regexp.MustCompile("^[a-zA-Z0-9_\\-./]+$")
var beginsWith = regexp.MustCompile("^[./]")

// isWrongFirstSymbol проверяет первый символ названия ветки.
func isWrongFirstSymbol(branchName string) bool {
	return beginsWith.MatchString(branchName)
}

// IsBranchNameTooLong проверяет длину названия ветки.
func IsBranchNameTooLong(branchName string, maxLength int) bool {
	return len([]rune(branchName)) > maxLength
}

// branchNameContainsForbiddenSymbols проверяет, содержатся ли в названии ветки символы, которые не разрешены.
func branchNameContainsForbiddenSymbols(branchName string) bool {
	return !containsOnly.MatchString(branchName)
}

// ValidateBranchNameIsRefName валидирует, что название ветки - корректное ссылочное имя Git.
// Проверка в соответствии с [СТ]. Возвращает true, если имя корректное.
//
// [СТ]: https://sberworks.ru/wiki/pages/viewpage.action?pageId=595139024#:~:text=%D0%9D%D0%B0%D0%B7%D0%B2%D0%B0%D0%BD%D0%B8%D0%B5%20%D0%B2%D0%B5%D1%82%D0%BA%D0%B8%20%D0%B4%D0%BE%D0%BB%D0%B6%D0%BD%D0%BE,%D1%81%D1%81%D1%8B%D0%BB%D0%BE%D1%87%D0%BD%D1%8B%D0%BC%20%D0%B8%D0%BC%D0%B5%D0%BD%D0%B5%D0%BC%20Git%22]
func ValidateBranchNameIsRefName(branchName string) bool {
	return !isWrongFirstSymbol(branchName) && !branchNameContainsForbiddenSymbols(branchName)
}
