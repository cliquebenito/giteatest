package role_model

import (
	"context"
	"fmt"
	"strconv"

	"code.gitea.io/gitea/modules/trace"
	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"xorm.io/builder"

	"code.gitea.io/gitea/models"
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/organization"
	"code.gitea.io/gitea/models/repo"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/tenant"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/sbt/audit"
	"code.gitea.io/gitea/modules/setting"
)

// configureRoleModel возвращает за конфигурированную модель контроля доступа
/*
	Конфигурация модели содержит разделы:
		r - request_definition - Раздел определяет аргументы в функции e.Enforce(...).
		p - policy_definition - Раздел определяет значение политики.
		g - role_definition - Данный раздел используется для определения отношений наследования ролей RBAC.
Так же при использовании в разделе matchers служит построителем графических отношений,
который использует график для сравнения объекта запроса с объектом политики.
		e - policy_effect - это определение для политического эффекта. Он определяет, следует ли одобрять запрос на доступ,
если запросу соответствуют несколько правил политики. Например, одно правило разрешает и другое отрицает.
		m - matchers - Раздел является определением для политических соответствий. matchers - это выражения,
которые определяют, как правила политики вычисляются в соответствии с запросом.

	Текущая настройка(описана в формате: (раздел)/(запись в разделе) - (значение записи) - (описание):
		r/r - sub, tenant, project, act - Поля имеют следующий смысл:
																				sub - объект доступа.
																				tenant - тенант к которому запрашивается доступ.
																				project - проект к которому запрашивается доступ.
																				act - действие, которое необходимо совершить.
		p/p - sub, tenant, project, act - Поля имеют следующий смысл:
																				sub - объект доступа.
																				tenant - тенант к которому предоставляется доступ.
																				project - проект к которому предоставляется доступ.
																				act - действие, которое разрешено совершать.
		g/g - _, _ - означает, что в отношениях наследования участвуют две стороны. Например, owner, own - означает,
что роль owner содержит в себе доступ к действию own
		e/e - some(where (p.eft == allow)) - если есть какое-либо согласованное правило политики allow, конечным эффектом является allow
		m/m - r.sub == p.sub && r.tenant == p.tenant && r.project == p.project && g(p.act, r.act) - проверяет что:
																																																- мы нашли политику для пользователя из запроса
																																																- тенант в политике и в запросе совпадают
																																																- проект в политике и в запросе совпадают
																																																- содержит ли политика запрашиваемое действие в списке разрешенных действий
*/
func configureRoleModel() model.Model {
	m := model.NewModel()
	m.AddDef("r", "r", "sub, tenant, project, act")
	m.AddDef("r", "r2", "sub, tenant, project")
	m.AddDef("r", "r3", "team, project, repository, act")
	m.AddDef("p", "p", "sub, tenant, project, act")
	m.AddDef("p", "p2", "project, act")
	m.AddDef("p", "p3", "sub, act")
	m.AddDef("p", "p4", "sub, tenant, project, team")
	m.AddDef("p", "p5", "team, project, repository, name") // cB_vB_vPR, create_branch, view_branch, view_Pr
	m.AddDef("g", "g", "_, _")
	m.AddDef("g", "g2", "_, _")
	m.AddDef("g", "g3", "_, _")
	m.AddDef("e", "e", "some(where (p.eft == allow))")
	m.AddDef("m", "m", "r.sub == p.sub && r.tenant == p.tenant && r.project == p.project && g(p.act, r.act)")
	// permission на inner source true идем в таблицу sc_tenant_organization и проверяем на соответствие tenant
	m.AddDef("m", "m2", "r.project == p2.project && g2(p2.act, r.act)")
	m.AddDef("m", "m3", "r.sub == p3.sub && g(p3.act, r.act)")
	m.AddDef("m", "m4", "r2.sub == p4.sub && r2.tenant == p4.tenant && r2.project == p4.project")
	m.AddDef("m", "m5", "r3.team == p5.team && r3.project == p5.project && r3.repository == p5.repository && g3(p5.name, r3.act)")
	return m
}

