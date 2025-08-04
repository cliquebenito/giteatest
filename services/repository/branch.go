// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repository

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"gitlab.com/gitlab-org/gitaly/v16/proto/go/gitalypb"

	"code.gitea.io/gitea/integration/gitaly"
	"code.gitea.io/gitea/modules/setting"

	"code.gitea.io/gitea/models"
	git_model "code.gitea.io/gitea/models/git"
	"code.gitea.io/gitea/models/git/protected_branch"
	repo_model "code.gitea.io/gitea/models/repo"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/notification"
	repo_module "code.gitea.io/gitea/modules/repository"
	pull_service "code.gitea.io/gitea/services/pull"
)

var EmptyBranchName = ""

// CreateNewBranch creates a new repository branch
func CreateNewBranch(ctx context.Context, doer *user_model.User, repo *repo_model.Repository, oldBranchName, branchName string) (err error) {
	ctxWithCancel, cancel := context.WithCancel(ctx)
	defer cancel()

	// Check if branch name can be used
	if err = checkBranchName(ctxWithCancel, repo, branchName); err != nil {
		return err
	}

	ctx2, conn, err := gitaly.NewRefClient(ctxWithCancel)
	if err != nil {
		return fmt.Errorf("New Commit Client failed, err: %s", err)
	}

	foundBranch, err := conn.FindBranch(ctx2, &gitalypb.FindBranchRequest{
		Repository: &gitalypb.Repository{
			GlRepository:  repo.Name,
			GlProjectPath: repo.OwnerName,
			StorageName:   setting.Gitaly.MainServerName,
			RelativePath:  repo.RepoPath(),
		},
		Name: []byte(oldBranchName),
	})

	if err != nil {
		return fmt.Errorf("FindBranch is failed: %v", err)
	}

	if foundBranch.String() == EmptyBranchName {
		return fmt.Errorf("start branch not exists")
	}

	ctx3, oc, err := gitaly.NewOperationClient(ctx2)
	if err != nil {
		return err
	}

	response, err := oc.UserCreateBranch(ctx3, &gitalypb.UserCreateBranchRequest{
		Repository: &gitalypb.Repository{
			GlRepository:                  repo.Name,
			GlProjectPath:                 repo.OwnerName,
			StorageName:                   setting.Gitaly.MainServerName,
			RelativePath:                  repo.RepoPath(),
			GitAlternateObjectDirectories: repo_module.PushingEnvironment(doer, repo),
		},
		BranchName: []byte(branchName),
		User: &gitalypb.User{
			GlId:       strconv.FormatInt(doer.ID, 10),
			Name:       []byte(doer.Name),
			Email:      []byte(doer.GetDefaultEmail()),
			GlUsername: doer.Name,
		},
		StartPoint: []byte(oldBranchName),
	})
	if err != nil {
		return err
	}

	if string(response.Branch.Name) == EmptyBranchName {
		return fmt.Errorf("UserCreateBranch is failed")
	}

	return nil
}

// GetBranches returns branches from the repository, skipping skip initial branches and
// returning at most limit branches, or all branches if limit is 0.
func GetBranches(ctx context.Context, repo *repo_model.Repository, skip, limit int) ([]*git.Branch, int, error) {
	return git.GetBranchesByPath(ctx, repo.OwnerName, repo.Name, repo.RepoPath(), skip, limit)
}

func GetBranchCommitID(ctx context.Context, repo *repo_model.Repository, branch string) (string, error) {
	return git.GetBranchCommitID(ctx, repo.OwnerName, repo.Name, repo.RepoPath(), branch)
}

// checkBranchName validates branch name with existing repository branches
func checkBranchName(ctx context.Context, repo *repo_model.Repository, name string) error {
	gitRepo, err := git.OpenRepository(ctx, repo.OwnerName, repo.Name, repo.RepoPath())
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	patterns := make([][]byte, 0)
	patterns = append(patterns, []byte(git.BranchPrefix))
	patterns = append(patterns, []byte(git.TagPrefix))

	ctxWithCancel, cancel := context.WithCancel(gitRepo.Ctx)
	defer cancel()
	listRefsReq, err := gitRepo.RefClient.ListRefs(ctxWithCancel, &gitalypb.ListRefsRequest{
		Repository: gitRepo.GitalyRepo,
		Patterns:   patterns,
		Head:       true,
		PeelTags:   false,
	})
	if err != nil {
		return fmt.Errorf("failed to get references: %w", err)
	}

	listRefs, err := listRefsReq.Recv()
	if err != nil {
		return fmt.Errorf("failed to recieve references: %w", err)
	}

	for _, ref := range listRefs.GetReferences() {
		refName := string(ref.GetName())
		branchRefName := strings.TrimPrefix(refName, git.BranchPrefix)
		switch {
		case branchRefName == name:
			return models.ErrBranchAlreadyExists{
				BranchName: name,
			}
		// If branchRefName like a/b but we want to create a branch named a then we have a conflict
		case strings.HasPrefix(branchRefName, name+"/"):
			return models.ErrBranchNameConflict{
				BranchName: branchRefName,
			}
			// Conversely if branchRefName like a but we want to create a branch named a/b then we also have a conflict
		case strings.HasPrefix(name, branchRefName+"/"):
			return models.ErrBranchNameConflict{
				BranchName: branchRefName,
			}
		case refName == git.TagPrefix+name:
			return models.ErrTagAlreadyExists{
				TagName: name,
			}
		}
	}

	return nil
}

