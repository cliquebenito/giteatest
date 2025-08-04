// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gitlab.com/gitlab-org/gitaly/v16/proto/go/gitalypb"

	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/setting"

	"code.gitea.io/gitea/models/db"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/util"
)

// UpdateRepositoryOwnerNames updates repository owner_names (this should only be used when the ownerName has changed case)
func UpdateRepositoryOwnerNames(ownerID int64, ownerName string) error {
	if ownerID == 0 {
		return nil
	}
	ctx, committer, err := db.TxContext(db.DefaultContext)
	if err != nil {
		return err
	}
	defer committer.Close()

	if _, err := db.GetEngine(ctx).Where("owner_id = ?", ownerID).Cols("owner_name").Update(&Repository{
		OwnerName: ownerName,
	}); err != nil {
		return err
	}

	return committer.Commit()
}

// UpdateRepositoryUpdatedTime updates a repository's updated time
func UpdateRepositoryUpdatedTime(repoID int64, updateTime time.Time) error {
	_, err := db.GetEngine(db.DefaultContext).Exec("UPDATE repository SET updated_unix = ? WHERE id = ?", updateTime.Unix(), repoID)
	return err
}

// UpdateRepositoryCols updates repository's columns
func UpdateRepositoryCols(ctx context.Context, repo *Repository, cols ...string) error {
	_, err := db.GetEngine(ctx).ID(repo.ID).Cols(cols...).Update(repo)
	return err
}

// ErrReachLimitOfRepo represents a "ReachLimitOfRepo" kind of error.
type ErrReachLimitOfRepo struct {
	Limit int
}

// IsErrReachLimitOfRepo checks if an error is a ErrReachLimitOfRepo.
func IsErrReachLimitOfRepo(err error) bool {
	_, ok := err.(ErrReachLimitOfRepo)
	return ok
}

func (err ErrReachLimitOfRepo) Error() string {
	return fmt.Sprintf("user has reached maximum limit of repositories [limit: %d]", err.Limit)
}

func (err ErrReachLimitOfRepo) Unwrap() error {
	return util.ErrPermissionDenied
}

// ErrRepoAlreadyExist represents a "RepoAlreadyExist" kind of error.
type ErrRepoAlreadyExist struct {
	Uname string
	Name  string
}

// IsErrRepoAlreadyExist checks if an error is a ErrRepoAlreadyExist.
func IsErrRepoAlreadyExist(err error) bool {
	_, ok := err.(ErrRepoAlreadyExist)
	return ok
}

func (err ErrRepoAlreadyExist) Error() string {
	return fmt.Sprintf("repository already exists [uname: %s, name: %s]", err.Uname, err.Name)
}

func (err ErrRepoAlreadyExist) Unwrap() error {
	return util.ErrAlreadyExist
}

// ErrRepoFilesAlreadyExist represents a "RepoFilesAlreadyExist" kind of error.
type ErrRepoFilesAlreadyExist struct {
	Uname string
	Name  string
}

// IsErrRepoFilesAlreadyExist checks if an error is a ErrRepoAlreadyExist.
func IsErrRepoFilesAlreadyExist(err error) bool {
	_, ok := err.(ErrRepoFilesAlreadyExist)
	return ok
}

func (err ErrRepoFilesAlreadyExist) Error() string {
	return fmt.Sprintf("repository files already exist [uname: %s, name: %s]", err.Uname, err.Name)
}

func (err ErrRepoFilesAlreadyExist) Unwrap() error {
	return util.ErrAlreadyExist
}

// CheckCreateRepository check if could created a repository
func CheckCreateRepository(doer, u *user_model.User, name string, overwriteOrAdopt bool) error {
	if !doer.CanCreateRepo() {
		return ErrReachLimitOfRepo{u.MaxRepoCreation}
	}

	if err := IsUsableRepoName(name); err != nil {
		return err
	}

	has, err := IsRepositoryModelOrDirExist(db.DefaultContext, u, name)
	if err != nil {
		return fmt.Errorf("IsRepositoryExist: %w", err)
	} else if has {
		return ErrRepoAlreadyExist{u.Name, name}
	}

	repoPath := RepoPath(u.Name, name)
	isExist, err := util.IsExist(repoPath)
	if err != nil {
		log.Error("Unable to check if %s exists. Error: %v", repoPath, err)
		return err
	}
	if !overwriteOrAdopt && isExist {
		return ErrRepoFilesAlreadyExist{u.Name, name}
	}
	return nil
}

