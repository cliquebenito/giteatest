package response

import (
	"time"
)

/*
CommitListResult структура ответа на запрос списка коммитов с общим количеством коммитов
*/
type CommitListResult struct {
	Total          int64     `json:"total"`
	Data           []*Commit `json:"data"`
	BeforeCommitId string    `json:"beforeCommitId,omitempty"`
	AfterCommitId  string    `json:"afterCommitId,omitempty"`
}

// Commit структура ответа на запрос информации о коммите
type Commit struct {
	*CommitMeta
	BranchName     *string        `json:"branch,omitempty"`
	RepoCommit     *RepoCommit    `json:"commit"`
	Parents        []*CommitMeta  `json:"parents"`
	Stats          *CommitStats   `json:"stats,omitempty"`
	Files          []*CommitFiles `json:"files,omitempty"`
	BeforeCommitId *string        `json:"beforeCommitId,omitempty"`
}

// CommitMeta метаданные коммита
type CommitMeta struct {
	SHA string `json:"sha"`
	// swagger:strfmt date-time
	Created time.Time `json:"created"`
}

// RepoCommit содержит информацию о коммите согласно контексту репозитория
type RepoCommit struct {
	Author    *CommitUser `json:"author"`
	Committer *CommitUser `json:"committer"`
	Message   string      `json:"message"`
	Tree      *CommitMeta `json:"tree"`
}

// Identity Автор либо коммитер
type Identity struct {
	Name  string `json:"name" binding:"MaxSize(100)"`
	Email string `json:"email" binding:"MaxSize(254)"`
}

// CommitUser содержит информацию о пользователе согласно контексту коммита
type CommitUser struct {
	Identity
	Date string `json:"date"`
}

// CommitStats статистика об изменениях в коммите
type CommitStats struct {
	Total        int `json:"total"`
	Additions    int `json:"additions"`
	Deletions    int `json:"deletions"`
	FilesChanged int `json:"numFiles"`
}

type CommitFiles struct {
	File   string `json:"file"`
	Status int8   `json:"status"`
}
