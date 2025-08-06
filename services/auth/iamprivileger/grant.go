package iamprivileger

import (
	"fmt"

	"code.gitea.io/gitea/models/organization"
	"code.gitea.io/gitea/models/role_model"
	"code.gitea.io/gitea/models/tenant"
	userModel "code.gitea.io/gitea/models/user"
	iampriveleges "code.gitea.io/gitea/modules/auth/iam/iamprivileges"
	"code.gitea.io/gitea/modules/log"
)

func (p Privileger) grantUserPermissionsToOrganization(
	user *userModel.User,
	tenant *tenant.ScTenant,
	organizationToPrivilege map[organization.Organization]iampriveleges.Privilege,
) error {
	log.Debug("iam:verify Trying to grant user permissions to organization")

	if len(organizationToPrivilege) == 0 {
		log.Debug("iam:verify No organization permissions to grant user. Exit")
		return nil
	}

	if err := role_model.RemoveExistingPrivilegesByTenantAndUserIDTx(p.casbinEnforcer, tenant.ID, user.ID); err != nil {
		return fmt.Errorf("remove existing privileges by tenant and user id: %w", err)
	}

	for org, privilege := range organizationToPrivilege {
		if err := role_model.GrantUserPermissionToOrganizationWithoutValidationTx(
			p.casbinEnforcer, user, tenant.ID, &org, privilege.Role,
		); err != nil {
			return fmt.Errorf("grant: %w", err)
		}
	}

	log.Debug("iam:verify Grant user permissions to organization successful")

	return nil
}
