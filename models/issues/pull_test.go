//go:build !correct

// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package issues_test

import (
	"context"
	"testing"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/default_reviewers"
	issues_model "code.gitea.io/gitea/models/issues"
	"code.gitea.io/gitea/models/review_settings"
	"code.gitea.io/gitea/models/unittest"

	"github.com/stretchr/testify/assert"
)

func TestPullRequest_LoadAttributes(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 1})
	assert.NoError(t, pr.LoadAttributes(db.DefaultContext))
	assert.NotNil(t, pr.Merger)
	assert.Equal(t, pr.MergerID, pr.Merger.ID)
}

func TestPullRequest_LoadIssue(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 1})
	assert.NoError(t, pr.LoadIssue(db.DefaultContext))
	assert.NotNil(t, pr.Issue)
	assert.Equal(t, int64(2), pr.Issue.ID)
	assert.NoError(t, pr.LoadIssue(db.DefaultContext))
	assert.NotNil(t, pr.Issue)
	assert.Equal(t, int64(2), pr.Issue.ID)
}

func TestPullRequest_LoadBaseRepo(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 1})
	assert.NoError(t, pr.LoadBaseRepo(db.DefaultContext))
	assert.NotNil(t, pr.BaseRepo)
	assert.Equal(t, pr.BaseRepoID, pr.BaseRepo.ID)
	assert.NoError(t, pr.LoadBaseRepo(db.DefaultContext))
	assert.NotNil(t, pr.BaseRepo)
	assert.Equal(t, pr.BaseRepoID, pr.BaseRepo.ID)
}

func TestPullRequest_LoadHeadRepo(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 1})
	assert.NoError(t, pr.LoadHeadRepo(db.DefaultContext))
	assert.NotNil(t, pr.HeadRepo)
	assert.Equal(t, pr.HeadRepoID, pr.HeadRepo.ID)
}

// TODO TestMerge

// TODO TestNewPullRequest

func TestPullRequestsNewest(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	prs, count, err := issues_model.PullRequests(1, &issues_model.PullRequestsOptions{
		ListOptions: db.ListOptions{
			Page: 1,
		},
		State:    "open",
		SortType: "newest",
		Labels:   []string{},
	})
	assert.NoError(t, err)
	assert.EqualValues(t, 3, count)
	if assert.Len(t, prs, 3) {
		assert.EqualValues(t, 5, prs[0].ID)
		assert.EqualValues(t, 2, prs[1].ID)
		assert.EqualValues(t, 1, prs[2].ID)
	}
}

func TestPullRequestsOldest(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	prs, count, err := issues_model.PullRequests(1, &issues_model.PullRequestsOptions{
		ListOptions: db.ListOptions{
			Page: 1,
		},
		State:    "open",
		SortType: "oldest",
		Labels:   []string{},
	})
	assert.NoError(t, err)
	assert.EqualValues(t, 3, count)
	if assert.Len(t, prs, 3) {
		assert.EqualValues(t, 1, prs[0].ID)
		assert.EqualValues(t, 2, prs[1].ID)
		assert.EqualValues(t, 5, prs[2].ID)
	}
}

func TestGetUnmergedPullRequest(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	pr, err := issues_model.GetUnmergedPullRequest(db.DefaultContext, 1, 1, "branch2", "master", issues_model.PullRequestFlowGithub)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), pr.ID)

	_, err = issues_model.GetUnmergedPullRequest(db.DefaultContext, 1, 9223372036854775807, "branch1", "master", issues_model.PullRequestFlowGithub)
	assert.Error(t, err)
	assert.True(t, issues_model.IsErrPullRequestNotExist(err))
}

func TestHasUnmergedPullRequestsByHeadInfo(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	exist, err := issues_model.HasUnmergedPullRequestsByHeadInfo(db.DefaultContext, 1, "branch2")
	assert.NoError(t, err)
	assert.True(t, exist)

	exist, err = issues_model.HasUnmergedPullRequestsByHeadInfo(db.DefaultContext, 1, "not_exist_branch")
	assert.NoError(t, err)
	assert.False(t, exist)
}

