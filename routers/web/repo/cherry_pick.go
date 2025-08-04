// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	contextDefault "context"
	"fmt"
	"strconv"
	"strings"

	"gitlab.com/gitlab-org/gitaly/v16/proto/go/gitalypb"
	"google.golang.org/grpc/status"

	"code.gitea.io/gitea/models"

	"code.gitea.io/gitea/models/unit"
	"code.gitea.io/gitea/modules/base"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/util"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/services/forms"
	"code.gitea.io/gitea/services/repository/files"
)

var tplCherryPick base.TplName = "repo/editor/cherry_pick"

// CherryPick handles cherrypick GETs
func CherryPick(ctx *context.Context) {
	ctx.Data["SHA"] = ctx.Params(":sha")
	cherryPickCommit, err := ctx.Repo.GitRepo.GetCommit(ctx.Params(":sha"))
	if err != nil {
		if git.IsErrNotExist(err) {
			ctx.NotFound("Missing Commit", err)
			return
		}
		ctx.ServerError("GetCommit", err)
		return
	}

	if ctx.FormString("cherry-pick-type") == "revert" {
		ctx.Data["CherryPickType"] = "revert"
		ctx.Data["commit_summary"] = "revert " + ctx.Params(":sha")
		ctx.Data["commit_message"] = "revert " + cherryPickCommit.Message()
	} else {
		ctx.Data["CherryPickType"] = "cherry-pick"
		splits := strings.SplitN(cherryPickCommit.Message(), "\n", 2)
		ctx.Data["commit_summary"] = splits[0]
		ctx.Data["commit_message"] = ""
		if len(splits) > 1 {
			ctx.Data["commit_message"] = splits[1]
		}
	}

	canCommit := renderCommitRights(ctx)
	ctx.Data["TreePath"] = ""

	if canCommit {
		ctx.Data["commit_choice"] = frmCommitChoiceDirect
	} else {
		ctx.Data["commit_choice"] = frmCommitChoiceNewBranch
	}
	ctx.Data["new_branch_name"] = GetUniquePatchBranchName(ctx)
	ctx.Data["last_commit"] = ctx.Repo.CommitID
	ctx.Data["LineWrapExtensions"] = strings.Join(setting.Repository.Editor.LineWrapExtensions, ",")
	ctx.Data["BranchLink"] = ctx.Repo.RepoLink + "/src/" + ctx.Repo.BranchNameSubURL()

	ctx.HTML(200, tplCherryPick)
}

