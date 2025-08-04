package usecase

import (
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/setting"
)

type UC struct {
}

type OneWorkConfig struct {
	Enabled            bool   `json:"enabled"`
	ServiceURL         string `json:"service_url"`
	ToolKey            string `json:"tool_key"`
	ContextURI         string `json:"context_uri"`
	MenuKey            string `json:"menu_key"`
	LoginLink          string `json:"login_link"`
	LogoutLink         string `json:"logout_link"`
	FallbackTimeout    int64  `json:"fallback_timeout"`
	ExtendedAdminPanel bool   `json:"extended_admin_panel"`
}

func NewUsecase() *UC {
	return &UC{}
}

type FullConfigResponse struct {
	OneWork OneWorkConfig `json:"sbt.one_work"`
}

func (u UC) Configurations(ctx *context.APIContext) *FullConfigResponse {
	return &FullConfigResponse{
		OneWork: OneWorkConfig{
			Enabled:            setting.OneWork.Enabled,
			ServiceURL:         setting.OneWork.ServiceURL,
			ToolKey:            setting.OneWork.ToolKey,
			ContextURI:         setting.OneWork.ContextURI,
			MenuKey:            setting.OneWork.MenuKey,
			LoginLink:          setting.OneWork.LoginLink,
			LogoutLink:         setting.OneWork.LogoutLink,
			FallbackTimeout:    int64(setting.OneWork.FallbackTimeout.Seconds()),
			ExtendedAdminPanel: setting.OneWork.ExtendedAdminPanel,
		},
	}
}
