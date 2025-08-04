package role_model

import (
	"context"
	"fmt"
	"strings"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/sbt/audit"
	"code.gitea.io/gitea/modules/setting"
)

// syncPrivilegesFromConfig - синхронизация привилегий при инициализации ролевой модели
func syncPrivilegesFromConfig(ctx context.Context) error {
	return db.WithTx(ctx, func(ctx context.Context) error {
		customGroups, err := GetAllCustomPrivilegesGroup(ctx)
		if err != nil {
			return fmt.Errorf("failed to get custom groups: %w", err)
		}

		mapCustomGroups := convertCustomGroupsToMap(customGroups)
		existingGroup := make(map[string]interface{}, len(mapCustomGroups))

		startIndex := 4 // первые 4 индекса занимает дефолтная ролевка, поэтому начинаем с 5-ого

		for code, group := range setting.SourceControlCustomGroups.CustomGroups {
			if !validateCustomGroup(code, group) {
				return fmt.Errorf("validating custom group '%s' is based sc configuration", code)
			}
			startIndex++

			if _, ok := mapCustomGroups[code]; !ok {
				if err := grantCustomPrivileges(ctx, code, group, startIndex); err != nil {
					return fmt.Errorf("granting custom privileges: %w", err)
				}
			}
			if err := updateCustomPrivileges(ctx, code, group, startIndex); err != nil {
				return fmt.Errorf("updating custom privileges: %w", err)
			}

			existingGroup[code] = nil
		}

		for key := range mapCustomGroups {
			if _, ok := existingGroup[key]; !ok {
				if err := removePrivileges(ctx, key); err != nil {
					return err
				}
			}
		}

		return nil
	})
}

// grantCustomPrivileges - выдача кастомных привилегий
func grantCustomPrivileges(ctx context.Context, code string, group setting.CustomGroup, index int) error {
	auditParams := map[string]string{
		"group_code": code,
		"name":       group.Name,
		"privileges": group.Privileges,
	}
	customPrivilege := NewScCustomPrivilegesGroup(code, group.Name, group.Privileges)
	if err := AddCustomPrivilegesGroup(ctx, customPrivilege); err != nil {
		auditParams["error"] = "Error has occurred while adding custom group"
		audit.CreateAndSendEvent(audit.AddCustomPrivilegesEvent, audit.EmptyRequiredField, audit.EmptyRequiredField, audit.StatusFailure, audit.EmptyRequiredField, auditParams)
		return err
	}
	if err := addCustomPrivilegesToPolicy(code, group, index); err != nil {
		auditParams["error"] = "Error has occurred while adding custom group to policy"
		audit.CreateAndSendEvent(audit.AddCustomPrivilegesEvent, audit.EmptyRequiredField, audit.EmptyRequiredField, audit.StatusFailure, audit.EmptyRequiredField, auditParams)
		return err
	}

	audit.CreateAndSendEvent(audit.AddCustomPrivilegesEvent, audit.EmptyRequiredField, audit.EmptyRequiredField, audit.StatusSuccess, audit.EmptyRequiredField, auditParams)
	return nil

}

