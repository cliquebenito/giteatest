package casbin_role_manager

import (
	"context"

	"code.gitea.io/gitea/models/organization"
	"code.gitea.io/gitea/models/role_model"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/trace"
)

type manager struct{}

func New() manager {
	return manager{}
}

func (m manager) CheckUserPermissionToOrganization(
	ctx context.Context,
	sub *user_model.User,
	tenantId string,
	org *organization.Organization,
	action role_model.Action,
) (bool, error) {
	logTracer := trace.NewLogTracer()
	message := logTracer.CreateTraceMessage(ctx)
	err := logTracer.Trace(message)
	if err != nil {
		log.Error("Error has occurred while creating trace message: %v", err)
	}
	defer func() {
		err = logTracer.TraceTime(message)
		if err != nil {
			log.Error("Error has occurred while creating trace time message: %v", err)
		}
	}()

	return role_model.CheckUserPermissionToOrganization(ctx, sub, tenantId, org, action)
}