// GrantUserPermissionToOrganization назначает пользователю роль в проекте под тенантом
func GrantUserPermissionToOrganization(sub *user_model.User, tenantId string, org *organization.Organization, role Role) error {
	if err := validateNewPrivileges(sub, tenantId, org, role); err != nil {
		log.Error("Error has occurred while validating new privileges. Error: %v", err)
		return err
	}

	return grantUserPermissionToOrganization(sub, tenantId, org, role)
}

func grantUserPermissionToOrganization(sub *user_model.User, tenantId string, org *organization.Organization, role Role) error {
	if err := removeExistingPrivilegesInOrg(sub, org); err != nil {
		log.Error("Error has occurred while removing userId: %d privileges in orgId: %d. Error: %v", sub.ID, org.ID, err)
		return err
	}

	team, err := organization.GetOwnerTeam(context.Background(), org.ID)
	if err != nil {
		log.Error("Error has occurred while getting owner team for orgId: %d. Error: %v", org.ID, err)
		return err
	}

	if err := models.AddTeamMember(team, sub.ID); err != nil {
		log.Error("Error has occurred while adding teamId: %d member userId: %d. Error: %v", team.ID, sub.ID, err)
		return err
	}

	if _, err := securityEnforcer.AddPolicy(strconv.FormatInt(sub.ID, 10), tenantId, strconv.FormatInt(org.ID, 10), role.String()); err != nil {
		log.Error("Error has occurred while adding %v policy to projectId: %d for userId: %d under tenantId: %v. Error: %v", role.String(), org.ID, sub.ID, tenantId, err)
		return err
	}
	if err := securityEnforcer.SavePolicy(); err != nil {
		log.Error("Error has occurred while saving %v policy to projectId: %d for userId: %d under tenantId: %v. Error: %v", role.String(), org.ID, sub.ID, tenantId, err)
		return err
	}

	log.Debug("%v policy to projectId: %d for userId: '%d' under tenantId: %v successful granted", role.String(), org.ID, sub.ID, tenantId)
	return nil
}

func AddProjectToInnerSource(org *organization.Organization) error {
	if _, err := securityEnforcer.AddNamedPolicy("p2", strconv.FormatInt(org.ID, 10), InnerSource); err != nil {
		log.Error("Error has occurred while adding projectId: %d to %v projects. Error: %v", org.ID, InnerSource, err)
		return err
	}
	if err := securityEnforcer.SavePolicy(); err != nil {
		log.Error("Error has occurred while saving policy with adding projectId: %d to %v projects. Error: %v", org.ID, InnerSource, err)
		return err
	}

	log.Debug("projectId: %d to %v projects successful added", org.ID, InnerSource)
	return nil
}

// GrantUserTuz выдает пользователю роль tuz
func GrantUserTuz(sub *user_model.User) error {
	auditParams := map[string]string{
		"subject_name": sub.Name,
	}
	if _, err := securityEnforcer.AddNamedPolicy("p3", strconv.FormatInt(sub.ID, 10), TUZ.String()); err != nil {
		log.Error("Error has occurred while adding user: %d to %v. Error: %v", sub.ID, TUZ.String(), err)
		auditParams["error"] = "Error has occurred while adding tuz named policy for user"
		audit.CreateAndSendEvent(audit.UserTuzRightsGrantedEvent, audit.EmptyRequiredField, audit.EmptyRequiredField, audit.StatusFailure, audit.EmptyRequiredField, auditParams)
		return err
	}
	if err := securityEnforcer.SavePolicy(); err != nil {
		log.Error("Error has occurred while saving policy with adding user: %d to %v. Error: %v", sub.ID, TUZ.String(), err)
		auditParams["error"] = "Error has occurred while saving tuz policy for user"
		audit.CreateAndSendEvent(audit.UserTuzRightsGrantedEvent, audit.EmptyRequiredField, audit.EmptyRequiredField, audit.StatusFailure, audit.EmptyRequiredField, auditParams)
		return err
	}

	log.Debug("user: %d successful added to %v", sub.ID, TUZ.String())
	audit.CreateAndSendEvent(audit.UserTuzRightsGrantedEvent, audit.EmptyRequiredField, audit.EmptyRequiredField, audit.StatusSuccess, audit.EmptyRequiredField, auditParams)
	return nil
}

