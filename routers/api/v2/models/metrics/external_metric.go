package metrics

// ExternalMetricOptions модель для запроса внутренней метрики
type ExternalMetricOptions struct {
	RepoKey    string `json:"repo_key"`
	TenantKey  string `json:"tenant_key"`
	ProjectKey string `json:"project_key"`
}

// swagger:response externalMetricGetResponse
type ExternalMetricGetResponse struct {
	Value int    `json:"value"`
	Text  string `json:"text"`
}

type SetExternalMetricRequest struct {
	Value int    `json:"value" binding:"Required"`
	Text  string `json:"text" binding:"Required"`
}
