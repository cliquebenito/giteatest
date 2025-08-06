// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"fmt"

	"code.gitea.io/gitea/modules/base"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/services/forms"
	"code.gitea.io/gitea/services/repository"
)

const (
	tplProtectedBranch base.TplName = "repo/settings/protected_branch"
)

// RenameBranchPost responses for rename a branch
func RenameBranchPost(ctx *context.Context) {
	form := web.GetForm(ctx).(*forms.RenameBranchForm)

	if !ctx.Repo.CanCreateBranch() {
		ctx.NotFound("RenameBranch", nil)
		return
	}

	if ctx.HasError() {
		ctx.Flash.Error(ctx.GetErrMsg())
		ctx.Redirect(fmt.Sprintf("%s/branches", ctx.Repo.RepoLink))
		return
	}

	if err := repository.RenameBranch(ctx, ctx.Repo.Repository, ctx.Doer, ctx.Repo.GitRepo, form.From, form.To); err != nil {
		log.Error("Error has occurred while renaming branch: %v", err)
		ctx.ServerError("RenameBranch", err)
		ctx.Redirect(fmt.Sprintf("%s/branches", ctx.Repo.RepoLink))
		return
	}

	ctx.Flash.Success(ctx.Tr("repo.settings.rename_branch_success", form.From, form.To))
	ctx.Redirect(fmt.Sprintf("%s/branches", ctx.Repo.RepoLink))
}
