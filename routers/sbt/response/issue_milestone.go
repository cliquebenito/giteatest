package response

import "time"

// Milestone коллекция задач объединенных в цель для выполнения в репозитории
type Milestone struct {
	ID           int64     `json:"id"`
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	State        StateType `json:"state"`
	OpenIssues   int       `json:"open_issues"`
	ClosedIssues int       `json:"closed_issues"`
	// swagger:strfmt date-time
	Created time.Time `json:"created_at"`
	// swagger:strfmt date-time
	Updated *time.Time `json:"updated_at"`
	// swagger:strfmt date-time
	Closed *time.Time `json:"closed_at"`
	// swagger:strfmt date-time
	Deadline *time.Time `json:"due_on"`
}
