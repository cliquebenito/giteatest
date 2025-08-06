// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package stats

import (
	"fmt"

	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/graceful"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/process"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/services"
)

// DBIndexer implements Indexer interface to use database's like search
type DBIndexer struct{}

// Index repository status function
func (db *DBIndexer) Index(id int64) error {
	ctx, _, finished := process.GetManager().AddContext(graceful.GetManager().ShutdownContext(), fmt.Sprintf("Stats.DB Index Repo[%d]", id))
	defer finished()

	repo, err := repo_model.GetRepositoryByID(ctx, id)
	if err != nil {
		log.Error("Error has occurred while getting repository by ID: %v", err)
		return fmt.Errorf("get repository by ID: %w", err)
	}
	if repo.IsEmpty {
		return nil
	}

	status, err := repo_model.GetIndexerStatus(ctx, repo, repo_model.RepoIndexerTypeStats)
	if err != nil {
		log.Error("Error has occurred while getting indexer status: %v", err)
		return fmt.Errorf("get indexer status: %w", err)
	}

	gitRepo, err := git.OpenRepository(ctx, repo.OwnerName, repo.Name, repo.RepoPath())
	if err != nil {
		if err.Error() == "no such file or directory" {
			return nil
		}
		log.Error("Error has occurred while opening git repository: %v", err)
		return fmt.Errorf("open repository: %w", err)
	}
	defer gitRepo.Close()

	// Get latest commit for default branch
	commitID, err := gitRepo.GetBranchCommitID(repo.DefaultBranch)
	if err != nil {
		if git.IsErrBranchNotExist(err) || git.IsErrNotExist(err) || setting.IsInTesting {
			log.Debug("Unable to get commit ID for default branch %s in %s ... skipping this repository", repo.DefaultBranch, repo.RepoPath())
			return nil
		}
		log.Error("Error has occurred while getting commit ID for default branch '%s' in repo with path '%s': %v", repo.DefaultBranch, repo.RepoPath(), err)
		return fmt.Errorf("get commit ID for default branch '%s' in repo with path '%s': %w", repo.DefaultBranch, repo.RepoPath(), err)
	}

	// Do not recalculate stats if already calculated for this commit
	if status.CommitSha == commitID {
		return nil
	}

	fileLicensesInfo, err := services.GetLicensesInfoForRepo(gitRepo, commitID, repo.ID, repo.OwnerName, repo.DefaultBranch)
	if err != nil {
		log.Error("Error has occurred while getting licenses info for repo with ID '%d': %v", repo.ID, err)
		return fmt.Errorf("get licenses info for repo with ID '%d': %w", repo.ID, err)
	}

	if err := repo_model.UpsertInfoLicense(repo.ID, commitID, repo.DefaultBranch, fileLicensesInfo); err != nil {
		log.Error("Error has occurred while insert or update information about license for repo with ID '%d': %v", repo.ID, err)
		return fmt.Errorf("insert or update information about license for repo with ID '%d': %w", repo.ID, err)
	}

	// Calculate and save language statistics to database AAA
	stats, err := gitRepo.GetLanguageStats(commitID)
	if err != nil {
		if !setting.IsInTesting {
			log.Error("Error has occurred while getting language stats for commit '%s' for default branch '%s' in repo with path '%s': %v", commitID, repo.DefaultBranch, repo.RepoPath(), err)
		}
		return fmt.Errorf("get language stats: %w", err)
	}

	if err := repo_model.UpdateLanguageStats(repo, commitID, stats); err != nil {
		log.Error("Error has occurred while updating language stats for commit '%s' for default branch '%s' in repo with path '%s': %v", commitID, repo.DefaultBranch, repo.RepoPath(), err)
		return fmt.Errorf("update language stats for commit '%s' for default branch '%s' in repo with pth '%s': %w", commitID, repo.DefaultBranch, repo.RepoPath(), err)
	}

	log.Debug("DBIndexer completed language stats for ID %s for default branch %s in %s. stats count: %d", commitID, repo.DefaultBranch, repo.RepoPath(), len(stats))
	return nil
}

// Close dummy function
func (db *DBIndexer) Close() {
}
