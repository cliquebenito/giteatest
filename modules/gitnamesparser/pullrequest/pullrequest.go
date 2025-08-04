package pullrequest

import (
	"errors"
	"fmt"
	"regexp"

	"code.gitea.io/gitea/models/gitnames"
	"code.gitea.io/gitea/modules/gitnamesparser"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/routers/private/pull_request_reader"
)

type pullRequestLinksParser struct {
}

func NewParser() pullRequestLinksParser {
	return pullRequestLinksParser{}
}

// Parse метод ищет коды юнитов TaskTracker в названии ветки, МРа, коммитов
func (p pullRequestLinksParser) Parse(header pull_request_reader.PullRequestHeader) (gitnames.PullRequestLinks, error) {
	gatherCommitsLinks := func(prl *gitnames.PullRequestLinks) error {
		commitLinks, err := p.gatherCommitsLinks(header.CommitNames)
		if err != nil {
			return err
		}

		prl.CommitsLinks = commitLinks

		return nil
	}

	gatherBranchLinks := func(prl *gitnames.PullRequestLinks) error {
		branchLinks, err := p.gatherBranchLinks(header.BranchName)
		if err != nil {
			return err
		}

		prl.BranchLinks = branchLinks

		return nil
	}

	pullRequestLinks, err := p.gatherPullRequestsLinks(header)
	if err != nil {
		if emptyErr := new(gitnamesparser.UnitCodeNotFoundError); errors.As(err, &emptyErr) {
			if err = gatherBranchLinks(&pullRequestLinks); err != nil {
				log.Debug("unit_linker: try to parse branch: %v: err: %v", header.PullRequestName, err)
			}

			if err = gatherCommitsLinks(&pullRequestLinks); err != nil {
				log.Debug("unit_linker: try to parse commits: %v: err: %v", header.PullRequestName, err)
			}
		} else {
			return gitnames.PullRequestLinks{}, fmt.Errorf("pull request links: %w", err)
		}
	}

	if err = gatherBranchLinks(&pullRequestLinks); err != nil {
		if emptyErr := new(gitnamesparser.UnitCodeNotFoundError); errors.As(err, &emptyErr) {
			log.Debug("unit_linker: try to parse branch: %v: err: %v", header.PullRequestName, err)
		}
	}

	if header.CommitNames == nil || len(header.CommitNames) == 0 {
		return pullRequestLinks, nil
	}

	if err = gatherCommitsLinks(&pullRequestLinks); err != nil {
		if emptyErr := new(gitnamesparser.UnitCodeNotFoundError); errors.As(err, &emptyErr) {
			log.Debug("unit_linker: try to parse commits: %v: err: %v", header.PullRequestName, err)
		}
	}

	if pullRequestLinks.IsEmpty() {
		log.Debug("unit_linker (pull requests): Nothing to link")
	}

	return pullRequestLinks, nil
}

var pullRequestCodeRE = regexp.MustCompile("[A-Z_0-9]{1,30}-[0-9]{1,30}")
var branchCodeRE = regexp.MustCompile("[^/_a-zа-я-][A-Z_0-9]{1,30}-[0-9]{1,30}")

func (p pullRequestLinksParser) gatherPullRequestsLinks(header pull_request_reader.PullRequestHeader) (gitnames.PullRequestLinks, error) {
	pullRequestCodes, prDescription, err :=
		gitnamesparser.ParseCodesAndDescription(header.PullRequestName, pullRequestCodeRE)
	if err != nil {
		return gitnames.PullRequestLinks{}, fmt.Errorf("pull request name: %w", err)
	}

	pullRequest := gitnames.PullRequestLinks{
		Base: gitnames.Base{Description: prDescription},
	}

	for _, code := range pullRequestCodes {
		pullRequest.Base.LinkedUnits = append(
			pullRequest.Base.LinkedUnits, gitnames.UnitCode{Code: code},
		)
	}

	return pullRequest, nil
}

func (p pullRequestLinksParser) gatherBranchLinks(branchName string) (gitnames.BranchLinks, error) {
	branchCodes, branchDescription, err :=
		gitnamesparser.ParseCodesAndDescription(branchName, branchCodeRE)
	if err != nil {
		return gitnames.BranchLinks{}, fmt.Errorf("branch name: %w", err)
	}

	branchLinks := gitnames.BranchLinks{
		Base: gitnames.Base{Description: branchDescription},
	}

	for _, code := range branchCodes {
		branchLinks.Base.LinkedUnits = append(branchLinks.Base.LinkedUnits, gitnames.UnitCode{Code: code})
	}

	return branchLinks, nil
}

func (p pullRequestLinksParser) gatherCommitsLinks(commitNames []string) ([]gitnames.CommitLinks, error) {
	if commitNames == nil || len(commitNames) == 0 {
		return nil, gitnamesparser.NewEmptyCommitLinksError()
	}

	var commitsLinks []gitnames.CommitLinks
	for _, commitName := range commitNames {
		commitCodes, commitDescription, err := gitnamesparser.ParseCodesAndDescription(commitName, pullRequestCodeRE)
		if err != nil {
			continue
		}

		commitLinks := gitnames.CommitLinks{
			Base: gitnames.Base{Description: commitDescription},
		}

		for _, code := range commitCodes {
			commitLinks.LinkedUnits = append(
				commitLinks.LinkedUnits, gitnames.UnitCode{Code: code},
			)
		}

		commitsLinks = append(commitsLinks, commitLinks)
	}

	return commitsLinks, nil
}
