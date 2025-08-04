package user_accesser

import (
	"context"

	"code.gitea.io/gitea/routers/web/user/accesser"
)

type requestAccesser struct{}

func New() requestAccesser {
	return requestAccesser{}
}

func (a requestAccesser) IsReadAccessGranted(ctx context.Context, request accesser.UserAccessRequest) (bool, error) {
	// if restricted, it is available to any authorized user within the tenant.
	// if private, it is available only to him/herself

	if request.IsVisibilityPrivate() {
		if request.IsOwner() {
			return true, nil
		}

		return false, nil
	}

	if request.IsVisibilityLimited() {
		if request.IsDoerAndTargetUserInTheSameTenant() {
			return true, nil
		}

		return false, nil
	}

	return true, nil
}
