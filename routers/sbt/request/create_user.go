package request

import (
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/web/middleware"
	"fmt"
	"gitea.com/go-chi/binding"
	"github.com/gobwas/glob"
	"net/http"
	"strings"
)

/*
RegisterUser - структура запроса для регистрации пользователя
*/
type RegisterUser struct {
	UserName string `json:"username" binding:"Required;SbtMaxSize(50);SbtMinSize(2)"`
	Email    string `json:"email" binding:"Required;SbtMaxSize(255);SbtMinSize(5);SbtEmail"`
	Password string `json:"password" binding:"Required;SbtMaxSize(254)"`
}

// String возвращает строковое представление запроса, без пароля пользователя
func (f *RegisterUser) String() string {
	return fmt.Sprintf("[ Username: %s, Email: %s ]", f.UserName, f.Email)
}

/*
Validate метод валидации полей запроса, вызываемый в методе bind
*/
func (f *RegisterUser) Validate(req *http.Request, errs binding.Errors) binding.Errors {
	ctx := context.GetValidateContext(req)
	return middleware.Validate(errs, ctx.Data, f, ctx.Locale)
}

/*
IsEmailDomainListed сhecks whether the domain of an email address matches a list of domains
*/
func IsEmailDomainListed(globs []glob.Glob, email string) bool {
	if len(globs) == 0 {
		return false
	}

	n := strings.LastIndex(email, "@")
	if n <= 0 {
		return false
	}

	domain := strings.ToLower(email[n+1:])

	for _, g := range globs {
		if g.Match(domain) {
			return true
		}
	}

	return false
}

/*
IsEmailDomainAllowed
Validates that the email address provided by the user matches what has been configured.
The email is marked as allowed if it matches any of the domains in the whitelist or if it doesn't match any of
domains in the blocklist, if any such list is not empty.
*/
func (f *RegisterUser) IsEmailDomainAllowed() bool {
	if len(setting.Service.EmailDomainAllowList) == 0 {
		return !IsEmailDomainListed(setting.Service.EmailDomainBlockList, f.Email)
	}

	return IsEmailDomainListed(setting.Service.EmailDomainAllowList, f.Email)
}
