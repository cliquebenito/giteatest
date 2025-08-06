package review_settings_db

import (
	"context"
	"fmt"

	"code.gitea.io/gitea/models/review_settings"
	"xorm.io/builder"
)

// DeleteReviewSettingsByRepoID удаление review settings по repo id
func (r reviewSettingsDB) DeleteReviewSettingsByRepoID(_ context.Context, repoID int64, branchName string) error {
	_, err := r.engine.Where(builder.And(builder.Eq{"branch_name": branchName}, builder.Eq{"repo_id": repoID})).Delete(new(review_settings.ReviewSettings))
	if err != nil {
		return fmt.Errorf("delete review settings by repo id: %w", err)
	}
	return nil
}
