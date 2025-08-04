// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package pull

import (
	contextDefault "context"
	"strconv"

	"gitlab.com/gitlab-org/gitaly/v16/proto/go/gitalypb"

	"code.gitea.io/gitea/modules/log"
)

//// doMergeStyleSquash gets a commit author signature for squash commits
//func getAuthorSignatureSquash(ctx *mergeContext) (*git.Signature, error) {
//	if err := ctx.pr.Issue.LoadPoster(ctx); err != nil {
//		log.Error("%-v Issue[%d].LoadPoster: %v", ctx.pr, ctx.pr.Issue.ID, err)
//		return nil, err
//	}
//
//	// Try to get an signature from the same user in one of the commits, as the
//	// poster email might be private or commits might have a different signature
//	// than the primary email address of the poster.
//	gitRepo, closer, err := git.RepositoryFromContextOrOpen(ctx, ctx.pr.BaseRepo.OwnerName, ctx.pr.BaseRepo.Name, ctx.tmpBasePath)
//	if err != nil {
//		log.Error("%-v Unable to open base repository: %v", ctx.pr, err)
//		return nil, err
//	}
//	defer closer.Close()
//
//	commits, err := gitRepo.CommitsBetweenIDs(trackingBranch, "HEAD")
//	if err != nil {
//		log.Error("%-v Unable to get commits between: %s %s: %v", ctx.pr, "HEAD", trackingBranch, err)
//		return nil, err
//	}
//
//	uniqueEmails := make(container.Set[string])
//	for _, commit := range commits {
//		if commit.Author != nil && uniqueEmails.Add(commit.Author.Email) {
//			commitUser, _ := user_model.GetUserByEmail(ctx, commit.Author.Email)
//			if commitUser != nil && commitUser.ID == ctx.pr.Issue.Poster.ID {
//				return commit.Author, nil
//			} else if commitUser == nil && commit.Author.Name != "" {
//				commitUser, _ = user_model.GetUserByName(ctx, commit.Author.Name)
//				if commitUser != nil && commitUser.ID == ctx.pr.Issue.Poster.ID {
//					return commit.Author, nil
//				}
//			}
//		}
//	}
//	return ctx.pr.Issue.Poster.NewGitSig(), nil
//}

// doMergeStyleSquash squashes the tracking branch on the current HEAD (=base)
func doMergeStyleSquash(ctx *mergeContext, message string) error {
	if err := ctx.pr.Issue.LoadPoster(ctx); err != nil {
		log.Error("%-v Issue[%d].LoadPoster: %v", ctx.pr, ctx.pr.Issue.ID, err)
		return err
	}
	sig := ctx.pr.Issue.Poster.NewGitSig()
	ctxWithCancel, cancel := contextDefault.WithCancel(ctx.gitRepo.Ctx)
	defer cancel()
	//todo update versions v 17 2 0
	squashClient, err := ctx.gitRepo.OperationClient.UserSquash(ctxWithCancel, &gitalypb.UserSquashRequest{
		Repository: ctx.gitRepo.GitalyRepo,
		User: &gitalypb.User{
			GlId:       strconv.FormatInt(ctx.doer.ID, 10),
			Name:       []byte(ctx.committer.Name),
			Email:      []byte(ctx.committer.GetDefaultEmail()),
			GlUsername: ctx.committer.Name,
		},
		StartSha: ctx.pr.MergeBase,
		EndSha:   ctx.pr.HeadCommitID,
		Author: &gitalypb.User{
			GlId:       strconv.FormatInt(sig.ID, 10),
			Name:       []byte(sig.Name),
			Email:      []byte(sig.GetDefaultEmail()),
			GlUsername: sig.Name,
		},
		CommitMessage: []byte(message),
	})
	if err != nil {
		return err
	}
	_, err = ctx.gitRepo.OperationClient.UserFFBranch(ctxWithCancel, &gitalypb.UserFFBranchRequest{
		Repository: ctx.gitRepo.GitalyRepo,
		User: &gitalypb.User{
			GlId:       strconv.FormatInt(ctx.doer.ID, 10),
			Name:       []byte(ctx.committer.Name),
			Email:      []byte(ctx.committer.GetDefaultEmail()),
			GlUsername: ctx.committer.Name,
		},
		CommitId: squashClient.GetSquashSha(),
		Branch:   []byte(ctx.pr.BaseBranch),
	})
	if err != nil {
		return err
	}

	ctx.pr.MergedCommitID = squashClient.GetSquashSha()
	return nil
}
