// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package user

import (
	context_general "context"
	"fmt"
	"net/http"

	"code.gitea.io/gitea/models/organization"
	"code.gitea.io/gitea/models/perm"
	access_model "code.gitea.io/gitea/models/perm/access"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/role_model"
	tenat_model "code.gitea.io/gitea/models/tenant"
	trace_model "code.gitea.io/gitea/models/trace"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/log"
	api "code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/modules/trace"
	"code.gitea.io/gitea/routers/api/v1/utils"
	"code.gitea.io/gitea/services/convert"
)

// listUserRepos - List the repositories owned by the given user.
func listUserRepos(ctx *context.APIContext, u *user_model.User, private bool) {
	ctxTrace := context_general.WithValue(ctx, trace_model.Key, "v1")
	ctxTrace = context_general.WithValue(ctxTrace, trace_model.EndpointKey, ctx.Req.RequestURI)
	ctxTrace = context_general.WithValue(ctxTrace, trace_model.FrontedKey, false)

	logTracer := trace.NewLogTracer()
	message := logTracer.CreateTraceMessage(ctx)
	err := logTracer.Trace(message)
	if err != nil {
		log.Error("Error has occurred while creating trace message: %v", err)
	}
	defer func() {
		err = logTracer.TraceTime(message)
		if err != nil {
			log.Error("Error has occurred while creating trace time message: %v", err)
		}
	}()

	opts := utils.GetListOptions(ctx)
	orgName := ctx.Params(":org")
	org, err := organization.GetOrgByName(ctx, orgName)
	if err != nil {
		if organization.IsErrOrgNotExist(err) {
			ctx.Error(http.StatusUnprocessableEntity, fmt.Sprintf("Organization with name %s doesn't exist", orgName), err)
		} else {
			ctx.Error(http.StatusInternalServerError, fmt.Sprintf("Error has occurred while getting organization by name %s", orgName), err)
		}
		return
	}
	tenantID, err := tenat_model.GetTenantByOrgIdOrDefault(ctx, org.ID)
	if err != nil {
		log.Error("Error has occurred while getting tenant by organization id: %v", err)
		ctx.Error(http.StatusInternalServerError, "Error has occurred while getting tenant by organization id", err)
		return
	}

	repositoriesOrg, err := organization.GetOrgRepositories(ctx, org.ID)
	if err != nil {
		log.Error("Error has occurred while getting repositories by org_id %d : %v", org.ID, err)
		ctx.Error(http.StatusInternalServerError, "Error has occurred while getting repositories by org_id", err)
		return
	}

	allowRepoIDs := make([]int64, 0)
	for idx := range repositoriesOrg {

		action := role_model.READ
		if repositoriesOrg[idx].IsPrivate {
			action = role_model.READ_PRIVATE
		}
		allowed, err := role_model.CheckUserPermissionToOrganization(ctx, u, tenantID, &organization.Organization{ID: org.ID}, action)
		if err != nil {
			log.Error("Error has occurred while checking user permission to organization: %v", err)
			continue
		}
		if !allowed {
			allow, err := role_model.CheckUserPermissionToTeam(ctx, u, tenantID, &organization.Organization{ID: org.ID}, &repo_model.Repository{ID: repositoriesOrg[idx].ID}, role_model.ViewBranch.String())
			if err != nil {
				log.Error("Error has occurred while checking user permission to organization: %v", err)
				ctx.JSON(http.StatusInternalServerError, fmt.Sprintf("Error has occurred while checking user's permissions: %v", err))
				return
			}
			if !allow {
				continue
			}
		}
		allowRepoIDs = append(allowRepoIDs, repositoriesOrg[idx].ID)
	}

	repos, count, err := repo_model.GetUserRepositories(&repo_model.SearchRepoOptions{
		Actor:          u,
		OwnerID:        u.ID,
		OwnerIDs:       []int64{u.ID},
		Private:        private,
		ListOptions:    opts,
		OrderBy:        "id ASC",
		AllowedRepoIDs: allowRepoIDs,
	})
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "GetUserRepositories", err)
		return
	}

	if err := repos.LoadAttributes(ctx); err != nil {
		ctx.Error(http.StatusInternalServerError, "RepositoryList.LoadAttributes", err)
		return
	}

	apiRepos := make([]*api.Repository, 0, len(repos))
	for i := range repos {
		access, err := access_model.AccessLevel(ctx, ctx.Doer, repos[i])
		if err != nil {
			ctx.Error(http.StatusInternalServerError, "AccessLevel", err)
			return
		}
		if ctx.IsSigned && ctx.Doer.IsAdmin || access >= perm.AccessModeRead {
			apiRepos = append(apiRepos, convert.ToRepo(ctx, repos[i], access))
		}
	}

	ctx.SetLinkHeader(int(count), opts.PageSize)
	ctx.SetTotalCountHeader(count)
	ctx.JSON(http.StatusOK, &apiRepos)
}

// ListUserRepos - list the repos owned by the given user.
func ListUserRepos(ctx *context.APIContext) {
	// swagger:operation GET /users/{username}/repos user userListRepos
	// ---
	// summary: List the repos owned by the given user
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
	//     "$ref": "#/responses/RepositoryList"

	private := ctx.IsSigned
	listUserRepos(ctx, ctx.ContextUser, private)
}

// ListMyRepos - list the repositories you own or have access to.
func ListMyRepos(ctx *context.APIContext) {
	// swagger:operation GET /user/repos user userCurrentListRepos
	// ---
	// summary: List the repos that the authenticated user owns
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
	//     "$ref": "#/responses/RepositoryList"

	opts := &repo_model.SearchRepoOptions{
		ListOptions:        utils.GetListOptions(ctx),
		Actor:              ctx.Doer,
		OwnerID:            ctx.Doer.ID,
		Private:            ctx.IsSigned,
		IncludeDescription: true,
	}

	var err error
	repos, count, err := repo_model.SearchRepository(ctx, opts)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "SearchRepository", err)
		return
	}

	results := make([]*api.Repository, len(repos))
	for i, repo := range repos {
		if err = repo.LoadOwner(ctx); err != nil {
			ctx.Error(http.StatusInternalServerError, "LoadOwner", err)
			return
		}
		accessMode, err := access_model.AccessLevel(ctx, ctx.Doer, repo)
		if err != nil {
			ctx.Error(http.StatusInternalServerError, "AccessLevel", err)
		}
		results[i] = convert.ToRepo(ctx, repo, accessMode)
	}

	ctx.SetLinkHeader(int(count), opts.ListOptions.PageSize)
	ctx.SetTotalCountHeader(count)
	ctx.JSON(http.StatusOK, &results)
}

// ListOrgRepos - list the repositories of an organization.
func ListOrgRepos(ctx *context.APIContext) {
	// swagger:operation GET /orgs/{org}/repos organization orgListRepos
	// ---
	// summary: List an organization's repos
	// deprecated: true
	// produces:
	// - application/json
	// parameters:
	// - name: org
	//   in: path
	//   description: name of the organization
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
	//     "$ref": "#/responses/RepositoryList"

	listUserRepos(ctx, ctx.Org.Organization.AsUser(), ctx.IsSigned)
}
