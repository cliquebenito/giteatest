package response

// PullsList DTO для ответов на запрос списка ПРов репозитория
type PullsList struct {
	Total int             `json:"total"`
	Data  *[]*PullRequest `json:"data"`
}
