package repo_marks

import (
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/modules/timeutil"
)

func init() {
	db.RegisterModel(new(RepoMarks))
}

// RepoMarks Структура для хранения отметок репозитория (например, отметка CodeHub)
type RepoMarks struct {
	ID        int64              `xorm:"PK AUTOINCR" json:"-"`
	ExpertID  int64              `xorm:"NOT NULL" json:"expert_id"`
	RepoID    int64              `xorm:"UNIQUE(repo_mark) INDEX NOT NULL" json:"repo_id"`
	MarkKey   string             `xorm:"UNIQUE(repo_mark) VARCHAR(255) NOT NULL" json:"mark_key"`
	CreatedAt timeutil.TimeStamp `xorm:"CREATED" json:"-"`
	UpdatedAt timeutil.TimeStamp `xorm:"UPDATED" json:"-"`
}
