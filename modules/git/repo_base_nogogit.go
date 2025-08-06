// Copyright 2015 The Gogs Authors. All rights reserved.
// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

//go:build !gogit

package git

import (
	"bufio"
	"context"

	"code.gitea.io/gitea/integration/gitaly"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"

	"gitlab.com/gitlab-org/gitaly/v16/proto/go/gitalypb"
)

// Repository represents a Git repository.
type Repository struct {
	Owner string
	Name  string
	Path  string

	RepoClient       *gitaly.RepositoryClient
	RefClient        *gitaly.RefClient
	CommitClient     *gitaly.CommitClient
	BlobClient       *gitaly.BlobClient
	OperationClient  *gitaly.OperationClient
	SSHServiceClient *gitaly.SSHServiceClient
	DiffClient       *gitaly.DiffClient
	ConflictsClient  *gitaly.ConflictsClient

	GitalyRepo *gitalypb.Repository

	tagCache *ObjectCache

	gpgSettings *GPGSettings

	batchCancel context.CancelFunc
	batchReader *bufio.Reader
	batchWriter WriteCloserError

	checkCancel context.CancelFunc
	checkReader *bufio.Reader
	checkWriter WriteCloserError

	Ctx             context.Context
	LastCommitCache *LastCommitCache
}

// openRepositoryWithDefaultContext opens the repository at the given path with DefaultContext.
func openRepositoryWithDefaultContext(repoPath string) (*Repository, error) {
	//todo fix tests
	return OpenRepository(DefaultContext, repoPath, repoPath, repoPath)
}

// OpenRepository opens the repository at the given path with the provided context.
func OpenRepository(ctx context.Context, owner, name, path string) (*Repository, error) {
	// todo: refactor
	ctx1, rc, err := gitaly.NewRepositoryClient(ctx)
	if err != nil {
		return nil, err
	}
	ctx2, refc, err := gitaly.NewRefClient(ctx1)
	if err != nil {
		return nil, err
	}
	ctx3, cc, err := gitaly.NewCommitClient(ctx2)
	if err != nil {
		return nil, err
	}
	ctx4, bc, err := gitaly.NewBlobClient(ctx3)
	if err != nil {
		return nil, err
	}
	ctx5, oc, err := gitaly.NewOperationClient(ctx4)
	if err != nil {
		return nil, err
	}
	ctx6, ssshc, err := gitaly.NewSSHClient(ctx5)
	if err != nil {
		return nil, err
	}
	ctx7, dc, err := gitaly.NewDiffClient(ctx6)
	if err != nil {
		return nil, err
	}
	ctx8, confc, err := gitaly.NewConflictsClient(ctx7)
	if err != nil {
		return nil, err
	}

	repo := &Repository{
		Path:             path,
		Owner:            owner,
		Name:             name,
		RepoClient:       rc,
		RefClient:        refc,
		CommitClient:     cc,
		BlobClient:       bc,
		OperationClient:  oc,
		SSHServiceClient: ssshc,
		DiffClient:       dc,
		ConflictsClient:  confc,
		GitalyRepo: &gitalypb.Repository{
			GlRepository:  name,
			GlProjectPath: owner,
			RelativePath:  path,
			StorageName:   setting.Gitaly.MainServerName,
		},
		tagCache: newObjectCache(),
		Ctx:      ctx8,
	}

	return repo, nil
}

// CatFileBatch obtains a CatFileBatch for this repository
func (repo *Repository) CatFileBatch(ctx context.Context) (WriteCloserError, *bufio.Reader, func()) {
	if repo.batchCancel == nil || repo.batchReader.Buffered() > 0 {
		log.Debug("Opening temporary cat file batch for: %s", repo.Path)
		return CatFileBatch(ctx, repo.Path)
	}
	return repo.batchWriter, repo.batchReader, func() {}
}

// CatFileBatchCheck obtains a CatFileBatchCheck for this repository
func (repo *Repository) CatFileBatchCheck(ctx context.Context) (WriteCloserError, *bufio.Reader, func()) {
	if repo.checkCancel == nil || repo.checkReader.Buffered() > 0 {
		log.Debug("Opening temporary cat file batch-check: %s", repo.Path)
		return CatFileBatchCheck(ctx, repo.Path)
	}
	return repo.checkWriter, repo.checkReader, func() {}
}

// Close this repository, in particular close the underlying gogitStorage if this is not nil
func (repo *Repository) Close() (err error) {
	if repo == nil {
		return
	}
	if repo.batchCancel != nil {
		repo.batchCancel()
		repo.batchReader = nil
		repo.batchWriter = nil
		repo.batchCancel = nil
	}
	if repo.checkCancel != nil {
		repo.checkCancel()
		repo.checkCancel = nil
		repo.checkReader = nil
		repo.checkWriter = nil
	}
	repo.LastCommitCache = nil
	repo.tagCache = nil
	return err
}
