package iamprivileger

import (
	"context"
	"fmt"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/organization"
	"code.gitea.io/gitea/models/role_model"
	"code.gitea.io/gitea/models/tenant"
	userModel "code.gitea.io/gitea/models/user"
	iampriveleges "code.gitea.io/gitea/modules/auth/iam/iamprivileges"
	"code.gitea.io/gitea/modules/auth/iam/iamtoken"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/services/casbingormadapter"
	"github.com/casbin/casbin/v2"
)

type Privileger struct {
	casbinEnforcer  casbin.IEnforcer
	casbinDBAdapter *casbingormadapter.Adapter
	engine          db.Engine
}

func New(casbinEnforcer casbin.IEnforcer, engine db.Engine) (Privileger, error) {
	rawAdapter := casbinEnforcer.GetAdapter()

	adapter, ok := rawAdapter.(*casbingormadapter.Adapter)
	if !ok {
		return Privileger{}, fmt.Errorf("get casbin adapter: can not cast %T to gorm adapter", rawAdapter)
	}

	return Privileger{
		casbinEnforcer:  casbinEnforcer,
		casbinDBAdapter: adapter,
		engine:          engine,
	}, nil
}

func (p Privileger) ApplyPrivileges(
	ctx context.Context,
	user *userModel.User,
	token iamtoken.IAMJWT,
	privileges iampriveleges.SourceControlPrivilegesByTenant,
) error {
	tenantName := token.TenantName

	log.Debug(fmt.Sprintf("ApplyPrivileges: %+v\n", privileges))
	log.Debug(fmt.Sprintf("Token: %+v\n", token))
	log.Debug(fmt.Sprintf("User: %+v\n", user))

	scTenant, err := p.getTenantByName(ctx, tenantName)
	if err != nil {
		log.Error("Error has occurred while getting tenant by name %v", err)
		return fmt.Errorf("get tenant by name: %w", err)
	}

	if scTenant == nil {
		return nil
	}

	tenantOrganizations, err := p.getOrganizationsByTenant(ctx, scTenant, privileges)
	if err != nil {
		log.Error("Error has occurred while getting orgs by name %v", err)
		return fmt.Errorf("get orgs by name: %w", err)
	}

	tenantPrivileges := privileges[scTenant.OrgKey]
	organizationToPrivileges := p.getOrganizationToPrivileges(tenantOrganizations, tenantPrivileges)

	if err = p.grantUserPermissionsToOrganization(
		user,
		scTenant,
		organizationToPrivileges,
	); err != nil {
		if role_model.IsErrRoleAlreadyExists(err) || organization.IsErrLastOrgOwner(err) {
			return nil
		}

		log.Error("Error has occurred while granting permission to organization user %v", err)
		return fmt.Errorf("grant user %s permission to organization %s: %w", user.Name, scTenant.Name, err)
	}

	return nil
}

func (p Privileger) getOrganizationToPrivileges(
	organizations []*organization.Organization,
	privileges iampriveleges.Privileges,
) map[organization.Organization]iampriveleges.Privilege {

	orgToPrivilegeForProject := make(map[string]iampriveleges.Privileges)

	for _, privilege := range privileges {
		orgToPrivilegeForProject[privilege.ProjectName] = append(
			orgToPrivilegeForProject[privilege.ProjectName],
			privilege,
		)
	}

	orgToMaxPrivilegeForProject := make(map[string]iampriveleges.Privilege)

	for projectName, privilegesForProject := range orgToPrivilegeForProject {
		maxPrivilege, err := privilegesForProject.GetMaxPrivilege()
		if err != nil {
			log.Debug(fmt.Sprintf("Failed to get max privilege for project %s: %v", projectName, err))
			continue
		}

		orgToMaxPrivilegeForProject[maxPrivilege.ProjectName] = maxPrivilege
	}

	orgToPrivileges := make(map[organization.Organization]iampriveleges.Privilege)

	for _, org := range organizations {
		maxPrivilege, exists := orgToMaxPrivilegeForProject[org.LowerName]
		if !exists {
			log.Debug(fmt.Sprintf("Organization %s has no max privilege for project %d", org.Name, org.ID))
			continue
		}

		orgToPrivileges[*org] = maxPrivilege
	}

	return orgToPrivileges
}

func (p Privileger) getTenantByName(ctx context.Context, tenantName string) (*tenant.ScTenant, error) {
	if tenantName == "" {
		return nil, nil
	}

	tenants, err := tenant.GetTenants(ctx)
	if err != nil {
		return nil, fmt.Errorf("get tenants: %w", err)
	}

	if len(tenants) == 0 {
		return nil, NewErrTenantNotFound(tenantName, fmt.Errorf("tenant not found"))
	}

	for _, tenantObj := range tenants {
		log.Debug(fmt.Sprintf("tenantObj: %+v\n", tenantObj))
		if tenantName == tenantObj.Name {
			return tenantObj, nil
		}
	}

	return nil, NewErrTenantNotFound(tenantName, fmt.Errorf("tenant not found"))
}

// getOrganizationsByTenant метод отдает организации по тенанту
func (p Privileger) getOrganizationsByTenant(
	ctx context.Context,
	scTenant *tenant.ScTenant,
	privileges iampriveleges.SourceControlPrivilegesByTenant,
) ([]*organization.Organization, error) {
	if scTenant == nil {
		return nil, nil
	}

	// сперва находим все активные организации
	allOrg, err := organization.GetAllActiveOrganization(ctx)
	if err != nil {
		return nil, fmt.Errorf("get all organizations: %w", err)
	}

	if len(allOrg) == 0 {
		return nil, NewErrorOrganizationNotFound("", fmt.Errorf("organizations not found"))
	}

	log.Debug(fmt.Sprintf("Error has occurred while getting all active organization: %+v\n", allOrg))

	// поиск идет по org_key, ибо он не меняется в отличии от tenant_name
	log.Debug(fmt.Sprintf("Error has occurred while getting organizations by tenant: scTenant.OrgKey: %+v\n", scTenant.OrgKey))
	projectNames, err := privileges.UniqProjectNamesByTenantName(scTenant.OrgKey)
	if err != nil {
		log.Error("Error has occurred while getting unique project names by tenant name %v", err)
		return nil, fmt.Errorf("get unique project names by tenant name %s: %w", scTenant.Name, err)
	}

	log.Debug(fmt.Sprintf("Error has occurred while getting organizations by tenant: projectNames: %+v\n", projectNames))

	organizations := make([]*organization.Organization, 0)
	for _, projectName := range projectNames {
		for _, orgObj := range allOrg {
			if orgObj.Name == projectName {
				organizations = append(organizations, orgObj)
			}
		}
	}

	if len(organizations) == 0 {
		return nil, NewErrorOrganizationNotFound("", fmt.Errorf("organizations %v not found", projectNames))
	}

	log.Debug(fmt.Sprintf("Error has occurred while getting organizations by tenant: organizations: %+v\n", organizations))
	return organizations, nil
}
