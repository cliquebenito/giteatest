package accesser

import "code.gitea.io/gitea/models/role_model"

// OrgAccessRequest описывает запрос на аутентификацию для организации
type OrgAccessRequest struct {
	DoerID         int64
	TargetTenantID string
	TargetOrgID    int64
	Action         role_model.Action
}
