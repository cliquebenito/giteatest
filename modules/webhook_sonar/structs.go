package webhook_sonar

// SonarConditions структура поля conditions для webhook
type SonarConditions struct {
	ErrorThreshold string `json:"errorThreshold"`
	Metric         string `json:"metric"`
	OnLeakPeriod   bool   `json:"onLeakPeriod"`
	Operator       string `json:"operator"`
	Status         string `json:"status"`
	Value          string `json:"value"`
}

// SonarQualityGate структура qualityGate из webhook
type SonarQualityGate struct {
	Conditions []SonarConditions `json:"conditions"`
	Name       string            `json:"name"`
	Status     string            `json:"status"`
}

// WebHook структура webhook из sonarQube
type WebHook struct {
	ServerUrl   string           `json:"serverUrl"`
	TaskId      string           `json:"taskId"`
	Status      string           `json:"status"`
	Revision    string           `json:"revision"`
	AnalysedAt  string           `json:"analysedAt"`
	Branch      Branch           `json:"branch"`
	Project     Project          `json:"project"`
	QualityGate SonarQualityGate `json:"qualityGate"`
}

// SonarMetrics структура для парсинга метрик из sonarQube
type SonarMetrics struct {
	ID          string `json:"id"`
	Key         string `json:"key"`
	Type        string `json:"type"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Domain      string `json:"domain"`
	Direction   int    `json:"direction"`
	Qualitative bool   `json:"qualitative"`
	Hidden      bool   `json:"hidden"`
	Value       string `json:"value"`
}

// MetricsSonar список метрик из sonarQube webhook
type MetricsSonar struct {
	Metrics []SonarMetrics `json:"metrics"`
}

// SonarMeasures струкутра дополнительных метрик
type SonarMeasures struct {
	Metric string `json:"metric"`
	Period struct {
		Index     int    `json:"index"`
		BestValue bool   `json:"bestValue"`
		Value     string `json:"value"`
	}
	BestValue bool   `json:"bestValue"`
	Value     string `json:"value"`
	Component string `json:"component"`
}

// MeasureSonar информация о дополительных метриках
type MeasureSonar struct {
	Measures []SonarMeasures `json:"measures"`
}

// Branch информация о ветке из webhook
type Branch struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	IsMain bool   `json:"isMain"`
	Url    string `json:"url"`
}

// Project информация о проекте из webhook
type Project struct {
	Key  string `json:"key"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

// ResponseMetrics структура поля metrics в ответе SonarResponseForRepository
type ResponseMetrics struct {
	Key            string `json:"key"`
	Name           string `json:"name"`
	Type           string `json:"type"`
	Domain         string `json:"domain"`
	Value          string `json:"value"`
	AuxMetricKey   string `json:"aux_metric_key"`
	AuxMetricName  string `json:"aux_metric_name"`
	AuxMetricType  string `json:"aux_metric_type"`
	AuxMetricValue string `json:"aux_metric_value"`
}

// SonarResponseForRepository ответ для отображения информации на странице repository
type SonarResponseForRepository struct {
	SonarQubeStatus string            `json:"sonarQubeStatus"`
	AnalysedAt      string            `json:"analysedAt"`
	SonarUrl        string            `json:"sonarUrl"`
	SonarProjectKey string            `json:"sonarProjectKey"`
	Metrics         []ResponseMetrics `json:"metrics"`
}

// ConditionsMetrics информация о метриках для QualityGatesForProject
type ConditionsMetrics struct {
	Status         string `json:"status"`
	MetricKey      string `json:"metricKey"`
	Comparator     string `json:"comparator"`
	ErrorThreshold string `json:"errorThreshold,omitempty"`
	ActualValue    string `json:"actualValue"`
}

// PeriodAnalyseAt информация о периоде проведения анализа
type PeriodAnalyseAt struct {
	Mode      string `json:"mode"`
	Date      string `json:"date"`
	Parameter string `json:"parameter"`
}

// QualityGatesForProject ответ об анализе по все qualityGatesMetrics
type QualityGatesForProject struct {
	ProjectStatus struct {
		Status            string              `json:"status"`
		IgnoredConditions bool                `json:"ignoredConditions"`
		CaycStatus        string              `json:"caycStatus"`
		Conditions        []ConditionsMetrics `json:"conditions"`
		Period            PeriodAnalyseAt     `json:"period"`
	} `json:"projectStatus"`
}

// ResponseForPagePullRequest разрешение на слияние по ответу из sonarQube
type ResponseForPagePullRequest struct {
	UrlToSonarQube string `json:"urlToSonarQube"`
	Status         string `json:"status"`
}

// PullRequestsInformation ответ об статусах pull requests из sonarqube
type PullRequestsInformation struct {
	PullRequests []struct {
		Key    string `json:"key"`
		Title  string `json:"title"`
		Branch string `json:"branch"`
		Base   string `json:"base"`
		Status struct {
			QualityGateStatus string `json:"qualityGateStatus"`
		} `json:"status"`
		AnalysisDate string `json:"analysisDate"`
		Url          string `json:"url"`
		Target       string `json:"target"`
	} `json:"pullRequests"`
}
