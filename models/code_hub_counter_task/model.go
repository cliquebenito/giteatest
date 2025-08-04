package code_hub_counter_task

import (
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/modules/timeutil"
)

func init() {
	db.RegisterModel(new(CodeHubCounterTasks))
}

type CodeHubAction string

const (
	CloneRepositoryAction CodeHubAction = "clone_repository"
)

type Status string

const (
	StatusDone     Status = "done"
	StatusUnlocked Status = "unlocked"
	StatusLocked   Status = "locked"
)

// CodeHubCounterTasks модель для хранения тасок для подсчета статистики по уникальным использованиям репозитория
type CodeHubCounterTasks struct {
	ID int64 `xorm:"PK AUTOINCR" json:"-"`

	UserID int64 `xorm:"NOT NULL" json:"user_id"`
	RepoID int64 `xorm:"NOT NULL" json:"repo_id"`

	Action CodeHubAction `xorm:"NOT NULL" json:"action"`
	Status Status        `xorm:"NOT NULL" json:"status"`

	CreatedAt timeutil.TimeStamp `xorm:"CREATED" json:"-"`
	UpdatedAt timeutil.TimeStamp `xorm:"UPDATED" json:"-"`
}
