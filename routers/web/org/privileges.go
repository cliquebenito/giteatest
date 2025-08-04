package org

import (
	"net/http"
	"strconv"
	"strings"

	"code.gitea.io/gitea/models"
	"code.gitea.io/gitea/models/organization"
	"code.gitea.io/gitea/models/organization/custom"
	"code.gitea.io/gitea/models/role_model"
	"code.gitea.io/gitea/models/tenant"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/base"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/json"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/sbt/audit"
	auditutils "code.gitea.io/gitea/modules/sbt/audit/utils"
	"code.gitea.io/gitea/modules/templates"
	"code.gitea.io/gitea/modules/trace"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/routers/web/user/accesser"
	"code.gitea.io/gitea/routers/web/user/team_server"
	"code.gitea.io/gitea/services/forms"
)

const (
	// tplPrivileges путь до шаблона для просмотра привилегий организации
	tplPrivileges base.TplName = "org/privileges/view"
)

// Privileges метод для заполнения и возврата шаблона для просмотра групп привилегий
func Privileges(ctx *context.Context) {
	privileges, err := role_model.GetPrivilegesByOrgId(ctx.Org.Organization.ID)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, err.Error())
		return
	}

	tenantId, err := tenant.GetTenantByOrgIdOrDefault(ctx, ctx.Org.Organization.ID)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, err.Error())
		return
	}

	for _, privilege := range privileges {
		privilege.User.Avatar = string(templates.Avatar(ctx, privilege.User, 24, "tiny"))
	}

	ctx.Data["PageIsOrgPrivileges"] = true
	ctx.Data["Privileges"] = privileges
	ctx.Data["TenantID"] = tenantId
	roles := role_model.GetUserRoles()
	roleNames := role_model.GetUserRoleNames()
	ctx.Data["Roles"] = roles
	ctx.Data["RoleNames"] = roleNames
	ctx.HTML(http.StatusOK, tplPrivileges)
}

// GrantPrivileges метод для заполнения формы добавления привилегий
func GrantPrivileges(ctx *context.Context) {
	tenantId, err := tenant.GetTenantByOrgIdOrDefault(ctx, ctx.Org.Organization.ID)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, err.Error())
		return
	}
	filterName := strings.ToLower(ctx.FormString("search"))
	users, err := role_model.GetUsersForAssigment(ctx, tenantId, filterName, ctx.Org.Organization)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, err.Error())
		return
	}

	for _, user := range users {
		user.Avatar = string(templates.Avatar(ctx, user, 24, "tiny"))
	}

	ctx.Data["UsersForAssigment"] = users

	result := make(map[string][]*user_model.User)
	result["users"] = users

	ctx.JSON(http.StatusOK, result)
}

