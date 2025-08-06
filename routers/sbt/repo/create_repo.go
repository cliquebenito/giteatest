package repo

import (
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/organization"
	repoModel "code.gitea.io/gitea/models/repo"
	userModel "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/cache"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/label"
	repoModule "code.gitea.io/gitea/modules/repository"
	"code.gitea.io/gitea/modules/util"
	"code.gitea.io/gitea/modules/web"
	apiError "code.gitea.io/gitea/routers/sbt/apierror"
	sbtCache "code.gitea.io/gitea/routers/sbt/cache"
	"code.gitea.io/gitea/routers/sbt/logger"
	"code.gitea.io/gitea/routers/sbt/request"
	repoService "code.gitea.io/gitea/services/repository"
	"fmt"
	"net/http"
)

func CreateRepo(ctx *context.Context) {
	req := web.GetForm(ctx).(*request.CreateRepo)

	owner := checkContextUser(ctx, req.OrgId)
	if ctx.Written() {
		return
	}

	CreateUserRepo(ctx, owner, *req)
}

// CreateUserRepo создать репозиторий
func CreateUserRepo(ctx *context.Context, owner *userModel.User, reqOpt request.CreateRepo) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	if reqOpt.AutoInit && reqOpt.Readme == "" {
		reqOpt.Readme = "Default"
	}

	// If the readme template does not exist, a 400 will be returned.
	if reqOpt.AutoInit && len(reqOpt.Readme) > 0 && !util.SliceContains(repoModule.Readmes, reqOpt.Readme) {
		ctx.JSON(http.StatusBadRequest, apiError.ReadmeTemplateNotExists())
		return
	}

	opt := repoModule.CreateRepoOptions{
		Name:          reqOpt.Name,
		Description:   reqOpt.Description,
		IssueLabels:   reqOpt.IssueLabels,
		Gitignores:    reqOpt.Gitignores,
		License:       reqOpt.License,
		Readme:        reqOpt.Readme,
		IsPrivate:     reqOpt.Private,
		AutoInit:      reqOpt.AutoInit,
		DefaultBranch: reqOpt.DefaultBranch,
		TrustModel:    repoModel.ToTrustModel(reqOpt.TrustModel),
		IsTemplate:    reqOpt.Template,
	}

	repo, err := repoService.CreateRepository(ctx, ctx.Doer, owner, opt)

	if err != nil {
		log.Error("Error has occurred while creating new repo for user: %s with error message: %s", ctx.Doer.Name, err)

		if repoModel.IsErrRepoAlreadyExist(err) {
			ctx.JSON(http.StatusBadRequest, apiError.RepoAlreadyExists())
		} else if db.IsErrNameReserved(err) || db.IsErrNamePatternNotAllowed(err) {
			ctx.JSON(http.StatusBadRequest, apiError.RepoWrongName())
		} else if label.IsErrTemplateLoad(err) {
			ctx.JSON(http.StatusBadRequest, apiError.RepoWrongLabels())
		} else if repoModel.IsErrCreateUserRepo(err) {
			ctx.JSON(http.StatusBadRequest, apiError.UserRepoCreate())
		} else {
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		}

		return
	}

	ctx.JSON(http.StatusCreated, request.Repo{ID: repo.ID})

	cache.RemoveItem(sbtCache.GenerateRepoListKey(owner.Name) + "*")
}

// checkContextUser метод получения владельца создаваемого репозитория (это может быть организация или текущий пользователь)
func checkContextUser(ctx *context.Context, uid int64) *userModel.User {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	//Если uid == 0 или uid == Doer.ID т.е. идентификатор текущего пользователя, тогда возвращаем текущего пользователя
	//Иначе uid это id организации.
	if uid == ctx.Doer.ID || uid == 0 {
		return ctx.Doer
	}

	// Получаем организацию по идентификатору
	org, err := userModel.GetUserByID(ctx, uid)
	if userModel.IsErrUserNotExist(err) {
		return ctx.Doer
	}

	if err != nil {
		log.Error("Error has occurred while getting user by userId: %d. Error message: %v", uid, err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		return nil
	}

	// Если полученный id не организация, то возвращаем ошибку
	if !org.IsOrganization() {
		log.Debug("Can not create repository by userId: %s, because it is not an organization", uid)
		ctx.JSON(http.StatusBadRequest, apiError.UserNotOrganization())
		return nil
	}

	if !ctx.Doer.IsAdmin {
		canCreate, err := organization.OrgFromUser(org).CanCreateOrgRepo(ctx.Doer.ID)
		if err != nil {
			log.Error("Error has occurred while getting user with username: %s permission for creating repository in organization with orgId: %d. Error message: %v", ctx.Doer.Name, uid, err)
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
			return nil

		} else if !canCreate {
			log.Debug("User with username: %d have not permission to create repository in organization with id: %d", ctx.Doer.Name, uid)
			ctx.JSON(http.StatusBadRequest, apiError.UserInsufficientPermission(ctx.Doer.Name, fmt.Sprintf("create repository in orgId: %d", uid)))
			return nil
		}
	}

	return org
}
