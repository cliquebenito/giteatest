// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repository

import (
	"code.gitea.io/gitea/models/db"
	repo_model "code.gitea.io/gitea/models/repo"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/notification"
	repo_module "code.gitea.io/gitea/modules/repository"
	"code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/modules/util"
	"context"
	"fmt"
	"gitlab.com/gitlab-org/gitaly/v16/proto/go/gitalypb"
	"strings"
)

// ErrForkAlreadyExist represents a "ForkAlreadyExist" kind of error.
type ErrForkAlreadyExist struct {
	Uname    string
	RepoName string
	ForkName string
}

// IsErrForkAlreadyExist checks if an error is an ErrForkAlreadyExist.
func IsErrForkAlreadyExist(err error) bool {
	_, ok := err.(ErrForkAlreadyExist)
	return ok
}

func (err ErrForkAlreadyExist) Error() string {
	return fmt.Sprintf("repository is already forked by user [uname: %s, repo path: %s, fork path: %s]", err.Uname, err.RepoName, err.ForkName)
}

func (err ErrForkAlreadyExist) Unwrap() error {
	return util.ErrAlreadyExist
}

// ForkRepoOptions contains the fork repository options
type ForkRepoOptions struct {
	BaseRepo    *repo_model.Repository
	Name        string
	Description string
}

// ForkRepository forks a repository
func ForkRepository(ctx context.Context, doer, owner *user_model.User, opts ForkRepoOptions) (*repo_model.Repository, error) {
	// Fork is prohibited, if user has reached maximum limit of repositories
	if !owner.CanForkRepo() {
		return nil, repo_model.ErrReachLimitOfRepo{
			Limit: owner.MaxRepoCreation,
		}
	}

	forkedRepo, err := repo_model.GetUserFork(ctx, opts.BaseRepo.ID, owner.ID)
	if err != nil {
		return nil, err
	}
	if forkedRepo != nil {
		return nil, ErrForkAlreadyExist{
			Uname:    owner.Name,
			RepoName: opts.BaseRepo.FullName(),
			ForkName: forkedRepo.FullName(),
		}
	}

	repo := &repo_model.Repository{
		OwnerID:       owner.ID,
		Owner:         owner,
		OwnerName:     owner.Name,
		Name:          opts.Name,
		LowerName:     strings.ToLower(opts.Name),
		Description:   opts.Description,
		DefaultBranch: opts.BaseRepo.DefaultBranch,
		IsPrivate:     opts.BaseRepo.IsPrivate || opts.BaseRepo.Owner.Visibility == structs.VisibleTypePrivate,
		IsEmpty:       opts.BaseRepo.IsEmpty,
		IsFork:        true,
		ForkID:        opts.BaseRepo.ID,
	}

	oldRepoPath := opts.BaseRepo.RepoPath()

	defer func() {
		panicErr := recover()
		if panicErr == nil {
			return
		}

		panic(panicErr)
	}()

	err = db.WithTx(ctx, func(txCtx context.Context) error {
		if err = repo_module.CreateRepositoryByExample(txCtx, doer, owner, repo, false, true); err != nil {
			return err
		}

		if err = repo_model.IncrementRepoForkNum(txCtx, opts.BaseRepo.ID); err != nil {
			return err
		}

		repoPath := repo_model.RepoPath(owner.Name, repo.Name)

		oldRepository, err := git.OpenRepository(ctx, opts.BaseRepo.OwnerName, opts.BaseRepo.Name, oldRepoPath)
		if err != nil {
			return err
		}

		_, err = oldRepository.RepoClient.CreateFork(oldRepository.Ctx, &gitalypb.CreateForkRequest{
			Repository: &gitalypb.Repository{
				StorageName:   oldRepository.GitalyRepo.StorageName,
				RelativePath:  repoPath,
				GlRepository:  repo.Name,
				GlProjectPath: owner.Name,
			},
			SourceRepository: oldRepository.GitalyRepo,
		})
		if err != nil {
			return err
		}

		if err = repo_module.CreateDelegateHooks(repoPath); err != nil {
			return fmt.Errorf("createDelegateHooks: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// even if below operations failed, it could be ignored. And they will be retried
	if err := repo_module.UpdateRepoSize(ctx, repo); err != nil {
		log.Error("Failed to update size for repository: %v", err)
	}
	if err := repo_model.CopyLanguageStat(opts.BaseRepo, repo); err != nil {
		log.Error("Copy language stat from oldRepo failed: %v", err)
	}

	gitRepo, err := git.OpenRepository(ctx, repo.OwnerName, repo.Name, repo.RepoPath())
	if err != nil {
		log.Error("Open created git repository failed: %v", err)
	} else {
		defer gitRepo.Close()
		if err := repo_module.SyncReleasesWithTags(repo, gitRepo); err != nil {
			log.Error("Sync releases from git tags failed: %v", err)
		}
	}

	notification.NotifyForkRepository(ctx, doer, opts.BaseRepo, repo)

	return repo, nil
}

// ConvertForkToNormalRepository convert the provided repo from a forked repo to normal repo
func ConvertForkToNormalRepository(ctx context.Context, repo *repo_model.Repository) error {
	err := db.WithTx(ctx, func(ctx context.Context) error {
		repo, err := repo_model.GetRepositoryByID(ctx, repo.ID)
		if err != nil {
			return err
		}

		if !repo.IsFork {
			return nil
		}

		if err := repo_model.DecrementRepoForkNum(ctx, repo.ForkID); err != nil {
			log.Error("Unable to decrement repo fork num for old root repo %d of repository %-v whilst converting from fork. Error: %v", repo.ForkID, repo, err)
			return err
		}

		repo.IsFork = false
		repo.ForkID = 0

		if err := repo_module.UpdateRepository(ctx, repo, false); err != nil {
			log.Error("Unable to update repository %-v whilst converting from fork. Error: %v", repo, err)
			return err
		}

		return nil
	})

	return err
}
