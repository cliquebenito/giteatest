package request

import (
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/web/middleware"
	"gitea.com/go-chi/binding"
	"net/http"
	"strconv"
)

/*
UserSettingsOptional - структура запроса на изменение настроек профиля пользователя
Все поля являются не обязательными.
*/
type UserSettingsOptional struct {
	Name                *string `json:"username" binding:"SbtMaxSize(50);SbtMinSize(2)"`
	FullName            *string `json:"full_name" binding:"SbtMaxSize(100)"`
	Website             *string `json:"website" binding:"SbtUrl;SbtMaxSize(255)"`
	Location            *string `json:"location" binding:"SbtMaxSize(50)"`
	Description         *string `json:"description" binding:"SbtMaxSize(255)"`
	Visibility          *string `json:"visibility" binding:"SbtIn(public,limited,private)"`
	KeepEmailPrivate    *bool   `json:"hide_email"`
	KeepActivityPrivate *bool   `json:"hide_activity"`
}

// Validate validates the fields
func (f *UserSettingsOptional) Validate(req *http.Request, errs binding.Errors) binding.Errors {
	ctx := context.GetValidateContext(req)
	return middleware.Validate(errs, ctx.Data, f, ctx.Locale)
}

func (f *UserSettingsOptional) ToString() string {
	var res string

	res = "{"
	if f.Name != nil {
		res += "\"username\":\"" + *f.Name + "\","
	}
	if f.FullName != nil {
		res += "\"full_name\":\"" + *f.FullName + "\","
	}
	if f.Website != nil {
		res += "\"website\":\"" + *f.Website + "\","
	}
	if f.Location != nil {
		res += "\"location\":\"" + *f.Location + "\","
	}
	if f.Description != nil {
		res += "\"description\":\"" + *f.Description + "\","
	}
	if f.Visibility != nil {
		res += "\"visibility\":\"" + *f.Visibility + "\","
	}
	if f.KeepEmailPrivate != nil {
		res += "\"hide_email\":\"" + strconv.FormatBool(*f.KeepEmailPrivate) + "\","
	}
	if f.KeepActivityPrivate != nil {
		res += "\"hide_activity\":\"" + strconv.FormatBool(*f.KeepActivityPrivate) + "\""
	}
	res += "}"

	return res
}

// UserAvatar структура запроса на смену аватара пользователя
type UserAvatar struct {
	Image string `json:"image" binding:"Required"`
}

// Validate validates the fields
func (f *UserAvatar) Validate(req *http.Request, errs binding.Errors) binding.Errors {
	ctx := context.GetValidateContext(req)
	return middleware.Validate(errs, ctx.Data, f, ctx.Locale)
}

// ChangePassword структура изменения пароля пользователя
type ChangePassword struct {
	OldPassword string `json:"old_password" binding:"Required;SbtMaxSize(254)"`
	NewPassword string `json:"new_password" binding:"Required;SbtMaxSize(254)"`
}

// Validate validates the fields
func (f *ChangePassword) Validate(req *http.Request, errs binding.Errors) binding.Errors {
	ctx := context.GetValidateContext(req)
	return middleware.Validate(errs, ctx.Data, f, ctx.Locale)
}
