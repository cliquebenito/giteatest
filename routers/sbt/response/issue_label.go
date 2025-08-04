package response

// Label метка для пулл реквеста или задачи
type Label struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Exclusive   bool   `json:"exclusive"`
	Color       string `json:"color"`
	Description string `json:"description"`
	URL         string `json:"url"`
}
