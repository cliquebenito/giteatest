package project

import (
	cctx "context"
	"errors"
	"fmt"
	"strconv"

	"github.com/google/uuid"

	"code.gitea.io/gitea/models/organization"
	"code.gitea.io/gitea/models/role_model"
	tenant_model "code.gitea.io/gitea/models/tenant"
	"code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/sbt/audit"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/services/forms"
)

// CreateProject cоздание проекта под тенантом
func CreateProject(ctx context.APIContext, projectRequest forms.CreateProjectRequest) (*forms.CreateProjectResponse, error) {
	auditParams := map[string]string{
		"tenant_key":   projectRequest.TenantKey,
		"project_key":  projectRequest.ProjectKey,
		"project_name": projectRequest.Name,
	}

	u := ctx.Doer
	var userName string
	var userID string
	if u != nil {
		userName = u.Name
		userID = strconv.FormatInt(u.ID, 10)
	} else {
		userName = audit.EmptyRequiredField
		userID = audit.EmptyRequiredField
	}

	// Публичные проекты не создаем
	if setting.SourceControl.Enabled && setting.SourceControl.TenantWithRoleModeEnabled && structs.VisibleType.IsPublic(projectRequest.Visibility) {
		auditParams["error"] = "Creating public projects is forbidden"
		audit.CreateAndSendEvent(audit.ProjectCreateEvent, userName, userID, audit.StatusFailure, audit.EmptyRequiredField, auditParams)
		log.Debug("Incorrect visibility while updating project with orgName '%s'", projectRequest.Name)
		return nil, ErrVisibilityIncorrect{Visibility: projectRequest.Visibility}
	}

	tenant, has, err := tenant_model.GetTenantByOrgKey(ctx, projectRequest.TenantKey)
	if err != nil {
		auditParams["error"] = "Error has occurred while getting tenant"
		audit.CreateAndSendEvent(audit.ProjectCreateEvent, userName, userID, audit.StatusFailure, audit.EmptyRequiredField, auditParams)
		log.Error("Error has occurred while getting tenant by orgKey '%s'. Error: %v", projectRequest.TenantKey, err)
		return nil, fmt.Errorf("failed to get tenant by orgKey %s: %w", projectRequest.TenantKey, err)
	}

	if !tenant.IsActive {
		auditParams["error"] = "Error has occurred while checking tenant, tenant is inactive"
		audit.CreateAndSendEvent(audit.ProjectCreateEvent, userName, userID, audit.StatusFailure, audit.EmptyRequiredField, auditParams)
		log.Debug("Tenant is inactive")
		return nil, tenant_model.ErrTenantNotActive{TenantKey: projectRequest.TenantKey}
	}

	if !has {
		auditParams["error"] = "Error has occurred while checking tenant, tenant not exists"
		audit.CreateAndSendEvent(audit.ProjectCreateEvent, userName, userID, audit.StatusFailure, audit.EmptyRequiredField, auditParams)
		log.Debug("Tenant not exists by orgKey '%s'", projectRequest.TenantKey)
		return nil, tenant_model.ErrTenantKeyNotExists{TenantKey: projectRequest.TenantKey}
	}

	// Проверка проекта на существование
	_, has, err = tenant_model.GetTenantOrganizationsByProjectKey(ctx, projectRequest.ProjectKey)
	if err != nil {
		auditParams["error"] = "Error has occurred while getting tenant organization"
		audit.CreateAndSendEvent(audit.ProjectCreateEvent, userName, userID, audit.StatusFailure, audit.EmptyRequiredField, auditParams)
		log.Error("Error has occurred while getting tenant organization by projectKey '%s'. Error: %v", projectRequest.ProjectKey, err)
		return nil, err
	}
	if has {
		auditParams["error"] = "Error has occurred while checking project, project is exists"
		audit.CreateAndSendEvent(audit.ProjectCreateEvent, userName, userID, audit.StatusFailure, audit.EmptyRequiredField, auditParams)
		log.Debug("Project with projectKey '%s' is exists", projectRequest.ProjectKey)
		return nil, tenant_model.ErrProjectKeyAlreadyUsed{ProjectKey: projectRequest.ProjectKey}
	}

	// Создание проекта под тенантом
	org := &organization.Organization{
		Name:        projectRequest.Name,
		Description: projectRequest.Description,
		Visibility:  projectRequest.Visibility,
		IsActive:    true,
	}

	createdOrg, err := createOrg(ctx, org, projectRequest.TenantKey, projectRequest.ProjectKey)
	if err != nil {
		audit.CreateAndSendEvent(audit.ProjectCreateEvent, userName, userID, audit.StatusFailure, audit.EmptyRequiredField, auditParams)
		log.Error("Error has occurred while creating organization with name '%s' projectKey '%s' and orgKey '%s'. Error: %v", projectRequest.Name, projectRequest.ProjectKey, projectRequest.TenantKey, err)
		return nil, fmt.Errorf("create organizarion with name '%s', projectKey '%s' and orgKey '%s': %w", projectRequest.Name, projectRequest.ProjectKey, projectRequest.TenantKey, err)
	}
	audit.CreateAndSendEvent(audit.ProjectCreateEvent, userName, userID, audit.StatusSuccess, audit.EmptyRequiredField, auditParams)

	// Добавление в _casbin_rule
	if setting.SourceControl.Enabled && setting.SourceControl.TenantWithRoleModeEnabled {
		if structs.VisibleType.IsLimited(org.Visibility) {
			if err = role_model.AddProjectToInnerSource(createdOrg); err != nil {
				log.Error("Error has occurred while adding project to inner source: %v", err)
				return nil, fmt.Errorf("add project to inner source: %w", err)
			}
		}

		if err = role_model.GrantUserPermissionToOrganization(ctx.Doer, tenant.ID, org, role_model.OWNER); err != nil {
			log.Error("Error has occurred while graining user permission to organization: %v", err)
			return nil, fmt.Errorf("create project: %w", err)
		}
	}

	return &forms.CreateProjectResponse{
		Id:         createdOrg.ID,
		Name:       createdOrg.Name,
		ProjectKey: projectRequest.ProjectKey,
		Visibility: createdOrg.Visibility,
		Uri:        fmt.Sprintf("/%s", createdOrg.Name),
	}, nil
}

