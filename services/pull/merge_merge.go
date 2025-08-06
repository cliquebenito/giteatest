// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package pull

import (
	contextDefault "context"
	"strconv"

	"gitlab.com/gitlab-org/gitaly/v16/proto/go/gitalypb"
)

// doMergeStyleMerge merges the tracking into the current HEAD - which is assumed to tbe staging branch (equal to the pr.BaseBranch)
func doMergeStyleMerge(ctx *mergeContext, message string) error {
	ctxWithCancel, cancel := contextDefault.WithCancel(ctx.gitRepo.Ctx)
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
		CommitId: ctx.pr.HeadCommitID,
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

	_, err = mergeClient.Recv()
	if err != nil {
		return err
	}

	ctx.pr.MergedCommitID = resp.GetCommitId()
	return nil
}