type CheckUserPermissionFnType func(ctx context.Context, sub *user_model.User, tenantId string, org *organization.Organization, action Action) (bool, error)

// CheckUserPermissionToOrganization проверяет доступ пользователя к проекту под тенантом
// Deprecated: should use models/role_model/casbin_role_manager implementation
func CheckUserPermissionToOrganization(ctx context.Context, sub *user_model.User, tenantId string, org *organization.Organization, action Action) (bool, error) {
	logTracer := trace.NewLogTracer()
	traceParams := map[string]interface{}{
		"userId":   sub.ID,
		"orgId":    org.ID,
		"tenantId": tenantId,
		"action":   action.String(),
	}

	message := logTracer.CreateTraceMessageWithParams(ctx, traceParams)
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

	permitted, err := securityEnforcer.Enforce(strconv.FormatInt(sub.ID, 10), tenantId, strconv.FormatInt(org.ID, 10), action.String())
	if err != nil {
		log.Error("Error has occurred while checking %v permission to projectId: %d for userId: %d under tenantId: %v. Error: %v", action.String(), org.ID, sub.ID, tenantId, err)
		return false, err
	}
	permittedFromInnerSource, errFromInnerSource := securityEnforcer.Enforce(casbin.EnforceContext{
		RType: "r",
		PType: "p2",
		EType: "e",
		MType: "m2",
	}, strconv.FormatInt(sub.ID, 10), tenantId, strconv.FormatInt(org.ID, 10), action.String())
	if errFromInnerSource != nil {
		log.Error("Error has occurred while checking %v permission to projectId: %d for userId: %d under tenantId: %v. Error: %v", action.String(), org.ID, sub.ID, tenantId, errFromInnerSource)
		return false, errFromInnerSource
	}

	permittedTuz, errTuz := securityEnforcer.Enforce(casbin.EnforceContext{
		RType: "r",
		PType: "p3",
		EType: "e",
		MType: "m3",
	}, strconv.FormatInt(sub.ID, 10), tenantId, strconv.FormatInt(org.ID, 10), action.String())
	if errTuz != nil {
		log.Error("Error has occurred while checking %v permission to projectId: %d for userId: %d under tenantId: %v. Error: %v", action.String(), org.ID, sub.ID, tenantId, errTuz)
		return false, errTuz
	}
	if !permittedTuz && !permitted && permittedFromInnerSource {
		tenantOrg, err := tenant.GetTenantOrganizationsByOrgId(db.DefaultContext, org.ID)
		if err != nil {
			log.Error("Error has occurred while checking %s permission to projectId: %d for userId: %d. Error: %v", action.String(), org.ID, sub.ID, err)
			return false, err
		}
		permittedFromInnerSource = tenantOrg.TenantID == tenantId
	}

	return permitted || permittedFromInnerSource || permittedTuz, nil
}

// CheckUserPermissionToTeam проверяет доступ пользователя к репозиторию по кастомным привилегиям +++
// Deprecated: should use models/role_model/custom_casbin_role_manager implementation
func CheckUserPermissionToTeam(ctx context.Context, sub *user_model.User, tenantId string, org *organization.Organization, repo *repo.Repository, action string) (bool, error) {
	logTracer := trace.NewLogTracer()
	traceParams := map[string]interface{}{
		"userId":   sub.ID,
		"orgId":    org.ID,
		"tenantId": tenantId,
		"action":   action,
		"custom":   true,
	}

	message := logTracer.CreateTraceMessageWithParams(ctx, traceParams)
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

	if org == nil || sub == nil || repo == nil {
		return false, fmt.Errorf("invalid input: organization, user, or repository is nil")
	}
	permitted, err := securityEnforcer.Enforce(casbin.EnforceContext{
		RType: "r2",
		PType: "p4",
		EType: "e",
		MType: "m4",
	}, strconv.FormatInt(sub.ID, 10), tenantId, strconv.FormatInt(org.ID, 10))
	if err != nil {
		log.Error("Error has occurred while checking action: %v permission for userID: %v, projectID: %v, repoID: %v, branch: %v. Error: %v", action, sub.ID, org.ID, repo.ID, err)
		return false, fmt.Errorf("matching policies: %w", err)
	}

	teams, err := organization.GetUserOrgTeams(db.DefaultContext, org.ID, sub.ID)
	if err != nil {
		log.Error("Error has occurred while getting user organization teams for userID: %v, projectID: %v. Error: %v", sub.ID, org.ID, err)
		return false, fmt.Errorf("getting teams by org: %w", err)
	}

	for _, t := range teams {
		permittedForTeam, err := securityEnforcer.Enforce(casbin.EnforceContext{
			RType: "r3",
			PType: "p5",
			EType: "e",
			MType: "m5",
		}, t.Name, strconv.FormatInt(org.ID, 10), strconv.FormatInt(repo.ID, 10), action)
		if err != nil {
			log.Error("Error has occurred while checking action: %v permission for userID: %v, projectID: %v, repoID: %v, branch: %v. Error: %v", action, sub.ID, org.ID, repo.ID, err)
			return false, fmt.Errorf("matching policies: %w", err)
		}

		if permitted && permittedForTeam {
			return true, nil
		}
	}

	return false, nil
}

