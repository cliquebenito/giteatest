package v1_29

import (
	"fmt"

	"xorm.io/xorm"

	"code.gitea.io/gitea/models/code_hub_counter_task"
	"code.gitea.io/gitea/models/code_hub_unique_usages"
	"code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/modules/timeutil"
)

// CreateCodeHubCounterTable функция для создания миграции таблицы internal_metric_counter
func CreateCodeHubCounterTable(x *xorm.Engine) error {
	type CodeHubCounter struct {
		ID            int64              `xorm:"PK AUTOINCR" json:"-"`
		RepoID        int64              `xorm:"UNIQUE(s) INDEX NOT NULL" json:"repo_id"`
		NumUniqUsages int                `xorm:"NOT NULL DEFAULT 0" json:"num_uniq_usages"`
		UpdatedAt     timeutil.TimeStamp `xorm:"UPDATED" json:"-"`
		CreatedAt     timeutil.TimeStamp `xorm:"CREATED" json:"-"`
	}

	var repositories []*repo.Repository
	if err := x.Table("repository").Find(&repositories); err != nil {
		return fmt.Errorf("get repositories: %w", err)
	}
	if err := x.Sync(new(CodeHubCounter)); err != nil {
		return fmt.Errorf("create sc_repo_key table: %w", err)
	}
	if len(repositories) == 0 {
		return nil
	}

	codeHubCounters := make([]*CodeHubCounter, len(repositories))
	for idx, repository := range repositories {
		codeHubCounters[idx] = &CodeHubCounter{
			RepoID:        repository.ID,
			NumUniqUsages: 0,
			UpdatedAt:     timeutil.TimeStampNow(),
			CreatedAt:     timeutil.TimeStampNow(),
		}
	}
	if _, err := x.Insert(&codeHubCounters); err != nil {
		return fmt.Errorf("insert code hub counter: %w", err)
	}
	return nil
}

func CreateCodeHubUniqueUsagesTable(x *xorm.Engine) error {
	return x.Sync(new(code_hub_unique_usages.CodeHubUniqueUsages))
}

func CreateCodeHubCounterTasksTable(x *xorm.Engine) error {
	return x.Sync(new(code_hub_counter_task.CodeHubCounterTasks))
}
