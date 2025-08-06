package org_accesser

import (
	"context"
	"fmt"

	"code.gitea.io/gitea/models/organization"
	"code.gitea.io/gitea/models/role_model"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/trace"
	"code.gitea.io/gitea/routers/web/user/accesser"
)

type casbinPermissioner interface {
	CheckUserPermissionToOrganization(
		ctx context.Context,
		sub *user_model.User,
		tenantId string,
		org *organization.Organization,
		action role_model.Action,
	) (bool, error)
}

type requestAccesser struct {
	casbinPermissioner
}

func New(casbinPermissioner casbinPermissioner) requestAccesser {
	return requestAccesser{casbinPermissioner: casbinPermissioner}
}

func (a requestAccesser) IsReadAccessGranted(ctx context.Context, request accesser.OrgAccessRequest) (bool, error) {
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

	action := role_model.READ
	doer := &user_model.User{ID: request.DoerID}
	targetOrg := &organization.Organization{ID: request.TargetOrgID}

	hasPermission, err := a.casbinPermissioner.
		CheckUserPermissionToOrganization(ctx, doer, request.TargetTenantID, targetOrg, action)
	if err != nil {
		log.Error("Error has occurred while checking permission: %v", err)
		return false, fmt.Errorf("check premission: %w", err)
	}

	return hasPermission, nil
}

func (a requestAccesser) IsAccessGranted(ctx context.Context, request accesser.OrgAccessRequest) (bool, error) {
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

	doer := &user_model.User{ID: request.DoerID}
	targetOrg := &organization.Organization{ID: request.TargetOrgID}
	hasPermission, err := a.casbinPermissioner.
		CheckUserPermissionToOrganization(ctx, doer, request.TargetTenantID, targetOrg, request.Action)
	if err != nil {
		log.Error("Error has occurred while checking permission: %v", err)
		return false, fmt.Errorf("check premission: %w", err)
	}

	return hasPermission, nil
}
