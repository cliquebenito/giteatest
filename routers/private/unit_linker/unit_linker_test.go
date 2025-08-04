//go:build !correct

package unit_linker

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"code.gitea.io/gitea/models/gitnames"
	"code.gitea.io/gitea/routers/private/pull_request_reader"
	"code.gitea.io/gitea/routers/private/task_tracker_client"
	"code.gitea.io/gitea/routers/private/unit_linker/mocks"
)

var (
	testCtx      = mock.AnythingOfType("context.backgroundCtx")
	testUnitLink = mock.AnythingOfType("unit_links.AllUnitLinks")
)

func TestUnitLinkerUseCase_LinkPullRequest(t *testing.T) {
	ctx := context.Background()
	branchParserMock := mocks.NewBranchHeaderParser(t)
	pullRequestParserMock := mocks.NewPullRequestHeaderParser(t)
	taskTrackerClientMock := mocks.NewTaskTrackerClient(t)
	unitLinkerDBMock := mocks.NewUnitLinkDB(t)
	pullRequestReaderDBMock := mocks.NewPullRequestReader(t)

	const withValidation = false

	mockedUnitLinker := NewUnitLinker(
		branchParserMock,
		pullRequestParserMock,
		unitLinkerDBMock,
		pullRequestReaderDBMock,
		taskTrackerClientMock,
		withValidation,
	)

	readerReturnValue := pull_request_reader.PullRequestHeader{
		PullRequestName: "[GITRU-1] The first pull request ever",
		BranchName:      "feature/GITRU-13",
		CommitNames:     []string{"Init commit", "GITRU-2 Another day in paradise"},
	}

	parserReturnValue := gitnames.PullRequestLinks{
		Base:         gitnames.Base{LinkedUnits: []gitnames.UnitCode{{Code: "GITRU-1"}}},
		CommitsLinks: []gitnames.CommitLinks{{gitnames.Base{Description: "", LinkedUnits: []gitnames.UnitCode{{Code: "GITRU-2"}}}}},
	}

	request := PullRequestLinkRequest{BranchName: "feature/GITRU-13", PullRequestID: 1}

	pullRequestReaderDBMock.
		On("ReadByID", testCtx, request.PullRequestID, pull_request_reader.MergedPullRequestStatus).
		Return(readerReturnValue, nil)

	pullRequestParserMock.
		On("Parse", readerReturnValue).
		Return(parserReturnValue, nil)

	unitLinkerDBMock.
		On("UpdateLinks", testCtx, request.PullRequestID, testUnitLink).
		Return(nil)

	err := mockedUnitLinker.LinkPullRequest(ctx, request)
	require.NoError(t, err)
}

func TestUnitLinkerUseCase_LinkPullRequest_with_validation(t *testing.T) {
	ctx := context.Background()
	branchParserMock := mocks.NewBranchHeaderParser(t)
	pullRequestParserMock := mocks.NewPullRequestHeaderParser(t)
	taskTrackerClientMock := mocks.NewTaskTrackerClient(t)
	unitLinkerDBMock := mocks.NewUnitLinkDB(t)
	pullRequestReaderDBMock := mocks.NewPullRequestReader(t)

	const withValidation = true

	mockedUnitLinker := NewUnitLinker(
		branchParserMock,
		pullRequestParserMock,
		unitLinkerDBMock,
		pullRequestReaderDBMock,
		taskTrackerClientMock,
		withValidation,
	)

	readerReturnValue := pull_request_reader.PullRequestHeader{
		PullRequestName: "[GITRU-1] The first pull request ever",
		BranchName:      "feature/GITRU-13",
		CommitNames:     []string{"Init commit", "GITRU-2 Another day in paradise"},
	}

	parserReturnValue := gitnames.PullRequestLinks{
		Base:         gitnames.Base{LinkedUnits: []gitnames.UnitCode{{Code: "GITRU-1"}}},
		CommitsLinks: []gitnames.CommitLinks{{gitnames.Base{Description: "", LinkedUnits: []gitnames.UnitCode{{Code: "GITRU-2"}}}}},
	}

	request := PullRequestLinkRequest{BranchName: "feature/GITRU-13", PullRequestID: 1}
	checkCodesRequest := []gitnames.UnitCode{{Code: "GITRU-1"}, {Code: "GITRU-2"}}
	checkCodesResponse := task_tracker_client.CheckCodesResponse{Units: []task_tracker_client.Unit{{Code: "GITRU-1", IsExists: true}, {Code: "GITRU-2", IsExists: true}}}

	pullRequestReaderDBMock.
		On("ReadByID", testCtx, request.PullRequestID, pull_request_reader.MergedPullRequestStatus).
		Return(readerReturnValue, nil)

	pullRequestParserMock.
		On("Parse", readerReturnValue).
		Return(parserReturnValue, nil)

	taskTrackerClientMock.
		On("CheckCodes", testCtx, checkCodesRequest).
		Return(checkCodesResponse, nil)

	unitLinkerDBMock.
		On("UpdateLinks", testCtx, request.PullRequestID, testUnitLink).
		Return(nil)

	err := mockedUnitLinker.LinkPullRequest(ctx, request)
	require.NoError(t, err)
}
