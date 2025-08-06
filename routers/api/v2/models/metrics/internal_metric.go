package metrics

// InternalMetricGetOptions модель для запроса внутренней метрики
type InternalMetricGetOptions struct {
	RepoKey    string `json:"repo_key"`
	TenantKey  string `json:"tenant_key"`
	ProjectKey string `json:"project_key"`
	Metric     string `json:"metric"`
}

// swagger:response internalMetricGetResponse
type InternalMetricGetResponse struct {
	Value int `json:"value"`
}
