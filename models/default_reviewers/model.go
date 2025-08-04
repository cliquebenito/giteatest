package default_reviewers

import "code.gitea.io/gitea/models/db"

func init() {
	db.RegisterModel(new(DefaultReviewers))
}

type DefaultReviewers struct {
	ID                   int64   `xorm:"pk autoincr"`
	ReviewSettingID      int64   `xorm:"NOT NULL" json:"review_setting_id"`
	RequiredApprovals    int64   `xorm:"NOT NULL DEFAULT 0"`
	DefaultReviewersList []int64 `xorm:"JSON TEXT"`
}
