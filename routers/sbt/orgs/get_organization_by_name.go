package orgs

import (
	"code.gitea.io/gitea/models/organization"
	"code.gitea.io/gitea/modules/context"
	apiError "code.gitea.io/gitea/routers/sbt/apierror"
	"code.gitea.io/gitea/routers/sbt/convert"
	"code.gitea.io/gitea/routers/sbt/logger"
	"net/http"
)

// GetOrganizationByName метод получения организации по ее имени
// Организация доступна в случае:
// - Если текущий пользователь является владельцем/участником организации
// - Если организация публично доступная
// В случае если организация не доступна для просмотра возвращается статус 400 (StatusBadRequest)
func GetOrganizationByName(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	if !organization.HasOrgOrUserVisible(ctx, ctx.Org.Organization.AsUser(), ctx.Doer) {
		log.Debug("Error organization: %s not found by name because organization is not visible for current user", ctx.Org.Organization.Name)
		ctx.JSON(http.StatusBadRequest, apiError.OrganizationNotFoundByNameError(ctx.Org.Organization.Name))
		return
	}

	ctx.JSON(http.StatusOK, convert.ToOrganization(ctx, ctx.Org.Organization))
}
