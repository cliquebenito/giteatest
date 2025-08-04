package response

import (
	"code.gitea.io/gitea/models/organization"
	"code.gitea.io/sdk/gitea"
	"time"
)

// PullRequest пулл реквест
type PullRequest struct {
	ID        int64           `json:"id"`
	Index     int64           `json:"number"`
	Poster    *User           `json:"user"`
	Title     string          `json:"title"`
	Body      string          `json:"body"`
	Labels    []*Label        `json:"labels"`
	Milestone *Milestone      `json:"milestone"`
	Reviewers []*RepoReviewer `json:"reviewers"`
	State     StateType       `json:"state"`
	IsLocked  bool            `json:"is_locked"`
	Comments  int             `json:"comments"`

	Mergeable           bool       `json:"mergeable"`
	HasMerged           bool       `json:"merged"`
	Merged              *time.Time `json:"merged_at"`
	MergedCommitID      *string    `json:"merge_commit_sha"`
	MergedBy            *User      `json:"merged_by"`
	AllowMaintainerEdit bool       `json:"allow_maintainer_edit"`

	Base      *PRBranchInfo `json:"base"`
	Head      *PRBranchInfo `json:"head"`
	MergeBase string        `json:"merge_base"`

	Deadline *time.Time `json:"due_date"`

	Created *time.Time `json:"created_at"`
	Updated *time.Time `json:"updated_at"`
	Closed  *time.Time `json:"closed_at"`
}

// PRBranchInfo ветки пулл реквеста
type PRBranchInfo struct {
	Name       string      `json:"label"`
	Ref        string      `json:"ref"`
	Sha        string      `json:"sha"`
	RepoID     int64       `json:"repo_id"`
	Repository *Repository `json:"repo"`
}

// RepoReviewer структура ревьюера
type RepoReviewer struct {
	ReviewerUser *User                 `json:"reviewer,omitempty"`
	ReviewerTeam *organization.Team    `json:"reviewerTeam,omitempty"`
	ReviewState  gitea.ReviewStateType `json:"reviewState"`
}
