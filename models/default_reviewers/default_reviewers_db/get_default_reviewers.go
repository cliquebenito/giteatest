package default_reviewers_db

import (
	"context"
	"fmt"

	"code.gitea.io/gitea/models/default_reviewers"
	"xorm.io/builder"
)

// GetDefaultReviewers Получить ревьюверов по reviewSettingID
func (r defaultReviewersDB) GetDefaultReviewers(ctx context.Context, settingID int64) ([]*default_reviewers.DefaultReviewers, error) {
	reviewers := make([]*default_reviewers.DefaultReviewers, 0)

	err := r.engine.Where(builder.Eq{"review_setting_id": settingID}).Find(&reviewers)
	if err != nil {
		return nil, fmt.Errorf("find default reviewers: %w", err)
	}
	return reviewers, nil
}
