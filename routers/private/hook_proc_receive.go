// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package private

import (
	"code.gitea.io/gitea/modules/sbt/audit"
	"net/http"
	"strconv"

	repo_model "code.gitea.io/gitea/models/repo"
	gitea_context "code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/private"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/services/agit"
)

// HookProcReceive proc-receive hook - only handles agit Proc-Receive requests at present
func HookProcReceive(ctx *gitea_context.PrivateContext) {
	opts := web.GetForm(ctx).(*private.HookOptions)

	auditParams := map[string]string{
		"repository":    ctx.Repo.Repository.Name,
		"repository_id": strconv.FormatInt(ctx.Repo.Repository.ID, 10),
		"owner":         ctx.Repo.Repository.OwnerName,
	}

	if !git.SupportProcReceive {
		ctx.Status(http.StatusNotFound)
		auditParams["error"] = "Git not supported proc-receive"
		audit.CreateAndSendEvent(audit.ChangesPushEvent, opts.UserName, strconv.FormatInt(opts.UserID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}

	results, err := agit.ProcReceive(ctx, ctx.Repo.Repository, ctx.Repo.GitRepo, opts)
	if err != nil {
		if repo_model.IsErrUserDoesNotHaveAccessToRepo(err) {
			auditParams["error"] = "User does not have access to repo"
			audit.CreateAndSendEvent(audit.ChangesPushEvent, opts.UserName, strconv.FormatInt(opts.UserID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			ctx.Error(http.StatusBadRequest, "UserDoesNotHaveAccessToRepo", err.Error())
		} else {
			log.Error(err.Error())
			auditParams["error"] = ""
			audit.CreateAndSendEvent(audit.ChangesPushEvent, opts.UserName, strconv.FormatInt(opts.UserID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			ctx.JSON(http.StatusInternalServerError, private.Response{
				Err: err.Error(),
			})
		}

		return
	}

	ctx.JSON(http.StatusOK, private.HookProcReceiveResult{
		Results: results,
	})
}