// CherryPickPost handles cherrypick POSTs
func CherryPickPost(ctx *context.Context) {
	form := web.GetForm(ctx).(*forms.CherryPickForm)

	sha := ctx.Params(":sha")
	ctx.Data["SHA"] = sha
	if form.Revert {
		ctx.Data["CherryPickType"] = "revert"
	} else {
		ctx.Data["CherryPickType"] = "cherry-pick"
	}

	canCommit := renderCommitRights(ctx)
	branchName := ctx.Repo.BranchName
	if form.CommitChoice == frmCommitChoiceNewBranch {
		branchName = form.NewBranchName
	}
	ctx.Data["commit_summary"] = form.CommitSummary
	ctx.Data["commit_message"] = form.CommitMessage
	ctx.Data["commit_choice"] = form.CommitChoice
	ctx.Data["new_branch_name"] = form.NewBranchName
	ctx.Data["last_commit"] = ctx.Repo.CommitID
	ctx.Data["LineWrapExtensions"] = strings.Join(setting.Repository.Editor.LineWrapExtensions, ",")
	ctx.Data["BranchLink"] = ctx.Repo.RepoLink + "/src/" + ctx.Repo.BranchNameSubURL()

	if ctx.HasError() {
		ctx.HTML(200, tplCherryPick)
		return
	}

	// Cannot commit to a an existing branch if user doesn't have rights
	if branchName == ctx.Repo.BranchName && !canCommit {
		ctx.Data["Err_NewBranchName"] = true
		ctx.Data["commit_choice"] = frmCommitChoiceNewBranch
		ctx.RenderWithErr(ctx.Tr("repo.editor.cannot_commit_to_protected_branch", branchName), tplCherryPick, &form)
		return
	}

	message := strings.TrimSpace(form.CommitSummary)
	if message == "" {
		if form.Revert {
			message = ctx.Tr("repo.commit.revert-header", sha)
		} else {
			message = ctx.Tr("repo.commit.cherry-pick-header", sha)
		}
	}

	form.CommitMessage = strings.TrimSpace(form.CommitMessage)
	if len(form.CommitMessage) > 0 {
		message += "\n\n" + form.CommitMessage
	}

	opts := &files.ApplyDiffPatchOptions{
		LastCommitID: form.LastCommit,
		OldBranch:    ctx.Repo.BranchName,
		NewBranch:    branchName,
		Message:      message,
	}

	// First lets try the simple plain read-tree -m approach
	opts.Content = sha
	if err := opts.Validate(ctx, ctx.Repo.Repository, ctx.Doer); err != nil {
		if models.IsErrBranchAlreadyExists(err) {
			// User has specified a branch that already exists
			branchErr := err.(models.ErrBranchAlreadyExists)
			ctx.Data["Err_NewBranchName"] = true
			ctx.RenderWithErr(ctx.Tr("repo.editor.branch_already_exists", branchErr.BranchName), tplCherryPick, &form)
			return
		} else if models.IsErrCommitIDDoesNotMatch(err) {
			ctx.RenderWithErr(ctx.Tr("repo.editor.file_changed_while_editing", ctx.Repo.RepoLink+"/compare/"+form.LastCommit+"..."+ctx.Repo.CommitID), tplPatchFile, &form)
			return
		}
		ctx.RenderWithErr(ctx.Tr("repo.editor.fail_to_apply_patch", err.Error()), tplCherryPick, &form)
		return
	}

	ctxWithCancel, cancel := contextDefault.WithCancel(ctx.Repo.GitRepo.Ctx)
	defer cancel()
	if form.Revert {
		userRevertRequest := &gitalypb.UserRevertRequest{
			Repository: ctx.Repo.GitRepo.GitalyRepo,
			User: &gitalypb.User{
				GlId:       strconv.FormatInt(ctx.Doer.ID, 10),
				Name:       []byte(ctx.Doer.Name),
				Email:      []byte(ctx.Doer.GetDefaultEmail()),
				GlUsername: ctx.Doer.LowerName,
			},
			Commit:          &gitalypb.GitCommit{Id: sha},
			BranchName:      []byte(branchName),
			Message:         []byte(message),
			StartBranchName: []byte(ctx.Repo.BranchName),
			ExpectedOldOid:  form.LastCommit,
		}
		if _, err := ctx.Repo.GitRepo.OperationClient.UserRevert(ctxWithCancel, userRevertRequest); err != nil {
			revertStatusError, _ := status.FromError(err)

			rightCommitId := form.LastCommit
			if ctx.Repo.Commit.ParentCount() != 0 {
				if rightCommit, err := ctx.Repo.Commit.Parent(0); err == nil {
					rightCommitId = rightCommit.ID.String()
				}
			}

			rawDiff, err := ctx.Repo.GitRepo.DiffClient.RawDiff(ctxWithCancel, &gitalypb.RawDiffRequest{
				Repository:    ctx.Repo.GitRepo.GitalyRepo,
				LeftCommitId:  sha,
				RightCommitId: rightCommitId,
			})
			if err != nil {
				ctx.RenderWithErr(ctx.Tr("repo.editor.fail_to_apply_patch", revertStatusError.Message()), tplCherryPick, &form)
				return
			}
			recv, err := rawDiff.Recv()
			if err != nil {
				ctx.RenderWithErr(ctx.Tr("repo.editor.fail_to_apply_patch", revertStatusError.Message()), tplCherryPick, &form)
				return
			}

			opts.Content = fmt.Sprintf("revert %s\n\n", sha) + string(recv.Data)
			ctx.Data["FileContent"] = opts.Content
			ctx.RenderWithErr(ctx.Tr("repo.editor.fail_to_apply_patch", revertStatusError.Message()), tplPatchFile, &form)
			return
		}
	} else {
		userCherryPickRequest := &gitalypb.UserCherryPickRequest{
			Repository: ctx.Repo.GitRepo.GitalyRepo,
			User: &gitalypb.User{
				GlId:       strconv.FormatInt(ctx.Doer.ID, 10),
				Name:       []byte(ctx.Doer.Name),
				Email:      []byte(ctx.Doer.GetDefaultEmail()),
				GlUsername: ctx.Doer.LowerName,
			},
			Commit:            &gitalypb.GitCommit{Id: sha},
			BranchName:        []byte(branchName),
			Message:           []byte(message),
			StartBranchName:   []byte(ctx.Repo.BranchName),
			ExpectedOldOid:    form.LastCommit,
			CommitAuthorName:  []byte(ctx.Doer.Name),
			CommitAuthorEmail: []byte(ctx.Doer.GetDefaultEmail()),
		}
		if _, err := ctx.Repo.GitRepo.OperationClient.UserCherryPick(ctxWithCancel, userCherryPickRequest); err != nil {
			cherryPickStatusError, _ := status.FromError(err)
			rawDiff, err := ctx.Repo.GitRepo.DiffClient.RawPatch(ctx.Repo.GitRepo.Ctx, &gitalypb.RawPatchRequest{
				Repository:    ctx.Repo.GitRepo.GitalyRepo,
				LeftCommitId:  form.LastCommit,
				RightCommitId: sha,
			})
			if err != nil {
				ctx.RenderWithErr(ctx.Tr("repo.editor.fail_to_apply_patch", cherryPickStatusError.Message()), tplCherryPick, &form)
				return
			}
			recv, err := rawDiff.Recv()
			if err != nil {
				ctx.RenderWithErr(ctx.Tr("repo.editor.fail_to_apply_patch", cherryPickStatusError.Message()), tplCherryPick, &form)
				return
			}
			opts.Content = string(recv.Data)
			ctx.Data["FileContent"] = opts.Content
			ctx.RenderWithErr(ctx.Tr("repo.editor.fail_to_apply_patch", cherryPickStatusError.Message()), tplPatchFile, &form)
			return
		}
	}

	if form.CommitChoice == frmCommitChoiceNewBranch && ctx.Repo.Repository.UnitEnabled(ctx, unit.TypePullRequests) {
		ctx.Redirect(ctx.Repo.RepoLink + "/compare/" + util.PathEscapeSegments(ctx.Repo.BranchName) + "..." + util.PathEscapeSegments(form.NewBranchName))
	} else {
		ctx.Redirect(ctx.Repo.RepoLink + "/src/branch/" + util.PathEscapeSegments(branchName))
	}
}
