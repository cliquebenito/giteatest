// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package pull

import (
	"context"
	"strconv"
	"strings"

	"gitlab.com/gitlab-org/gitaly/v16/proto/go/gitalypb"

	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/log"
)

// getRebaseAmendMessage composes the message to amend commits in rebase merge of a pull request.
func getRebaseAmendMessage(ctx *mergeContext, baseGitRepo *git.Repository) (message string, err error) {
	ctxWithCancel, cancel := context.WithCancel(ctx.gitRepo.Ctx)
	defer cancel()
	commitMessageClient, err := ctx.gitRepo.CommitClient.GetCommitMessages(ctxWithCancel, &gitalypb.GetCommitMessagesRequest{
		Repository: ctx.gitRepo.GitalyRepo,
		CommitIds:  []string{ctx.pr.MergedCommitID},
	})
	if err != nil {
		return message, err
	}
	commitMessage, err := commitMessageClient.Recv()
	if err != nil {
		return message, err
	}

	commitTitle, commitBody, _ := strings.Cut(string(commitMessage.Message), "\n")
	extraVars := map[string]string{"CommitTitle": strings.TrimSpace(commitTitle), "CommitBody": strings.TrimSpace(commitBody)}

	message, body, err := getMergeMessage(ctx, baseGitRepo, ctx.pr, repo_model.MergeStyleRebase, extraVars)
	if err != nil || message == "" {
		return "", err
	}

	if len(body) > 0 {
		message = message + "\n\n" + body
	}
	return message, err
}

// Perform rebase merge without merge commit.
func doMergeRebaseFastForward(ctx *mergeContext) error {
	// Original repo to read template from.
	baseGitRepo, err := git.OpenRepository(ctx, ctx.pr.BaseRepo.OwnerName, ctx.pr.BaseRepo.Name, ctx.pr.BaseRepo.RepoPath())
	if err != nil {
		log.Error("Unable to get Git repo for rebase: %v", err)
		return err
	}
	defer baseGitRepo.Close()

	// Amend last commit message based on template, if one exists
	newMessage, err := getRebaseAmendMessage(ctx, baseGitRepo)
	if err != nil {
		log.Error("Unable to get commit message for amend: %v", err)
		return err
	}
	ctxWithCancel, cancel := context.WithCancel(ctx.gitRepo.Ctx)
	defer cancel()
	if newMessage != "" {
		_, err = ctx.gitRepo.OperationClient.UserFFBranch(ctxWithCancel, &gitalypb.UserFFBranchRequest{
			Repository: ctx.gitRepo.GitalyRepo,
			User: &gitalypb.User{
				GlId:       strconv.FormatInt(ctx.doer.ID, 10),
				Name:       []byte(ctx.committer.Name),
				Email:      []byte(ctx.committer.GetDefaultEmail()),
				GlUsername: ctx.committer.Name,
			},
			CommitId: ctx.pr.MergedCommitID,
			Branch:   []byte(ctx.pr.BaseBranch),
		})
		if err != nil {
			return err
		}
	}

	return nil
}

// Perform rebase merge with merge commit.
func doMergeRebaseMergeCommit(ctx *mergeContext, message string) error {
	ctxWithCancel, cancel := context.WithCancel(ctx.gitRepo.Ctx)
	defer cancel()
	mergeClient, err := ctx.gitRepo.OperationClient.UserMergeBranch(ctxWithCancel)
	if err != nil {
		return err
	}
	req := &gitalypb.UserMergeBranchRequest{
		Repository: ctx.gitRepo.GitalyRepo,
		User: &gitalypb.User{
			GlId:       strconv.FormatInt(ctx.doer.ID, 10),
			Name:       []byte(ctx.committer.Name),
			Email:      []byte(ctx.committer.GetDefaultEmail()),
			GlUsername: ctx.committer.Name,
		},
		CommitId: ctx.pr.MergedCommitID,
		Branch:   []byte(ctx.pr.BaseBranch),
		Message:  []byte(message),
		Apply:    false,
	}

	err = mergeClient.Send(req)
	if err != nil {
		return err
	}

	resp, err := mergeClient.Recv()
	if err != nil {
		return err
	}

	req.Apply = true
	err = mergeClient.Send(req)
	if err != nil {
		return err
	}

	ctx.pr.MergedCommitID = resp.GetCommitId()
	return nil
}

// doMergeStyleRebase rebases the tracking branch on the base branch as the current HEAD with or with a merge commit to the original pr branch
func doMergeStyleRebase(ctx *mergeContext, mergeStyle repo_model.MergeStyle, message string) error {
	if err := rebaseTrackingOnToBase(ctx, mergeStyle); err != nil {
		return err
	}

	if mergeStyle == repo_model.MergeStyleRebase {
		return doMergeRebaseFastForward(ctx)
	}

	return doMergeRebaseMergeCommit(ctx, message)
}
