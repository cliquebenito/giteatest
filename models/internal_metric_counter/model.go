package internal_metric_counter

import (
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/modules/timeutil"
)

func init() {
	db.RegisterModel(new(InternalMetricCounter))
}

// InternalMetricCounter структура для полей таблицы счетчика уникальных использований репозитория
type InternalMetricCounter struct {
	ID          int64              `xorm:"PK AUTOINCR" json:"-"`
	RepoID      int64              `xorm:"INDEX NOT NULL" json:"repo_id"`
	MetricValue int                `xorm:"NOT NULL DEFAULT 0" json:"value"`
	MetricKey   string             `xorm:"VARCHAR(255) NOT NULL" json:"key"`
	UpdatedAt   timeutil.TimeStamp `xorm:"UPDATED" json:"-"`
	CreatedAt   timeutil.TimeStamp `xorm:"CREATED" json:"-"`
}
