package convert

import (
	issuesModel "code.gitea.io/gitea/models/issues"
	repoModel "code.gitea.io/gitea/models/repo"
	userModel "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/routers/sbt/logger"
	"code.gitea.io/gitea/routers/sbt/response"
	"context"
	"fmt"
	"net/url"
	"strings"
)

// ToIssue конвертирует Issue в response.Issue
func ToIssue(ctx context.Context, issue *issuesModel.Issue, log logger.Logger) *response.Issue {
	if err := issue.LoadLabels(ctx); err != nil {
		return &response.Issue{}
	}
	if err := issue.LoadPoster(ctx); err != nil {
		return &response.Issue{}
	}
	if err := issue.LoadRepo(ctx); err != nil {
		return &response.Issue{}
	}

	responseIssue := &response.Issue{
		ID:          issue.ID,
		Index:       issue.Index,
		Poster:      ToUser(ctx, issue.Poster, nil),
		Title:       issue.Title,
		Body:        issue.Content,
		Attachments: ToAttachments(issue.Attachments),
		Ref:         issue.Ref,
		State:       GetIssueState(issue),
		IsLocked:    issue.IsLocked,
		Created:     issue.CreatedUnix.AsTime(),
		Updated:     issue.UpdatedUnix.AsTime(),
	}

	if issue.Repo != nil {
		if err := issue.Repo.LoadOwner(ctx); err != nil {
			return &response.Issue{}
		}
		responseIssue.Labels = ToLabelList(issue.Labels, issue.Repo, issue.Repo.Owner, log)
		responseIssue.Repo = &response.RepositoryMeta{
			ID:       issue.Repo.ID,
			Name:     issue.Repo.Name,
			Owner:    issue.Repo.OwnerName,
			FullName: issue.Repo.FullName(),
		}
	}

	if issue.ClosedUnix != 0 {
		responseIssue.Closed = issue.ClosedUnix.AsTimePtr()
	}

	if err := issue.LoadMilestone(ctx); err != nil {
		return &response.Issue{}
	}
	if issue.Milestone != nil {
		responseIssue.Milestone = ToMilestone(issue.Milestone)
	}

	if err := issue.LoadAssignees(ctx); err != nil {
		return &response.Issue{}
	}
	if len(issue.Assignees) > 0 {
		for _, assignee := range issue.Assignees {
			responseIssue.Assignees = append(responseIssue.Assignees, ToUser(ctx, assignee, nil))
		}
	}
	if issue.IsPull {
		if err := issue.LoadPullRequest(ctx); err != nil {
			return &response.Issue{}
		}
		if issue.PullRequest != nil {
			responseIssue.PullRequest = &response.PullRequestMeta{
				HasMerged: issue.PullRequest.HasMerged,
			}
			if issue.PullRequest.HasMerged {
				responseIssue.PullRequest.Merged = issue.PullRequest.MergedUnix.AsTimePtr()
			}
		}
	}
	if issue.DeadlineUnix != 0 {
		responseIssue.Deadline = issue.DeadlineUnix.AsTimePtr()
	}

	return responseIssue
}

// State returns string representation of issue status.
func GetIssueState(issue *issuesModel.Issue) response.StateType {
	if issue.IsClosed {
		return response.StateClosed
	}
	return response.StateOpen
}

// State returns string representation of milestone status.
func GetMilestoneState(m *issuesModel.Milestone) response.StateType {
	if m.IsClosed {
		return response.StateClosed
	}
	return response.StateOpen
}

// ToLabelList converts list of Label to API format
func ToLabelList(labels []*issuesModel.Label, repo *repoModel.Repository, org *userModel.User, log logger.Logger) []*response.Label {
	result := make([]*response.Label, len(labels))
	for i := range labels {
		result[i] = ToLabel(labels[i], repo, org, log)
	}
	return result
}

// ToLabel converts Label to API format
func ToLabel(label *issuesModel.Label, repo *repoModel.Repository, org *userModel.User, log logger.Logger) *response.Label {
	result := &response.Label{
		ID:          label.ID,
		Name:        label.Name,
		Exclusive:   label.Exclusive,
		Color:       strings.TrimLeft(label.Color, "#"),
		Description: label.Description,
	}

	// calculate URL
	if label.BelongsToRepo() && repo != nil {
		if repo != nil {
			result.URL = fmt.Sprintf("%s/labels/%d", repo.APIURL(), label.ID)
		} else {
			log.Error("ToLabel did not get repo to calculate url for label with id '%d'", label.ID)
		}
	} else { // BelongsToOrg
		if org != nil {
			result.URL = fmt.Sprintf("%sapi/v1/orgs/%s/labels/%d", setting.AppURL, url.PathEscape(org.Name), label.ID)
		} else {
			log.Error("ToLabel did not get org to calculate url for label with id '%d'", label.ID)
		}
	}

	return result
}

// ToAPIMilestone converts Milestone into API Format
func ToMilestone(m *issuesModel.Milestone) *response.Milestone {
	apiMilestone := &response.Milestone{
		ID:           m.ID,
		State:        GetMilestoneState(m),
		Title:        m.Name,
		Description:  m.Content,
		OpenIssues:   m.NumOpenIssues,
		ClosedIssues: m.NumClosedIssues,
		Created:      m.CreatedUnix.AsTime(),
		Updated:      m.UpdatedUnix.AsTimePtr(),
	}
	if m.IsClosed {
		apiMilestone.Closed = m.ClosedDateUnix.AsTimePtr()
	}
	if m.DeadlineUnix.Year() < 9999 {
		apiMilestone.Deadline = m.DeadlineUnix.AsTimePtr()
	}
	return apiMilestone
}
