// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package org

import (
	"context"
	"fmt"

	"code.gitea.io/gitea/models"
	"code.gitea.io/gitea/models/db"
	org_model "code.gitea.io/gitea/models/organization"
	packages_model "code.gitea.io/gitea/models/packages"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/role_model"
	"code.gitea.io/gitea/models/tenant"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/storage"
	"code.gitea.io/gitea/modules/util"
	user_service "code.gitea.io/gitea/services/user"
)

// DeleteOrganization completely and permanently deletes everything of organization.
func DeleteOrganization(org *org_model.Organization) error {
	ctx, commiter, err := db.TxContext(db.DefaultContext)
	if err != nil {
		return err
	}
	defer commiter.Close()

	// Check ownership of repository.
	count, err := repo_model.CountRepositories(ctx, repo_model.CountRepositoryOptions{OwnerID: org.ID})
	if err != nil {
		return fmt.Errorf("GetRepositoryCount: %w", err)
	} else if count > 0 {
		return models.ErrUserOwnRepos{UID: org.ID}
	}

	// Check ownership of packages.
	if ownsPackages, err := packages_model.HasOwnerPackages(ctx, org.ID); err != nil {
		return fmt.Errorf("HasOwnerPackages: %w", err)
	} else if ownsPackages {
		return models.ErrUserOwnPackages{UID: org.ID}
	}

	if err := org_model.DeleteOrganization(ctx, org); err != nil {
		return fmt.Errorf("DeleteOrganization: %w", err)
	}

	if err := commiter.Commit(); err != nil {
		return err
	}

	// FIXME: system notice
	// Note: There are something just cannot be roll back,
	//	so just keep error logs of those operations.
	path := user_model.UserPath(org.Name)

	if err := util.RemoveAll(path); err != nil {
		return fmt.Errorf("failed to RemoveAll %s: %w", path, err)
	}

	if len(org.Avatar) > 0 {
		avatarPath := org.CustomAvatarRelativePath()
		if err := storage.Avatars.Delete(avatarPath); err != nil {
			return fmt.Errorf("failed to remove %s: %w", avatarPath, err)
		}
	}
	tenantID, err := tenant.GetTenantByOrgIdOrDefault(ctx, org.ID)
	if err != nil {
		log.Error("Error has occurred while getting tenant by organization id %d: %v", org.ID, err)
		return fmt.Errorf("DeleteOrganization failed: %w", err)
	}
	if err := org_model.RemoveRelationTenantOrganization(ctx, org.ID, tenantID); err != nil {
		log.Error("Error has occurred while removing relation tenantID %s, organization id: %d : %v", tenantID, org.ID, err)
		return fmt.Errorf("DeleteOrganization failed: %w", err)
	}
	if err := role_model.RemoveExistingPrivilegesByTenantAndOrgID(tenantID, org.ID); err != nil {
		log.Error("Error has occurred while removing relation tenantID %s, organization id: %d : %v", tenantID, org.ID, err)
		return fmt.Errorf("DeleteOrganization failed: %w", err)
	}

	return nil
}

// RenameOrganization renames an organization.
func RenameOrganization(ctx context.Context, org *org_model.Organization, newName string) error {
	return user_service.RenameUser(ctx, org.AsUser(), newName)
}

// GetOrganizations получаем список организаций по organization_id
func GetOrganizations(ctx context.Context, organizationIDs []int64) ([]*org_model.Organization, error) {
	organizations, err := org_model.GetOrganizationByIDs(ctx, organizationIDs)
	if err != nil {
		return nil, err
	}
	return organizations, nil
}
