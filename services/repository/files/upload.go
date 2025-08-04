// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package files

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"

	"gitlab.com/gitlab-org/gitaly/v16/proto/go/gitalypb"

	repo_model "code.gitea.io/gitea/models/repo"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/git"
)

// UploadRepoFileOptions contains the uploaded repository file options
type UploadRepoFileOptions struct {
	LastCommitID string
	OldBranch    string
	NewBranch    string
	TreePath     string
	Message      string
	Files        []string // In UUID format.
	Signoff      bool
}

// UploadRepoFiles uploads files to the given repository
func UploadRepoFiles(ctx context.Context, repo *repo_model.Repository, doer *user_model.User, opts *UploadRepoFileOptions) error {
	if len(opts.Files) == 0 {
		return nil
	}

	uploads, err := repo_model.GetUploadsByUUIDs(opts.Files)
	if err != nil {
		return fmt.Errorf("GetUploadsByUUIDs [uuids: %v]: %w", opts.Files, err)
	}

	gitRepo, closer, err := git.RepositoryFromContextOrOpen(ctx, repo.OwnerName, repo.Name, repo.RepoPath())
	if err != nil {
		return err
	}
	defer closer.Close()

	// make author and committer the doer
	author := doer
	committer := doer

	requestMessages := make([]*gitalypb.UserCommitFilesRequest, 0)

	header := &gitalypb.UserCommitFilesRequest{
		UserCommitFilesRequestPayload: &gitalypb.UserCommitFilesRequest_Header{
			Header: &gitalypb.UserCommitFilesRequestHeader{
				Repository: gitRepo.GitalyRepo,
				User: &gitalypb.User{
					GlId:       strconv.Itoa(int(committer.ID)),
					Name:       []byte(committer.Name),
					Email:      []byte(committer.GetDefaultEmail()),
					GlUsername: committer.Name,
				},
				BranchName:        []byte(opts.NewBranch),
				CommitMessage:     []byte(opts.Message),
				CommitAuthorName:  []byte(author.Name),
				CommitAuthorEmail: []byte(author.GetDefaultEmail()),
				StartBranchName:   []byte(opts.OldBranch),
				Force:             false,
				StartSha:          opts.LastCommitID,
			},
		},
	}

	requestMessages = append(requestMessages, header)

	for _, upload := range uploads {
		requestMessages = append(requestMessages, newActionRequest(gitalypb.UserCommitFilesActionHeader_CREATE, opts.TreePath+"/"+upload.Name, ""))

		file, err := os.Open(upload.LocalPath())
		if err != nil {
			return err
		}
		defer file.Close()

		content, err := io.ReadAll(file)
		if err != nil {
			return err
		}
		requestMessages = append(requestMessages, &gitalypb.UserCommitFilesRequest{
			UserCommitFilesRequestPayload: &gitalypb.UserCommitFilesRequest_Action{
				Action: &gitalypb.UserCommitFilesAction{
					UserCommitFilesActionPayload: &gitalypb.UserCommitFilesAction_Content{
						Content: content,
					},
				},
			},
		})
	}

	ctxWithCancel, cancel := context.WithCancel(gitRepo.Ctx)
	defer cancel()
	userCommitFilesClient, err := gitRepo.OperationClient.UserCommitFiles(ctxWithCancel)
	if err != nil {
		return err
	}

	for _, reqMes := range requestMessages {
		err = userCommitFilesClient.Send(reqMes)
		if err != nil {
			return err
		}
	}

	recv, err := userCommitFilesClient.CloseAndRecv()
	if err != nil || recv.IndexError != "" || recv.PreReceiveError != "" {
		return err
	}

	if repo.IsEmpty {
		_ = repo_model.UpdateRepositoryCols(ctx, &repo_model.Repository{ID: repo.ID, IsEmpty: false}, "is_empty")
	}

	return repo_model.DeleteUploads(uploads...)
}
