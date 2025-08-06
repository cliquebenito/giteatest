package templates

import (
	"context"

	"code.gitea.io/gitea/models/organization"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/role_model"
	trace_model "code.gitea.io/gitea/models/trace"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/trace"
)

// CheckPrivileges проверяет привилегии на основании данных из шаблона
func CheckPrivileges(traceID string, endpoint string, userId int64, tenant string, orgId int64, action string) bool {
	ctx := context.WithValue(context.Background(), trace_model.Key, traceID)
	ctx = context.WithValue(ctx, trace_model.EndpointKey, endpoint)
	ctx = context.WithValue(ctx, trace_model.FrontedKey, true)

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

	return role_model.ConvertAndCheckPrivileges(ctx, userId, tenant, orgId, action)
}

// CheckPrivilegesByRoleAndCustom проверяeм кастомные привилегии для пользователя на основании данных из шаблона
func CheckPrivilegesByRoleAndCustom(traceID string, endpoint string, userId int64, tenantId string, orgId, repoID int64, actionCustom string) bool {
	ctx := context.WithValue(context.Background(), trace_model.Key, traceID)
	ctx = context.WithValue(ctx, trace_model.EndpointKey, endpoint)
	ctx = context.WithValue(ctx, trace_model.FrontedKey, true)

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

	allow, err := role_model.CheckUserPermissionToTeam(ctx, &user_model.User{ID: userId}, tenantId, &organization.Organization{ID: orgId}, &repo_model.Repository{ID: repoID}, actionCustom)
	if err != nil || !allow {
		return false
	}
	return allow
}
