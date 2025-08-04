package task_tracker_client

const (
	PriorityAttributeName = "priority"
	StatusAttributeName   = "workflow_status"
)

type GetDescriptionsFilters struct {
	Units []string `json:"units"`
}

type GetDescriptionsAttributes []string

func getDefaultAttributes() GetDescriptionsAttributes {
	return []string{
		PriorityAttributeName,
		StatusAttributeName,
	}
}

type GetDescriptionsPage struct {
	Size int `json:"size"`
	Page int `json:"page"`
}

func getDefaultPage() GetDescriptionsPage {
	const pageSize = 100
	const page = 0
	return GetDescriptionsPage{Page: page, Size: pageSize}
}

// GetDescriptionsRequest модель запроса расширенной информации в TaskTracker
type GetDescriptionsRequest struct {
	Filters    GetDescriptionsFilters    `json:"filters"`
	Page       GetDescriptionsPage       `json:"page"`
	Attributes GetDescriptionsAttributes `json:"attributes"`
}

type GetDescriptionsError struct {
	Title string `json:"title"`
	Code  string `json:"code"`

	Description string `json:"description"`

	HTTPStatusCode int `json:"http_status_code"`
}

type GetDescriptionsUnit struct {
	Code    string `json:"code"`
	Summary string `json:"summary"`

	Description interface{} `json:"description"`
}

type GetDescriptionsAttribute struct {
	Code string `json:"code"`
	Name string `json:"name"`
	Type string `json:"type"`

	Description string `json:"description"`
}

type GetDescriptionsValue struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type GetDescriptionsAttributeAndValue struct {
	Attribute GetDescriptionsAttribute `json:"attribute"`
	Value     GetDescriptionsValue     `json:"value"`
}

type GetDescriptionsContent struct {
	Unit GetDescriptionsUnit `json:"unit"`

	AttributesAndValues []GetDescriptionsAttributeAndValue `json:"attributes"`
}

// GetDescriptionsResponse модель ответа по расширенной информации в TaskTracker
type GetDescriptionsResponse struct {
	Content []GetDescriptionsContent `json:"content"`
}
