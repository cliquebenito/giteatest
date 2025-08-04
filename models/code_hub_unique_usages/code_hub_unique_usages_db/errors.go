package code_hub_unique_usages_db

import "fmt"

type RepoNotFoundError struct {
	RepoID int64
}

func NewRepoNotFoundError(repoID int64) *RepoNotFoundError {
	return &RepoNotFoundError{RepoID: repoID}
}

func (e *RepoNotFoundError) Error() string {
	return fmt.Sprintf("repo '%d' not found", e.RepoID)
}
