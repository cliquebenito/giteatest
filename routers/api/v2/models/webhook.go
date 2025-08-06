package models

import (
	"fmt"
	"strings"

	"code.gitea.io/gitea/models/webhook"
	webhook2 "code.gitea.io/gitea/modules/webhook"
)

// CreateHookOption options when create a hook
// swagger:model CreateHookOption
type CreateHookOption struct {
	Type                string
	Config              CreateHookOptionConfig `json:"config" binding:"Required"`
	Events              []string               `json:"events" binding:"Required"`
	BranchFilter        string                 `json:"branch_filter" binding:"GlobPattern"`
	AuthorizationHeader string                 `json:"authorization_header"`
	Active              *bool                  `json:"active"`
	OwnerID             int64
	RepoID              int64
}

type CreateHookOptionConfig struct {
	ContentType string `json:"content_type" binding:"Required"`
	Url         string `json:"url" binding:"Required"`
	Secret      string `json:"secret"`
}

func (c *CreateHookOption) Validate() error {
	if !webhook.IsValidHookContentType(c.Config.ContentType) {
		return ErrInvalidHookContentType{}
	}
	c.Type = webhook2.SOURCECONTROL
	if c.Config.Url == "" {
		return fmt.Errorf("Err: config url is required")
	}
	if c.Config.ContentType == "" {
		return fmt.Errorf("Err: content type is required")
	}
	if !strings.HasPrefix(c.Config.Url, "http") {
		return fmt.Errorf("Err: incorrect URL prefix")
	}
	if c.Active == nil {
		t := true
		c.Active = &t
	}
	return nil
}
