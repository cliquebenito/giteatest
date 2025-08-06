// Copyright 2015 The Gogs Authors. All rights reserved.
// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package user

import (
	"fmt"

	"code.gitea.io/gitea/models/tenant"
	"code.gitea.io/gitea/modules/log"
	context_service "code.gitea.io/gitea/services/context"

	"code.gitea.io/gitea/models/role_model"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/setting"
)

// Action response for follow/unfollow user request
func Action(ctx *context.Context) {
	var err error

	if setting.SourceControl.MultiTenantEnabled {
		context_service.UserAssignmentWeb()(ctx)

		tenantIDsDoer, err := getTenantsForUserOrOrganization(ctx, ctx.Doer)
		if err != nil {
			log.Error("Error has occurred while getting tenants for ctx.Doer: %v", err)
			ctx.NotFound(ctx.Req.URL.RequestURI(), nil)
			return
		}

		tenantIDsContextUser, err := getTenantsForUserOrOrganization(ctx, ctx.ContextUser)
		if err != nil {
			log.Error("Error has occurred while getting tenants for ctx.ContextUser: %v", err)
			ctx.NotFound(ctx.Req.URL.RequestURI(), nil)
			return
		}

		var isSequenceTenant bool
		for _, tenantIDDoer := range tenantIDsDoer {
			for _, tenantIDContextUser := range tenantIDsContextUser {
				if tenantIDDoer == tenantIDContextUser {
					isSequenceTenant = true
					break
				}
			}
		}
		if !isSequenceTenant {
			log.Warn("user does not have access to organization")
			ctx.NotFound(ctx.Req.URL.RequestURI(), nil)
			return
		}
	}

	switch ctx.FormString("action") {
	case "follow":
		err = user_model.FollowUser(ctx.Doer.ID, ctx.ContextUser.ID)
	case "unfollow":
		err = user_model.UnfollowUser(ctx.Doer.ID, ctx.ContextUser.ID)
	}

	if err != nil {
		ctx.ServerError(fmt.Sprintf("Action (%s)", ctx.FormString("action")), err)
		return
	}
	// FIXME: We should check this URL and make sure that it's a valid Gitea URL
	ctx.RedirectToFirst(ctx.FormString("redirect_to"), ctx.ContextUser.HomeLink())
}

// getTenantsForUser получает идентификаторы тенантов, в которых состоит пользователь из контекста.
func getTenantsForUser(ctx *context.Context, user *user_model.User) ([]string, error) {
	tenantIDsUser, err := role_model.GetUserTenantIDsOrDefaultTenantID(user)
	if err != nil {
		log.Error("Error has occurred while getting tenants for user: %v", err)
		return nil, fmt.Errorf("getting users from tenants for user: %w", err)
	}
	return tenantIDsUser, nil
}

// getTenantsForOrganization получает тенант id для организации из контекста.
func getTenantsForOrganization(ctx *context.Context, user *user_model.User) ([]string, error) {
	tenantOrganization, err := tenant.GetTenantOrganizationsByOrgId(ctx, user.ID)
	if err != nil {
		log.Error("Error has occurred while getting tenants for organization by org id: %v", err)
		return nil, fmt.Errorf("getting tenants for organization by org id: %w", err)
	}
	return []string{tenantOrganization.TenantID}, nil
}

// getTenantsForUserOrOrganization проверяет, является ли user пользователем или организацией,
// и в зависимости от этого вызывает нужный метод для получения идентификаторов тенантов.
func getTenantsForUserOrOrganization(ctx *context.Context, user *user_model.User) ([]string, error) {
	var tenantIDsForUser []string
	var err error
	if user.IsOrganization() {
		if tenantIDsForUser, err = getTenantsForOrganization(ctx, user); err != nil {
			log.Error("Error has occurred while getting tenant for organization: %v", err)
			return nil, fmt.Errorf("getting tenant for organization: %w", err)
		}
	} else {
		if tenantIDsForUser, err = getTenantsForUser(ctx, user); err != nil {
			log.Error("Error has occurred while getting tenant for user: %v", err)
			return nil, fmt.Errorf("getting tenant for user: %w", err)
		}
	}
	return tenantIDsForUser, nil
}
