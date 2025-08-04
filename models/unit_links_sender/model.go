package unit_links_sender

import (
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/modules/timeutil"
)

func init() {
	db.RegisterModel(new(UnitLinksSenderTasks))
}

type Action string

const (
	SendAddPullRequestLinksAction     Action = "add_pull_request"
	SendDeletePullRequestLinksAction  Action = "delete_pull_request"
	SendUpdatePullRequestStatusAction Action = "update_pull_request_status"
)

type Status string

const (
	StatusDone     Status = "done"
	StatusUnlocked Status = "unlocked"
	StatusLocked   Status = "locked"
)

// UnitLinksSenderTasks структура полей для таблицы unit_links_sender_tasks
type UnitLinksSenderTasks struct {
	ID int64 `xorm:"PK AUTOINCR" json:"-"`

	Payload  string `xorm:"NOT NULL TEXT JSON" json:"payload"`
	Action   Action `xorm:"NOT NULL" json:"action"`
	UserName string `xorm:"NOT NULL" json:"user_name"`

	PullRequestID  int64  `xorm:"NOT NULL" json:"pull_request_id"`
	PullRequestURL string `xorm:"NOT NULL" json:"pull_request_url"`

	Status Status `xorm:"NOT NULL" json:"status"`

	CreatedAt timeutil.TimeStamp `xorm:"CREATED" json:"-"`
	UpdatedAt timeutil.TimeStamp `xorm:"UPDATED" json:"-"`
}