func TestGetUnmergedPullRequestsByHeadInfo(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	prs, err := issues_model.GetUnmergedPullRequestsByHeadInfo(1, "branch2")
	assert.NoError(t, err)
	assert.Len(t, prs, 1)
	for _, pr := range prs {
		assert.Equal(t, int64(1), pr.HeadRepoID)
		assert.Equal(t, "branch2", pr.HeadBranch)
	}
}

func TestGetUnmergedPullRequestsByBaseInfo(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	prs, err := issues_model.GetUnmergedPullRequestsByBaseInfo(1, "master")
	assert.NoError(t, err)
	assert.Len(t, prs, 1)
	pr := prs[0]
	assert.Equal(t, int64(2), pr.ID)
	assert.Equal(t, int64(1), pr.BaseRepoID)
	assert.Equal(t, "master", pr.BaseBranch)
}

func TestGetPullRequestByIndex(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	pr, err := issues_model.GetPullRequestByIndex(db.DefaultContext, 1, 2)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), pr.BaseRepoID)
	assert.Equal(t, int64(2), pr.Index)

	_, err = issues_model.GetPullRequestByIndex(db.DefaultContext, 9223372036854775807, 9223372036854775807)
	assert.Error(t, err)
	assert.True(t, issues_model.IsErrPullRequestNotExist(err))

	_, err = issues_model.GetPullRequestByIndex(db.DefaultContext, 1, 0)
	assert.Error(t, err)
	assert.True(t, issues_model.IsErrPullRequestNotExist(err))
}

func TestGetPullRequestByID(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	pr, err := issues_model.GetPullRequestByID(db.DefaultContext, 1)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), pr.ID)
	assert.Equal(t, int64(2), pr.IssueID)

	_, err = issues_model.GetPullRequestByID(db.DefaultContext, 9223372036854775807)
	assert.Error(t, err)
	assert.True(t, issues_model.IsErrPullRequestNotExist(err))
}

func TestGetPullRequestByIssueID(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	pr, err := issues_model.GetPullRequestByIssueID(db.DefaultContext, 2)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), pr.IssueID)

	_, err = issues_model.GetPullRequestByIssueID(db.DefaultContext, 9223372036854775807)
	assert.Error(t, err)
	assert.True(t, issues_model.IsErrPullRequestNotExist(err))
}

func TestPullRequest_Update(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 1})
	pr.BaseBranch = "baseBranch"
	pr.HeadBranch = "headBranch"
	pr.Update()

	pr = unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: pr.ID})
	assert.Equal(t, "baseBranch", pr.BaseBranch)
	assert.Equal(t, "headBranch", pr.HeadBranch)
	unittest.CheckConsistencyFor(t, pr)
}

func TestPullRequest_UpdateCols(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	pr := &issues_model.PullRequest{
		ID:         1,
		BaseBranch: "baseBranch",
		HeadBranch: "headBranch",
	}
	assert.NoError(t, pr.UpdateCols("head_branch"))

	pr = unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 1})
	assert.Equal(t, "master", pr.BaseBranch)
	assert.Equal(t, "headBranch", pr.HeadBranch)
	unittest.CheckConsistencyFor(t, pr)
}

func TestPullRequestList_LoadAttributes(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	prs := []*issues_model.PullRequest{
		unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 1}),
		unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 2}),
	}
	assert.NoError(t, issues_model.PullRequestList(prs).LoadAttributes())
	for _, pr := range prs {
		assert.NotNil(t, pr.Issue)
		assert.Equal(t, pr.IssueID, pr.Issue.ID)
	}

	assert.NoError(t, issues_model.PullRequestList([]*issues_model.PullRequest{}).LoadAttributes())
}

// TODO TestAddTestPullRequestTask