// GrantPrivilegesPost метод для обработки запроса на добавление привилегии
func GrantPrivilegesPost(ctx *context.Context) {
	form := web.GetForm(ctx).(*forms.GrantPrivilegesForm)
	orgName := audit.EmptyRequiredField
	if ctx.Org != nil && ctx.Org.Organization != nil {
		orgName = ctx.Org.Organization.Name
	}

	auditValues := auditutils.NewRequiredAuditParams(ctx)
	auditParams := map[string]string{
		"project":          orgName,
		"affected_user_id": auditValues.DoerID,
		"affected_user":    auditValues.DoerName,
	}

	if form == nil {
		auditParams["error"] = "Error has occurred while validating form"
		ctx.Error(http.StatusBadRequest)
		return
	}
	newValue := struct {
		UserID   int64
		OrgID    int64
		Role     string
		TenantID string
	}{
		UserID:   form.UserId,
		OrgID:    form.OrgId,
		Role:     form.Role,
		TenantID: form.TenantId,
	}

	newValueBytes, marshalErr := json.Marshal(newValue)
	if marshalErr != nil {
		auditParams["error"] = "Error has occurred while marshalling form"
		audit.CreateAndSendEvent(audit.PrivilegesGrantEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
		ctx.Error(http.StatusBadRequest)
		return
	}

	auditParams["new_value"] = string(newValueBytes)

	u := &user_model.User{
		ID: form.UserId,
	}
	org := &organization.Organization{
		ID: form.OrgId,
	}
	role, ok := role_model.GetRoleByString(form.Role)
	if !ok {
		auditParams["error"] = "Error has occurred while searching for a group privileges"
		audit.CreateAndSendEvent(audit.PrivilegesGrantEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
		err := &role_model.ErrNonExistentRole{Role: form.Role}
		ctx.Error(http.StatusBadRequest, err.Error())
		return
	}

	privileges, err := role_model.GetPrivilegesByTenant(form.TenantId)
	if err != nil {
		auditParams["error"] = "Error has occurred while getting privileges"
		audit.CreateAndSendEvent(audit.PrivilegesGrantEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
		ctx.NotFound("Not found target tenant with GetPrivilegesByTenant", err)
		return
	}
	err = role_model.GrantUserPermissionToOrganization(u, form.TenantId, org, role)
	if err != nil {
		auditParams["error"] = "Error has occurred while granting privileges"
		audit.CreateAndSendEvent(audit.PrivilegesGrantEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
		ctx.Error(http.StatusInternalServerError, err.Error())
		return
	}

	for i, privilege := range privileges {
		if privilege.Org.ID == org.ID {
			continue
		}
		if i == len(privileges)-1 {
			auditParams["error"] = "Error occurred while searching by tenant for a group privileges"
			audit.CreateAndSendEvent(audit.PrivilegesGrantEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
			ctx.NotFound("Not found target tenant with privileges", err)
			return
		}
	}

	audit.CreateAndSendEvent(audit.PrivilegesGrantEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusSuccess, auditValues.RemoteAddress, auditParams)
	ctx.Status(http.StatusOK)
}

type Server struct {
	casbinPermissioner context.RolePermissioner
	casbinCustomRepo   custom.CustomPrivileger
	creator            *team_server.Server
}

func NewServerForPrivileges(casbinPermissioner context.RolePermissioner, custom custom.CustomPrivileger, creator *team_server.Server) Server {
	return Server{
		creator:            creator,
		casbinCustomRepo:   custom,
		casbinPermissioner: casbinPermissioner}
}

// RevokePrivilegesPost метод для обработки запроса на удаление привилегии
func (s *Server) RevokePrivilegesPost(ctx *context.Context) {
	form := web.GetForm(ctx).(*forms.GrantPrivilegesForm)
	auditValues := auditutils.NewRequiredAuditParams(ctx)
	orgName := audit.EmptyRequiredField
	if ctx.Org != nil && ctx.Org.Organization != nil {
		orgName = ctx.Org.Organization.Name
	}
	auditParams := map[string]string{
		"project":          orgName,
		"affected_user_id": auditValues.DoerID,
		"affected_user":    auditValues.DoerName,
	}

	oldValue := struct {
		UserID   int64
		OrgID    int64
		Role     string
		TenantID string
	}{
		UserID:   form.UserId,
		OrgID:    form.OrgId,
		Role:     form.Role,
		TenantID: form.TenantId,
	}

	oldValueBytes, _ := json.Marshal(oldValue)
	auditParams["old_value"] = string(oldValueBytes)

	if ctx.Written() {
		auditParams["error"] = "Error occurred while validating form"
		audit.CreateAndSendEvent(audit.PrivilegesRevokeEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
		return
	}

	u := &user_model.User{
		ID: form.UserId,
	}
	org := &organization.Organization{
		ID: form.OrgId,
	}
	role, ok := role_model.GetRoleByString(form.Role)
	if !ok {
		auditParams["error"] = "Error has occurred while searching for a group privileges"
		audit.CreateAndSendEvent(audit.PrivilegesRevokeEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
		err := &role_model.ErrNonExistentRole{Role: form.Role}
		ctx.Error(http.StatusBadRequest, err.Error())
		return
	}
	if err := role_model.RevokeUserPermissionToOrganization(u, form.TenantId, org, role, true); err != nil {
		auditParams["error"] = "Error has occurred while revoke for a group privileges"
		audit.CreateAndSendEvent(audit.PrivilegesRevokeEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
		ctx.Error(http.StatusInternalServerError, err.Error())
		return
	}

	teams, err := organization.GetUserOrgTeams(ctx, org.ID, u.ID)
	if err != nil {
		auditParams["error"] = "Error has occurred while getting teams for user"
		audit.CreateAndSendEvent(audit.PrivilegesRevokeEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
		ctx.Error(http.StatusInternalServerError, err.Error())
		return
	}

	tenantID, err := tenant.GetTenantByOrgIdOrDefault(ctx, org.ID)
	if err != nil {
		auditParams["error"] = "Error has occurred while getting tenant"
		audit.CreateAndSendEvent(audit.PrivilegesRevokeEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
		ctx.Error(http.StatusInternalServerError, err.Error())
		return
	}

	for _, team := range teams {
		err = models.RemoveTeamMember(team, u.ID)
		if err != nil {
			if organization.IsErrLastOrgOwner(err) {
				ctx.Flash.Error(ctx.Tr("form.last_org_owner"))
				auditParams["error"] = "Cannot remove the last user from the 'owners' team"
				audit.CreateAndSendEvent(audit.PrivilegesRevokeEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
				return
			} else {
				auditParams["error"] = "Error has occurred while removing team member"
				audit.CreateAndSendEvent(audit.PrivilegesRevokeEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
				ctx.Error(http.StatusInternalServerError, err.Error())
				return
			}
		}
		if err := s.creator.RemoveCustomPrivilege(accesser.RepoAccessRequest{
			DoerID:         u.ID,
			TargetTenantID: tenantID,
			OrgID:          ctx.Org.Organization.ID,
			Team:           team,
		}); err != nil {
			auditParams["error"] = "Error has occurred while removing user custom privileges"
			audit.CreateAndSendEvent(audit.PrivilegesRevokeEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			log.Error("TeamsAction role_model.RemoveUserFromTeamCustomPrivilege failed while removing user's custom privileges from team : %v", err)
			ctx.Error(http.StatusInternalServerError, err.Error())
			return
		}
	}
	audit.CreateAndSendEvent(audit.PrivilegesRevokeEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusSuccess, auditValues.RemoteAddress, auditParams)
	ctx.Status(http.StatusOK)
}

// CheckPrivileges проверяет привилегии на основании данных из запроса
func CheckPrivileges(ctx *context.Context) {
	logTracer := trace.NewLogTracer()
	message := logTracer.CreateTraceMessage(ctx)
	errTrace := logTracer.Trace(message)
	if errTrace != nil {
		log.Error("Error has occurred while creating trace message: %v", errTrace)
	}
	defer func() {
		errTrace = logTracer.TraceTime(message)
		if errTrace != nil {
			log.Error("Error has occurred while creating trace time message: %v", errTrace)
		}
	}()

	form := web.GetForm(ctx).(*forms.CheckPrivilegesForm)

	if ctx.Written() {
		return
	}

	result := map[string]bool{
		"is_allow": role_model.ConvertAndCheckPrivileges(ctx, form.UserId, form.TenantId, form.OrgId, form.Action),
	}

	ctx.JSON(http.StatusOK, result)
}

// GetPrivilegesForUser возвращает список привилегий пользователя в проекте
func GetPrivilegesForUser(ctx *context.Context) {
	form := web.GetForm(ctx).(*forms.UserPrivilegesForm)

	if ctx.Written() {
		return
	}

	userRole, err := role_model.GetRoleForUser(form.UserId, form.OrgId, form.TenantId)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, err.Error())
		return
	}

	result := map[string][]role_model.Action{
		"privileges": role_model.GetActionsForRole(userRole),
	}

	ctx.JSON(http.StatusOK, result)
}
