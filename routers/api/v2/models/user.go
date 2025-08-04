package models

import (
	"fmt"
	"time"

	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/routers/sbt/request"
)

// nameSize максимальная длинна имени пользователя
const nameSize = 40

// CreateUserRequest структура для создания пользователя
type CreateUserRequest struct {
	UserKey    string     `json:"user_key" binding:"Required"`
	Name       string     `json:"name" binding:"Required"`
	Email      string     `json:"email" binding:"Required"`
	FullName   string     `json:"full_name"`
	Created    *time.Time `json:"created_at"`
	Visibility string     `json:"visibility" binding:"In(,public,limited,private)"`
	Restricted *bool      `json:"restricted"`
}

// CreateUserResponse структура ответа созданного пользователя
type CreateUserResponse struct {
	ID       int64  `json:"id"`
	UserKey  string `json:"user_key"`
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	Email    string `json:"email"`
}

func (c *CreateUserRequest) Validate() bool {

	if len(c.Name) > nameSize {
		return false
	}
	if len(setting.Service.EmailDomainAllowList) == 0 {
		return !request.IsEmailDomainListed(setting.Service.EmailDomainBlockList, c.Email)
	}

	return request.IsEmailDomainListed(setting.Service.EmailDomainAllowList, c.Email)
}

// ProjectInfoRequest структура запроса для получения информации о проекте
type UserInfoRequest struct {
	UserKey string `json:"user_key" binding:"Required"`
}

// ProjectInfoResponse структура ответа информации о проекте
type UserInfoResponse struct {
	ID         int64               `json:"id"`
	UserKey    string              `json:"user_key"`
	Name       string              `json:"login_name"`
	FullName   string              `json:"full_name"`
	Email      string              `json:"email"`
	Visibility structs.VisibleType `json:"visibility"`
}

// BindFromContext получение данных из контекста
func (c *UserInfoRequest) BindFromContext(ctx *context.APIContext) error {
	if ctx == nil || ctx.Base == nil || ctx.Base.Req == nil {
		return fmt.Errorf("Err: request incorrect")
	}
	if err := ctx.Base.Req.ParseForm(); err != nil {
		return fmt.Errorf("parse form: %w", err)
	}
	c.UserKey = ctx.Base.Req.Form.Get("user_key")
	return nil
}

func (c *UserInfoRequest) Validate() error {
	if c.UserKey == "" {
		return fmt.Errorf("user_key are require")
	}
	return nil
}
