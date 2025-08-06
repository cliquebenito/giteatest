package role_model

import (
	"context"
	"fmt"
	"strconv"

	"github.com/casbin/casbin/v2"

	"code.gitea.io/gitea/models"
	"code.gitea.io/gitea/models/organization"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/log"
)

// RemoveExistingPrivilegesByTenantAndUserIDTx удаляет привилегии в тенанте по пользователю
func RemoveExistingPrivilegesByTenantAndUserIDTx(enforcer casbin.IEnforcer, tenantID string, userID int64) error {
	privileges, err := GetPrivilegesByUserId(userID)
	if err != nil {
		log.Error("Error has occurred while getting privileges by userId: %d. Error: %v", userID, err)
		return fmt.Errorf("get privileges by userId: %d. Error: %w", userID, err)
	}
	for _, privilege := range privileges {
		if tenantID == privilege.TenantID {
			if err := RevokeUserPermissionToOrganizationTx(enforcer, privilege.User, privilege.TenantID, privilege.Org, privilege.Role); err != nil {
				return fmt.Errorf("revoke user permission to organization: %w", err)
			}
		}
	}
	return nil
}

// RemoveExistingPrivilegesByTenantAndOrgID удаляет привилегии в тенанте по проекту
func RemoveExistingPrivilegesByTenantAndOrgID(tenantID string, orgID int64) error {
	privileges, err := GetPrivilegesByTenant(tenantID)
	if err != nil {
		log.Error("Error has occurred while getting privileges: %v", err)
		return err
	}
	for _, privilege := range privileges {
		if orgID == privilege.Org.ID {
			if err := RevokeUserPermissionToOrganization(privilege.User, privilege.TenantID, privilege.Org, privilege.Role, true); err != nil {
				log.Error("Error has occurred while revoking permission: %v", err)
				return err
			}
		}
	}
	return nil
}

// RevokeUserPermissionToOrganizationTx снимает с пользователя роль в проекте под тенантом
func RevokeUserPermissionToOrganizationTx(enforcer casbin.IEnforcer, sub *user_model.User, tenantId string, org *organization.Organization, role Role) error {
	if err := models.RemoveOrgUser(org.ID, sub.ID); err != nil {
		log.Error("Error has occurred while removing userId: %d from orgId: %d. Error: %v", sub.ID, org.ID, err)
		return fmt.Errorf("remove user from org: %w", err)
	}
	if _, err := enforcer.RemovePolicy(strconv.FormatInt(sub.ID, 10), tenantId, strconv.FormatInt(org.ID, 10), role.String()); err != nil {
		log.Error("Error has occurred while removing %v policy to projectId: %d for userId: %d under tenantId: %v. Error: %v", role.String(), org.ID, sub.ID, tenantId, err)
		return fmt.Errorf("remove policy: %v", err)
	}

	log.Debug("%v policy to projectId: %d for userId: %d under tenantId: %v successful revoked", role.String(), org.ID, sub.ID, tenantId)
	return nil
}

// GrantUserPermissionToOrganization назначает пользователю роль в проекте под тенантом
func GrantUserPermissionToOrganizationTx(enforcer casbin.IEnforcer, sub *user_model.User, tenantId string, org *organization.Organization, role Role) error {
	if err := validateNewPrivileges(sub, tenantId, org, role); err != nil {
		log.Error("Error has occurred while validating new privileges. Error: %v", err)
		return fmt.Errorf("validate new Privileges: %v", err)
	}

	return grantUserPermissionToOrganizationTx(enforcer, sub, tenantId, org, role)
}

// GrantUserPermissionToOrganizationWithoutValidationTx назначает пользователю роль в проекте под тенантом в транзакции без валидации
func GrantUserPermissionToOrganizationWithoutValidationTx(enforcer casbin.IEnforcer, sub *user_model.User, tenantId string, org *organization.Organization, role Role) error {
	return grantUserPermissionToOrganizationTx(enforcer, sub, tenantId, org, role)
}

// grantUserPermissionToOrganization назначает пользователю роль в проекте под тенантом
func grantUserPermissionToOrganizationTx(enforcer casbin.IEnforcer, sub *user_model.User, tenantId string, org *organization.Organization, role Role) error {
	if err := removeExistingPrivilegesInOrgTx(enforcer, sub, org); err != nil {
		log.Error("Error has occurred while removing userId: %d privileges in orgId: %d. Error: %v", sub.ID, org.ID, err)
		return fmt.Errorf("remove existing privileges in org: %v", err)
	}

	team, err := organization.GetOwnerTeam(context.Background(), org.ID)
	if err != nil {
		log.Error("Error has occurred while getting owner team for orgId: %d. Error: %v", org.ID, err)
		return fmt.Errorf("get owner team for org: %v", err)
	}

	if err := models.AddTeamMember(team, sub.ID); err != nil {
		log.Error("Error has occurred while adding teamId: %d member userId: %d. Error: %v", team.ID, sub.ID, err)
		return fmt.Errorf("add team member: %v", err)
	}

	if _, err := enforcer.AddPolicy(strconv.FormatInt(sub.ID, 10), tenantId, strconv.FormatInt(org.ID, 10), role.String()); err != nil {
		log.Error("Error has occurred while adding %v policy to projectId: %d for userId: %d under tenantId: %v. Error: %v", role.String(), org.ID, sub.ID, tenantId, err)
		return fmt.Errorf("add policy: %v", err)
	}

	log.Debug("%v policy to projectId: %d for userId: '%d' under tenantId: %v successful granted", role.String(), org.ID, sub.ID, tenantId)
	return nil
}

// removeExistingPrivilegesInOrg удаляет привилегии пользователя в проекте
func removeExistingPrivilegesInOrgTx(enforcer casbin.IEnforcer, sub *user_model.User, org *organization.Organization) error {
	privileges, err := GetPrivilegesByUserId(sub.ID)
	if err != nil {
		log.Error("Error has occurred while getting privileges by userId: %d. Error: %v", sub.ID, err)
		return fmt.Errorf("get Privileges by userId: %v", err)
	}

	for _, privilege := range privileges {
		if org.ID == privilege.Org.ID {
			if err := RevokeUserPermissionToOrganizationTx(enforcer, privilege.User, privilege.TenantID, privilege.Org, privilege.Role); err != nil {
				return fmt.Errorf("revoke user permission to org: %v", err)
			}
			break
		}
	}
	return nil
}