// ChangeRepositoryName changes all corresponding setting from old repository name to new one.
func ChangeRepositoryName(ctx context.Context, doer *user_model.User, repo *Repository, newRepoName string) (err error) {
	oldRepoName := repo.Name
	newRepoName = strings.ToLower(newRepoName)

	if err = IsUsableRepoName(newRepoName); err != nil {
		return err
	}

	if err := repo.LoadOwner(ctx); err != nil {
		return err
	}

	gitRepo, read, err := git.RepositoryFromContextOrOpen(ctx, repo.Owner.Name, repo.Name, repo.RepoPath())
	if err != nil {
		return err
	}

	fmt.Println(read)
	newRepoPath := strings.Split(repo.RepoPath(), repo.LowerName)[0] + newRepoName + ".git"

	url := repo.Link()
	fmt.Println(url)

	newRepo, err := gitRepo.RepoClient.CreateRepository(ctx, &gitalypb.CreateRepositoryRequest{
		Repository: &gitalypb.Repository{
			GlRepository:  newRepoName,
			GlProjectPath: repo.OwnerName,
			RelativePath:  newRepoPath,
			StorageName:   setting.Gitaly.MainServerName,
		},
		DefaultBranch: []byte(repo.DefaultBranch),
	})

	if err != nil {
		return err
	}

	if newRepo.String() != "" {
	}

	path, err := gitRepo.RepoClient.ReplicateRepository(ctx, &gitalypb.ReplicateRepositoryRequest{
		Repository: &gitalypb.Repository{
			GlRepository:  newRepoName,
			GlProjectPath: repo.OwnerName,
			RelativePath:  newRepoPath,
			StorageName:   setting.Gitaly.MainServerName,
		},
		Source: gitRepo.GitalyRepo,
		ReplicateObjectDeduplicationNetworkMembership: true,
	})
	fmt.Println(path)

	////paths, err := gitRepo.GetTree("")
	//
	//res, err := gitRepo.RepoClient.CreateRepositoryFromURL(ctx, &gitalypb.CreateRepositoryFromURLRequest{
	//	Repository: &gitalypb.Repository{
	//		StorageName:  setting.Gitaly.MainServerName,
	//		RelativePath: newRepoPath,
	//	},
	//	Mirror: true,
	//	Url:    "/Users/21905502/Downloads/gitea/data/sourcecontrol-repositories/check_delete_repo/newrepopath.git",
	//	//HttpAuthorizationHeader: url,
	//	//Mirror: false,
	//	//ResolvedAddress: newRepoPath,
	//})

	defer gitRepo.Close()
	if err != nil {
		return
	}

	resp, err := gitRepo.RepoClient.RemoveRepository(ctx, &gitalypb.RemoveRepositoryRequest{
		Repository: gitRepo.GitalyRepo,
	})

	if err != nil {
		return
	}

	if resp.String() != "" {
		return
	}

	has, err := IsRepositoryModelOrDirExist(ctx, repo.Owner, newRepoName)
	if err != nil {
		return fmt.Errorf("IsRepositoryExist: %w", err)
	} else if has {
		return ErrRepoAlreadyExist{repo.Owner.Name, newRepoName}
	}

	newRepoPath = RepoPath(repo.Owner.Name, newRepoName)
	if err = util.Rename(repo.RepoPath(), newRepoPath); err != nil {
		return fmt.Errorf("rename repository directory: %w", err)
	}

	wikiPath := repo.WikiPath()
	isExist, err := util.IsExist(wikiPath)
	if err != nil {
		log.Error("Unable to check if %s exists. Error: %v", wikiPath, err)
		return err
	}
	if isExist {
		if err = util.Rename(wikiPath, WikiPath(repo.Owner.Name, newRepoName)); err != nil {
			return fmt.Errorf("rename repository wiki: %w", err)
		}
	}

	ctx, committer, err := db.TxContext(db.DefaultContext)
	if err != nil {
		return err
	}
	defer committer.Close()

	if err := NewRedirect(ctx, repo.Owner.ID, repo.ID, oldRepoName, newRepoName); err != nil {
		return err
	}

	return committer.Commit()
}

// UpdateRepoSize updates the repository size, calculating it using getDirectorySize
func UpdateRepoSize(ctx context.Context, repoID, size int64) error {
	if _, err := db.GetEngine(ctx).ID(repoID).Cols("size").NoAutoTime().Update(&Repository{Size: size}); err != nil {
		return fmt.Errorf("update repo size in database: %w", err)
	}

	return nil
}

// UpdateRepoDefaultBranch updates the repository default branch in database.
func UpdateRepoDefaultBranch(ctx context.Context, repo *Repository, newDefaultBranchName string) error {
	if _, err := db.GetEngine(ctx).ID(repo.ID).Cols("default_branch").Update(&Repository{DefaultBranch: newDefaultBranchName}); err != nil {
		return fmt.Errorf("update repo default branch: %w", err)
	}

	return nil
}

// UpdateRepoEmptyStatus обновляет поле is_empty для репозитория в БД.
func UpdateRepoEmptyStatus(ctx context.Context, repo *Repository, empty bool) error {
	if _, err := db.GetEngine(ctx).ID(repo.ID).Cols("is_empty").Update(&Repository{IsEmpty: empty}); err != nil {
		log.Error("Error has occurred while updating repository empty status: %v", err)
		return fmt.Errorf("update repo empty status: %w", err)
	}

	return nil
}

// ErrCreateUserRepo represents a "CreateUserRepo" kind of error.
type ErrCreateUserRepo struct{}

// IsErrCreateUserRepo checks if an error is a ErrCreateUserRepo.
func IsErrCreateUserRepo(err error) bool {
	_, ok := err.(ErrCreateUserRepo)
	return ok
}

func (err ErrCreateUserRepo) Error() string {
	return fmt.Sprintf("Creating a repository outside the project is prohibited")
}