// RevokeUserPermissionToOrganization снимает с пользователя роль в проекте под тенантом
func RevokeUserPermissionToOrganization(sub *user_model.User, tenantId string, org *organization.Organization, role Role, permanentRemove bool) error {
	if permanentRemove {
		if err := models.RemoveOrgUser(org.ID, sub.ID); err != nil {
			log.Error("Error has occurred while removing userId: %d from orgId: %d. Error: %v", sub.ID, org.ID, err)
			return err
		}
	}
	if _, err := securityEnforcer.RemovePolicy(strconv.FormatInt(sub.ID, 10), tenantId, strconv.FormatInt(org.ID, 10), role.String()); err != nil {
		log.Error("Error has occurred while removing %v policy to projectId: %d for userId: %d under tenantId: %v. Error: %v", role.String(), org.ID, sub.ID, tenantId, err)
		return err
	}
	if err := securityEnforcer.SavePolicy(); err != nil {
		log.Error("Error has occurred while removing %v policy to projectId: %d for userId: %d under tenantId: %v. Error: %v", role.String(), org.ID, sub.ID, tenantId, err)
		return err
	}

	log.Debug(" %v policy to projectId: %d for userId: %d under tenantId: %v successful revoked", role.String(), org.ID, sub.ID, tenantId)
	return nil
}

func RemoveProjectToInnerSource(org *organization.Organization) error {
	if _, err := securityEnforcer.RemoveNamedPolicy("p2", strconv.FormatInt(org.ID, 10), InnerSource); err != nil {
		log.Error("Error has occurred while removing projectId: %d from %v projects. Error: %v", org.ID, InnerSource, err)
		return err
	}
	if err := securityEnforcer.SavePolicy(); err != nil {
		log.Error("Error has occurred while saving policy with removing projectId: %d from %v projects. Error: %v", org.ID, InnerSource, err)
		return err
	}

	log.Debug("projectId: %d from %v projects successful removed", org.ID, InnerSource)
	return nil
}

// GetAllPrivileges метод для получения всех привилегий
func GetAllPrivileges() ([]EnrichedPrivilege, error) {
	policy, err := securityEnforcer.GetPolicy()
	if err != nil {
		log.Error("Error has occurred while loading policy. Error: %v", err)
		return nil, fmt.Errorf("get policy. Error: %w", err)
	}

	privileges := convertStringToPrivilegeArray(policy)
	enrichedPrivileges, err := enrichPrivileges(privileges)
	if err != nil {
		log.Error("Error has occurred while enriching privileges. Error: %v", err)
		return nil, err
	}
	return enrichedPrivileges, nil
}

