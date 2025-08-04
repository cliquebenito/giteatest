package orgs

import (
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/organization"
	userModel "code.gitea.io/gitea/models/user"
	ctx "code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/structs"
	apiError "code.gitea.io/gitea/routers/sbt/apierror"
	sbtConvert "code.gitea.io/gitea/routers/sbt/convert"
	"code.gitea.io/gitea/routers/sbt/logger"
	"code.gitea.io/gitea/routers/sbt/response"
	"code.gitea.io/gitea/routers/sbt/user"
	"net/http"
)

// SearchOrgs поиск организаций по критериям (имя, сортировка, параметры пагинирования)
func SearchOrgs(ctx *ctx.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	pageSize := ctx.FormInt("limit")
	if pageSize == 0 {
		pageSize = setting.UI.ExplorePagingNum
	}
	pageOpts := db.ListOptions{
		Page:     ctx.FormInt("page"),
		PageSize: pageSize,
	}

	visibleTypes := []structs.VisibleType{structs.VisibleTypePublic}
	if ctx.Doer != nil {
		visibleTypes = append(visibleTypes, structs.VisibleTypeLimited, structs.VisibleTypePrivate)
	}

	users, count, err := userModel.SearchUsers(&userModel.SearchUserOptions{
		ListOptions: pageOpts,
		Actor:       ctx.Doer,
		Type:        userModel.UserTypeOrganization,
		OrderBy:     user.GetSearchOrderQuery(ctx.FormString("sort")),
		Visible:     visibleTypes,
		Keyword:     ctx.FormTrim("q"),
	})

	if err != nil {
		log.Error("An error occurred while search users, err: %v", err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())

		return
	}

	apiOrgs := make([]*response.Organization, 0, len(users))

	for i := range users {
		apiOrgs = append(apiOrgs, sbtConvert.ToOrganization(ctx, organization.OrgFromUser(users[i])))
	}

	ctx.JSON(http.StatusOK,
		response.OrganizationListResult{
			Total: count,
			Data:  apiOrgs,
		})
}