// GetProject получение проекта по tenantKey и projectKey
func GetProject(ctx context.APIContext, tenantKey, projectKey string) (forms.ProjectInfoResponse, error) {
	tenantOrganization, err := tenant_model.GetTenantOrganizationsByKeys(ctx, tenantKey, projectKey)
	if err != nil {
		if errors.Is(err, tenant_model.ErrTenantOrganizationsNotExists{}) {
			log.Error("Error has occurred while getting tenant by keys, tenantKey: %s, projectKey: %s. Error: %v", tenantKey, projectKey, err)
			return forms.ProjectInfoResponse{}, fmt.Errorf("get tenant organization by keys, project key :%s,tenant key:%s, error: %w", projectKey, tenantKey, err)
		}
		log.Error("Error has occurred while getting tenant organizations by projectKey '%s'. Error: %v", projectKey, err)
		return forms.ProjectInfoResponse{}, err
	}
	if tenantOrganization == nil {
		log.Debug("Organization not found by projectID '%s'", projectKey)
		return forms.ProjectInfoResponse{}, err
	}
	organizations, err := organization.GetOrganizationByIDs(ctx, []int64{tenantOrganization.OrganizationID})
	if err != nil {
		log.Error("Error has occurred while getting organizations by IDs '%v'. Error: %v", tenantOrganization.OrganizationID, err)
		return forms.ProjectInfoResponse{}, err
	}
	if len(organizations) == 0 {
		log.Debug("Organization not found by projectKey '%s'", projectKey)
		return forms.ProjectInfoResponse{}, fmt.Errorf("Organization not found")
	}
	return forms.ProjectInfoResponse{
		Id:         organizations[0].ID,
		Name:       organizations[0].Name,
		ProjectKey: projectKey,
		Visibility: organizations[0].Visibility,
		Uri:        fmt.Sprintf("/%s", organizations[0].Name),
	}, nil
}

// createOrg создание проекта и tenantOrganization
func createOrg(ctx context.APIContext, org *organization.Organization, orgKey, projectKey string) (*organization.Organization, error) {
	tenant, has, err := tenant_model.GetTenantByOrgKey(ctx, orgKey)
	if err != nil {
		return nil, err
	}
	if !has {
		log.Debug("Tenant not found by orgKey '%s'", orgKey)
		return nil, tenant_model.ErrTenantByKeysNotExists{OrgKey: orgKey, ProjectKey: projectKey}
	}
	userAdmin, err := user.GetAdminUser()
	if err != nil {
		log.Error("Error has occurred while getting admin user. Error: %v", err)
		return nil, err
	}
	err = organization.CreateOrganization(org, userAdmin)
	if err != nil {
		if user.IsErrUserAlreadyExist(err) {
			log.Error("Error has occurred while creating organization. Error: %v", err)
			return nil, fmt.Errorf("%w", ErrProjectNameAlreadyUsed{Name: org.Name})
		}
		log.Error("Error has occurred while creating organization. Error: %v", err)
	}
	tenantOrganization := &tenant_model.ScTenantOrganizations{
		ID:             uuid.NewString(),
		TenantID:       tenant.ID,
		OrganizationID: org.ID,
		OrgKey:         orgKey,
		ProjectKey:     projectKey,
	}
	err = tenant_model.InsertTenantOrganization(ctx, tenantOrganization)
	if err != nil {
		log.Error("Error has occurred while inserting tenant organization. Error: %v", err)
		return nil, err
	}
	return org, nil
}

// GetProjectByKeys получение проекта по tenantKey и projectKey
func GetProjectByKeys(ctx cctx.Context, tenantKey, projectKey string) (*user.User, error) {
	tenantOrganization, err := tenant_model.GetTenantOrganizationsByKeys(ctx, tenantKey, projectKey)
	if err != nil {
		if errors.Is(err, tenant_model.ErrTenantOrganizationsNotExists{}) {
			log.Error("Error has occurred while getting tenant by keys, tenantKey: %s, projectKey: %s. Error: %v", tenantKey, projectKey, err)
			return nil, fmt.Errorf("get tenant organization by keys, project key :%s,tenant key:%s, error: %w", projectKey, tenantKey, err)
		}
		log.Error("Error has occurred while getting tenant organizations by projectKey '%s'. Error: %v", projectKey, err)
		return nil, err
	}
	if tenantOrganization == nil {
		log.Debug("Organization not found by projectID '%s'", projectKey)
		return nil, err
	}
	organizations, err := organization.GetOrganizationByIDs(ctx, []int64{tenantOrganization.OrganizationID})
	if err != nil {
		log.Error("Error has occurred while getting organizations by IDs '%v'. Error: %v", tenantOrganization.OrganizationID, err)
		return nil, err
	}
	if len(organizations) == 0 {
		log.Debug("Organization not found by projectKey '%s'", projectKey)
		return nil, fmt.Errorf("Err: organization not found [project_key: %s]", projectKey)
	}
	return mapOrganizationToUser(organizations[0]), nil
}
func mapOrganizationToUser(org *organization.Organization) *user.User {
	if org == nil {
		return nil
	}

	return (*user.User)(org)
}
