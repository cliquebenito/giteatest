package pull_request_sender

// FromUnitStatusPr статус pull requests при обновлении статуса pr
type FromUnitStatusPr string

const (
	PRStatusOpen   FromUnitStatusPr = "opened"
	PRStatusClosed FromUnitStatusPr = "closed"
	PRStatusMerged FromUnitStatusPr = "merged"
)

// UpdatePullRequestStatusOptions параметры для добавления записи об обновления статуса pr
type UpdatePullRequestStatusOptions struct {
	UserName, PullRequestURL string
	PullRequestStatus        FromUnitStatusPr
	FromUnitID               int64
}
