// Copyright 2015 The Gogs Authors. All rights reserved.
// Copyright 2018 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package org

import (
	"net/http"

	"github.com/google/uuid"

	activities_model "code.gitea.io/gitea/models/activities"
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/organization"
	"code.gitea.io/gitea/models/perm"
	"code.gitea.io/gitea/models/role_model"
	"code.gitea.io/gitea/models/tenant"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
	api "code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/routers/api/v1/user"
	"code.gitea.io/gitea/routers/api/v1/utils"
	"code.gitea.io/gitea/services/convert"
	"code.gitea.io/gitea/services/org"
)

func listUserOrgs(ctx *context.APIContext, u *user_model.User) {
	listOptions := utils.GetListOptions(ctx)
	showPrivate := ctx.IsSigned && (ctx.Doer.IsAdmin || ctx.Doer.ID == u.ID)

	opts := organization.FindOrgOptions{
		ListOptions:    listOptions,
		UserID:         u.ID,
		IncludePrivate: showPrivate,
	}
	orgs, err := organization.FindOrgs(opts)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "FindOrgs", err)
		return
	}
	maxResults, err := organization.CountOrgs(opts)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "CountOrgs", err)
		return
	}

	apiOrgs := make([]*api.Organization, len(orgs))
	for i := range orgs {
		apiOrgs[i] = convert.ToOrganization(ctx, orgs[i])
	}

	ctx.SetLinkHeader(int(maxResults), listOptions.PageSize)
	ctx.SetTotalCountHeader(maxResults)
	ctx.JSON(http.StatusOK, &apiOrgs)
}

// ListMyOrgs list all my orgs
func ListMyOrgs(ctx *context.APIContext) {
	// swagger:operation GET /user/orgs organization orgListCurrentUserOrgs
	// ---
	// summary: List the current user's organizations
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
	listUserOrgs(ctx, ctx.Doer)
}

// ListUserOrgs list user's orgs
func ListUserOrgs(ctx *context.APIContext) {
	// swagger:operation GET /users/{username}/orgs organization orgListUserOrgs
	// ---
	// summary: List a user's organizations
	// deprecated: true
	// produces:
	// - application/json
	// parameters:
	// - name: username
	//   in: path
	//   description: username of user
	//   type: string
	//   required: true
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
	listUserOrgs(ctx, ctx.ContextUser)
}

// GetUserOrgsPermissions get user permissions in organization
func GetUserOrgsPermissions(ctx *context.APIContext) {
	// swagger:operation GET /users/{username}/orgs/{org}/permissions organization orgGetUserPermissions
	// ---
	// summary: Get user permissions in organization
	// deprecated: true
	// produces:
	// - application/json
	// parameters:
	// - name: username
	//   in: path
	//   description: username of user
	//   type: string
	//   required: true
	// - name: org
	//   in: path
	//   description: name of the organization
	//   type: string
	//   required: true
	// responses:
	//   "200":
	//     "$ref": "#/responses/OrganizationPermissions"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"

	var o *user_model.User
	if o = user.GetUserByParamsName(ctx, ":org"); o == nil {
		return
	}

	op := api.OrganizationPermissions{}

	if !organization.HasOrgOrUserVisible(ctx, o, ctx.ContextUser) {
		ctx.NotFound("HasOrgOrUserVisible", nil)
		return
	}

	org := organization.OrgFromUser(o)
	authorizeLevel, err := org.GetOrgUserMaxAuthorizeLevel(ctx.ContextUser.ID)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "GetOrgUserAuthorizeLevel", err)
		return
	}

	if authorizeLevel > perm.AccessModeNone {
		op.CanRead = true
	}
	if authorizeLevel > perm.AccessModeRead {
		op.CanWrite = true
	}
	if authorizeLevel > perm.AccessModeWrite {
		op.IsAdmin = true
	}
	if authorizeLevel > perm.AccessModeAdmin {
		op.IsOwner = true
	}

	op.CanCreateRepository, err = org.CanCreateOrgRepo(ctx.ContextUser.ID)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "CanCreateOrgRepo", err)
		return
	}

	ctx.JSON(http.StatusOK, op)
}

// GetAll return list of all public organizations
func GetAll(ctx *context.APIContext) {
	// swagger:operation Get /orgs organization orgGetAll
	// ---
	// summary: Get list of organizations
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

	vMode := []api.VisibleType{api.VisibleTypePublic}
	if ctx.IsSigned {
		vMode = append(vMode, api.VisibleTypeLimited)
		if ctx.Doer.IsAdmin {
			vMode = append(vMode, api.VisibleTypePrivate)
		}
	}

	listOptions := utils.GetListOptions(ctx)

	publicOrgs, maxResults, err := user_model.SearchUsers(&user_model.SearchUserOptions{
		Actor:       ctx.Doer,
		ListOptions: listOptions,
		Type:        user_model.UserTypeOrganization,
		OrderBy:     db.SearchOrderByAlphabetically,
		Visible:     vMode,
	})
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "SearchOrganizations", err)
		return
	}
	orgs := make([]*api.Organization, len(publicOrgs))
	for i := range publicOrgs {
		orgs[i] = convert.ToOrganization(ctx, organization.OrgFromUser(publicOrgs[i]))
	}

	ctx.SetLinkHeader(int(maxResults), listOptions.PageSize)
	ctx.SetTotalCountHeader(maxResults)
	ctx.JSON(http.StatusOK, &orgs)
}

