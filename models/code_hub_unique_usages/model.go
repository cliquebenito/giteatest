package code_hub_unique_usages

import (
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/modules/timeutil"
)

func init() {
	db.RegisterModel(new(CodeHubUniqueUsages))
}

// CodeHubUniqueUsages структура для полей таблицы для хранения уникальных использований репозитория
type CodeHubUniqueUsages struct {
	ID        int64              `xorm:"PK AUTOINCR" json:"-"`
	RepoID    int64              `xorm:"UNIQUE(repo_user) INDEX(s) NOT NULL" json:"repo_id"`
	UserID    int64              `xorm:"UNIQUE(repo_user) INDEX(s) NOT NULL" json:"user_id"`
	UpdatedAt timeutil.TimeStamp `xorm:"UPDATED" json:"-"`
	CreatedAt timeutil.TimeStamp `xorm:"CREATED" json:"-"`
}
