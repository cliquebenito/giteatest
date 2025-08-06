package v1_27

import (
	"fmt"
	"strconv"

	"code.gitea.io/gitea/models/repo"
	"xorm.io/xorm"
)

// CreateScRepoKeyTable функция для создания миграции таблицы sc_repo_key
func CreateScRepoKeyTable(x *xorm.Engine) error {
	var repositories []*repo.Repository
	if err := x.Table("repository").Find(&repositories); err != nil {
		return fmt.Errorf("get repositories: %w", err)
	}
	if err := x.Sync(new(repo.ScRepoKey)); err != nil {
		return fmt.Errorf("create sc_repo_key table: %w", err)
	}
	if len(repositories) == 0 {
		return nil
	}
	repoKeys := make([]*repo.ScRepoKey, len(repositories))
	for idx, repository := range repositories {
		repoKeys[idx] = &repo.ScRepoKey{
			RepoID: strconv.FormatInt(repository.ID, 10),
		}
	}
	if _, err := x.Insert(&repoKeys); err != nil {
		return fmt.Errorf("insert repo key: %w", err)
	}
	return nil
}