// CheckIsUserTuz проверяет, является ли пользователь ТУЗом
func CheckIsUserTuz(userId int64) (bool, error) {
	formattingUserId := []string{strconv.FormatInt(userId, 10)}
	filterPolicy, err := securityEnforcer.GetFilteredNamedPolicy("p3", 0, formattingUserId...)
	if err != nil {
		log.Error("Error has occurred while loading filtered named policy. Error: %v", err)
		return false, fmt.Errorf("get filtered named policy. Error: %w", err)
	}
	for _, policy := range filterPolicy {
		if len(policy) == 2 && policy[1] == TUZ.String() {
			return true, nil
		}
	}
	return false, nil
}

// GetPrivilegesByUserId метод для получения всех привилегий пользователя
func GetPrivilegesByUserId(userId int64) ([]EnrichedPrivilege, error) {
	formattingUserId := []string{strconv.FormatInt(userId, 10)}

	filterPolicy, err := securityEnforcer.GetFilteredPolicy(0, formattingUserId...)
	if err != nil {
		log.Error("Error has occurred while loading filtered policy. Error: %v", err)
		return nil, fmt.Errorf("get filtered policy. Error: %w", err)
	}

	privileges := convertStringToPrivilegeArray(filterPolicy)
	enrichedPrivileges, err := enrichPrivileges(privileges)
	if err != nil {
		log.Error("Error has occurred while enriching Privileges. Error: %v", err)
		return nil, err
	}

	return enrichedPrivileges, nil
}

// GetUsersForAssigment выдает список пользователей,	которых можно добавить в группу привилегий в проекте
func GetUsersForAssigment(ctx context.Context, tenantId, userName string, org *organization.Organization) (filteredUsers []*user_model.User, err error) {
	if userName == "" || org == nil {
		return nil, fmt.Errorf("missing organization or user name")
	}

	users, err := user_model.GetUserByChartName(ctx, userName)
	if err != nil {
		log.Error("Error has occurred while getting user by name. Error: %v", err)
		return nil, err
	}

	if !setting.SourceControl.MultiTenantEnabled {
		return
	}

	casbinPrivileges, err := GetAllPrivileges()
	if err != nil {
		log.Error("Error has get all Privileges. Error: %v", err)
		return
	}

	defaultTenant, err := tenant.GetDefaultTenant(ctx)
	if err != nil {
		log.Error("Error has get default tenant. Error: %v", err)
		return
	}

	// Получаем список индексов пользователей
	filteredUserIDs := make(map[int64]struct{})
	for _, user := range users {
		filteredUserIDs[user.ID] = struct{}{}
	}

	// Оставляем пользователей, которых еще нет в организации
	for _, casbinPrivilege := range casbinPrivileges {
		if casbinPrivilege.Org.ID == org.ID {
			delete(filteredUserIDs, casbinPrivilege.User.ID)
		}
	}

	// Оставляем только пользователей из текущего или дефолтного тенанта
	for _, casbinPrivilege := range casbinPrivileges {
		if !(casbinPrivilege.TenantID == tenantId || casbinPrivilege.TenantID == defaultTenant.ID) {
			delete(filteredUserIDs, casbinPrivilege.User.ID)
		}
	}

	// Восстанавливаем список пользователей по их id
	for _, user := range users {
		var isTuz bool
		if isTuz, err = CheckIsUserTuz(user.ID); err != nil {
			log.Error("Error has occurred while check if user is tuz. Error: %v", err)
			return nil, err
		}
		if isTuz {
			continue
		}
		if _, ok := filteredUserIDs[user.ID]; ok {
			filteredUsers = append(filteredUsers, user)
		}
	}
	return
}

// GetPrivilegesByTenant метод для получения всех привилегий в тенанте
func GetPrivilegesByTenant(tenantId string) ([]EnrichedPrivilege, error) {
	filterPolicy, err := securityEnforcer.GetFilteredPolicy(1, []string{tenantId}...)
	if err != nil {
		log.Error("Error has occurred while loading filtered policy. Error: %v", err)
		return nil, fmt.Errorf("get filtered policy. Error: %w", err)
	}

	privileges := convertStringToPrivilegeArray(filterPolicy)
	enrichedPrivileges, err := enrichPrivileges(privileges)
	if err != nil {
		log.Error("Error has occurred while enriching Privileges. Error: %v", err)
		return nil, err
	}

	return enrichedPrivileges, nil
}

