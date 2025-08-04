// Copyright 2015 The Gogs Authors. All rights reserved.
// Copyright 2018 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package git

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"gitlab.com/gitlab-org/gitaly/v16/proto/go/gitalypb"

	"code.gitea.io/gitea/modules/log"
)

// BranchPrefix base dir of the branch information file store on git
const BranchPrefix = "refs/heads/"

// AGit Flow

// PullRequestPrefix special ref to create a pull request: refs/for/<targe-branch>/<topic-branch>
// or refs/for/<targe-branch> -o topic='<topic-branch>'
const PullRequestPrefix = "refs/for/"

// TODO: /refs/for-review for suggest change interface

// IsReferenceExist returns true if given reference exists in the repository.
func IsReferenceExist(ctx context.Context, owner, repoName, repoPath, name string) bool {
	gitRepo, err := OpenRepository(ctx, owner, repoName, repoPath)
	if err != nil {
		return false
	}

	return gitRepo.IsReferenceExist(name)
}

// IsBranchExist returns true if given branch exists in the repository.
func IsBranchExist(ctx context.Context, owner, repoName, repoPath, name string) bool {
	gitRepo, err := OpenRepository(ctx, owner, repoName, repoPath)
	if err != nil {
		return false
	}

	return gitRepo.IsBranchExist(name)
}

// Branch represents a Git branch.
type Branch struct {
	Name string
	Path string

	gitRepo *Repository
}

// GetHEADBranch returns corresponding branch of HEAD.
func (repo *Repository) GetHEADBranch() (*Branch, error) {
	if repo == nil {
		log.Error("Error has occurred while getting all local branches: nil repo was passed")
		return nil, fmt.Errorf("nil repo")
	}
	stdout, _, err := NewCommand(repo.Ctx, "symbolic-ref", "HEAD").RunStdString(&RunOpts{Dir: repo.Path})
	if err != nil {
		log.Error("Error has occurred while getting HEAD branch: %v", err)
		return nil, fmt.Errorf("get HEAD branch: %w", err)
	}
	stdout = strings.TrimSpace(stdout)

	if !strings.HasPrefix(stdout, BranchPrefix) {
		log.Error("Error has occurred while getting HEAD branch: invalid HEAD branch: %v", stdout)
		return nil, fmt.Errorf("invalid HEAD branch: %v", stdout)
	}

	return &Branch{
		Name:    stdout[len(BranchPrefix):],
		Path:    stdout,
		gitRepo: repo,
	}, nil
}

// SetDefaultBranch sets HEAD (default branch of repository on server) on branch with given name.
func (repo *Repository) SetDefaultBranch(name string) error {
	if repo == nil {
		log.Error("Error has occurred while getting all local branches: nil repo was passed")
		return fmt.Errorf("nil repo")
	}

	request := &gitalypb.WriteRefRequest{
		Repository: repo.GitalyRepo,
		Ref:        []byte("HEAD"),
		Revision:   []byte(BranchPrefix + name),
	}

	if _, err := repo.RepoClient.WriteRef(repo.Ctx, request); err != nil {
		log.Error("Error has occurred while setting default branch to %s: %v", name, err)
		return fmt.Errorf("set default branch to %s: %w", name, err)
	}

	return nil
}

// GetDefaultBranch gets default branch of repository.
//
// Order of choosing a default branch:
//  1. If there are no branches, return an empty string.
//  2. If there is only one branch, return the only branch.
//  3. If a branch exists that matches HEAD, return the HEAD reference name.
//  4. If a branch exists named refs/heads/main, return refs/heads/main.
//  5. If a branch exists named refs/heads/master, return refs/heads/master.
//  6. Return the first branch (as per default ordering by git).
func (repo *Repository) GetDefaultBranch() (string, error) {
	if repo == nil {
		log.Error("Error has occurred while getting all local branches: nil repo was passed")
		return "", fmt.Errorf("nil repo")
	}

	ctx, cancel := context.WithCancel(repo.Ctx)
	defer cancel()
	defaultBranch, err := repo.RefClient.FindDefaultBranchName(ctx, &gitalypb.FindDefaultBranchNameRequest{
		Repository: repo.GitalyRepo,
		HeadOnly:   false,
	})
	if err != nil {
		log.Error("Error has occurred while getting default branch: %v", err)
		return "", fmt.Errorf("get default branch: %w", err)
	}

	return strings.TrimPrefix(string(defaultBranch.GetName()), BranchPrefix), nil
}

