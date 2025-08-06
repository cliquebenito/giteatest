package unit_linker

import (
	"context"
	"errors"
	"fmt"

	"code.gitea.io/gitea/models/gitnames"
	"code.gitea.io/gitea/models/unit_links"
	"code.gitea.io/gitea/routers/private/pull_request_reader"
	"code.gitea.io/gitea/routers/private/task_tracker_client"
)

// //go:generate mockery --name=branchHeaderParser --exported
type branchHeaderParser interface {
	Parse(branchName string) (gitnames.BranchLinks, error)
}

// //go:generate mockery --name=pullRequestHeaderParser --exported
type pullRequestHeaderParser interface {
	Parse(pull_request_reader.PullRequestHeader) (gitnames.PullRequestLinks, error)
}

// //go:generate mockery --name=unitLinkDB --exported
type unitLinkDB interface {
	UpdateLinks(
		ctx context.Context,
		fromUnitID int64,
		links unit_links.AllUnitLinks,
		userName, pullRequestURL string,
	) error
	RemoveLinks(
		ctx context.Context,
		fromUnitID int64,
		links unit_links.AllUnitLinks,
		userName, pullRequestURL string,
	) error
}

// //go:generate mockery --name=taskTrackerClient --exported
type taskTrackerClient interface {
	CheckCodes(ctx context.Context, codes []gitnames.UnitCode) (task_tracker_client.CheckCodesResponse, error)
}

// //go:generate mockery --name=pullRequestReader --exported
type pullRequestReader interface {
	ReadByID(
		ctx context.Context,
		id int64,
		status pull_request_reader.PullRequestStatus,
	) (pull_request_reader.PullRequestHeader, error)
}

// UnitLinker объект для создания связей
type UnitLinker struct {
	branchHeaderParser
	pullRequestHeaderParser
	unitLinkDB
	pullRequestReader
	taskTrackerClient
	withUnitValidation bool
}

// NewUnitLinker создает unit_linker
func NewUnitLinker(
	branchHeaderParser branchHeaderParser,
	pullRequestHeaderParser pullRequestHeaderParser,
	unitLinkDB unitLinkDB, pullRequestReader pullRequestReader,
	taskTrackerClient taskTrackerClient,
	WithUnitValidation bool,
) UnitLinker {
	return UnitLinker{
		unitLinkDB:         unitLinkDB,
		branchHeaderParser: branchHeaderParser,
		pullRequestReader:  pullRequestReader,
		taskTrackerClient:  taskTrackerClient,
		withUnitValidation: WithUnitValidation,

		pullRequestHeaderParser: pullRequestHeaderParser,
	}
}

// LinkPullRequest позволяет привязать МР к юнитам TaskTracker
func (u UnitLinker) LinkPullRequest(ctx context.Context, request PullRequestLinkRequest) error {
	if err := request.Validate(); err != nil {
		return fmt.Errorf("validate request: %w", err)
	}

	prID := request.PullRequestID
	prStatus := request.PullRequestStatus
	userName := request.UserName

	prHeader, err := u.pullRequestReader.ReadByID(ctx, prID, prStatus)
	if err != nil {
		return fmt.Errorf("read pull request header, id: '%d': %w", prID, err)
	}

	prHeader.BranchName = request.BranchName

	rawPRLinks, err := u.pullRequestHeaderParser.Parse(prHeader)
	if err != nil {
		return fmt.Errorf("parse pull request header, id: '%d': %w", prID, err)
	}

	links, err := u.getPullRequestLinks(ctx, prID, rawPRLinks)
	if err != nil {
		if handledErr := handleGetLinksErrors(err); handledErr != nil {
			return fmt.Errorf("get codes, id: '%d': %w", prID, err)
		}
	}

	if err = u.unitLinkDB.UpdateLinks(ctx, prID, links, userName, prHeader.PullRequestURL); err != nil {
		return fmt.Errorf("create link, id: '%d': %w", prID, err)
	}

	return nil
}

// UnlinkPullRequest позволяет отвязать МР от юнитов TaskTracker
func (u UnitLinker) UnlinkPullRequest(ctx context.Context, request PullRequestLinkRequest) error {
	if err := request.Validate(); err != nil {
		return fmt.Errorf("validate request: %w", err)
	}

	prID := request.PullRequestID
	prStatus := request.PullRequestStatus
	userName := request.UserName

	prHeader, err := u.pullRequestReader.ReadByID(ctx, prID, prStatus)
	if err != nil {
		return fmt.Errorf("read pull request header, id: '%d': %w", prID, err)
	}

	prHeader.BranchName = request.BranchName

	rawLinks, err := u.pullRequestHeaderParser.Parse(prHeader)
	if err != nil {
		return fmt.Errorf("parse pull request header, id: '%d': %w", prID, err)
	}

	links, err := u.getPullRequestLinks(ctx, prID, rawLinks)
	if err != nil {
		if handledErr := handleGetLinksErrors(err); handledErr != nil {
			return fmt.Errorf("get codes, id: '%d': %w", prID, err)
		}
	}

	if err = u.unitLinkDB.RemoveLinks(ctx, prID, links, userName, prHeader.PullRequestURL); err != nil {
		return fmt.Errorf("remove link, id: '%d': %w", prID, err)
	}

	return nil
}

func (u UnitLinker) getPullRequestLinks(ctx context.Context, prID int64, rawLinks gitnames.PullRequestLinks) (unit_links.AllUnitLinks, error) {
	codes, err := rawLinks.GetUniqCodes()
	if err != nil {
		return nil, fmt.Errorf("get codes: %w", err)
	}

	var links unit_links.AllUnitLinks

	if !u.withUnitValidation {
		for _, code := range codes {
			link := unit_links.UnitLinks{
				IsActive:     true,
				FromUnitID:   prID,
				ToUnitID:     code.Code,
				FromUnitType: unit_links.PullRequestFromUnitType,
			}
			links = append(links, link)
		}

		return links, nil
	}

	checkResponse, err := u.taskTrackerClient.CheckCodes(ctx, codes)
	if err != nil {
		return nil, fmt.Errorf("check pull request codes: %w", err)
	}

	var checkedLinks unit_links.AllUnitLinks
	for _, unit := range checkResponse.Units {
		if !unit.IsExists {
			continue
		}

		link := unit_links.UnitLinks{
			IsActive:     true,
			FromUnitID:   prID,
			ToUnitID:     unit.Code,
			FromUnitType: unit_links.PullRequestFromUnitType,
		}

		checkedLinks = append(checkedLinks, link)
	}

	return checkedLinks, nil
}

func handleGetLinksErrors(err error) error {
	if targetErr := new(gitnames.EmptyUnitCodesListError); errors.As(err, &targetErr) {
		return nil
	}

	return err
}