// RemoveExistingPrivilegesByTenant удаляет привилегии в тенанте +++
func RemoveExistingPrivilegesByTenant(tenantID string) error {
	privilegesByTenant, err := GetPrivilegesByTenant(tenantID)
	if err != nil {
		log.Error("Error has occurred while getting Privileges by tenantID %s: %v", tenantID, err)
		return err
	}

	for _, privilege := range privilegesByTenant {
		if err := RevokeUserPermissionToOrganization(privilege.User, privilege.TenantID, privilege.Org, privilege.Role, true); err != nil {
			log.Error("Error has occurred while removing user's permission to organization")
			return fmt.Errorf("revoking user's permmissions: %w", err)
		}
	}
	return nil
}

// GetPrivilegesByOrgId метод для получения всех привилегий в проекте
func GetPrivilegesByOrgId(orgId int64) ([]EnrichedPrivilege, error) {
	orgPrivileges, err := securityEnforcer.GetFilteredPolicy(2, []string{strconv.FormatInt(orgId, 10)}...)
	if err != nil {
		log.Error("Error has occurred while loading filtered policy. Error: %v", err)
		return nil, fmt.Errorf("get filtered policy. Error: %w", err)
	}

	privileges := convertStringToPrivilegeArray(orgPrivileges)
	enrichedPrivileges, err := enrichPrivileges(privileges)
	if err != nil {
		log.Error("Error has occurred while enriching Privileges. Error: %v", err)
		return nil, err
	}

	return enrichedPrivileges, nil
}

// ConvertAndCheckPrivileges подготавливает данные и проверяет доступ пользователя к проекту под тенантом
func ConvertAndCheckPrivileges(ctx context.Context, userId int64, tenantId string, orgId int64, action string) bool {
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

	act, ok := GetActionByString(action)
	if !ok {
		return false
	}

	allowed, err := CheckUserPermissionToOrganization(ctx, &user_model.User{ID: userId}, tenantId, &organization.Organization{ID: orgId}, act)
	if err != nil || !allowed {
		return false
	}
	return true
}

// CheckPermissionForUserOfTeam проверяем права доступа для пользователя в команде +++
func CheckPermissionForUserOfTeam(ctx context.Context, userID int64, orgID, repoID int64, action string) (bool, error) {
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

	tenantID, err := GetUserTenantId(ctx, userID)
	if err != nil {
		log.Error("Error has occurred while getting tenant by user ID %d: %v", userID, err)
		return false, fmt.Errorf("getting tenant by user: %w", err)
	}

	if allowed, err := CheckUserPermissionToTeam(ctx, &user_model.User{ID: userID}, tenantID, &organization.Organization{ID: orgID}, &repo.Repository{ID: repoID}, action); err != nil || !allowed {
		return false, fmt.Errorf("check user permission to team. Error: %w", err)
	}
	return true, nil
}

// GetUserTenantId метод для получения идентификатора тенанта пользователя
func GetUserTenantId(ctx context.Context, userId int64) (string, error) {
	filterPolicy, err := securityEnforcer.GetFilteredPolicy(0, []string{strconv.FormatInt(userId, 10)}...)
	if err != nil {
		log.Error("Error has occurred while loading filtered policy. Error: %v", err)
		return "", fmt.Errorf("get filtered policy. Error: %w", err)
	}

	privileges := convertStringToPrivilegeArray(filterPolicy)
	if len(privileges) > 0 {
		return privileges[0].tenant, nil
	}
	defaultTenant, err := tenant.GetDefaultTenant(ctx)
	if err != nil {
		return "", err
	}
	return defaultTenant.ID, nil
}

