// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package user

import (
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/role_model"
	"code.gitea.io/gitea/models/tenant"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/services/convert"
	"fmt"
	"net/http"
)

// Search search users
// if setting.SbtKeycloakForm.Enabled search in keycloak, otherwise in db - todo after full integration with СУДИР
// now search users in keycloak and db
func Search(ctx *context.Context) {
	orgID := ctx.FormInt64("uid")
	listOptions := db.ListOptions{
		Page:     ctx.FormInt("page"),
		PageSize: convert.ToCorrectPageSize(ctx.FormInt("limit")),
	}

	var usersResult []*user_model.User
	userMap := make(map[string]*user_model.User, 0)
	var maxResultsRes int64
	var err error
	if setting.SbtKeycloakForm.Enabled {
		users, err := user_model.SearchUsersInKeycloak(ctx.FormTrim("q"))
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, map[string]interface{}{
				"ok":    false,
				"error": err.Error(),
			})
			return
		}
		maxResults, err := user_model.GetUsersTotalCountFromKeycloak()
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, map[string]interface{}{
				"ok":    false,
				"error": err.Error(),
			})
			return
		}
		for _, user := range users {
			userMap[user.Email] = user
		}
		maxResultsRes += maxResults
	}
	tenantID, err := tenant.GetTenantByOrgIdOrDefault(ctx, orgID)
	if err != nil {
		log.Error("Search tenant.GetTenantByOrgIdOrDefault failed while getting tenant id by org id: %v", err)
		ctx.JSON(http.StatusNotFound, fmt.Sprintf("Search tenant.GetTenantByOrgIdOrDefault failed: %v", err))
		return
	}
	privilegesByTenantID, err := role_model.GetPrivilegesByTenant(tenantID)
	if err != nil {
		log.Error("Search role_model.GetPrivilegesByTenant failed while getting group privileges by tenant id: %v", err)
		ctx.JSON(http.StatusNotFound, fmt.Sprintf("Search role_model.GetPrivilegesByTenant failed: %v", err))
		return
	}
	userIDs := make([]int64, len(privilegesByTenantID))
	for idx := range privilegesByTenantID {
		userIDs[idx] = privilegesByTenantID[idx].User.ID
	}
	users, maxResults, err := user_model.SearchUsers(&user_model.SearchUserOptions{
		Actor:       ctx.Doer,
		Keyword:     ctx.FormTrim("q"),
		UID:         ctx.FormInt64("uid"),
		Type:        user_model.UserTypeIndividual,
		IsActive:    ctx.FormOptionalBool("active"),
		ListOptions: listOptions,
		UserIDs:     userIDs,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, map[string]interface{}{
			"ok":    false,
			"error": err.Error(),
		})
		return
	}

	for _, user := range users {
		if _, ok := userMap[user.Email]; !ok {
			userMap[user.Email] = user
			maxResultsRes++
		}
	}

	for _, user := range userMap {
		usersResult = append(usersResult, user)
	}

	maxResultsRes += maxResults
	ctx.SetTotalCountHeader(maxResults)

	ctx.JSON(http.StatusOK, map[string]interface{}{
		"ok":   true,
		"data": convert.ToUsers(ctx, ctx.Doer, usersResult),
	})
}
