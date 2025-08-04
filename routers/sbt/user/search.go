package user

import (
	"code.gitea.io/gitea/models/db"
	userModel "code.gitea.io/gitea/models/user"
	ctx "code.gitea.io/gitea/modules/context"
	apiError "code.gitea.io/gitea/routers/sbt/apierror"
	sbtConvert "code.gitea.io/gitea/routers/sbt/convert"
	"code.gitea.io/gitea/routers/sbt/logger"
	"code.gitea.io/gitea/routers/sbt/response"
	"code.gitea.io/gitea/services/convert"
	"net/http"
)

// Search поиск пользователей по критериям (имя, сортировка, параметры пагинирования)
func Search(ctx *ctx.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)
	listOptions := db.ListOptions{
		Page:     ctx.FormInt("page"),
		PageSize: convert.ToCorrectPageSize(ctx.FormInt("limit")),
	}

	users, count, err := userModel.SearchUsers(&userModel.SearchUserOptions{
		Actor:       ctx.Doer,
		Keyword:     ctx.FormTrim("q"),
		Type:        userModel.UserTypeIndividual,
		IsActive:    ctx.FormOptionalBool("active"),
		OrderBy:     GetSearchOrderQuery(ctx.FormString("sort")),
		ListOptions: listOptions,
	})
	if err != nil {
		log.Error("An error occurred while search users, err: %v", err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		return
	}

	apiUsers := make([]*response.User, 0, len(users))

	for i := range users {
		apiUsers = append(apiUsers, sbtConvert.ToUser(ctx, users[i], ctx.Doer))
	}

	ctx.JSON(http.StatusOK,
		response.UserListResults{
			Total: count,
			Data:  apiUsers,
		})
}

// GetSearchOrderQuery Формируем сортировку в виде части SQL запроса
// мы не можем использовать orderBy из `models.SearchOrderByXxx`, потому, что может произойти JOIN различных таблиц с одинаковыми именами колонок
func GetSearchOrderQuery(name string) db.SearchOrderBy {
	switch name {
	case "newest":
		return "`user`.id DESC"
	case "oldest":
		return "`user`.id ASC"
	case "recentupdate":
		return "`user`.updated_unix DESC"
	case "leastupdate":
		return "`user`.updated_unix ASC"
	case "reversealphabetically":
		return "`user`.name DESC"
	case "lastlogin":
		return "`user`.last_login_unix ASC"
	case "reverselastlogin":
		return "`user`.last_login_unix DESC"
	case "alphabetically":
		return "`user`.name ASC"
	default:
		return "`user`.updated_unix DESC"
	}
}
