package response

import "time"

// StateType состояние Issue
type StateType string

const (
	// StateOpen пулл реквест открыт
	StateOpen StateType = "open"
	// StateClosed пулл реквест закрыт
	StateClosed StateType = "closed"
)

type Issue struct {
	ID               int64         `json:"id"`
	Index            int64         `json:"number"`
	Poster           *User         `json:"user"`
	OriginalAuthor   string        `json:"original_author"`
	OriginalAuthorID int64         `json:"original_author_id"`
	Title            string        `json:"title"`
	Body             string        `json:"body"`
	Ref              string        `json:"ref"`
	Attachments      []*Attachment `json:"assets"`
	Labels           []*Label      `json:"labels"`
	Milestone        *Milestone    `json:"milestone"`
	Assignees        []*User       `json:"assignees"`
	// Whether the issue is open or closed
	//
	// type: string
	// enum: open,closed
	State    StateType  `json:"state"`
	IsLocked bool       `json:"is_locked"`
	Created  time.Time  `json:"created_at"`
	Updated  time.Time  `json:"updated_at"`
	Closed   *time.Time `json:"closed_at"`
	Deadline *time.Time `json:"due_date"`

	PullRequest *PullRequestMeta `json:"pull_request"`
	Repo        *RepositoryMeta  `json:"repository"`
}

// PullRequestMeta метаданные пулл реквеста
type PullRequestMeta struct {
	HasMerged bool       `json:"merged"`
	Merged    *time.Time `json:"merged_at"`
}

// RepositoryMeta метаданные репозитория
type RepositoryMeta struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Owner    string `json:"owner"`
	FullName string `json:"full_name"`
}
