// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package pull

import (
	"context"
	"strconv"

	"gitlab.com/gitlab-org/gitaly/v16/proto/go/gitalypb"

	issues_model "code.gitea.io/gitea/models/issues"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/git"
)

// updateHeadByRebaseOnToBase handles updating a PR's head branch by rebasing it on the PR current base branch
func updateHeadByRebaseOnToBase(ctx context.Context, pr *issues_model.PullRequest, doer *user_model.User, message string) error {
	gitRepo, err := git.OpenRepository(ctx, pr.HeadRepo.OwnerName, pr.HeadRepo.Name, pr.HeadRepo.RepoPath())
	if err != nil {
		return err
	}

	if pr.HeadCommitID == "" {
		pr.HeadCommitID, err = git.GetFullCommitID(gitRepo.Ctx, gitRepo.Path, pr.HeadBranch)
		if err != nil {
			pr.HeadCommitID = pr.HeadBranch
		}
	}

	ctxWithCancel, cancel := context.WithCancel(gitRepo.Ctx)
	defer cancel()
	rebaseClient, err := gitRepo.OperationClient.UserRebaseConfirmable(ctxWithCancel)
	if err != nil {
		return err
	}
	header := &gitalypb.UserRebaseConfirmableRequest{
		UserRebaseConfirmableRequestPayload: &gitalypb.UserRebaseConfirmableRequest_Header_{
			Header: &gitalypb.UserRebaseConfirmableRequest_Header{
				Repository: gitRepo.GitalyRepo,
				User: &gitalypb.User{
					GlId:       strconv.FormatInt(doer.ID, 10),
					Name:       []byte(doer.Name),
					Email:      []byte(doer.GetDefaultEmail()),
					GlUsername: doer.Name,
				},
				Branch:           []byte(pr.HeadBranch),
				BranchSha:        pr.HeadCommitID,
				RemoteRepository: gitRepo.GitalyRepo,
				RemoteBranch:     []byte(pr.BaseBranch),
			},
		},
	}
	err = rebaseClient.Send(header)
	if err != nil {
		return err
	}

	_, err = rebaseClient.Recv()
	if err != nil {
		return err
	}

	apply := &gitalypb.UserRebaseConfirmableRequest{
		UserRebaseConfirmableRequestPayload: &gitalypb.UserRebaseConfirmableRequest_Apply{
			Apply: true,
		},
	}
	err = rebaseClient.Send(apply)
	if err != nil {
		return err
	}
	_, err = rebaseClient.Recv()
	if err != nil {
		return err
	}
	return nil
}
