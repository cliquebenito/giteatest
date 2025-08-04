package v1_33

import (
	"fmt"

	"code.gitea.io/gitea/models/default_reviewers"
	"code.gitea.io/gitea/models/git/protected_branch"
	"code.gitea.io/gitea/models/review_settings"
	"xorm.io/xorm"
)

func CreateReviewSettingsAndDefaultReviewersTable(x *xorm.Engine) error {
	if err := x.Sync(new(review_settings.ReviewSettings)); err != nil {
		return fmt.Errorf("failed to sync ReviewSettings model: %w", err)
	}

	if err := x.Sync(new(default_reviewers.DefaultReviewers)); err != nil {
		return fmt.Errorf("failed to sync DefaultReviewers model: %w", err)
	}

	var branches []protected_branch.ProtectedBranch
	if err := x.Find(&branches); err != nil {
		return err
	}

	for _, branch := range branches {
		rs := &review_settings.ReviewSettings{
			RepoID:                        branch.RepoID,
			RuleName:                      branch.RuleName,
			EnableMergeWhitelist:          branch.EnableMergeWhitelist,
			MergeWhitelistUserIDs:         branch.MergeWhitelistUserIDs,
			EnableStatusCheck:             branch.EnableStatusCheck,
			StatusCheckContexts:           branch.StatusCheckContexts,
			EnableDefaultReviewers:        branch.EnableApprovalsWhitelist,
			BlockOnRejectedReviews:        branch.BlockOnRejectedReviews,
			BlockOnOfficialReviewRequests: branch.BlockOnOfficialReviewRequests,
			BlockOnOutdatedBranch:         branch.BlockOnOutdatedBranch,
			DismissStaleApprovals:         branch.DismissStaleApprovals,
			EnableSonarQube:               branch.EnableSonarQube,
			CreatedUnix:                   branch.CreatedUnix,
			UpdatedUnix:                   branch.UpdatedUnix,
		}
		if _, err := x.Insert(rs); err != nil {
			return err
		}

		if branch.EnableApprovalsWhitelist {
			dr := &default_reviewers.DefaultReviewers{
				ReviewSettingID:      rs.ID,
				RequiredApprovals:    branch.RequiredApprovals,
				DefaultReviewersList: branch.ApprovalsWhitelistUserIDs,
			}
			if _, err := x.Insert(dr); err != nil {
				return err
			}
		}
	}

	return nil
}
