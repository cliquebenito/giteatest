package admin

import "code.gitea.io/gitea/routers/web/explore"

type Server struct {
	explore.Server
}

func New(exploreServer explore.Server) Server {
	return Server{exploreServer}
}
