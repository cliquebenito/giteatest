package repo

import (
	"code.gitea.io/gitea/models/organization"
	repoModel "code.gitea.io/gitea/models/repo"
	userModel "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/cache"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/web"
	apiError "code.gitea.io/gitea/routers/sbt/apierror"
	sbtCache "code.gitea.io/gitea/routers/sbt/cache"
	"code.gitea.io/gitea/routers/sbt/logger"
	"code.gitea.io/gitea/routers/sbt/request"
	repoService "code.gitea.io/gitea/services/repository"
	"net/http"
)

// ForkRepo делает форк репозитория
func ForkRepo(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	req := web.GetForm(ctx).(*request.ForkRepo)

	var forker *userModel.User // user/org that will own the fork
	if req.Organization == "" {
		forker = ctx.Doer
	} else {
		org, err := organization.GetOrgByName(ctx, req.Organization)
		if err != nil {
			if organization.IsErrOrgNotExist(err) {
				log.Debug("Organization with name: %s not found", req.Organization)
				ctx.JSON(http.StatusBadRequest, apiError.OrganizationNotFoundByNameError(req.Organization))
			} else {
				log.Error("Error has occurred while checking organization name: %s with error message: %s", req.Organization, err)
				ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
			}
			return
		}

		if !ctx.Doer.IsAdmin {
			isMember, err := org.IsOrgMember(ctx.Doer.ID)
			if err != nil {
				log.Error("Error has occurred while checking member right of user: in organization: %s with error message: %s", ctx.Doer.Name, req.Organization, err)
				ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
				return
			} else if !isMember {
				log.Debug("User:%s is not a member of organization with name: %s ", ctx.Doer.Name, req.Organization)
				ctx.JSON(http.StatusBadRequest, apiError.UserIsNotPartOfOrganizationError(ctx.Doer.Name, req.Organization))
				return
			}
		}
		forker = org.AsUser()
	}

	fork, err := repoService.ForkRepository(ctx, ctx.Doer, forker, repoService.ForkRepoOptions{
		BaseRepo:    ctx.Repo.Repository,
		Name:        req.Name,
		Description: req.Description,
	})
	if err != nil {
		switch {
		case repoModel.IsErrReachLimitOfRepo(err):
			log.Debug("Can not fork repository: %s for user: %s. Repository count limit is reached.", ctx.Repo.Repository.FullName(), forker.Name)
			ctx.JSON(http.StatusBadRequest, apiError.RepoCountLimitIsReached())
		case repoModel.IsErrRepoAlreadyExist(err):
			log.Debug("Can not fork repository: %s for user: %s. Repository already exists.", ctx.Repo.Repository.FullName(), forker.Name)
			ctx.JSON(http.StatusBadRequest, apiError.RepoAlreadyExists())
		case repoService.IsErrForkAlreadyExist(err):
			log.Debug("Can not fork repository with repoId: %d because fork already exist repoName: %s for user: %s", ctx.Repo.Repository.ID, err.(repoService.ErrForkAlreadyExist).ForkName, forker.Name)
			ctx.JSON(http.StatusBadRequest, apiError.ForkAlreadyExist(err.(repoService.ErrForkAlreadyExist).ForkName))
		case repoModel.IsErrCreateUserRepo(err):
			log.Debug("Can not migrate repository %s. Creating a repository outside the project is prohibited", ctx.Repo.Repository.FullName())
			ctx.JSON(http.StatusBadRequest, apiError.UserRepoCreate())
		default:
			log.Error("Error has occurred while forking repo: %s for user: %s with error message: %s", ctx.Repo.Repository.FullName(), forker.Name, err)
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		}
		return
	}

	ctx.JSON(http.StatusCreated, request.Repo{ID: fork.ID})

	cache.RemoveItem(sbtCache.GenerateRepoListKey(ctx.Doer.Name) + "*")
}
