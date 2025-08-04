package controller

import (
	"net/http"

	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/routers/sc/configurations/usecase"
)

type Server struct {
	uc *usecase.UC
}

func NewScConfigurations(uc *usecase.UC) *Server {
	return &Server{uc: uc}
}

func (s Server) Configurations(ctx *context.APIContext) {
	payload := s.uc.Configurations(ctx)
	ctx.JSON(http.StatusOK, payload)
}