func TestPullRequest_IsWorkInProgress(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 2})
	pr.LoadIssue(db.DefaultContext)

	assert.False(t, pr.IsWorkInProgress())

	pr.Issue.Title = "WIP: " + pr.Issue.Title
	assert.True(t, pr.IsWorkInProgress())

	pr.Issue.Title = "[wip]: " + pr.Issue.Title
	assert.True(t, pr.IsWorkInProgress())
}

func TestPullRequest_GetWorkInProgressPrefixWorkInProgress(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 2})
	pr.LoadIssue(db.DefaultContext)

	assert.Empty(t, pr.GetWorkInProgressPrefix(db.DefaultContext))

	original := pr.Issue.Title
	pr.Issue.Title = "WIP: " + original
	assert.Equal(t, "WIP:", pr.GetWorkInProgressPrefix(db.DefaultContext))

	pr.Issue.Title = "[wip] " + original
	assert.Equal(t, "[wip]", pr.GetWorkInProgressPrefix(db.DefaultContext))
}

func TestDeleteOrphanedObjects(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	countBefore, err := db.GetEngine(db.DefaultContext).Count(&issues_model.PullRequest{})
	assert.NoError(t, err)

	_, err = db.GetEngine(db.DefaultContext).Insert(&issues_model.PullRequest{IssueID: 1000}, &issues_model.PullRequest{IssueID: 1001}, &issues_model.PullRequest{IssueID: 1003})
	assert.NoError(t, err)

	orphaned, err := db.CountOrphanedObjects(db.DefaultContext, "pull_request", "issue", "pull_request.issue_id=issue.id")
	assert.NoError(t, err)
	assert.EqualValues(t, 3, orphaned)

	err = db.DeleteOrphanedObjects(db.DefaultContext, "pull_request", "issue", "pull_request.issue_id=issue.id")
	assert.NoError(t, err)

	countAfter, err := db.GetEngine(db.DefaultContext).Count(&issues_model.PullRequest{})
	assert.NoError(t, err)
	assert.EqualValues(t, countBefore, countAfter)
}
func TestParseCodeOwnersLine(t *testing.T) {
	type CodeOwnerTest struct {
		Line   string
		Tokens []string
	}

	given := []CodeOwnerTest{
		{Line: "", Tokens: nil},
		{Line: "# comment", Tokens: []string{}},
		{Line: "!.* @user1", Tokens: []string{"!.*", "@user1"}},
		{Line: `.*\\.js @user2 #comment`, Tokens: []string{`.*\.js`, "@user2"}},
		{Line: `docs/(aws|google|azure)/[^/]*\\.(md|txt) @user3`, Tokens: []string{`docs/(aws|google|azure)/[^/]*\.(md|txt)`, "@user3"}},
		{Line: `\#path @user3`, Tokens: []string{`#path`, "@user3"}},
		{Line: `path\ with\ spaces/ @user3`, Tokens: []string{`path with spaces/`, "@user3"}},
	}

	for _, g := range given {
		tokens := issues_model.TokenizeCodeOwnersLine(g.Line)
		assert.Equal(t, g.Tokens, tokens, "Codeowners tokenizer failed")
	}
}

func TestHasEnoughApprovals_DisabledReviewers(t *testing.T) {
	ctx := context.Background()

	rs := &review_settings.ReviewSettings{EnableDefaultReviewers: false}
	dr := []*default_reviewers.DefaultReviewers{
		{RequiredApprovals: 1, DefaultReviewersList: []int64{1, 2}},
	}
	pr := &issues_model.PullRequest{}

	assert.True(t, issues_model.HasEnoughApprovals(ctx, rs, dr, pr))
}

func TestHasEnoughApprovals_NoApproversReturned(t *testing.T) {
	ctx := context.Background()

	rs := &review_settings.ReviewSettings{EnableDefaultReviewers: true}
	dr := []*default_reviewers.DefaultReviewers{
		{RequiredApprovals: 1, DefaultReviewersList: []int64{1, 2}},
	}
	pr := &issues_model.PullRequest{}

	original := issues_model.GetGrantedApprovalsIDs
	issues_model.GetGrantedApprovalsIDsFunc = func(_ context.Context, _ *review_settings.ReviewSettings, _ *issues_model.PullRequest) map[int64]struct{} {
		return nil
	}
	defer func() { issues_model.GetGrantedApprovalsIDsFunc = original }()

	assert.False(t, issues_model.HasEnoughApprovals(ctx, rs, dr, pr))
}

