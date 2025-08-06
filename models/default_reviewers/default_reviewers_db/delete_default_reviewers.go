package default_reviewers_db

import (
	"context"
	"fmt"

	"code.gitea.io/gitea/models/default_reviewers"
	"xorm.io/builder"
)

// DeleteDefaultReviewers удаление default reviewers
func (r defaultReviewersDB) DeleteDefaultReviewers(ctx context.Context, defaultReviewers []*default_reviewers.DefaultReviewers) error {
	_, err := r.engine.Delete(defaultReviewers)
	if err != nil {
		return fmt.Errorf("delete default reviewers: %w", err)
	}
	return nil
}

// DeleteDefaultReviewersBySettingID удаление default reviewers по review setting id
func (r defaultReviewersDB) DeleteDefaultReviewersBySettingID(ctx context.Context, settingID int64) error {
	_, err := r.engine.Where(builder.Eq{"review_setting_id": settingID}).Delete(new(default_reviewers.DefaultReviewers))
	if err != nil {
		return fmt.Errorf("delete default reviewers by setting id: %w", err)
	}
	return nil
}
