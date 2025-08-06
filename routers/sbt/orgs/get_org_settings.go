package orgs

import (
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/routers/sbt/convert"
	"code.gitea.io/gitea/routers/sbt/logger"
	"net/http"
)

// GetOrgSettings метод получения настроек организации
func GetOrgSettings(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	ctx.JSON(http.StatusOK, convert.ToOrganizationSettings(ctx.Org.Organization))
}
