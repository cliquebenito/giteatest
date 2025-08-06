package custom_casbin_role_manager

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"code.gitea.io/gitea/modules/trace"
	"github.com/casbin/casbin/v2"

	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/role_model"
	"code.gitea.io/gitea/modules/log"
)

type manager struct {
	casbinEnforcer casbin.IEnforcer
}

func NewManager(casbinEnforcer casbin.IEnforcer) *manager {
	return &manager{
		casbinEnforcer: casbinEnforcer,
	}
}

func (m *manager) CheckCustomPrivileges(ctx context.Context, privilege ConfCustomPrivilege) (bool, error) {
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

	if privilege.Org == nil || privilege.User == nil {
		return false, fmt.Errorf("invalid input: user or organization is nil")
	}

	return role_model.CheckUserPermissionToTeam(ctx, privilege.User, privilege.TenantID, privilege.Org, &repo_model.Repository{ID: privilege.RepoID}, privilege.CustomPrivilege)
}

// GrantCustomPrivilegeTeamUser метод для назначения кастомных привилегий пользователю в проекте и определенной команде
func (m *manager) GrantCustomPrivilegeTeamUser(privilege GrantCustomPrivilege) error {
	if privilege.User == nil || privilege.Org == nil || privilege.Team == nil {
		log.Error("invalid input: user or team or org is nil, user:%v,org:%v,team:%v", privilege.User.LowerName, privilege.Org.LowerName, privilege.Team.Name)
		return fmt.Errorf("invalid input: user or organization or team is nil")
	}
	if _, err := m.casbinEnforcer.AddNamedPolicy("p4", strconv.FormatInt(privilege.User.ID, 10), privilege.TenantID, strconv.FormatInt(privilege.Org.ID, 10), privilege.Team.Name); err != nil {
		log.Error("Error has occurred while adding policy for userID: %v, tenantID: %v, projectID: %v, teamName: %v. Error: %v", privilege.User.ID, privilege.TenantID, privilege.Org.ID, privilege.Team.Name, err)
		return fmt.Errorf("add policy: %w", err)
	}
	if err := m.casbinEnforcer.SavePolicy(); err != nil {
		log.Error("Error has occurred while adding policy for userID: %v, tenantID: %v, projectID: %v, teamName: %v. Error: %v", privilege.User.ID, privilege.TenantID, privilege.Org.ID, privilege.Team.Name, err)
		return fmt.Errorf("save custom policies: %w", err)
	}
	return nil
}

func (m *manager) GetCustomPrivilegesForUser(orgID, userID, tenant, teamName string) bool {
	hasNamedPolicy, err := m.casbinEnforcer.HasNamedPolicy("p4", userID, tenant, orgID, teamName)
	if err != nil {
		log.Error("Error has occurred while checking custom privileges for userID: %v, tenantID: %v, projectID: %v, teamName: %v. Error: %v", userID, tenant, orgID, teamName, err)
		return false
	}
	return hasNamedPolicy
}

func (m *manager) RemoveUserFromTeamCustomPrivilege(privilege GrantCustomPrivilege) error {
	if privilege.Org == nil || privilege.Team == nil || privilege.User == nil {
		return fmt.Errorf("invalid input: user or organization or team is required")
	}
	if _, err := m.casbinEnforcer.RemoveNamedPolicy("p4", strconv.FormatInt(privilege.User.ID, 10), privilege.TenantID, strconv.FormatInt(privilege.Org.ID, 10), privilege.Team.Name); err != nil {
		log.Error("Error has occurred while removing policy for userID: %v, tenantID: %v, projectID: %v, teamName: %v. Error: %v", privilege.User.ID, privilege.TenantID, privilege.Org.ID, privilege.Team.Name, err)
		return fmt.Errorf("removing custom policies: %w", err)
	}
	if err := m.casbinEnforcer.SavePolicy(); err != nil {
		log.Error("Error has occurred while removing policy for userID: %v, tenantID: %v, projectID: %v, teamName: %v. Error: %v", privilege.User.ID, privilege.TenantID, privilege.Org.ID, privilege.Team.Name, err)
		return fmt.Errorf("save custom policies: %w", err)
	}
	return nil
}

