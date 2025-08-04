package repo_mark

type Mark struct {
	Label    string `json:"label"`
	ExpertID int64  `json:"expert_id"`
}

// GetRepoMarksResponse модель ответа для запроса отметок репозитория
type GetRepoMarksResponse struct {
	Marks []Mark `json:"marks"`
}