// Create api for create organization
func Create(ctx *context.APIContext) {
	// swagger:operation POST /orgs organization orgCreate
	// ---
	// summary: Create an organization
	// deprecated: true
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
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
	if !ctx.Doer.CanCreateOrganization() {
		ctx.Error(http.StatusForbidden, "Create organization not allowed", nil)
		return
	}

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
		Name:                      form.UserName,
		FullName:                  form.FullName,
		Description:               form.Description,
		Website:                   form.Website,
		Location:                  form.Location,
		IsActive:                  true,
		Type:                      user_model.UserTypeOrganization,
		Visibility:                visibility,
		RepoAdminChangeTeamAccess: form.RepoAdminChangeTeamAccess,
	}
	if err := organization.CreateOrganization(org, ctx.Doer); err != nil {
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

// Get get an organization
func Get(ctx *context.APIContext) {
	// swagger:operation GET /orgs/{org} organization orgGet
	// ---
	// summary: Get an organization
	// deprecated: true
	// produces:
	// - application/json
	// parameters:
	// - name: org
	//   in: path
	//   description: name of the organization to get
	//   type: string
	//   required: true
	// responses:
	//   "200":
	//     "$ref": "#/responses/Organization"

	if !organization.HasOrgOrUserVisible(ctx, ctx.Org.Organization.AsUser(), ctx.Doer) {
		ctx.NotFound("HasOrgOrUserVisible", nil)
		return
	}
	ctx.JSON(http.StatusOK, convert.ToOrganization(ctx, ctx.Org.Organization))
}

// Edit change an organization's information
func Edit(ctx *context.APIContext) {
	// swagger:operation PATCH /orgs/{org} organization orgEdit
	// ---
	// summary: Edit an organization
	// deprecated: true
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: org
	//   in: path
	//   description: name of the organization to edit
	//   type: string
	//   required: true
	// - name: body
	//   in: body
	//   required: true
	//   schema:
	//     "$ref": "#/definitions/EditOrgOption"
	// responses:
	//   "200":
	//     "$ref": "#/responses/Organization"
	form := web.GetForm(ctx).(*api.EditOrgOption)
	org := ctx.Org.Organization
	org.FullName = form.FullName
	org.Description = form.Description
	org.Website = form.Website
	org.Location = form.Location
	if form.Visibility != "" {
		org.Visibility = api.VisibilityModes[form.Visibility]
		if org.Visibility == api.VisibleTypePublic && setting.SourceControl.Enabled && setting.SourceControl.TenantWithRoleModeEnabled {
			log.Debug("Incorrect visibility while updating project with orgName '%s'", org.Name)
			ctx.JSON(http.StatusBadRequest, "Incorrect visibility")
			return
		}
	}
	if form.RepoAdminChangeTeamAccess != nil {
		org.RepoAdminChangeTeamAccess = *form.RepoAdminChangeTeamAccess
	}
	if err := user_model.UpdateUserCols(ctx, org.AsUser(),
		"full_name", "description", "website", "location",
		"visibility", "repo_admin_change_team_access",
	); err != nil {
		ctx.Error(http.StatusInternalServerError, "EditOrganization", err)
		return
	}

	ctx.JSON(http.StatusOK, convert.ToOrganization(ctx, org))
}

// Delete an organization
func Delete(ctx *context.APIContext) {
	// swagger:operation DELETE /orgs/{org} organization orgDelete
	// ---
	// summary: Delete an organization
	// deprecated: true
	// produces:
	// - application/json
	// parameters:
	// - name: org
	//   in: path
	//   description: organization that is to be deleted
	//   type: string
	//   required: true
	// responses:
	//   "204":
	//     "$ref": "#/responses/empty"

	if err := org.DeleteOrganization(ctx.Org.Organization); err != nil {
		ctx.Error(http.StatusInternalServerError, "DeleteOrganization", err)
		return
	}
	ctx.Status(http.StatusNoContent)
}

func ListOrgActivityFeeds(ctx *context.APIContext) {
	// swagger:operation GET /orgs/{org}/activities/feeds organization orgListActivityFeeds
	// ---
	// summary: List an organization's activity feeds
	// deprecated: true
	// produces:
	// - application/json
	// parameters:
	// - name: org
	//   in: path
	//   description: name of the org
	//   type: string
	//   required: true
	// - name: date
	//   in: query
	//   description: the date of the activities to be found
	//   type: string
	//   format: date
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
	//     "$ref": "#/responses/ActivityFeedsList"
	//   "404":
	//     "$ref": "#/responses/notFound"

	includePrivate := false
	if ctx.IsSigned {
		if ctx.Doer.IsAdmin {
			includePrivate = true
		} else {
			org := organization.OrgFromUser(ctx.ContextUser)
			isMember, err := org.IsOrgMember(ctx.Doer.ID)
			if err != nil {
				ctx.Error(http.StatusInternalServerError, "IsOrgMember", err)
				return
			}
			includePrivate = isMember
		}
	}

	listOptions := utils.GetListOptions(ctx)

	opts := activities_model.GetFeedsOptions{
		RequestedUser:  ctx.ContextUser,
		Actor:          ctx.Doer,
		IncludePrivate: includePrivate,
		Date:           ctx.FormString("date"),
		ListOptions:    listOptions,
	}

	feeds, count, err := activities_model.GetFeeds(ctx, opts)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "GetFeeds", err)
		return
	}
	ctx.SetTotalCountHeader(count)

	ctx.JSON(http.StatusOK, convert.ToActivities(ctx, feeds, ctx.Doer))
}
