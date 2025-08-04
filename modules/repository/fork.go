// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repository

import (
	"context"
	"fmt"

	"code.gitea.io/gitea/models/organization"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/role_model"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/trace"
)

// CanUserForkRepo returns true if specified user can fork repository.
func CanUserForkRepo(user *user_model.User, repo *repo_model.Repository) (bool, error) {
	if user == nil {
		return false, nil
	}
	if repo.OwnerID != user.ID && !repo_model.HasForkedRepo(user.ID, repo.ID) {
		return true, nil
	}
	ownedOrgs, err := organization.GetOrgsCanCreateRepoByUserID(user.ID)
	if err != nil {
		return false, err
	}
	for _, org := range ownedOrgs {
		if repo.OwnerID != org.ID && !repo_model.HasForkedRepo(org.ID, repo.ID) {
			return true, nil
		}
	}
	return false, nil
}

func CheckUserForkRepository(ctx context.Context, user *user_model.User, orgID int64, tenantID string) (bool, error) {
	// все привилегии пользователя
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

	allPrivileges, err := role_model.GetAllPrivileges()
	if err != nil {
		log.Error("Error has occurred while getting all privileges: %v", err)
		return false, fmt.Errorf("get all privileges: %w", err)
	}
	for _, privilege := range allPrivileges {
		if privilege.User.ID == user.ID {
			// если мы сейчас в этой организации - пропускаем
			if privilege.Org.ID == orgID {
				continue
			}

			// проверяем, является ли пользователь может создать репозиторий в организации
			if ok, err := role_model.CheckUserPermissionToOrganization(ctx, user, tenantID, privilege.Org, role_model.CREATE); err != nil {
				log.Error("Error has occured while checking user permission to organization: %v", err)
				return false, fmt.Errorf("check user permission to organization: %w", err)
			} else if ok {
				// если есть хотя бы еще одна организация в которой пользователь может создавать репозитории
				return true, nil
			}
		}
	}
	return false, nil
}