// GetUserTenantIDsOrDefaultTenantID метод для получения идентификаторов тенанта пользователя
func GetUserTenantIDsOrDefaultTenantID(user *user_model.User) (ids []string, err error) {
	if user == nil {
		return ids, fmt.Errorf("user does not to identificate")
	}
	var orgs []*organization.TeamUser
	err = db.GetEngine(db.DefaultContext).Where(builder.Eq{"uid": user.ID}).Find(&orgs)
	if err != nil {
		return
	}
	if len(orgs) == 0 { //у нас в запросе пользователь без организаций
		if tenantObj, err := tenant.GetDefaultTenant(db.DefaultContext); err == nil {
			ids = []string{tenantObj.ID}
		}
		return
	}

	tenantIDs := make(map[string]struct{})

	for _, org := range orgs {
		var scTenantOrg *tenant.ScTenantOrganizations
		scTenantOrg, err = tenant.GetTenantOrganizationsByOrgId(db.DefaultContext, org.OrgID)
		if err != nil {
			return
		}
		if _, ok := tenantIDs[scTenantOrg.TenantID]; !ok {
			tenantIDs[scTenantOrg.TenantID] = struct{}{}
			ids = append(ids, scTenantOrg.TenantID)
		}
	}
	return
}

// GetRepoTenantId метод выдает тенант id исходя из репозитория
func GetRepoTenantId(repo *repo_model.Repository) (id string, err error) {
	if repo == nil || repo.OwnerID == 0 {
		return "", fmt.Errorf("repository owner id not found")
	}

	var tenants []*tenant.ScTenantOrganizations
	err = db.GetEngine(db.DefaultContext).Where(builder.Eq{"organization_id": repo.OwnerID}).Find(&tenants)
	return tenants[0].TenantID, err
}

// GetActionsForRole возвращает доступные действия для роли
func GetActionsForRole(role Role) []Action {
	policies, err := securityEnforcer.GetFilteredGroupingPolicy(0, []string{role.String()}...)
	if err != nil {
		log.Error("Error has occurred while loading filtered policy. Error: %v", err)
		return nil
	}

	actions := make([]Action, 0, len(policies))
	for _, policy := range policies {
		if convertedAction, ok := GetActionByString(policy[1]); ok {
			actions = append(actions, convertedAction)
		}
	}
	return actions
}

// GetRoleForUser Получает роль пользователя в проекте
func GetRoleForUser(userId int64, orgId int64, tenantId string) (Role, error) {
	userPrivileges, err := GetPrivilegesByUserId(userId)
	if err != nil {
		log.Error("Error has occurred while getting Privileges by userId. Error: %v", err)
		return 0, err
	}

	for _, privilege := range userPrivileges {
		if privilege.Org != nil && privilege.Org.ID == orgId && privilege.TenantID == tenantId {
			return privilege.Role, nil
		}
	}
	return 0, nil
}

// validateNewPrivileges валидация новой привилегии
func validateNewPrivileges(sub *user_model.User, tenantId string, org *organization.Organization, role Role) error {
	if sub == nil || org == nil {
		return fmt.Errorf("sub or org is required")
	}
	privilegesByUserID, err := GetPrivilegesByUserId(sub.ID)
	if err != nil {
		log.Error("Error has occurred while getting Privileges by userId: %d. Error: %v", sub.ID, err)
		return err
	}

	for _, privilege := range privilegesByUserID {
		if org.ID == privilege.Org.ID && role == privilege.Role {
			return &ErrRoleAlreadyExists{UserID: sub.ID, TenantID: tenantId, OrgID: org.ID, Role: role.String()}
		}
	}

	return nil
}

// removeExistingPrivilegesInOrg удаляет привилегии пользователя в проекте
func removeExistingPrivilegesInOrg(sub *user_model.User, org *organization.Organization) error {
	if sub == nil || org == nil {
		return fmt.Errorf("sub or org is required")
	}

	privileges, err := GetPrivilegesByUserId(sub.ID)
	if err != nil {
		log.Error("Error has occurred while getting Privileges by userId: %d. Error: %v", sub.ID, err)
		return err
	}

	for _, privilege := range privileges {
		if org.ID == privilege.Org.ID {
			if err := RevokeUserPermissionToOrganization(privilege.User, privilege.TenantID, privilege.Org, privilege.Role, false); err != nil {
				log.Error("Error has occurred while removing user's permission to organization")
				return fmt.Errorf("revoking user's permissions:%w", err)
			}
			break
		}
	}
	return nil
}

// GetSecurityEnforcer возвращает экземпляр casbin
func GetSecurityEnforcer() *casbin.SyncedEnforcer {
	return securityEnforcer
}
