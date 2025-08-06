package task_tracker_client

// CheckCodesRequest модель запроса к TaskTracker
type CheckCodesRequest struct {
	Codes []string `json:"unit_codes"`
}

type Unit struct {
	Code string `json:"code"`

	IsExists bool `json:"is_exists"`
}

// CheckCodesResponse модель ответа TaskTracker
type CheckCodesResponse struct {
	Units []Unit `json:"units"`
}