func (m *manager) RemoveCustomPrivileges(teamName string) error {
	oldPolicies, err := m.casbinEnforcer.GetFilteredNamedPolicy("p5", 0, teamName)
	if err != nil {
		log.Error("Error has occurred while getting custom policies for team %s: %v", teamName, err)
		return fmt.Errorf("getting policies: %w", err)
	}

	if len(oldPolicies) == 0 {
		return nil
	}

	for _, policy := range oldPolicies {
		if _, err := m.casbinEnforcer.RemoveNamedPolicy("p5", policy[0], policy[1], policy[2], policy[3]); err != nil {
			log.Error("Error has occurred while removing custom policy for team: %v, projectID: %v, repoID: %v, branch: %v. Error: %v", policy[0], policy[1], policy[2], policy[3], err)
			return fmt.Errorf("removing policies: %w", err)
		}
	}

	if err := m.casbinEnforcer.SavePolicy(); err != nil {
		log.Error("Error has occurred while creating custom privileges for team: %v, projectID: %v, repoID: %v, branch: %v. Error: %v", err)
		return fmt.Errorf("saving policies: %w", err)
	}
	return nil
}

// CheckGroupingPolicy проверяем наличие групповых политик для переданного поля
func (m *manager) CheckGroupingPolicy(fieldIndex int, fieldName string) bool {
	groupingPrivileges, err := m.casbinEnforcer.GetFilteredNamedGroupingPolicy("g3", fieldIndex, fieldName)
	if err != nil {
		log.Error("Error has occurred while getting custom grouping privileges for teams: %v. Error: %v", fieldName, err)
		return false
	}
	return len(groupingPrivileges) > 0
}

// RemoveCustomPrivilegesByParams удаляем p4 групповые привилегии по переданным параметрам
func (m *manager) RemoveCustomPrivilegesByParams(fieldIndex int, fieldName string) error {
	if _, err := m.casbinEnforcer.RemoveFilteredNamedPolicy("p4", fieldIndex, fieldName); err != nil {
		log.Error("Error has occurred while removing custom privileges for team: %v, projectID: %v, repoID: %v, branch: %v. Error: %v", err)
		return fmt.Errorf("removing filtering policies: %w", err)
	}

	if err := m.casbinEnforcer.SavePolicy(); err != nil {
		log.Error("Error has occurred while removing custom privileges for team: %v, projectID: %v, repoID: %v, branch: %v. Error: %v", err)
		return fmt.Errorf("saving policies: %w", err)
	}
	return nil
}

// RemoveExistingPrivilegesByTenantAndOrgID удаляет привилегии в тенанте по проекту
func (m *manager) RemoveExistingPrivilegesByTenantAndOrgID(tenantID string, orgID int64) error {
	privileges, err := role_model.GetPrivilegesByTenant(tenantID)
	if err != nil {
		log.Error("Error has occurred while getting privileges: %v", err)
		return err
	}
	for _, privilege := range privileges {
		if orgID == privilege.Org.ID {
			if err := role_model.RevokeUserPermissionToOrganization(privilege.User, privilege.TenantID, privilege.Org, privilege.Role, true); err != nil {
				log.Error("Error has occurred while revoking permission: %v", err)
				return err
			}
		}
	}
	return nil
}

func (m *manager) RemoveCustomPrivilegesByArrays(removeCustomPrivileges [][]string) error {
	if _, err := m.casbinEnforcer.RemoveNamedPolicies("p5", removeCustomPrivileges); err != nil {
		log.Error("Error has occurred while removing p5 policies: %v", err)
		return fmt.Errorf("removing named policies: %w", err)
	}

	if err := m.casbinEnforcer.SavePolicy(); err != nil {
		log.Error("Error has occurred while saving policies: %v", err)
		return fmt.Errorf("saving policies: %w", err)
	}
	return nil
}

// CreateCustomGroupingPrivileges создаем групповые политики для команды под проектом +++
func (m *manager) CreateCustomGroupingPrivileges(name string, groupingPolicies []string) error {
	for _, groupPolicy := range groupingPolicies {
		if _, err := m.casbinEnforcer.AddNamedGroupingPolicy("g3", name, groupPolicy); err != nil {
			log.Error("Error has occurred while creating custom grouping Privileges for teams: %v. Error: %v", groupingPolicies, err)
			return fmt.Errorf("adding name policies: %w", err)
		}
	}

	if err := m.casbinEnforcer.SavePolicy(); err != nil {
		log.Error("Error has occurred while creating custom grouping Privileges for teams: %v. Error: %v", groupingPolicies, err)
		return fmt.Errorf("saving policies: %w", err)
	}

	return nil
}

