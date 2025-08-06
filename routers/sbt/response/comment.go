package response

import (
	"time"
)

// Comment комментарий в пулл реквесте
type Comment struct {
	ID             int64         `json:"id"`
	Type           string        `json:"type"`
	Poster         *User         `json:"poster"`
	Body           string        `json:"body"`
	Attachments    []*Attachment `json:"attachments"`
	Patch          string        `json:"patch"`
	Reactions      []*Reaction   `json:"reactions"`
	TreePath       string        `json:"treePath"`
	Created        time.Time     `json:"createdAt"`
	Updated        time.Time     `json:"updatedAt"`
	EditCounts     int           `json:"editCounts"`
	IsOwner        bool          `json:"isOwner"`
	IsAuthor       bool          `json:"isAuthor"`
	IsCollaborator bool          `json:"isCollaborator"`
}

// Reaction реакция (смайлик) на комментарий
type Reaction struct {
	Content      string `json:"content"`
	UserId       int64  `json:"userId"`
	UserName     string `json:"userName"`
	UserFullName string `json:"userFullName"`
}

// CommentHistory история комментария
type CommentHistory struct {
	HistoryId    int64     `json:"historyId"`
	Action       string    `json:"action"`
	UserId       int64     `json:"userId"`
	UserName     string    `json:"userName"`
	UserFullName string    `json:"userFullName"`
	Updated      time.Time `json:"updatedAt"`
}

// CommentHistoryDetail детали изменений
type CommentHistoryDetail struct {
	Current       string `json:"current"`
	Previous      string `json:"previous"`
	PreviousId    int64  `json:"previousId"`
	Diff          []Diff `json:"diff"`
	CanSoftDelete bool   `json:"canSoftDelete"`
}

// Diff операция diff
type Diff struct {
	Type string `json:"type"`
	Text string `json:"text"`
}