// updateCustomPrivileges - обновление кастомных привилегий
func updateCustomPrivileges(ctx context.Context, code string, group setting.CustomGroup, index int) error {
	auditParams := map[string]string{
		"group_code": code,
		"name":       group.Name,
		"privileges": group.Privileges,
	}
	// Получаем политики для того, чтобы сравнить старые с новыми
	policies, err := securityEnforcer.GetFilteredGroupingPolicy(0, code)
	if err != nil {
		auditParams["error"] = "Error has occurred while getting grouping policy for role"
		audit.CreateAndSendEvent(audit.UpdateCustomPrivilegesEvent, audit.EmptyRequiredField, audit.EmptyRequiredField, audit.StatusFailure, audit.EmptyRequiredField, auditParams)
		return fmt.Errorf("failed to get grouping policy for code %s: %w", code, err)
	}

	// Получить все p-привязки, где v3 == code
	assignedPolicy, err := securityEnforcer.GetFilteredPolicy(3, code)
	if err != nil {
		auditParams["error"] = "Error has occurred while getting policy for role"
		audit.CreateAndSendEvent(audit.UpdateCustomPrivilegesEvent, audit.EmptyRequiredField, audit.EmptyRequiredField, audit.StatusFailure, audit.EmptyRequiredField, auditParams)
		return fmt.Errorf("failed to get policy for code %s: %w", code, err)
	}
	// TODO: мы должны проверить следующие случаи:
	// если есть назначения, то обновляем на старые (нужно для корректной работы при обновлении конфига); не можем просто return
	// есть нет назначений и привилегии не совпадают, то обновляем (нужно для корректной работы при обновлении конфига); не можем просто return
	// если нет назначений и привилегию различаются, то обновляем - идеальный кейс

	// Есть назначения (например, роль используется в p)
	if len(assignedPolicy) > 0 {
		if !policyContainsAllPrivileges(policies, group.Privileges) {
			auditParams["error"] = "Error has occurred while getting policy for role"
			audit.CreateAndSendEvent(audit.UpdateCustomPrivilegesEvent, audit.EmptyRequiredField, audit.EmptyRequiredField, audit.StatusFailure, audit.EmptyRequiredField, auditParams)
			return fmt.Errorf("cannot update role '%s': privileges differ from already assigned ones", code)
		}
		//  Привилегии совпадают — обновляем
		customPrivilege := NewScCustomPrivilegesGroup(code, group.Name, group.Privileges)
		if err := UpdateCustomPrivilegesGroupByCode(ctx, customPrivilege); err != nil {
			auditParams["error"] = "Error has occurred while updating privileges group"
			audit.CreateAndSendEvent(audit.UpdateCustomPrivilegesEvent, audit.EmptyRequiredField, audit.EmptyRequiredField, audit.StatusFailure, audit.EmptyRequiredField, auditParams)
			return fmt.Errorf("failed to update privileges group: %w", err)
		}
		if err := updateCustomPrivilegesToPolicy(code, group, index); err != nil {
			auditParams["error"] = "Error has occurred while updating casbin policy"
			audit.CreateAndSendEvent(audit.UpdateCustomPrivilegesEvent, audit.EmptyRequiredField, audit.EmptyRequiredField, audit.StatusFailure, audit.EmptyRequiredField, auditParams)

			return fmt.Errorf("failed to update casbin policy: %w", err)
		}

		audit.CreateAndSendEvent(audit.UpdateCustomPrivilegesEvent, audit.EmptyRequiredField, audit.EmptyRequiredField, audit.StatusSuccess, audit.EmptyRequiredField, auditParams)
		return nil
	}

	// Нет назначений — создаём
	customPrivilege := NewScCustomPrivilegesGroup(code, group.Name, group.Privileges)
	if err := UpdateCustomPrivilegesGroupByCode(ctx, customPrivilege); err != nil {
		auditParams["error"] = "Error has occurred while creating privileges group"
		audit.CreateAndSendEvent(audit.UpdateCustomPrivilegesEvent, audit.EmptyRequiredField, audit.EmptyRequiredField, audit.StatusFailure, audit.EmptyRequiredField, auditParams)
		return fmt.Errorf("failed to create privileges group: %w", err)
	}
	if err := updateCustomPrivilegesToPolicy(code, group, index); err != nil {
		auditParams["error"] = "Error has occurred while creating casbin policy"
		audit.CreateAndSendEvent(audit.UpdateCustomPrivilegesEvent, audit.EmptyRequiredField, audit.EmptyRequiredField, audit.StatusFailure, audit.EmptyRequiredField, auditParams)
		return fmt.Errorf("failed to create casbin policy: %w", err)
	}

	audit.CreateAndSendEvent(audit.UpdateCustomPrivilegesEvent, audit.EmptyRequiredField, audit.EmptyRequiredField, audit.StatusSuccess, audit.EmptyRequiredField, auditParams)
	return nil
}

func removePrivileges(ctx context.Context, key string) error {
	if err := DeleteCustomPrivilegesGroupByCode(ctx, key); err != nil {
		return err
	}
	if err := removeCustomPrivilegesToPolicy(key); err != nil {
		return err
	}
	return nil
}
func policyContainsAllPrivileges(policy [][]string, privileges string) bool {

	// Сформируем множество привилегий из политики
	policySet := make(map[string]struct{})
	for _, rule := range policy {
		if len(rule) >= 2 {
			p := strings.TrimSpace(rule[1])
			if p != "" {
				policySet[p] = struct{}{}
			}
		}
	}

	reqPrivs := strings.Split(privileges, ",")
	for _, p := range reqPrivs {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if _, ok := policySet[p]; !ok {
			return false // хотя бы одной привилегии нет
		}
	}

	return true // новые привилегии == старые привилегии
}

func convertCustomGroupsToMap(groups []*ScCustomPrivilegesGroup) map[string]*ScCustomPrivilegesGroup {
	result := make(map[string]*ScCustomPrivilegesGroup)
	for _, group := range groups {
		result[group.Code] = group
	}
	return result
}

