package review_settings_db

import (
	"context"
	"fmt"

	"code.gitea.io/gitea/models/review_settings"
	"xorm.io/builder"
)

// GetReviewSettings Получить ревьюверов по repoID
func (r reviewSettingsDB) GetReviewSettings(_ context.Context, repoID int64) ([]*review_settings.ReviewSettings, error) {
	settings := make([]*review_settings.ReviewSettings, 0)

	err := r.engine.Where(builder.Eq{"repo_id": repoID}).Find(&settings)
	if err != nil {
		return nil, fmt.Errorf("find review settings: %w", err)
	}
	if len(settings) == 0 {
		return nil, NewReviewSettingsDoesntExistsError(repoID, "")
	}
	return settings, nil
}

// GetReviewSettingsByBranchPattern Получить ревьюверов по branch name
func (r reviewSettingsDB) GetReviewSettingsByBranchPattern(_ context.Context, repoID int64, branchName string) (*review_settings.ReviewSettings, error) {
	settings := &review_settings.ReviewSettings{}

	has, err := r.engine.Where(builder.And(builder.Eq{"branch_name": branchName}, builder.Eq{"repo_id": repoID})).Get(settings)
	if err != nil {
		return nil, fmt.Errorf("find review settings: %w", err)
	}
	if !has {
		return nil, NewReviewSettingsDoesntExistsError(repoID, branchName)
	}
	return settings, nil
}
