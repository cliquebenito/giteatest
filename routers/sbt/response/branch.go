package response

// Branch ветка репозитория
type Branch struct {
	Name                          string         `json:"name"`
	Commit                        *PayloadCommit `json:"commit"`
	Protected                     bool           `json:"protected"`
	RequiredApprovals             int64          `json:"required_approvals"`
	EnableStatusCheck             bool           `json:"enable_status_check"`
	StatusCheckContexts           []string       `json:"status_check_contexts"`
	UserCanPush                   bool           `json:"user_can_push"`
	UserCanMerge                  bool           `json:"user_can_merge"`
	EffectiveBranchProtectionName string         `json:"effective_branch_protection_name"`
}

// BranchesListResult структура ответа на запрос списка веток репозитория с общим количеством веток
type BranchesListResult struct {
	Total int       `json:"total"`
	Data  []*Branch `json:"data"`
}