func (m *manager) UpdateCustomPrivileges(newPolicies [][]string) error {

	if len(newPolicies) == 0 {
		return nil
	}
	existsPrivileges, err := m.getAllCustomPrivilegesByIndex(0, newPolicies[0][0])
	if err != nil {
		log.Error("Error has occurred while getting: %v", err)
		return err
	}
	// customPolicy структура для фильтрации существующих привилегий
	type customPolicy struct {
		teamName  string
		projectID string
		repoID    string
	}
	// фильтруем уже существующие кастомные привилегии
	if len(existsPrivileges) > 0 {
		filteredNewPolicies := make([][]string, 0)
		uniquePolicies := make(map[customPolicy]string)
		for idx := range existsPrivileges {
			policyExist := existsPrivileges[idx]
			// проверка на длину 4, так как длина записи в casbine, policy 4 содержит 4 поля
			if len(policyExist) < 4 {
				continue
			}

			existPolicy := customPolicy{
				teamName:  policyExist[0],
				projectID: policyExist[1],
				repoID:    policyExist[2],
			}

			uniquePolicies[existPolicy] = policyExist[4]
		}

		for idx := range newPolicies {
			newPolicy := newPolicies[idx]
			// проверка на длину 4, так как длина записи в casbine, policy 4 содержит 4 поля
			if len(newPolicy) < 4 {
				continue
			}

			policy := customPolicy{
				teamName:  newPolicy[0],
				projectID: newPolicy[1],
				repoID:    newPolicy[2],
			}

			if oldCustomPrivileges, ok := uniquePolicies[policy]; ok {
				newPolicy[3] = convertConflictPolices(oldCustomPrivileges, newPolicy[3])
				if _, err := m.casbinEnforcer.RemoveNamedPolicy("p5", newPolicy[0], newPolicy[1], newPolicy[2], oldCustomPrivileges); err != nil {
					log.Error("Error has occurred while removing custom policyExist for team: %v, projectID: %v, repoID: %v, branch: %v, value: %v. Error: %v", newPolicy[0], newPolicy[1], newPolicy[2], newPolicy[3], oldCustomPrivileges, err)
					return fmt.Errorf("removing policies: %w", err)
				}
				if err := m.casbinEnforcer.SavePolicy(); err != nil {
					log.Error("Error has occurred while creating custom privileges for team: %v, projectID: %v, repoID: %v, branch: %v. Error: %v", err)
					return fmt.Errorf("saving policies: %w", err)
				}
			}
			filteredNewPolicies = append(filteredNewPolicies, newPolicy)
		}
		newPolicies = filteredNewPolicies
	}
	for _, policy := range newPolicies {
		if _, err := m.casbinEnforcer.AddNamedPolicy("p5", policy[0], policy[1], policy[2], policy[3]); err != nil {
			log.Error("Error has occurred while adding custom policy for team: %v, projectID: %v, repoID: %v, branch: %v. Error: %v", policy[0], policy[1], policy[2], policy[3], err)
			return fmt.Errorf("adding policies: %w", err)
		}
	}

	if err := m.casbinEnforcer.SavePolicy(); err != nil {
		log.Error("Error has occurred while creating custom privileges for team: %v, projectID: %v, repoID: %v, branch: %v. Error: %v", err)
		return fmt.Errorf("saving policies: %w", err)
	}
	return nil
}

// getAllCustomPrivilegesByIndex получаем p4 по индексу и название поля
func (m *manager) getAllCustomPrivilegesByIndex(fieldIndex int, fieldName string) ([][]string, error) {
	allCustomPolicies, err := m.casbinEnforcer.GetFilteredNamedPolicy("p4", fieldIndex, fieldName)
	if err != nil {
		log.Error("Error has occurred while getting p4 custom privileges: %v", err)
		return nil, err
	}
	return allCustomPolicies, nil
}

func convertConflictPolices(old, new string) string {
	oldCustomPrivileges := strings.Split(old, "_")
	newCustomPrivileges := strings.Split(new, "_")

	uniqueCustomPrivileges := make(map[string]struct{}, len(oldCustomPrivileges))
	for idx := range oldCustomPrivileges {
		uniqueCustomPrivileges[oldCustomPrivileges[idx]] = struct{}{}
	}

	for idx := range newCustomPrivileges {
		uniqueCustomPrivileges[newCustomPrivileges[idx]] = struct{}{}
	}

	newCustom := make([]role_model.CustomPrivilege, 0, len(uniqueCustomPrivileges))
	for privilege := range uniqueCustomPrivileges {
		newCustom = append(newCustom, role_model.PolicyOfNames[privilege])
	}
	return role_model.ConvertCustomPrivilegeToNameOfPolicy(newCustom)
}
