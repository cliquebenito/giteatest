package utils

import (
	"context"
	"fmt"

	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/repository"
)

// SetServerDefaultBranch получает дефолтную ветку репозитория из гитали (см. GetDefaultBranch),
// и затем устанавливает эту ветку в качестве HEAD на сервере. После этого обновляется информация в БД.
func SetServerDefaultBranch(ctx context.Context, repo *git.Repository) error {
	defaultBranchName, err := repo.GetDefaultBranch()
	if err != nil {
		log.Error("Error has occurred while getting default branch: %v", err)
		return fmt.Errorf("get default branch: %w", err)
	}

	if err = repo.SetDefaultBranch(defaultBranchName); err != nil {
		log.Error("Error has occurred while setting default branch: %v", err)
		return fmt.Errorf("set default branch: %w", err)
	}

	dbRepo, err := repo_model.GetRepositoryByOwnerAndName(ctx, repo.Owner, repo.Name)
	if err != nil {
		log.Error("Error has occurred while getting repository by owner and name: %v", err)
		return fmt.Errorf("get repository by owner and name: %w", err)
	}

	if err = repo_model.UpdateRepoDefaultBranch(ctx, dbRepo, defaultBranchName); err != nil {
		log.Error("Error has occurred while updating repository default branch: %v", err)
		return fmt.Errorf("updatу repository default branch: %w", err)
	}

	if err = repository.UpdateRepoSizeForDefaultBranch(ctx, dbRepo); err != nil {
		log.Error("Error has occurred while updating repository size: %v", err)
		return fmt.Errorf("update repository size: %w", err)
	}

	return nil
}
