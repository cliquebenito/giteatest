package cache

import (
	"context"
	"errors"
	"fmt"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/role_model"
	"code.gitea.io/gitea/models/tenant"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/log"
)

var ErrTenantNotFound = errors.New("Err: tenant not exists")
var ErrProjectNotFound = errors.New("Err: project not exists")

// RequestCache структура для кэша запросов
type RequestCache struct {
	engine     db.Engine
	users      map[string]*user_model.User
	tenants    map[string]*tenant.ScTenantOrganizations
	privileges map[int64][]role_model.EnrichedPrivilege
}

// Инициализация нового кэша
func NewRequestCache(engine db.Engine) RequestCache {
	return RequestCache{
		engine:     engine,
		users:      make(map[string]*user_model.User),
		tenants:    make(map[string]*tenant.ScTenantOrganizations),
		privileges: make(map[int64][]role_model.EnrichedPrivilege),
	}
}

// GetUser получения пользователя
func (rc *RequestCache) GetUser(ctx context.Context, externalID string) (*user_model.User, error) {
	if user, exists := rc.users[externalID]; exists {
		return user, nil
	}
	// получаем login_name пользователя
	user, err := user_model.GetIAMUserByLoginName(ctx, rc.engine, externalID)
	if err != nil {
		log.Error("Error has occurred while getting user. Error: %v", err)
		return nil, fmt.Errorf("get user: %w", err)
	}
	if !user.IsActive {
		log.Debug("User is not active %s", user.Name)
		return nil, fmt.Errorf("user is not active")
	}
	rc.users[externalID] = user
	return user, nil
}

// GetTenant получения тенанта
func (rc *RequestCache) GetTenant(ctx context.Context, tenantKey, projectKey string) (*tenant.ScTenantOrganizations, error) {
	if tenantMap, exists := rc.tenants[projectKey]; exists {
		return tenantMap, nil
	}
	// получаем tenant из scTenantOrganization
	tenantMap, has, err := tenant.GetTenantOrganizationsByProjectKey(ctx, projectKey)
	if err != nil {
		log.Error("Error has occurred while getting tenant. Error: %v", err)
		return nil, fmt.Errorf("get tenant organization by project key:%s error: %w", projectKey, err)
	}
	if !has {
		log.Debug("Project with projectKey %s is exists", projectKey)
		return nil, ErrProjectNotFound
	}
	// проверяем, что тенант существует
	if tenantMap.OrgKey != tenantKey {
		log.Debug("Tenant key not found %s", tenantKey)
		return nil, ErrTenantNotFound
	}
	// проверяем, что тенант активный
	tenantStatus, err := tenant.GetTenantByID(ctx, tenantMap.TenantID)
	if err != nil {
		log.Error("Error has occurred while getting tenant. Error: %v", err)
		return nil, fmt.Errorf("get tenant by tenant ID: %s. %w", tenantMap.TenantID, err)
	}
	if !tenantStatus.IsActive {
		log.Debug("Tenant is not active:%s", tenantMap.TenantID)
		return nil, fmt.Errorf("tenant is not active")
	}
	project, err := user_model.GetUserByID(ctx, tenantMap.OrganizationID)
	if err != nil {
		log.Error("Error has occurred while getting project. Error: %v", err)
		return nil, fmt.Errorf("get tenant by org ID: %d. %w", tenantMap.OrganizationID, err)
	}
	if !project.IsActive {
		log.Debug("Project is not active:%s", project.Name)
		return nil, fmt.Errorf("project is not active")
	}
	rc.tenants[projectKey] = tenantMap
	return tenantMap, nil
}

// GetTenantByKeys получения тенанта по ключам
func (rc *RequestCache) GetTenantByKeys(ctx context.Context, tenantKey, projectKey string) (*tenant.ScTenantOrganizations, error) {
	keys := fmt.Sprintf("%s|%s", tenantKey, projectKey)
	if tenantMap, exists := rc.tenants[keys]; exists {
		return tenantMap, nil
	}
	// получаем tenant из scTenantOrganization
	tenantMap, err := tenant.GetTenantOrganizationsByKeys(ctx, tenantKey, projectKey)
	if err != nil {
		if errors.Is(err, tenant.ErrTenantOrganizationsNotExists{}) {
			log.Error("Error has occurred while getting tenant by keys, tenantKey: %s, projectKey: %s. Error: %v", tenantKey, projectKey, err)
			return nil, fmt.Errorf("get tenant organization by keys, project key :%s,tenant key:%s, error: %w", projectKey, tenantKey, err)
		}
		return nil, err
	}
	// проверяем, что тенант существует
	if tenantMap.OrgKey != tenantKey {
		log.Debug("Tenant key not found %s", tenantKey)
		return nil, ErrTenantNotFound
	}
	// проверяем, что тенант активный
	tenantStatus, err := tenant.GetTenantByID(ctx, tenantMap.TenantID)
	if err != nil {
		log.Error("Error has occurred while getting tenant. Error: %v", err)
		return nil, fmt.Errorf("get tenant by tenant ID: %s. %w", tenantMap.TenantID, err)
	}
	if !tenantStatus.IsActive {
		log.Debug("Tenant is not active:%s", tenantMap.TenantID)
		return nil, fmt.Errorf("tenant is not active")
	}
	rc.tenants[keys] = tenantMap
	return tenantMap, nil
}

// GetPrivileges получения привилегий
func (rc *RequestCache) GetPrivileges(ctx context.Context, orgID int64) ([]role_model.EnrichedPrivilege, error) {
	if privileges, exists := rc.privileges[orgID]; exists {
		return privileges, nil
	}
	privileges, err := role_model.GetPrivilegesByOrgId(orgID)
	if err != nil {
		if errors.Is(err, role_model.ErrNonExistentRole{}) {
			log.Error("Error has occurred while getting privileges. Error: %v", err)
			return nil, fmt.Errorf("get privileges: %w", err)
		}
		return nil, err
	}
	rc.privileges[orgID] = privileges
	return privileges, nil
}