// GetBranch returns a branch by it's name
func (repo *Repository) GetBranch(branch string) (*Branch, error) {
	if !repo.IsBranchExist(branch) {
		return nil, ErrBranchNotExist{branch}
	}
	return &Branch{
		Path:    repo.Path,
		Name:    branch,
		gitRepo: repo,
	}, nil
}

// GetAllLocalBranches finds all the local branches under `refs/heads/` for the specified repository.
func (repo *Repository) GetAllLocalBranches(limit int32) ([]*Branch, error) {
	if repo == nil {
		log.Error("Error has occurred while getting all local branches: nil repo was passed")
		return nil, fmt.Errorf("nil repo")
	}

	responseReceiver, err := repo.RefClient.FindLocalBranches(repo.Ctx, &gitalypb.FindLocalBranchesRequest{
		Repository: repo.GitalyRepo,
		PaginationParams: &gitalypb.PaginationParameter{
			Limit: limit,
		},
	})
	if err != nil {
		log.Error("Error has occurred while getting all local branches: %v", err)
		return nil, fmt.Errorf("get all local branches: %w", err)
	}

	response, err := responseReceiver.Recv()
	if err != nil {
		log.Error("Error has occurred while getting all local branches: %v", err)
		return nil, fmt.Errorf("get all local branches: %w", err)
	}

	gitalyBranches := response.GetLocalBranches()
	var branches []*Branch
	for _, gitalyBranch := range gitalyBranches {
		branches = append(branches, &Branch{
			Name:    string(gitalyBranch.Name),
			Path:    repo.Path,
			gitRepo: repo,
		})
	}

	return branches, nil
}

// GetNumLocalBranches returns number of branches under `refs/heads/` for the specified repository.
func (repo *Repository) GetNumLocalBranches() (int, error) {
	if repo == nil {
		log.Error("Error has occurred while getting number of local branches: nil repo was passed")
		return 0, fmt.Errorf("nil repo")
	}

	branches, err := repo.GetAllLocalBranches(-1)
	if err != nil {
		log.Error("Error has occurred while getting number of local branches: %v", err)
		return 0, fmt.Errorf("get branches: %w", err)
	}

	return len(branches), nil
}

// HasOnlyOneBranch checks if the repository has only one branch.
// Needed to check for correctly set default branch on server and not to fetch all branches&
func (repo *Repository) HasOnlyOneBranch() (bool, error) {
	if repo == nil {
		log.Error("Error has occurred while checking if repo has only one branch: nil repo was passed")
		return false, fmt.Errorf("nil repo")
	}

	// Нужно проверить, что подтянется хотя бы 2 ветки, если их больше одной
	branches, err := repo.GetAllLocalBranches(2)
	if err != nil {
		log.Error("Error has occurred while checking if repo has only one branch: %v", err)
		return false, fmt.Errorf("get branches: %w", err)
	}

	return len(branches) == 1, nil
}

// GetBranchesByPath returns a branch by it's path
// if limit = 0 it will not limit
func GetBranchesByPath(ctx context.Context, owner, name, path string, skip, limit int) ([]*Branch, int, error) {
	gitRepo, err := OpenRepository(ctx, owner, name, path)
	if err != nil {
		return nil, 0, err
	}
	defer gitRepo.Close()

	return gitRepo.GetBranches(skip, limit)
}

// GetBranchCommitID returns a branch commit ID by its name
func GetBranchCommitID(ctx context.Context, owner, name, path, branch string) (string, error) {
	gitRepo, err := OpenRepository(ctx, owner, name, path)
	if err != nil {
		return "", err
	}
	defer gitRepo.Close()
	// todo gitaly
	return gitRepo.GetBranchCommitID(branch)
}

// GetBranches returns a slice of *git.Branch
func (repo *Repository) GetBranches(skip, limit int) ([]*Branch, int, error) {
	brs, countAll, err := repo.GetBranchNames(skip, limit)
	if err != nil {
		return nil, 0, err
	}

	branches := make([]*Branch, len(brs))
	for i := range brs {
		branches[i] = &Branch{
			Path:    repo.Path,
			Name:    brs[i],
			gitRepo: repo,
		}
	}

	return branches, countAll, nil
}