func TestHasEnoughApprovals_SufficientApprovals(t *testing.T) {
	ctx := context.Background()

	rs := &review_settings.ReviewSettings{EnableDefaultReviewers: true}
	dr := []*default_reviewers.DefaultReviewers{
		{RequiredApprovals: 2, DefaultReviewersList: []int64{1, 2, 3}},
	}
	pr := &issues_model.PullRequest{}

	original := issues_model.GetGrantedApprovalsIDs
	issues_model.GetGrantedApprovalsIDsFunc = func(_ context.Context, _ *review_settings.ReviewSettings, _ *issues_model.PullRequest) map[int64]struct{} {
		return map[int64]struct{}{1: {}, 2: {}}
	}
	defer func() { issues_model.GetGrantedApprovalsIDsFunc = original }()

	assert.True(t, issues_model.HasEnoughApprovals(ctx, rs, dr, pr))
}

func TestHasEnoughApprovals_InsufficientApprovals(t *testing.T) {
	ctx := context.Background()

	rs := &review_settings.ReviewSettings{EnableDefaultReviewers: true}
	dr := []*default_reviewers.DefaultReviewers{
		{RequiredApprovals: 2, DefaultReviewersList: []int64{1, 2}},
	}
	pr := &issues_model.PullRequest{}

	original := issues_model.GetGrantedApprovalsIDs
	issues_model.GetGrantedApprovalsIDsFunc = func(_ context.Context, _ *review_settings.ReviewSettings, _ *issues_model.PullRequest) map[int64]struct{} {
		return map[int64]struct{}{1: {}}
	}
	defer func() { issues_model.GetGrantedApprovalsIDsFunc = original }()

	assert.False(t, issues_model.HasEnoughApprovals(ctx, rs, dr, pr))
}

func TestHasEnoughApprovals_MultipleGroups_AllMustPass(t *testing.T) {
	ctx := context.Background()

	rs := &review_settings.ReviewSettings{EnableDefaultReviewers: true}
	dr := []*default_reviewers.DefaultReviewers{
		{RequiredApprovals: 1, DefaultReviewersList: []int64{1, 2}},
		{RequiredApprovals: 1, DefaultReviewersList: []int64{3, 4}},
	}
	pr := &issues_model.PullRequest{}

	original := issues_model.GetGrantedApprovalsIDs
	issues_model.GetGrantedApprovalsIDsFunc = func(_ context.Context, _ *review_settings.ReviewSettings, _ *issues_model.PullRequest) map[int64]struct{} {
		return map[int64]struct{}{1: {}, 3: {}}
	}
	defer func() { issues_model.GetGrantedApprovalsIDsFunc = original }()

	assert.True(t, issues_model.HasEnoughApprovals(ctx, rs, dr, pr))
}

func TestHasEnoughApprovals_MultipleGroups_OneFails(t *testing.T) {
	ctx := context.Background()

	rs := &review_settings.ReviewSettings{EnableDefaultReviewers: true}
	dr := []*default_reviewers.DefaultReviewers{
		{RequiredApprovals: 1, DefaultReviewersList: []int64{1, 2}},
		{RequiredApprovals: 2, DefaultReviewersList: []int64{3, 4}},
	}
	pr := &issues_model.PullRequest{}

	original := issues_model.GetGrantedApprovalsIDs
	issues_model.GetGrantedApprovalsIDsFunc = func(_ context.Context, _ *review_settings.ReviewSettings, _ *issues_model.PullRequest) map[int64]struct{} {
		return map[int64]struct{}{1: {}, 3: {}}
	}
	defer func() { issues_model.GetGrantedApprovalsIDsFunc = original }()

	assert.False(t, issues_model.HasEnoughApprovals(ctx, rs, dr, pr))
}
