package gitnamesparser

import (
	"regexp"
	"strings"
)

// ParseCodesAndDescription функция ищет коды юнитов TaskTracker
func ParseCodesAndDescription(value string, regexp *regexp.Regexp) ([]string, string, error) {
	if value == "" || len(value) < 2 {
		return nil, "", NewUnitCodeNotFoundError(value)
	}

	description := strings.TrimSpace(value)
	codes := regexp.FindAllString(description, -1)

	if codes == nil {
		return nil, "", NewUnitCodeNotFoundError(description)
	}

	return codes, description, nil
}
