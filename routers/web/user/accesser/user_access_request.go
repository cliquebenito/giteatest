package accesser

import "code.gitea.io/gitea/modules/structs"

// UserAccessRequest описывает запрос на аутентификацию для пользователя
type UserAccessRequest struct {
	DoerID int64

	DoerTenantIDs   []string
	TargetTenantIDs []string

	TargetUserID int64

	Visibility structs.VisibleType
}

func (u UserAccessRequest) IsOwner() bool {
	return u.DoerID == u.TargetUserID
}

func (u UserAccessRequest) IsVisibilityPrivate() bool {
	return u.Visibility.IsPrivate()
}

func (u UserAccessRequest) IsVisibilityLimited() bool {
	return u.Visibility.IsLimited()
}

func (u UserAccessRequest) IsDoerAndTargetUserInTheSameTenant() bool {
	doerTenantIDsInMap := map[string]struct{}{}

	for _, doerTenantID := range u.DoerTenantIDs {
		doerTenantIDsInMap[doerTenantID] = struct{}{}
	}

	for _, targetTenantID := range u.TargetTenantIDs {
		if _, ok := doerTenantIDsInMap[targetTenantID]; ok {
			return true
		}
	}

	return false
}
