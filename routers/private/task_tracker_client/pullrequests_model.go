package task_tracker_client

import (
	"code.gitea.io/gitea/models/pull_request_sender"
)

// AddPullRequestLinkRequest payload для отправки события о создания pr
type AddPullRequestLinkRequest struct {
	UnitCodes []string                             `json:"unit_codes"`
	PrID      int64                                `json:"pr_id"`
	PrURL     string                               `json:"pr_url"`
	UserLogin string                               `json:"user_login"`
	PrStatus  pull_request_sender.FromUnitStatusPr `json:"pr_status"`
}

// DeletePullRequestLinkRequest payload для отправки события удаления pr
type DeletePullRequestLinkRequest struct {
	UnitCodes []string `json:"unit_codes"`
	PrID      int64    `json:"pr_id"`
	UserLogin string   `json:"user_login"`
}

// UpdatePullRequestStatusRequest payload при отправки события при обновлении pr
type UpdatePullRequestStatusRequest struct {
	PrStatus  pull_request_sender.FromUnitStatusPr `json:"pr_status"`
	UserLogin string                               `json:"user_login"`
}
