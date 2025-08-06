// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package private

import (
	"context"
	"fmt"
	"net/http"

	gitea_context "code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/private"
	"code.gitea.io/gitea/routers/private/hooks"
)

// This file contains common functions relating to setting the Repository for the internal routes

// RepoAssignment assigns the repository and gitrepository to the private context
func RepoAssignment(ctx *gitea_context.PrivateContext) context.CancelFunc {
	ownerName := ctx.Params(":owner")
	repoName := ctx.Params(":repo")

	repo := hooks.LoadRepository(ctx, ownerName, repoName)
	if ctx.Written() {
		// Error handled in loadRepository
		return nil
	}

	gitRepo, err := git.OpenRepository(ctx, repo.OwnerName, repo.Name, repo.RepoPath())
	if err != nil {
		log.Error("Failed to open repository: %s/%s Error: %v", ownerName, repoName, err)
		ctx.JSON(http.StatusInternalServerError, private.Response{
			Err: fmt.Sprintf("Failed to open repository: %s/%s Error: %v", ownerName, repoName, err),
		})
		return nil
	}

	ctx.Repo = &gitea_context.Repository{
		Repository: repo,
		GitRepo:    gitRepo,
	}

	// We opened it, we should close it
	cancel := func() {
		// If it's been set to nil then assume someone else has closed it.
		if ctx.Repo.GitRepo != nil {
			ctx.Repo.GitRepo.Close()
		}
	}

	return cancel
}