func validateCustomGroup(code string, group setting.CustomGroup) bool {
	switch code {
	case OWNER.String():
		return false
	case MANAGER.String():
		return false
	case WRITER.String():
		return false
	case READER.String():
		return false
	case TUZ.String():
		return false
	case ViewBranch.String():
		return false
	case ChangeBranch.String():
		return false
	case CreatePR.String():
		return false
	case ApprovePR.String():
		return false
	case MergePR.String():
		return false
	}

	return true
}

func addCustomPrivilegesToPolicy(code string, group setting.CustomGroup, index int) error {
	allRoles[Role(index)] = code
	userRoles[Role(index)] = code
	userRoleNames[Role(index)] = group.Name

	addedPrivileges := make(map[string]interface{}, len(group.Privileges))
	privilegesArray := strings.Split(group.Privileges, ",")

	for _, privilege := range privilegesArray {
		if _, ok := addedPrivileges[privilege]; ok {
			log.Fatal("Privilege '%s' is duplicate in group '%s'", privilege, code)
		}

		if _, ok := GetActionByString(privilege); !ok {
			if _, ok := GetCustomPrivilegesByString(privilege); !ok {
				log.Fatal("Privilege '%s' in group '%s' not found", privilege, code)
			}
		}

		if _, err := securityEnforcer.AddGroupingPolicy(code, privilege); err != nil {
			log.Error("Error has occurred while adding grouping policy with action: %v for role: %v. Error: %v", privilege, code, err)
			return fmt.Errorf("add grouping policy: %w", err)
		}
	}

	return nil
}

func updateCustomPrivilegesToPolicy(code string, group setting.CustomGroup, index int) error {
	allRoles[Role(index)] = code
	userRoles[Role(index)] = code
	userRoleNames[Role(index)] = group.Name

	addedPrivileges := make(map[string]interface{}, len(group.Privileges))
	privilegesArray := strings.Split(group.Privileges, ",")

	for _, privilege := range privilegesArray {
		if _, ok := addedPrivileges[privilege]; ok {
			log.Fatal("Privilege '%s' is duplicate in group '%s'", privilege, code)
		}

		if _, ok := GetActionByString(privilege); !ok {
			if _, ok := GetCustomPrivilegesByString(privilege); !ok {
				log.Fatal("Privilege '%s' in group '%s' not found", privilege, code)
			}
		}

	}

	// сначала валидация, потом удаление старых привилегий, потом добавление новых
	_, err := securityEnforcer.RemoveFilteredGroupingPolicy(0, code)
	if err != nil {
		log.Error("Error has occurred while removing grouping policies for role: %v. Error: %v", code, err)
		return fmt.Errorf("remove grouping policy: %w", err)
	}

	for _, privilege := range privilegesArray {
		if _, err := securityEnforcer.AddGroupingPolicy(code, privilege); err != nil {
			log.Error("Error has occurred while adding grouping policy with action: %v for role: %v. Error: %v", privilege, code, err)
			return fmt.Errorf("add grouping policy: %w", err)
		}
	}

	return nil
}

func removeCustomPrivilegesToPolicy(code string) error {
	auditParams := map[string]string{
		"group_code": code,
	}
	policy, err := securityEnforcer.GetFilteredPolicy(3, code)
	if err != nil {
		auditParams["error"] = "Error has occurred while getting policy for role"
		audit.CreateAndSendEvent(audit.RemoveCustomPrivilegesEvent, audit.EmptyRequiredField, audit.EmptyRequiredField, audit.StatusFailure, audit.EmptyRequiredField, auditParams)
		return fmt.Errorf("failed to get policy for code %s: %w", code, err)
	}
	if len(policy) != 0 {
		auditParams["error"] = "Error has occurred while removing custom group"
		audit.CreateAndSendEvent(audit.RemoveCustomPrivilegesEvent, audit.EmptyRequiredField, audit.EmptyRequiredField, audit.StatusFailure, audit.EmptyRequiredField, auditParams)
		return fmt.Errorf("removing custom group: group is not empty")
	}

	_, err = securityEnforcer.RemoveFilteredGroupingPolicy(0, code)
	if err != nil {
		auditParams["error"] = "Error has occurred while removing grouping policies for role"
		audit.CreateAndSendEvent(audit.RemoveCustomPrivilegesEvent, audit.EmptyRequiredField, audit.EmptyRequiredField, audit.StatusFailure, audit.EmptyRequiredField, auditParams)
		log.Error("Error has occurred while removing grouping policies for role: %v. Error: %v", code, err)
		return fmt.Errorf("removing grouping policy: %w", err)
	}

	audit.CreateAndSendEvent(audit.RemoveCustomPrivilegesEvent, audit.EmptyRequiredField, audit.EmptyRequiredField, audit.StatusSuccess, audit.EmptyRequiredField, auditParams)
	return nil
}
