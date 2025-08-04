package orgs

import (
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/organization"
	"code.gitea.io/gitea/modules/context"
	apiError "code.gitea.io/gitea/routers/sbt/apierror"
	convertSbt "code.gitea.io/gitea/routers/sbt/convert"
	"code.gitea.io/gitea/routers/sbt/logger"
	"code.gitea.io/gitea/routers/sbt/response"
	"code.gitea.io/gitea/services/convert"
	"net/http"
)

// GetCurrentUserListOrgs Метод получения списка организаций, принадлежащих текущему пользователю
func GetCurrentUserListOrgs(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	page := ctx.FormInt("page")
	if page <= 1 {
		page = 1
	}

	listOptions := db.ListOptions{
		PageSize: convert.ToCorrectPageSize(ctx.FormInt("limit")),
		Page:     page,
	}

	opts := organization.FindOrgOptions{
		ListOptions:    listOptions,
		UserID:         ctx.Doer.ID,
		IncludePrivate: true,
	}

	orgs, err := organization.FindOrgs(opts)
	if err != nil {
		log.Error("Error has occurred while getting organization list for current userName: %s. Error message: %v", ctx.Doer.Name, err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		return
	}

	total, err := organization.CountOrgs(opts)
	if err != nil {
		log.Error("Error has occurred while getting total count of organization list for current userName: %s. Error message: %v", ctx.Doer.Name, err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		return
	}

	resOrgs := make([]*response.Organization, len(orgs))
	for i := range orgs {
		resOrgs[i] = convertSbt.ToOrganization(ctx, orgs[i])
	}

	ctx.JSON(http.StatusOK, response.OrganizationListResult{Total: total, Data: resOrgs})
}
