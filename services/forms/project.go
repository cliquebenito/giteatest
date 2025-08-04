package forms

import (
	"fmt"
	"strings"

	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/structs"
)

// nameSize - максимальная длинна имени
// descSize - максимальная длинна описания
// projectKeySize - максимальная длинна ключа проекта
const (
	nameSize       = 40
	descSize       = 255
	projectKeySize = 50
)

// CreateProjectRequest структура запроса для создания проекта
type CreateProjectRequest struct {
	TenantKey   string              `json:"tenant_key" binding:"Required"`
	Name        string              `json:"name" binding:"Required"`
	ProjectKey  string              `json:"project_key" binding:"Required"`
	Description string              `json:"description"`
	Visibility  structs.VisibleType `json:"visibility" binding:"Required"`
}

// CreateProjectResponse структура ответа созданного проекта
type CreateProjectResponse struct {
	Id         int64               `json:"id"`
	Name       string              `json:"name" binding:"required"`
	ProjectKey string              `json:"project_key"`
	Visibility structs.VisibleType `json:"visibility"`
	Uri        string              `json:"uri"`
}

// Validate валидация запроса.
func (c *CreateProjectRequest) Validate() error {
	if len(c.Name) > nameSize || len(c.Name) <= 0 {
		log.Debug("incorrect length of the field name")
		return fmt.Errorf("Err: wrong name length")
	}
	if len(c.ProjectKey) > projectKeySize || len(c.ProjectKey) <= 0 {
		log.Debug("incorrect length of the field project_key")
		return fmt.Errorf("Err: wrong project_key length")
	}
	if len(c.Description) > descSize {
		log.Debug("incorrect length description")
		return fmt.Errorf("Err: incorrect length description")
	}
	if strings.Contains(c.Name, ".") {
		return fmt.Errorf("Err: name contains a dot (.)")
	}
	if !(c.Visibility.IsLimited() || c.Visibility.IsPrivate()) {
		return fmt.Errorf("Err: wrong visibility")
	}
	return nil
}

// ProjectInfoRequest структура запроса для получения информации о проекте
type ProjectInfoRequest struct {
	TenantKey  string `json:"tenant_key" binding:"Required"`
	ProjectKey string `json:"project_key" binding:"Required"`
}

// ProjectInfoResponse структура ответа информации о проекте
type ProjectInfoResponse struct {
	Id         int64               `json:"id"`
	Name       string              `json:"name"`
	ProjectKey string              `json:"project_key"`
	Visibility structs.VisibleType `json:"visibility"`
	Uri        string              `json:"uri"`
}

// BindFromContext получение данных из контекста
func (c *ProjectInfoRequest) BindFromContext(ctx *context.APIContext) error {
	if ctx.Base == nil || ctx.Base.Req == nil {
		return fmt.Errorf("invalid request")
	}
	if err := ctx.Base.Req.ParseForm(); err != nil {
		return fmt.Errorf("parse form: %w", err)
	}
	c.TenantKey = ctx.Base.Req.Form.Get("tenant_key")
	c.ProjectKey = ctx.Base.Req.Form.Get("project_key")
	return nil
}

func (c *ProjectInfoRequest) Validate() error {
	if c.TenantKey == "" || c.ProjectKey == "" {
		return fmt.Errorf("tenant key and project key are required")
	}
	return nil
}
