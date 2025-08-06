package external_metric_counter

import (
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/modules/timeutil"
)

func init() {
	db.RegisterModel(new(ExternalMetricCounter))
}

// ExternalMetricCounter структура для полей таблицы внешнего счетчика
type ExternalMetricCounter struct {
	ID          int64              `xorm:"PK AUTOINCR" json:"-"`
	RepoID      int64              `xorm:"UNIQUE(s) INDEX NOT NULL" json:"repo_id"`
	MetricValue int                `xorm:"NOT NULL DEFAULT 0" json:"value"`
	Text        string             `xorm:"VARCHAR(255) NOT NULL" json:"text"`
	UpdatedAt   timeutil.TimeStamp `xorm:"UPDATED" json:"-"`
	CreatedAt   timeutil.TimeStamp `xorm:"CREATED" json:"-"`
}
