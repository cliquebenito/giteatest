package user

import (
	activitiesModel "code.gitea.io/gitea/models/activities"
	"code.gitea.io/gitea/models/db"
	userModel "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/setting"
	apiError "code.gitea.io/gitea/routers/sbt/apierror"
	"code.gitea.io/gitea/routers/sbt/convert"
	"code.gitea.io/gitea/routers/sbt/logger"
	"code.gitea.io/gitea/routers/sbt/response"
	"net/http"
)

// GetActivityHeatMap возвращает мапу активности ТЕКУЩЕГО пользователя в виде пар (дата - количество активностей)
// В число активностей входят все активности совершенные в репозиториях пользователя, то есть не только те, которые совершает пользователь.
// Так же в хит мапу включены все активности, в том числе активности в приватных репозиториях
func GetActivityHeatMap(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	ctxUser := getDashboardContextUser(ctx)

	data, err := activitiesModel.GetUserHeatmapDataSbt(ctxUser, ctx.Org.Team, ctx.Doer, false, true)
	if err != nil {
		log.Error("Not able to get heatmap data due to error: %v", err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
	}

	ctx.JSON(http.StatusOK, convert.ToHeatMapData(data))
}

// GetActivities возвращает список активностей ТЕКУЩЕГО пользователя с пагинированием, с возможностью фильтрации по дате
// В список активностей входят все активности совершенные в репозиториях пользователя, то есть не только те, которые совершает пользователь.
// Так же включены активности в приватных репозиториях
func GetActivities(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	ctxUser := getDashboardContextUser(ctx)

	var (
		date = ctx.FormString("date")
		page = ctx.FormInt("page")
	)

	if page <= 1 {
		page = 1
	}

	feeds, count, err := activitiesModel.GetFeeds(ctx, activitiesModel.GetFeedsOptions{
		RequestedUser:   ctxUser,
		RequestedTeam:   ctx.Org.Team,
		Actor:           ctx.Doer,
		IncludePrivate:  true,
		OnlyPerformedBy: false,
		IncludeDeleted:  false,
		Date:            date,
		ListOptions: db.ListOptions{
			Page:     page,
			PageSize: setting.UI.FeedPagingNum,
		},
	})
	if err != nil {
		log.Error("Not able to get user activities data due to error: %v", err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
	}

	data := make([]*response.Action, len(feeds))

	for i := range feeds {
		data[i] = convert.ToAction(feeds[i])
	}

	ctx.JSON(http.StatusOK,
		response.ActionListResults{
			Total: count,
			Data:  data,
		})
}

// getDashboardContextUser получает пользователя для которого формируется доска активности.
func getDashboardContextUser(ctx *context.Context) *userModel.User {
	ctxUser := ctx.Doer
	orgName := ctx.Params(":org")
	if len(orgName) > 0 {
		ctxUser = ctx.Org.Organization.AsUser()
	}

	return ctxUser
}

// GetUserActivities метод получения списка активностей запрашиваемого пользователя с пагинацией и сортировкой по дате
// В список активностей включены только те действия, которые были совершены пользователем
// Этот метод не защищен аутентификацией, активности выводятся с учетом видимости репозиториев
func GetUserActivities(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	var (
		date = ctx.FormString("date")
		page = ctx.FormInt("page")
	)
	if page <= 1 {
		page = 1
	}

	showPrivate := ctx.IsSigned && (ctx.Doer.IsAdmin || ctx.Doer.ID == ctx.ContextUser.ID)

	feeds, count, err := activitiesModel.GetFeeds(ctx, activitiesModel.GetFeedsOptions{
		RequestedUser:   ctx.ContextUser,
		Actor:           ctx.Doer,
		IncludePrivate:  showPrivate,
		OnlyPerformedBy: true,
		IncludeDeleted:  false,
		Date:            date,
		ListOptions: db.ListOptions{
			Page:     page,
			PageSize: setting.UI.FeedPagingNum,
		},
	})
	if err != nil {
		log.Error("Not able to get activities for user with userId: %d due to error: %v", ctx.ContextUser.ID, err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
	}

	data := make([]*response.Action, len(feeds))

	for i := range feeds {
		data[i] = convert.ToAction(feeds[i])
	}

	ctx.JSON(http.StatusOK,
		response.ActionListResults{
			Total: count,
			Data:  data,
		})
}

// GetUserActivityHeatMap метод возвращает мапу активности пользователя в виде пар (дата - количество активностей)
// В число активностей входят только активности пользователя
// Так же в хит мапу включены все активности с учетом видимости репозиториев
func GetUserActivityHeatMap(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	showPrivate := ctx.IsSigned && (ctx.Doer.IsAdmin || ctx.Doer.ID == ctx.ContextUser.ID)

	data, err := activitiesModel.GetUserHeatmapDataSbt(ctx.ContextUser, ctx.Org.Team, ctx.Doer, true, showPrivate)
	if err != nil {
		log.Error("Not able to get heatmap data due to error: %v", err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
	}

	ctx.JSON(http.StatusOK, convert.ToHeatMapData(data))
}
