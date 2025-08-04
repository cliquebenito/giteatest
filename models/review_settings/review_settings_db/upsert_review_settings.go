package review_settings_db

import (
	"context"
	"encoding/json"
	"fmt"

	"code.gitea.io/gitea/models/review_settings"
	"code.gitea.io/gitea/modules/timeutil"
)

// UpsertReviewSettings Upsert настроек ревью
func (r reviewSettingsDB) UpsertReviewSettings(_ context.Context, rs *review_settings.ReviewSettings) error {
	jsonUserIDs, err := json.Marshal(rs.MergeWhitelistUserIDs)
	if err != nil {
		return fmt.Errorf("user ids: %w", err)
	}
	jsonContexts, err := json.Marshal(rs.StatusCheckContexts)
	if err != nil {
		return fmt.Errorf("marshal contexts: %w", err)
	}

	now := timeutil.TimeStampNow()

	if rs.CreatedUnix.IsZero() {
		rs.CreatedUnix = now
	}

	_, err = r.engine.Exec(`
		INSERT INTO review_settings (
			repo_id, branch_name,
			enable_merge_whitelist, merge_whitelist_user_i_ds,
			enable_status_check, status_check_contexts,
			enable_default_reviewers, block_on_rejected_reviews,
			block_on_official_review_requests, block_on_outdated_branch,
			dismiss_stale_approvals, enable_sonar_qube,
			created_unix, updated_unix
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(repo_id, branch_name) DO UPDATE SET
			enable_merge_whitelist = excluded.enable_merge_whitelist,
			merge_whitelist_user_i_ds = excluded.merge_whitelist_user_i_ds,
			enable_status_check = excluded.enable_status_check,
			status_check_contexts = excluded.status_check_contexts,
			enable_default_reviewers = excluded.enable_default_reviewers,
			block_on_rejected_reviews = excluded.block_on_rejected_reviews,
			block_on_official_review_requests = excluded.block_on_official_review_requests,
			block_on_outdated_branch = excluded.block_on_outdated_branch,
			dismiss_stale_approvals = excluded.dismiss_stale_approvals,
			enable_sonar_qube = excluded.enable_sonar_qube,
			updated_unix = excluded.updated_unix
	`, rs.RepoID, rs.RuleName,
		rs.EnableMergeWhitelist, string(jsonUserIDs),
		rs.EnableStatusCheck, string(jsonContexts),
		rs.EnableDefaultReviewers, rs.BlockOnRejectedReviews,
		rs.BlockOnOfficialReviewRequests, rs.BlockOnOutdatedBranch,
		rs.DismissStaleApprovals, rs.EnableSonarQube,
		rs.CreatedUnix, now)

	if err != nil {
		return fmt.Errorf("upsert review settings: %w", err)
	}

	return nil
}
