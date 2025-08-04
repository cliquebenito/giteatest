package pull

import (
	gocontext "context"
	"fmt"

	"code.gitea.io/gitea/models"
	"code.gitea.io/gitea/models/default_reviewers"
	"code.gitea.io/gitea/models/issues"
	"code.gitea.io/gitea/models/review_settings"
	"code.gitea.io/gitea/models/review_settings/review_settings_db"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/log"
)

type reviewSettingsManager struct {
	defaultReviewersDB
	reviewSettingsDB
}

func NewReviewSettings(defaultReviewersDB defaultReviewersDB, reviewSettingsDB reviewSettingsDB) *reviewSettingsManager {
	return &reviewSettingsManager{defaultReviewersDB: defaultReviewersDB, reviewSettingsDB: reviewSettingsDB}
}

// //go:generate mockery --name=defaultReviewersDB --exported
type defaultReviewersDB interface {
	GetDefaultReviewers(ctx gocontext.Context, settingID int64) ([]*default_reviewers.DefaultReviewers, error)
}

// //go:generate mockery --name=reviewSettingsDB --exported
type reviewSettingsDB interface {
	GetReviewSettings(_ gocontext.Context, repoID int64) ([]*review_settings.ReviewSettings, error)
}

type RequiredReviewCondition struct {
	BranchName       string
	RequiredApproves int
	Approved         int
}

var GetApprovesForDefaultReviewer = func(ctx gocontext.Context, reviewSetting *review_settings.ReviewSettings, dr *default_reviewers.DefaultReviewers, pr *issues.PullRequest) int {
	return issues.GetApprovesForDefaultReviewer(ctx, reviewSetting, dr, pr)
}

func (r *reviewSettingsManager) GetRequiredReviewConditions(ctx gocontext.Context, repoID int64, pr *issues.PullRequest) ([]*RequiredReviewCondition, error) {
	conditions := make([]*RequiredReviewCondition, 0)
	reviewSettings, err := r.GetMatchedReviewSetting(ctx, repoID, pr.BaseBranch)
	if err != nil && !review_settings_db.IsErrReviewSettingsDoesntExistsError(err) {
		log.Error("Error has occurred while getting review settings. Error: %v", err)
		return nil, fmt.Errorf("get review settings: %w", err)
	}
	for _, rs := range reviewSettings {
		defaultReviewers, err := r.GetDefaultReviewers(ctx, rs.ID)
		if err != nil {
			log.Error("Error has occurred while getting default reviewers. Error: %v", err)
			return nil, fmt.Errorf("get default reviewers: %w", err)
		}
		for _, dr := range defaultReviewers {
			cond := &RequiredReviewCondition{
				BranchName:       rs.RuleName,
				RequiredApproves: int(dr.RequiredApprovals),
			}
			cond.Approved = GetApprovesForDefaultReviewer(ctx, rs, dr, pr)
			if cond.Approved < cond.RequiredApproves {
				conditions = append(conditions, cond)
			}
		}
	}
	return conditions, nil
}

// GetReviewersForPullRequest возвращает id ревьюверов в соответствии с review settings
func (r *reviewSettingsManager) GetReviewersForPullRequest(ctx gocontext.Context, repoId int64, pr *issues.PullRequest) ([]int64, error) {
	reviewSettings, err := r.GetMatchedReviewSetting(ctx, repoId, pr.BaseBranch)
	if err != nil && !review_settings_db.IsErrReviewSettingsDoesntExistsError(err) {
		log.Error("Error has occurred while getting review settings. Error: %v", err)
		return nil, fmt.Errorf("get review settings: %w", err)
	}
	if reviewSettings == nil {
		return nil, nil
	}
	reviewersId := make(map[int64]struct{})
	reviewers := make([]int64, 0)

	for _, rs := range reviewSettings {
		defaultReviewers, err := r.GetDefaultReviewers(ctx, rs.ID)
		if err != nil {
			log.Error("Error has occurred while getting default reviewers. Error: %v", err)
			return nil, fmt.Errorf("get default reviewers: %w", err)
		}
		for _, dr := range defaultReviewers {
			for _, usrId := range dr.DefaultReviewersList {
				reviewersId[usrId] = struct{}{}
			}
		}
	}

	for id, _ := range reviewersId {
		reviewers = append(reviewers, id)
	}
	return reviewers, nil
}

func (r reviewSettingsManager) GetMatchedReviewSetting(ctx gocontext.Context, repoID int64, branchName string) ([]*review_settings.ReviewSettings, error) {
	repoReviewSettings, err := r.GetReviewSettings(ctx, repoID)
	reviewSettings := make([]*review_settings.ReviewSettings, 0)
	if err != nil && !review_settings_db.IsErrReviewSettingsDoesntExistsError(err) {
		log.Error("Error has occurred while getting review settings. Error: %v", err)
		return nil, fmt.Errorf("get review settings: %w", err)
	}
	for _, rs := range repoReviewSettings {
		if rs.Match(branchName) {
			reviewSettings = append(reviewSettings, rs)
		}
	}
	return reviewSettings, nil
}

func (r reviewSettingsManager) CheckReviewSettingsProtections(ctx gocontext.Context, pr *issues.PullRequest, skipProtectedFilesCheck bool) (err error) {
	if err = pr.LoadBaseRepo(ctx); err != nil {
		return fmt.Errorf("LoadBaseRepo: %w", err)
	}

	reviewSettings, err := r.GetMatchedReviewSetting(ctx, pr.BaseRepoID, pr.BaseBranch)
	if err != nil {
		return fmt.Errorf("get matched review settings: %v", err)
	}
	if reviewSettings == nil {
		return nil
	}

	isPass, err := IsPullCommitStatusPass(ctx, pr)
	if err != nil {
		return err
	}
	if !isPass {
		return models.ErrDisallowedToMerge{
			Reason: "Not all required status checks successful",
		}
	}

	for _, rs := range reviewSettings {

		defaultReviewers, err := r.GetDefaultReviewers(ctx, rs.ID)
		if err != nil {
			log.Error("Error has occurred while getting default reviewers. Error: %v", err)
			return fmt.Errorf("get default reviewers: %w", err)
		}

		if !issues.HasEnoughApprovals(ctx, rs, defaultReviewers, pr) {
			return models.ErrDisallowedToMerge{
				Reason: "Does not have enough approvals",
			}
		}
		if issues.MergeBlockedByRejectedReview(ctx, rs, pr) {
			return models.ErrDisallowedToMerge{
				Reason: "There are requested changes",
			}
		}
		if issues.MergeBlockedByOfficialReviewRequests(ctx, rs, pr) {
			return models.ErrDisallowedToMerge{
				Reason: "There are official review requests",
			}
		}

		if issues.MergeBlockedByOutdatedBranch(rs, pr) {
			return models.ErrDisallowedToMerge{
				Reason: "The head branch is behind the base branch",
			}
		}
	}

	if skipProtectedFilesCheck {
		return nil
	}

	return nil
}

func (r reviewSettingsManager) IsUserAllowedToMerge(ctx gocontext.Context, pr *issues.PullRequest, user *user_model.User) (bool, error) {
	if user == nil {
		return false, nil
	}
	reviewSettings, err := r.GetMatchedReviewSetting(ctx, pr.BaseRepoID, pr.BaseBranch)
	if err != nil {
		return false, fmt.Errorf("get matched review settings: %v", err)
	}
	if reviewSettings == nil {
		return true, nil
	}

	for _, rs := range reviewSettings {
		if !rs.IsUserMergeWhitelisted(user.ID) {
			return false, nil
		}
	}
	return true, nil
}
