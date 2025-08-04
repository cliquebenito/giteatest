// Copyright 2015 The Gogs Authors. All rights reserved.
// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package admin

import (
	"net/http"

	"github.com/google/uuid"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/organization"
	"code.gitea.io/gitea/models/role_model"
	"code.gitea.io/gitea/models/tenant"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
	api "code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/routers/api/v1/utils"
	"code.gitea.io/gitea/services/convert"
)

// CreateOrg api for create organization
func CreateOrg(ctx *context.APIContext) {
	// swagger:operation POST /admin/users/{username}/orgs admin adminCreateOrg
	// ---
	// summary: Create an organization
	// deprecated: true
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: username
	//   in: path
	//   description: username of the user that will own the created organization
	//   type: string
	//   required: true
	// - name: organization
	//   in: body
	//   required: true
	//   schema: { "$ref": "#/definitions/CreateOrgOption" }
	// responses:
	//   "201":
	//     "$ref": "#/responses/Organization"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "422":
	//     "$ref": "#/responses/validationError"
	if !setting.SourceControl.InternalProjectCreate && setting.SourceControl.Enabled && setting.SourceControl.TenantWithRoleModeEnabled {
		log.Debug("A variable in config for creating an internal project is disabled")
		ctx.Error(http.StatusForbidden, "Create internal project not allowed", nil)
		return
	}

	form := web.GetForm(ctx).(*api.CreateOrgOption)

	visibility := api.VisibleTypeLimited
	if form.Visibility != "" {
		visibility = api.VisibilityModes[form.Visibility]
		if visibility == api.VisibleTypePublic && setting.SourceControl.Enabled && setting.SourceControl.TenantWithRoleModeEnabled {
			log.Debug("Incorrect visibility while updating project with orgName '%s'", form.UserName)
			ctx.JSON(http.StatusBadRequest, "Incorrect visibility")
			return
		}
	}

	org := &organization.Organization{
		Name:        form.UserName,
		FullName:    form.FullName,
		Description: form.Description,
		Website:     form.Website,
		Location:    form.Location,
		IsActive:    true,
		Type:        user_model.UserTypeOrganization,
		Visibility:  visibility,
	}

	if err := organization.CreateOrganization(org, ctx.ContextUser); err != nil {
		if user_model.IsErrUserAlreadyExist(err) ||
			db.IsErrNameReserved(err) ||
			db.IsErrNameCharsNotAllowed(err) ||
			db.IsErrNamePatternNotAllowed(err) {
			ctx.Error(http.StatusUnprocessableEntity, "", err)
		} else {
			ctx.Error(http.StatusInternalServerError, "CreateOrganization", err)
		}
		return
	}

	if !setting.SourceControl.Enabled || !setting.SourceControl.TenantWithRoleModeEnabled {
		log.Debug("source.control.enabled or source.control.TenantWithRoleModeEnabled is not enabled")
		ctx.JSON(http.StatusCreated, convert.ToOrganization(ctx, org))
		return
	}

	if org.IsVisibilityLimited() {
		if err := role_model.AddProjectToInnerSource(org); err != nil {
			log.Error("Error has occurred while adding project as an inner source project: %v", err)
			ctx.JSON(http.StatusInternalServerError, "Error has occurred while adding project as an inner source project")
			return
		}
	}
	tenantId, err := role_model.GetUserTenantId(ctx, ctx.Doer.ID)
	if err != nil {
		log.Error("Error has occurred while getting user tenant: %v", err)
		ctx.JSON(http.StatusInternalServerError, "Error has occurred while getting user tenant")
		return
	}

	tenantOrganization := &tenant.ScTenantOrganizations{
		ID:             uuid.NewString(),
		TenantID:       tenantId,
		OrganizationID: org.ID,
	}
	if err := organization.CreateRelationTenantOrganization(tenantOrganization); err != nil {
		log.Error("Error has occurred while creating tenant-organization relation: %v", err)
		ctx.JSON(http.StatusInternalServerError, "Error has occurred while creating tenant-organization relation")
		return
	}

	if err := role_model.GrantUserPermissionToOrganization(ctx.Doer, tenantId, org, role_model.OWNER); err != nil {
		log.Error("Error has occurred while granting owner privileges to organization: %v", err)
		ctx.JSON(http.StatusInternalServerError, "Error has occurred while granting owner privileges to organization")
		return
	}

	ctx.JSON(http.StatusCreated, convert.ToOrganization(ctx, org))
}

// GetAllOrgs API for getting information of all the organizations
func GetAllOrgs(ctx *context.APIContext) {
	// swagger:operation GET /admin/orgs admin adminGetAllOrgs
	// ---
	// summary: List all organizations
	// deprecated: true
	// produces:
	// - application/json
	// parameters:
	// - name: page
	//   in: query
	//   description: page number of results to return (1-based)
	//   type: integer
	// - name: limit
	//   in: query
	//   description: page size of results
	//   type: integer
	// responses:
	//   "200":
	//     "$ref": "#/responses/OrganizationList"
	//   "403":
	//     "$ref": "#/responses/forbidden"

	listOptions := utils.GetListOptions(ctx)

	users, maxResults, err := user_model.SearchUsers(&user_model.SearchUserOptions{
		Actor:       ctx.Doer,
		Type:        user_model.UserTypeOrganization,
		OrderBy:     db.SearchOrderByAlphabetically,
		ListOptions: listOptions,
		Visible:     []api.VisibleType{api.VisibleTypePublic, api.VisibleTypeLimited, api.VisibleTypePrivate},
	})
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "SearchOrganizations", err)
		return
	}
	orgs := make([]*api.Organization, len(users))
	for i := range users {
		orgs[i] = convert.ToOrganization(ctx, organization.OrgFromUser(users[i]))
	}

	ctx.SetLinkHeader(int(maxResults), listOptions.PageSize)
	ctx.SetTotalCountHeader(maxResults)
	ctx.JSON(http.StatusOK, &orgs)
}
