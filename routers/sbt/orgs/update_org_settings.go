package orgs

import (
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/organization"
	repoModel "code.gitea.io/gitea/models/repo"
	userModel "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/cache"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/modules/web"
	apiError "code.gitea.io/gitea/routers/sbt/apierror"
	sbtCache "code.gitea.io/gitea/routers/sbt/cache"
	"code.gitea.io/gitea/routers/sbt/convert"
	"code.gitea.io/gitea/routers/sbt/logger"
	"code.gitea.io/gitea/routers/sbt/request"
	orgService "code.gitea.io/gitea/services/org"
	repoService "code.gitea.io/gitea/services/repository"
	"net/http"
	"strings"
)

// UpdateOrgSettings метод обновления настроек организации
func UpdateOrgSettings(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	req := web.GetForm(ctx).(*request.UpdateOrgSettingsOptional)
	org := ctx.Org.Organization

	if req.Name != nil && org.Name != *req.Name {
		updateOrganizationName(ctx, org, *req.Name)
		if ctx.Written() {
			return
		}

		org.Name = *req.Name
		org.LowerName = strings.ToLower(*req.Name)
	}

	if req.MaxRepoCreation != nil {
		if *req.MaxRepoCreation < -1 {
			org.MaxRepoCreation = -1
		} else {
			org.MaxRepoCreation = *req.MaxRepoCreation
		}
	}
	if req.FullName != nil {
		org.FullName = *req.FullName
	}
	if req.Description != nil {
		org.Description = *req.Description
	}
	if req.Website != nil {
		org.Website = *req.Website
	}
	if req.Location != nil {
		org.Location = *req.Location
	}
	if req.RepoAdminChangeTeamAccess != nil {
		org.RepoAdminChangeTeamAccess = *req.RepoAdminChangeTeamAccess
	}

	var visibilityChanged bool
	if req.Visibility != nil && org.Visibility.String() != *req.Visibility {
		v, _ := structs.VisibilityModes[*req.Visibility]
		org.Visibility = v
		visibilityChanged = true
	}

	//Обновление настроек организации
	if err := userModel.UpdateUser(ctx, org.AsUser(), false); err != nil {
		log.Error("Unknown error type has occurred while updating organization's settings orgId: %d. Error: %v", org.ID, err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		return
	}

	// Если изменился тип видимости организации, то изменяется таблица доступов к репозиториям
	if visibilityChanged {
		repos, _, err := repoModel.GetUserRepositories(&repoModel.SearchRepoOptions{
			Actor: org.AsUser(), Private: true, ListOptions: db.ListOptions{Page: 1, PageSize: org.NumRepos},
		})
		if err != nil {
			log.Error("Error has occurred while getting repositories list for orgId: %d. Error message: %v", org.ID, err)
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
			return
		}
		for _, repo := range repos {
			repo.OwnerName = org.Name
			if err := repoService.UpdateRepository(ctx, repo, true); err != nil {
				log.Error("Error has occurred while updating repository with repoId: %d. Error message: %v", repo.ID, err)
				ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
				return
			}
		}
	}

	ctx.JSON(http.StatusOK, convert.ToOrganizationSettings(org))

	cache.RemoveItem(sbtCache.GenerateUserKey(org.Name) + "*")
}

// updateOrganizationName метод переименования организации и обработки ошибок
func updateOrganizationName(ctx *context.Context, org *organization.Organization, newName string) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	if err := orgService.RenameOrganization(ctx, org, newName); err != nil {
		switch {
		case userModel.IsErrUserAlreadyExist(err):
			log.Debug("User with userName: %s can't change orgName: %s because new name: %s already exist", ctx.Doer.Name, org.Name, newName)
			ctx.JSON(http.StatusBadRequest, apiError.OrgsNameAlreadyExistError(newName))

		case db.IsErrNameReserved(err):
			log.Debug("User with userName: %s can't change orgName: %s because new name: %s is reserved", ctx.Doer.Name, org.Name, newName)
			ctx.JSON(http.StatusBadRequest, apiError.OrgsNameReservedError(newName))

		case db.IsErrNamePatternNotAllowed(err):
			log.Debug("User with userName: %s can't change orgName: %s because new name: %s pattern is not allowed", ctx.Doer.Name, org.Name, newName)
			ctx.JSON(http.StatusBadRequest, apiError.OrgsNamePatternNotAllowedError(newName))

		case db.IsErrNameCharsNotAllowed(err):
			log.Debug("User with userName: %s can't change orgName: %s because new name: %s has not allowed characters", ctx.Doer.Name, org.Name, newName)
			ctx.JSON(http.StatusBadRequest, apiError.OrgsNameHasNotAllowedCharsError(newName))

		default:
			log.Error("While user with username: %s changing organization's name orgId: %d unknown error type has occurred: %v", ctx.Doer.Name, org.ID, err)
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		}
	}
}