// CreateNewBranchFromCommit creates a new repository branch
func CreateNewBranchFromCommit(ctx context.Context, doer *user_model.User, repo *repo_model.Repository, commit, branchName string) (err error) {
	// Check if branch name can be used
	if err := checkBranchName(ctx, repo, branchName); err != nil {
		return err
	}

	ctx2, oc, err := gitaly.NewOperationClient(ctx)
	if err != nil {
		return err
	}

	response, err := oc.UserCreateBranch(ctx2, &gitalypb.UserCreateBranchRequest{
		Repository: &gitalypb.Repository{
			GlRepository:                  repo.Name,
			GlProjectPath:                 repo.OwnerName,
			StorageName:                   setting.Gitaly.MainServerName,
			RelativePath:                  repo.RepoPath(),
			GitAlternateObjectDirectories: repo_module.PushingEnvironment(doer, repo),
		},
		BranchName: []byte(branchName),
		User: &gitalypb.User{
			GlId:       strconv.FormatInt(doer.ID, 10),
			Name:       []byte(doer.Name),
			Email:      []byte(doer.GetDefaultEmail()),
			GlUsername: doer.Name,
		},
		StartPoint: []byte(commit),
	})
	if err != nil {
		return err
	}

	if string(response.Branch.Name) == EmptyBranchName {
		return fmt.Errorf("UserCreateBranch is failed")
	}

	return nil
}

// RenameBranch rename a branch TODO переписать! Не возвращать ошибки в виде строк
func RenameBranch(ctx context.Context, repo *repo_model.Repository, doer *user_model.User, gitRepo *git.Repository, from, to string) (err error) {
	if from == to {
		return fmt.Errorf("target_exist")
	}

	if gitRepo.IsBranchExist(to) {
		return fmt.Errorf("target_exist")
	}

	if !gitRepo.IsBranchExist(from) {
		return fmt.Errorf("target_not_exist")
	}

	if err := git_model.RenameBranch(ctx, repo, from, to, func(isDefault bool) error {
		branch, err := gitRepo.GetBranch(from)
		if err != nil {
			log.Error(err.Error())
		}
		commit, err := branch.GetCommit()
		if err != nil {
			log.Error(err.Error())
		}
		log.Info(commit.ID.String())

		err2 := gitRepo.RenameBranch(commit.ID.String(), doer.Name, doer.Email, from, to, doer.ID)
		if err2 != nil {
			return err2
		}

		if isDefault {
			err2 = gitRepo.SetDefaultBranch(to)
			if err2 != nil {
				return err2
			}
		}

		return nil
	}); err != nil {
		return fmt.Errorf("RenameBranch failed. err: %v", err)
	}
	refID, err := gitRepo.GetRefCommitID(git.BranchPrefix + to)
	if err != nil {
		return fmt.Errorf("GetRefCommitID failed. err: %v", err)
	}

	notification.NotifyDeleteRef(ctx, doer, repo, "branch", git.BranchPrefix+from)
	notification.NotifyCreateRef(ctx, doer, repo, "branch", git.BranchPrefix+to, refID)

	return nil
}

// enmuerates all branch related errors
var (
	ErrBranchIsDefault = errors.New("branch is default")
)

// DeleteBranch delete branch
func DeleteBranch(ctx context.Context, doer *user_model.User, repo *repo_model.Repository, gitRepo *git.Repository, branchName string) error {
	if branchName == repo.DefaultBranch {
		return ErrBranchIsDefault
	}

	isProtected, err := git_model.IsBranchProtected(ctx, repo.ID, branchName)
	if err != nil {
		return err
	}
	if isProtected {
		return protected_branch.NewBranchIsProtectedError(branchName)
	}

	commit, err := gitRepo.GetBranchCommit(branchName)
	if err != nil {
		log.Error("Error has occurred while getting branch by commit: %v", err)
		return fmt.Errorf("getting branch by commit: %w", err)
	}

	ctxWithCancel, cancel := context.WithCancel(ctx)
	defer cancel()
	client, err := gitRepo.OperationClient.UserDeleteBranch(ctxWithCancel, &gitalypb.UserDeleteBranchRequest{
		Repository: gitRepo.GitalyRepo,
		BranchName: []byte(branchName),
		User: &gitalypb.User{
			GlId:       strconv.Itoa(int(doer.ID)),
			Name:       []byte(doer.Name),
			Email:      []byte(doer.GetDefaultEmail()),
			GlUsername: doer.Name,
		},
		ExpectedOldOid: "",
	})
	if err != nil {
		return fmt.Errorf("error with repo.OperationClient, err: %s", err)
	}

	if client.String() != "" {
		return fmt.Errorf("error with client.String(), err: %s", client.String())
	}

	if err := pull_service.CloseBranchPulls(doer, repo.ID, branchName); err != nil {
		return err
	}

	// Don't return error below this
	if err := PushUpdate(
		&repo_module.PushUpdateOptions{
			RefFullName:  git.BranchPrefix + branchName,
			OldCommitID:  commit.ID.String(),
			NewCommitID:  git.EmptySHA,
			PusherID:     doer.ID,
			PusherName:   doer.Name,
			RepoUserName: repo.OwnerName,
			RepoName:     repo.Name,
		}); err != nil {
		log.Error("Update: %v", err)
	}

	if err := git_model.AddDeletedBranch(ctx, repo.ID, branchName, commit.ID.String(), doer.ID); err != nil {
		log.Warn("AddDeletedBranch: %v", err)
	}

	return nil
}
