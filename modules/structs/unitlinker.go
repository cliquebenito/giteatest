package structs

import "fmt"

type GetUnitLinksWithDescriptionRequest struct {
	PullRequestID int64 `json:"pull_request_id" binding:"Required"`
}

func (g GetUnitLinksWithDescriptionRequest) Validate() error {
	if g.PullRequestID < 1 {
		return fmt.Errorf("'pull_request_id' is positive interger")
	}

	return nil
}

type GetDescriptionError struct {
	Code        string `json:"code"`
	Description string `json:"description"`
}

type GetDescriptionUnit struct {
	Name   string `json:"name"`
	Code   string `json:"code"`
	Status string `json:"state"`
	URL    string `json:"url"`

	Priority string `json:"priority"`
}

type GetUnitLinksWithDescriptionResponse struct {
	Units  []GetDescriptionUnit  `json:"units"`
	Errors []GetDescriptionError `json:"errors"`
}
