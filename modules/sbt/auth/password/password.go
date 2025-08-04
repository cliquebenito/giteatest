package password

import (
	"code.gitea.io/gitea/modules/setting"
	"strings"
	"sync"
)

// complexity contains information about a particular kind of password complexity
// основа взята отсюда modules/auth/password/password.go
type complexity struct {
	ValidChars string
	TrNameOne  string
}

const (
	LOWER = "password_lowercase_one"
	UPPER = "password_uppercase_one"
	DIGIT = "password_digit_one"
	SPEC  = "password_special_one"
)

// Варианты текста объяснения ошибки в пароле в зависимости от включеных настроек политики паролей
var passwordComplexityExplanation = map[string]string{LOWER: "one lowercase chars", UPPER: "one uppercase chars", DIGIT: "one digit chars", SPEC: "one special chars"}

var (
	matchComplexityOnce sync.Once
	validChars          string
	requiredList        []complexity

	charComplexities = map[string]complexity{
		"lower": {
			`abcdefghijklmnopqrstuvwxyz`,
			LOWER,
		},
		"upper": {
			`ABCDEFGHIJKLMNOPQRSTUVWXYZ`,
			UPPER,
		},
		"digit": {
			`0123456789`,
			DIGIT,
		},
		"spec": {
			` !"#$%&'()*+,-./:;<=>?@[\]^_{|}~` + "`",
			SPEC,
		},
	}
)

// NewComplexity for preparation
func NewComplexity() {
	matchComplexityOnce.Do(func() {
		setupComplexity(setting.PasswordComplexity)
	})
}

func setupComplexity(values []string) {
	if len(values) != 1 || values[0] != "off" {
		for _, val := range values {
			if complex, ok := charComplexities[val]; ok {
				validChars += complex.ValidChars
				requiredList = append(requiredList, complex)
			}
		}
		if len(requiredList) == 0 {
			// No valid character classes found; use all classes as default
			for _, complex := range charComplexities {
				validChars += complex.ValidChars
				requiredList = append(requiredList, complex)
			}
		}
	}
	if validChars == "" {
		// No complexities to check; provide a sensible default for password generation
		validChars = charComplexities["lower"].ValidChars + charComplexities["upper"].ValidChars + charComplexities["digit"].ValidChars
	}
}

// IsComplexEnough return True and empty error message if password meets complexity settings
// or false and password complexity error message otherwise
func IsComplexEnough(pwd string) (bool, string) {
	NewComplexity()
	var passwordErrors []string
	if len(validChars) > 0 {
		for _, req := range requiredList {
			if !strings.ContainsAny(req.ValidChars, pwd) {
				passwordErrors = append(passwordErrors, passwordComplexityExplanation[req.TrNameOne])
			}
		}
	}

	if len(passwordErrors) > 0 {
		return false, strings.Join(passwordErrors, ", ")
	}
	return true, ""
}
