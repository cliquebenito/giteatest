package user_or_organization

import (
	"errors"
	"fmt"

	"code.gitea.io/gitea/models/role_model"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/trace"
	"code.gitea.io/gitea/routers/web/user/accesser"
)

func (s Server) isAccessGranted(ctx *context.Context) (bool, error) {
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

	if !s.isSourceControlTenantsAndRoleModelEnabled {
		return true, nil
	}

	if ctx.Doer == nil {
		return false, NewInternalServerError(fmt.Errorf("doer is empty"))
	}

	if ctx.ContextUser == nil {
		return false, NewInternalServerError(fmt.Errorf("context user is empty"))
	}

	if ctx.ContextUser.IsOrganization() {
		return s.isAccessToOrgProfileGranted(ctx)
	}

	return s.isAccessToUserProfileGranted(ctx)
}

func (s Server) isAccessToUserProfileGranted(ctx *context.Context) (bool, error) {
	log.Debug("Grants check for user profile access is started")

	// TODO: move db logic from handler https://sberworks.ru/jira/browse/VCS-1296
	doerTenantIDs, err := role_model.GetUserTenantIDsOrDefaultTenantID(ctx.Doer)
	if err != nil {
		return false, NewInternalServerError(fmt.Errorf("get doer tenant ids by user id: %w", err))
	}

	// TODO: move db logic from handler https://sberworks.ru/jira/browse/VCS-1296
	userTenantIDs, err := role_model.GetUserTenantIDsOrDefaultTenantID(ctx.ContextUser)
	if err != nil {
		return false, NewInternalServerError(fmt.Errorf("get target user tenant ids by user id: %w", err))
	}

	accessRequest := accesser.UserAccessRequest{
		DoerID: ctx.Doer.ID, TargetTenantIDs: userTenantIDs, DoerTenantIDs: doerTenantIDs,
		TargetUserID: ctx.ContextUser.ID, Visibility: ctx.ContextUser.Visibility,
	}

	goCtx := context.GetGoContextFromRequestOrDefault(ctx)

	isAccessGranted, err := s.userRequestAccesser.IsReadAccessGranted(goCtx, accessRequest)
	if err != nil {
		return false, NewInternalServerError(fmt.Errorf("is read access granted: %w", err))
	}

	if !isAccessGranted {
		log.Debug("Check for user profile access is finished: negative result")
		return false, nil
	}

	log.Debug("Check for user profile access is finished: positive result")

	return true, nil
}

func (s Server) isAccessToOrgProfileGranted(ctx *context.Context) (bool, error) {
	// TODO: move db logic from handler https://sberworks.ru/jira/browse/VCS-1296
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

	tenantID, err := role_model.GetUserTenantId(ctx, ctx.Doer.ID)
	if err != nil {
		return false, NewInternalServerError(fmt.Errorf("get tenant id by org id: %w", err))
	}

	accessRequest := accesser.OrgAccessRequest{
		DoerID: ctx.Doer.ID, TargetTenantID: tenantID,
		TargetOrgID: ctx.ContextUser.ID,
	}

	goCtx := context.GetGoContextFromRequestOrDefault(ctx)

	isAccessGranted, err := s.orgRequestAccesser.IsReadAccessGranted(goCtx, accessRequest)
	if err != nil {
		return false, NewInternalServerError(fmt.Errorf("is read access granted: %w", err))
	}

	if !isAccessGranted {
		log.Debug("Check for org profile access is finished: negative result")
		return false, nil
	}

	log.Debug("Check for org profile access is finished: positive result")

	return true, nil
}

func (s Server) handleServerErrors(ctx *context.Context, err error) {
	if internalServerError := new(InternalServerError); errors.As(err, &internalServerError) {
		ctx.ServerError("internal error", internalServerError)
		return
	}
}
