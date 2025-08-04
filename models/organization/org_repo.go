// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package organization

import (
	"context"

	"code.gitea.io/gitea/models/db"
	repo_model "code.gitea.io/gitea/models/repo"
)

// GetOrgRepositories get repos belonging to the given organization
func GetOrgRepositories(ctx context.Context, orgID int64) ([]*repo_model.Repository, error) {
	var orgRepos []*repo_model.Repository
	return orgRepos, db.GetEngine(ctx).Where("owner_id = ?", orgID).Find(&orgRepos)
}

// GetOrgRepositories get repos belonging to the given organization and private filter
func GetOrgRepositoriesIDWithPrivateFilter(orgID int64, includePrivate bool) ([]int64, error) {
	var orgRepoIds []int64

	if includePrivate {
		return orgRepoIds, db.GetEngine(db.DefaultContext).
			Select("id").
			Table("repository").
			Where("owner_id = ?", orgID).
			Find(&orgRepoIds)
	}
	return orgRepoIds, db.GetEngine(db.DefaultContext).
		Select("id").
		Table("repository").
		Where("owner_id = ?", orgID).
		And("is_private = false").
		Find(&orgRepoIds)
}
