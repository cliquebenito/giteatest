package response

// RepoListResults результаты поиска репозитория
type RepoListResults struct {
	Total int64         `json:"total"`
	Data  []*Repository `json:"data"`
}
