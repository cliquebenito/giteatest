package orgs

import (
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/organization"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/modules/web"
	apiError "code.gitea.io/gitea/routers/sbt/apierror"
	"code.gitea.io/gitea/routers/sbt/convert"
	"code.gitea.io/gitea/routers/sbt/logger"
	"code.gitea.io/gitea/routers/sbt/request"
	"net/http"
)

// CreateOrganization метод создания организации пользователем
func CreateOrganization(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	req := web.GetForm(ctx).(*request.CreateOrganizationOptional)

	if !ctx.Doer.CanCreateOrganization() {
		log.Debug("User with username: %s can not create organization", ctx.Doer.Name)
		ctx.JSON(http.StatusBadRequest, apiError.UserNotAllowedCreateOrgsError())
		return
	}

	org := &organization.Organization{
		Name:     req.Name,
		IsActive: true,
		Type:     user_model.UserTypeOrganization,
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
	if req.Visibility != nil {
		org.Visibility = structs.VisibilityModes[*req.Visibility]
	} else {
		org.Visibility = structs.VisibleTypePublic
	}

	if err := organization.CreateOrganization(org, ctx.Doer); err != nil {
		switch {
		case user_model.IsErrUserAlreadyExist(err):
			log.Debug("User username: %s can't create organization because organization with name: %s already exist", ctx.Doer.Name, req.Name)
			ctx.JSON(http.StatusBadRequest, apiError.OrgsNameAlreadyExistError(req.Name))

		case db.IsErrNameReserved(err):
			log.Debug("User username: %s can't create organization because organization with name: %s is reserved", ctx.Doer.Name, req.Name)
			ctx.JSON(http.StatusBadRequest, apiError.OrgsNameReservedError(req.Name))

		case db.IsErrNamePatternNotAllowed(err):
			log.Debug("User with userName: %s can't create organization because organization name: %s pattern is not allowed", ctx.Doer.Name, req.Name)
			ctx.JSON(http.StatusBadRequest, apiError.OrgsNamePatternNotAllowedError(req.Name))

		case db.IsErrNameCharsNotAllowed(err):
			log.Debug("User with userName: %s can't create organization because organization name: %s contains not allowed characters", ctx.Doer.Name, req.Name)
			ctx.JSON(http.StatusBadRequest, apiError.OrgsNameHasNotAllowedCharsError(req.Name))

		default:
			log.Error("While user with username: %s create organization unknown error type has occurred: %v", ctx.Doer.Name, err)
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		}
		return
	}

	newOrg, err := organization.GetOrgByName(ctx, req.Name)
	if err != nil {
		if user_model.IsErrUserNotExist(err) || organization.IsErrOrgNotExist(err) {
			log.Debug("Created organization with name: %s is not exist", req.Name)
			ctx.JSON(http.StatusBadRequest, apiError.OrganizationNotFoundByNameError(req.Name))
		} else {
			log.Error("While getting created organization with name: %s unknown error type has occurred: %v", req.Name, err)
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		}
		return
	}

	ctx.JSON(http.StatusCreated, convert.ToOrganization(ctx, newOrg))
}
