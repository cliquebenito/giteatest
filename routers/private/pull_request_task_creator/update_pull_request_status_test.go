package pull_request_task_creator

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"code.gitea.io/gitea/models/pull_request_sender"
	mocks_tasker "code.gitea.io/gitea/routers/private/pull_request_task_creator/mocks"
)

var (
	testCtx = mock.AnythingOfType("context.backgroundCtx")
)

func TestPullRequestSender_UpdateStatusOfPullRequest(t *testing.T) {
	ctx := context.Background()
	taskDb := mocks_tasker.NewTaskTrackerDB(t)

	mockPullRequest := NewPullRequestTaskCreator(
		taskDb,
	)

	testCases := []struct {
		name        string
		opts        pull_request_sender.UpdatePullRequestStatusOptions
		IssueID     int64
		expectedErr error
	}{
		{
			name: "Success case for a status 'open' pull request",
			opts: pull_request_sender.UpdatePullRequestStatusOptions{
				FromUnitID:        1,
				UserName:          "test_user",
				PullRequestStatus: pull_request_sender.PRStatusOpen,
			},
			IssueID:     1,
			expectedErr: nil,
		},
		{
			name: "Success case for a status 'merge' pull request",
			opts: pull_request_sender.UpdatePullRequestStatusOptions{
				FromUnitID:        1,
				UserName:          "test_user",
				PullRequestStatus: pull_request_sender.PRStatusMerged,
			},
			IssueID:     1,
			expectedErr: nil,
		},
		{
			name: "Success case for a status 'closed' pull request",
			opts: pull_request_sender.UpdatePullRequestStatusOptions{
				FromUnitID:        1,
				UserName:          "test_user",
				PullRequestStatus: pull_request_sender.PRStatusClosed,
			},
			IssueID:     1,
			expectedErr: nil,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			taskDb.On("PullRequestStatusUpdate", testCtx, tt.opts).Return(tt.expectedErr)
			req := PullRequestUpdateStatus{
				UserName:          tt.opts.UserName,
				PullRequestID:     tt.opts.FromUnitID,
				PullRequestStatus: tt.opts.PullRequestStatus,
			}
			err := mockPullRequest.UpdateStatusOfPullRequest(ctx, req)
			require.Equal(t, tt.expectedErr, err)
		})
	}
}
