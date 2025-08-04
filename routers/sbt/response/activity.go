package response

import (
	issues_model "code.gitea.io/gitea/models/issues"
	"time"
)

// Action активность пользователя
type Action struct {
	ID             int64                 `json:"id"`
	OpType         string                `json:"type"`
	UserID         int64                 `json:"userId"`
	UserName       string                `json:"userName"`
	RepoID         int64                 `json:"repoId"`
	RepoName       string                `json:"repoName"`
	RepoOwnerName  string                `json:"repoOwnerName"`
	Comment        *issues_model.Comment `json:"comment"`
	IsDeleted      bool                  `json:"isDeleted"`
	RefName        string                `json:"refName"`
	IsPrivate      bool                  `json:"isPrivate"`
	Content        string                `json:"content"`
	AdditionalInfo string                `json:"additionalInfo"`
	Created        time.Time             `json:"created"`
}

// ActionListResults список активностей пользователя с общим количеством
type ActionListResults struct {
	Total int64     `json:"total"`
	Data  []*Action `json:"data"`
}