// DeleteBranchOptions Option(s) for delete branch
type DeleteBranchOptions struct {
	Force bool
}

// DeleteBranch delete a branch by name on repository.
func (repo *Repository) DeleteBranch(name, commitID, userName, userEmail string, userID int64) error {
	if userEmail == "" {
		userEmail = "source_control@sbertech.ru"
	}

	ctxWithCancel, cancel := context.WithCancel(repo.Ctx)
	defer cancel()
	client, err := repo.OperationClient.UserDeleteBranch(ctxWithCancel, &gitalypb.UserDeleteBranchRequest{
		Repository: repo.GitalyRepo,
		BranchName: []byte(name),
		User: &gitalypb.User{
			GlId:  strconv.FormatInt(userID, 10),
			Name:  []byte(userName),
			Email: []byte(userEmail),
		},
		ExpectedOldOid: commitID,
	})
	if err != nil {
		return fmt.Errorf("error with repo.OperationClient, err: %s", err)
	}

	if client.String() != "" {
		return fmt.Errorf("error with client.String(), err: %s", client.String())
	}

	//cmd := NewCommand(repo.Ctx, "branch")
	//
	//if opts.Force {
	//	cmd.AddArguments("-D")
	//} else {
	//	cmd.AddArguments("-d")
	//}
	//
	//cmd.AddDashesAndList(name)
	//_, _, err := cmd.RunStdString(&RunOpts{Dir: repo.Path})

	return err
}

// CreateBranch create a new branch
func (repo *Repository) CreateBranch(branch, oldbranchOrCommit string) error {
	cmd := NewCommand(repo.Ctx, "branch")
	cmd.AddDashesAndList(branch, oldbranchOrCommit)

	_, _, err := cmd.RunStdString(&RunOpts{Dir: repo.Path})

	return err
}

// AddRemote adds a new remote to repository.
func (repo *Repository) AddRemote(name, url string, fetch bool) error {
	cmd := NewCommand(repo.Ctx, "remote", "add")
	if fetch {
		cmd.AddArguments("-f")
	}
	cmd.AddDynamicArguments(name, url)

	_, _, err := cmd.RunStdString(&RunOpts{Dir: repo.Path})
	return err
}

// RemoveRemote removes a remote from repository.
func (repo *Repository) RemoveRemote(name string) error {
	_, _, err := NewCommand(repo.Ctx, "remote", "rm").AddDynamicArguments(name).RunStdString(&RunOpts{Dir: repo.Path})
	return err
}

// GetCommit returns the head commit of a branch
func (branch *Branch) GetCommit() (*Commit, error) {
	return branch.gitRepo.GetBranchCommit(branch.Name)
}

// RenameBranch rename a branch
func (repo *Repository) RenameBranch(commitID, userName, userEmail, from, to string, userID int64) error {
	if userEmail == "" {
		userEmail = "source_control@sbertech.ru"
	}
	ctxWithCancel, cancel := context.WithCancel(repo.Ctx)
	defer cancel()
	resp, err := repo.OperationClient.UserCreateBranch(ctxWithCancel,
		&gitalypb.UserCreateBranchRequest{
			Repository: repo.GitalyRepo,
			BranchName: []byte(to),
			User: &gitalypb.User{
				GlId:  strconv.FormatInt(userID, 10),
				Name:  []byte(userName),
				Email: []byte(userEmail),
			},
			StartPoint: []byte(commitID),
		},
	)

	if err != nil {
		return err
	}

	err = repo.DeleteBranch(from, commitID, userName, userEmail, userID)
	if err != nil {
		return err
	}
	log.Info(resp.String())

	//todo добавить метод в гитали.

	//
	//branch, response, err := client.Repositories.RenameBranch(repo.Ctx, repo.Owner, repo.Name, from, to)
	//
	//fmt.Println(branch, response)
	//_, _, err := NewCommand(repo.Ctx, "branch", "-m").AddDynamicArguments(from, to).RunStdString(&RunOpts{Dir: repo.Path})
	//return err
	return nil
}
