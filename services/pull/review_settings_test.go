package pull

import (
	"context"
	"errors"
	"testing"

	"code.gitea.io/gitea/models/default_reviewers"
	"code.gitea.io/gitea/models/issues"
	"code.gitea.io/gitea/models/review_settings"
	"code.gitea.io/gitea/services/pull/mocks"
	"github.com/stretchr/testify/assert"
)

func stubGetApprovesForDefaultReviewer(_ context.Context, _ *review_settings.ReviewSettings, _ *default_reviewers.DefaultReviewers, _ *issues.PullRequest) int {
	return 1
}

func TestGetRequiredReviewConditions_Success(t *testing.T) {
	ctx := context.Background()
	repoID := int64(1)
	pr := &issues.PullRequest{BaseBranch: "main"}

	// stub function
	original := issues.GetApprovesForDefaultReviewer
	GetApprovesForDefaultReviewer = stubGetApprovesForDefaultReviewer
	defer func() { GetApprovesForDefaultReviewer = original }()

	rs := &review_settings.ReviewSettings{ID: 10, RuleName: "main"}
	dr := &default_reviewers.DefaultReviewers{RequiredApprovals: 2}

	reviewDB := new(mocks.DefaultReviewersDB)
	defaultDB := new(mocks.ReviewSettingsDB)

	defaultDB.On("GetReviewSettings", ctx, repoID).Return([]*review_settings.ReviewSettings{rs}, nil)
	reviewDB.On("GetDefaultReviewers", ctx, int64(10)).Return([]*default_reviewers.DefaultReviewers{dr}, nil)

	svc := NewReviewSettings(reviewDB, defaultDB)
	conds, err := svc.GetRequiredReviewConditions(ctx, repoID, pr)

	assert.NoError(t, err)
	assert.Len(t, conds, 1)
	assert.Equal(t, "main", conds[0].BranchName)
	assert.Equal(t, 2, conds[0].RequiredApproves)
	assert.Equal(t, 1, conds[0].Approved)
}

func TestGetRequiredReviewConditions_ReviewSettingsError(t *testing.T) {
	ctx := context.Background()
	repoID := int64(1)
	pr := &issues.PullRequest{BaseBranch: "main"}

	expectedErr := errors.New("db error")

	reviewDB := new(mocks.DefaultReviewersDB)
	defaultDB := new(mocks.ReviewSettingsDB)

	defaultDB.On("GetReviewSettings", ctx, repoID).Return(nil, expectedErr)

	svc := NewReviewSettings(reviewDB, defaultDB)
	_, err := svc.GetRequiredReviewConditions(ctx, repoID, pr)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get review settings")
}

func TestGetRequiredReviewConditions_DefaultReviewersError(t *testing.T) {
	ctx := context.Background()
	repoID := int64(1)
	pr := &issues.PullRequest{BaseBranch: "main"}

	original := issues.GetApprovesForDefaultReviewer
	GetApprovesForDefaultReviewer = stubGetApprovesForDefaultReviewer
	defer func() { GetApprovesForDefaultReviewer = original }()

	rs := &review_settings.ReviewSettings{ID: 10, RuleName: "main"}

	reviewDB := new(mocks.DefaultReviewersDB)
	defaultDB := new(mocks.ReviewSettingsDB)

	defaultDB.On("GetReviewSettings", ctx, repoID).Return([]*review_settings.ReviewSettings{rs}, nil)
	reviewDB.On("GetDefaultReviewers", ctx, int64(10)).Return(nil, errors.New("failed to get reviewers"))

	svc := NewReviewSettings(reviewDB, defaultDB)
	_, err := svc.GetRequiredReviewConditions(ctx, repoID, pr)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get default reviewers")
}

func TestGetReviewersForPullRequest(t *testing.T) {
	ctx := context.Background()
	repoID := int64(1)

	pr := &issues.PullRequest{
		BaseBranch: "main",
	}

	t.Run("successfully returns unique reviewer IDs", func(t *testing.T) {
		mockReviewSettingsDB := new(mocks.ReviewSettingsDB)
		mockDefaultReviewersDB := new(mocks.DefaultReviewersDB)

		rs := &review_settings.ReviewSettings{
			ID:       10,
			RuleName: "main",
		}

		mockReviewSettingsDB.
			On("GetReviewSettings", ctx, repoID).
			Return([]*review_settings.ReviewSettings{rs}, nil)

		mockDefaultReviewersDB.
			On("GetDefaultReviewers", ctx, rs.ID).
			Return([]*default_reviewers.DefaultReviewers{
				{DefaultReviewersList: []int64{1, 2}},
				{DefaultReviewersList: []int64{2, 3}},
			}, nil)

		svc := NewReviewSettings(mockDefaultReviewersDB, mockReviewSettingsDB)
		reviewers, err := svc.GetReviewersForPullRequest(ctx, repoID, pr)

		assert.NoError(t, err)
		assert.ElementsMatch(t, []int64{1, 2, 3}, reviewers)
	})

	t.Run("no review settings match", func(t *testing.T) {
		mockReviewSettingsDB := new(mocks.ReviewSettingsDB)
		mockDefaultReviewersDB := new(mocks.DefaultReviewersDB)

		rs := &review_settings.ReviewSettings{
			ID:       20,
			RuleName: "feature/*",
		}

		mockReviewSettingsDB.
			On("GetReviewSettings", ctx, repoID).
			Return([]*review_settings.ReviewSettings{rs}, nil)

		svc := NewReviewSettings(mockDefaultReviewersDB, mockReviewSettingsDB)
		reviewers, err := svc.GetReviewersForPullRequest(ctx, repoID, pr)

		assert.NoError(t, err)
		assert.ElementsMatch(t, []int64{}, reviewers)
	})

	t.Run("review settings db returns unexpected error", func(t *testing.T) {
		mockReviewSettingsDB := new(mocks.ReviewSettingsDB)
		mockDefaultReviewersDB := new(mocks.DefaultReviewersDB)

		mockReviewSettingsDB.
			On("GetReviewSettings", ctx, repoID).
			Return(nil, errors.New("db error"))

		svc := NewReviewSettings(mockDefaultReviewersDB, mockReviewSettingsDB)
		reviewers, err := svc.GetReviewersForPullRequest(ctx, repoID, pr)

		assert.Error(t, err)
		assert.Nil(t, reviewers)
	})

	t.Run("default reviewers DB returns error", func(t *testing.T) {
		mockReviewSettingsDB := new(mocks.ReviewSettingsDB)
		mockDefaultReviewersDB := new(mocks.DefaultReviewersDB)

		rs := &review_settings.ReviewSettings{
			ID:       30,
			RuleName: "main",
		}

		mockReviewSettingsDB.
			On("GetReviewSettings", ctx, repoID).
			Return([]*review_settings.ReviewSettings{rs}, nil)

		mockDefaultReviewersDB.
			On("GetDefaultReviewers", ctx, rs.ID).
			Return(nil, errors.New("error loading reviewers"))

		svc := NewReviewSettings(mockDefaultReviewersDB, mockReviewSettingsDB)
		reviewers, err := svc.GetReviewersForPullRequest(ctx, repoID, pr)

		assert.Error(t, err)
		assert.Nil(t, reviewers)
	})
}
