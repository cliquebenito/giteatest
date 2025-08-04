package admin

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"

	"github.com/google/uuid"

	"code.gitea.io/gitea/models/organization"
	repo_model "code.gitea.io/gitea/models/repo"
	tenat_model "code.gitea.io/gitea/models/tenant"
	"code.gitea.io/gitea/modules/base"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/json"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/sbt/audit"
	auditutils "code.gitea.io/gitea/modules/sbt/audit/utils"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/timeutil"
	org_service "code.gitea.io/gitea/services/org"
	"code.gitea.io/gitea/services/repository"
	tenant_service "code.gitea.io/gitea/services/tenant"
)

const (
	tplGetTenants base.TplName = "admin/tenant/list"
)

var (
	// регулярка для проверки имени тенанта при создании и редактировании
	re = regexp.MustCompile(`^[a-zA-Z0-9-_.]{1,50}$`)
)

type TenantWithProjects struct {
	ID        string                       `json:"id"`
	Name      string                       `json:"name"`
	Default   bool                         `json:"default"`
	IsActive  bool                         `json:"is_active"`
	CreatedAt timeutil.TimeStamp           `json:"created_at"`
	UpdatedAt timeutil.TimeStamp           `json:"update_at"`
	Projects  []*organization.Organization `json:"projects"`
}

// Tenants отрисовываем начальный шаблон
func Tenants(ctx *context.Context) {
	ctx.Data["PageIsAdminTenants"] = true
	ctx.Data["Title"] = ctx.Tr("admin.tenants")

	tenants, err := tenant_service.GetTenants(ctx)
	if err != nil {
		log.Error("Tenants tenant_service.GetTenants failed: %v", err)
		ctx.ServerError("Tenants tenant_service.GetTenants", err)
		return
	}
	ctx.Data["Tenants"] = tenants
	ctx.HTML(http.StatusOK, tplGetTenants)
}

// TenantsList получаем все tenants
func TenantsList(ctx *context.Context) {
	tenants, err := tenant_service.GetTenants(ctx)
	if err != nil {
		log.Error("TenantsList tenant_service.GetTenants failed: %v", err)
		ctx.ServerError("TenantsList tenant_service.GetTenants", err)
		return
	}
	tenantsProjects := make(map[string][]map[string][]*TenantWithProjects)
	for _, tenant := range tenants {
		organizationsRelationTenant, errGetTenantOrganizations := tenat_model.GetTenantOrganizations(ctx, tenant.ID)
		if errGetTenantOrganizations != nil {
			log.Error("TenantsList tenat_model.GetTenantOrganizations failed: %v", errGetTenantOrganizations)
			ctx.ServerError("tenat_model.GetTenantOrganizations: %v", errGetTenantOrganizations)
			return
		}
		tenantIDWithOrganizations := make(map[string][]*TenantWithProjects)
		organizationIDs := make([]int64, len(organizationsRelationTenant))
		for idx, org := range organizationsRelationTenant {
			organizationIDs[idx] = org.OrganizationID
		}
		organizations, errGetOrganizationByIDs := organization.GetOrganizationByIDs(ctx, organizationIDs)
		if errGetOrganizationByIDs != nil {
			log.Error("TenantsList organization.GetOrganizationByIDs failed: %v", errGetOrganizationByIDs)
			ctx.ServerError("organization.GetOrganizationByIDs: %v", errGetOrganizationByIDs)
			return
		}
		tenantIDWithOrganizations[tenant.Name] = append(tenantIDWithOrganizations[tenant.ID], &TenantWithProjects{
			ID:        tenant.ID,
			Name:      tenant.Name,
			Default:   tenant.Default,
			IsActive:  tenant.IsActive,
			CreatedAt: tenant.CreatedAt,
			UpdatedAt: tenant.UpdatedAt,
			Projects:  organizations,
		})
		tenantsProjects["tenants"] = append(tenantsProjects["tenants"], tenantIDWithOrganizations)
	}
	ctx.JSON(http.StatusOK, &tenantsProjects)
}

// Tenant получаем конкретый tenant
func Tenant(ctx *context.Context) {
	tenantID := ctx.Params("tenantid")
	tenant, err := tenant_service.TenantByID(ctx, tenantID)
	if err != nil {
		if tenat_model.IsErrorTenantNotExists(err) {
			log.Debug("Tenant tenant_service.UpdateTenant failed to get tenant %s: %v", tenantID, err)
			ctx.Error(http.StatusNotFound, fmt.Sprintf("tenant_service.TenantByID: %v", err))
		} else {
			log.Error("Tenant tenant_service.UpdateTenant failed to get tenant %s: %v", tenantID, err)
			ctx.Error(http.StatusInternalServerError, fmt.Sprintf("tenant_service.TenantByID: %v", err))
		}
		return
	}
	organizationsRelationTenant, err := tenat_model.GetTenantOrganizations(ctx, tenant.ID)
	organizationIDs := make([]int64, len(organizationsRelationTenant))
	for idx, org := range organizationsRelationTenant {
		organizationIDs[idx] = org.OrganizationID
	}
	organizations, err := organization.GetOrganizationByIDs(ctx, organizationIDs)
	if err != nil {
		log.Error("Tenant organization.GetOrganizationByIDs failed: %v", err)
		ctx.ServerError("organization.GetOrganizationByIDs: %v", err)
		return
	}

	tenantOrganizations := make(map[string]*TenantWithProjects)
	tenantOrganizations[tenant.Name] = &TenantWithProjects{
		ID:        tenant.ID,
		Name:      tenant.Name,
		Default:   tenant.Default,
		IsActive:  tenant.IsActive,
		CreatedAt: tenant.CreatedAt,
		UpdatedAt: tenant.UpdatedAt,
		Projects:  organizations,
	}
	ctx.JSON(http.StatusOK, &tenantOrganizations)
}

// NewTenant создаем новый tenant
func NewTenant(ctx *context.Context) {
	requiredAuditParams := auditutils.NewRequiredAuditParams(ctx)
	auditParams := map[string]string{
		"email":            ctx.Doer.Email,
		"affected_user_id": strconv.FormatInt(ctx.Doer.ID, 10),
	}
	if !setting.SourceControl.MultiTenantEnabled {
		auditParams["error"] = "Error has occurred while validate multi-tenant mode"
		audit.CreateAndSendEvent(audit.TenantCreateEvent, requiredAuditParams.DoerName, requiredAuditParams.DoerID, audit.StatusFailure, requiredAuditParams.RemoteAddress, auditParams)
		log.Warn("NewTenant permission denied")
		ctx.Error(http.StatusForbidden, ctx.Tr("admin.permission_denied"))
		return
	}

	form := ctx.Req.Form
	// если в форме только csrf токен выходим
	if (len(form) == 1 && form.Get("_csrf") != "") || len(form) == 0 {
		auditParams["error"] = "Error has occurred while creating tenant"
		audit.CreateAndSendEvent(audit.TenantCreateEvent, requiredAuditParams.DoerName, requiredAuditParams.DoerID, audit.StatusFailure, requiredAuditParams.RemoteAddress, auditParams)
		log.Debug("NewTenant enter required fields")
		ctx.Error(http.StatusBadRequest, "the csrf token was entered in request form")
		return
	}
	tenant := &tenat_model.ScTenant{
		ID:        uuid.NewString(),
		Default:   false,
		IsActive:  true,
		CreatedAt: timeutil.TimeStampNow(),
		UpdatedAt: timeutil.TimeStampNow(),
	}
	type auditValue struct {
		ID        string
		Default   bool
		IsActive  bool
		CreatedAt timeutil.TimeStamp
		UpdatedAt timeutil.TimeStamp
	}
	newValue := auditValue{
		ID:        tenant.ID,
		Default:   tenant.Default,
		IsActive:  tenant.IsActive,
		CreatedAt: tenant.CreatedAt,
		UpdatedAt: timeutil.TimeStampNow(),
	}
	newValueBytes, _ := json.Marshal(newValue)
	auditParams["new_value"] = string(newValueBytes)

	if name := form.Get("name"); name != "" {
		if ok := re.MatchString(name); ok {
			if _, has, err := tenat_model.GetTenantByNameWithFlag(ctx, name); err != nil || has {
				auditParams["error"] = "Error has occurred while creating the tenant"
				audit.CreateAndSendEvent(audit.TenantCreateEvent, requiredAuditParams.DoerName, requiredAuditParams.DoerID, audit.StatusFailure, requiredAuditParams.RemoteAddress, auditParams)
				log.Debug(fmt.Sprintf("NewTenant tenant with name %s already exists", name))
				ctx.Error(http.StatusBadRequest, ctx.Tr("admin.tenant.already_exists"))
				return
			}
			tenant.Name = name
			tenant.OrgKey = name
		} else {
			auditParams["error"] = "Error has occurred while creating the tenant"
			audit.CreateAndSendEvent(audit.TenantCreateEvent, requiredAuditParams.DoerName, requiredAuditParams.DoerID, audit.StatusFailure, requiredAuditParams.RemoteAddress, auditParams)
			log.Debug(fmt.Sprintf("NewTenant invalid name: %s", name))
			ctx.Error(http.StatusBadRequest, ctx.Tr("admin.tenant.invalid_tenant_name"))
			return
		}
	}

	err := tenant_service.AddTenant(ctx, tenant)
	if err != nil {
		auditParams["error"] = "Error has occurred while adding the tenant"
		audit.CreateAndSendEvent(audit.TenantCreateEvent, requiredAuditParams.DoerName, requiredAuditParams.DoerID, audit.StatusFailure, requiredAuditParams.RemoteAddress, auditParams)
		log.Debug("NewTenant tenant_service.AddTenant failed: %v", err)
		ctx.Error(http.StatusNotFound, fmt.Sprintf("tenant_service.AddTenant: %v", err))
		return
	}

	newValue.UpdatedAt = timeutil.TimeStampNow()
	newValueBytes, _ = json.Marshal(newValue)
	auditParams["new_value"] = string(newValueBytes)

	audit.CreateAndSendEvent(audit.TenantCreateEvent, requiredAuditParams.DoerName, requiredAuditParams.DoerID, audit.StatusSuccess, requiredAuditParams.RemoteAddress, auditParams)
	ctx.JSON(http.StatusCreated, &tenant)
}

// DeleteTenant удаляем tenant и связанные с ними organizations
func DeleteTenant(ctx *context.Context) {
	requiredAuditParams := auditutils.NewRequiredAuditParams(ctx)
	if !setting.SourceControl.MultiTenantEnabled {
		auditParams := map[string]string{
			"affected_user_id": strconv.FormatInt(ctx.Doer.ID, 10),
			"email":            ctx.Doer.Email,
		}
		auditParams["error"] = "Error has occurred while validate multi-tenant mode"
		audit.CreateAndSendEvent(audit.TenantDeleteEvent, requiredAuditParams.DoerName, requiredAuditParams.DoerID, audit.StatusFailure, requiredAuditParams.RemoteAddress, auditParams)
		log.Warn("DeleteTenant permission denied")
		ctx.Error(http.StatusForbidden, ctx.Tr("admin.permission_denied"))
		return
	}

	tenantID := ctx.Params("tenantid")
	auditParams := map[string]string{
		"tenant_id":        tenantID,
		"affected_user_id": strconv.FormatInt(ctx.Doer.ID, 10),
		"email":            ctx.Doer.Email,
	}

	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		auditParams["error"] = "Error has occurred while deleting tenant"
		audit.CreateAndSendEvent(audit.TenantDeleteEvent, requiredAuditParams.DoerName, requiredAuditParams.DoerID, audit.StatusFailure, requiredAuditParams.RemoteAddress, auditParams)
		log.Debug("DeleteTenant uuid.Parse failed: %v", err)
		ctx.Error(http.StatusBadRequest, fmt.Sprintf("DeleteTenant uuid.Parse failed: %v", err))
		return
	}
	tenant, err := tenant_service.TenantByID(ctx, tenantID)
	if err != nil {
		if tenat_model.IsErrorTenantNotExists(err) {
			auditParams["error"] = "Error has occurred while getting tenant"
			audit.CreateAndSendEvent(audit.TenantDeleteEvent, requiredAuditParams.DoerName, requiredAuditParams.DoerID, audit.StatusFailure, requiredAuditParams.RemoteAddress, auditParams)
			log.Debug("DeleteTenant tenant_service.TenantByID failed to get tenant %s: %v", tenantID, err)
			ctx.Error(http.StatusNotFound, fmt.Sprintf("DeleteTenant tenant_service.TenantByID: %v", err))
		} else {
			auditParams["error"] = "Error has occurred while getting tenant"
			audit.CreateAndSendEvent(audit.TenantDeleteEvent, requiredAuditParams.DoerName, requiredAuditParams.DoerID, audit.StatusFailure, requiredAuditParams.RemoteAddress, auditParams)
			log.Error("DeleteTenant tenant_service.TenantByID failed to get tenant %s: %v", tenantID, err)
			ctx.Error(http.StatusInternalServerError, fmt.Sprintf("DeleteTenant tenant_service.TenantByID: %v", err))
		}
		auditParams["error"] = "Error has occurred while getting tenant"
		audit.CreateAndSendEvent(audit.TenantDeleteEvent, requiredAuditParams.DoerName, requiredAuditParams.DoerID, audit.StatusFailure, requiredAuditParams.RemoteAddress, auditParams)
		return
	}
	type auditValue struct {
		ID         string
		Name       string
		OrgKey     string
		IsActive   bool
		TenantUUID string
	}
	oldValue := auditValue{
		ID:         tenant.ID,
		Name:       tenant.Name,
		OrgKey:     tenant.OrgKey,
		IsActive:   tenant.IsActive,
		TenantUUID: tenantUUID.String(),
	}
	oldValueBytes, _ := json.Marshal(oldValue)
	auditParams["old_value"] = string(oldValueBytes)

	tenantOrganizations, err := tenat_model.GetTenantOrganizations(ctx, tenant.ID)
	if err != nil {
		auditParams["error"] = "Error has occurred while getting tenant organizations"
		audit.CreateAndSendEvent(audit.TenantDeleteEvent, requiredAuditParams.DoerName, requiredAuditParams.DoerID, audit.StatusFailure, requiredAuditParams.RemoteAddress, auditParams)
		log.Error("DeleteTenant tenant_service.GetTenantOrganizations failed: %v", err)
		ctx.ServerError("DeleteTenant tenant_service.GetTenantOrganizations failed: %v", err)
		return
	}
	orgIDs := make([]int64, len(tenantOrganizations))
	for idx, tenOrg := range tenantOrganizations {
		orgIDs[idx] = tenOrg.OrganizationID
	}
	organizations, err := organization.GetOrganizationByIDs(ctx, orgIDs)
	if err != nil {
		auditParams["error"] = "Error has occurred while getting tenant organizations"
		audit.CreateAndSendEvent(audit.TenantDeleteEvent, requiredAuditParams.DoerName, requiredAuditParams.DoerID, audit.StatusFailure, requiredAuditParams.RemoteAddress, auditParams)
		log.Error("DeleteTenant organization.GetOrganizationByIDs tenantIDs: %v", err)
		ctx.ServerError("DeleteTenant organization.GetOrganizationByIDs failed: %v", err)
		return
	}
	repos, _, err := repo_model.GetUserRepositories(&repo_model.SearchRepoOptions{Actor: ctx.Doer, OwnerIDs: orgIDs})
	if err != nil {
		auditParams["error"] = "Error has occurred while getting tenant repositories"
		audit.CreateAndSendEvent(audit.TenantDeleteEvent, requiredAuditParams.DoerName, requiredAuditParams.DoerID, audit.StatusFailure, requiredAuditParams.RemoteAddress, auditParams)
		log.Error("DeleteTenant repo_model.GetUserRepositories failed: %v", err)
		ctx.ServerError("DeleteTenant repo_model.GetUserRepositories failed: %v", err)
		return
	}
	for _, rep := range repos {
		errDeleteRepository := repository.DeleteRepository(ctx, ctx.Doer, rep, true)
		if errDeleteRepository != nil {
			auditParams["error"] = "Error has occurred while deleting repository"
			audit.CreateAndSendEvent(audit.TenantDeleteEvent, requiredAuditParams.DoerName, requiredAuditParams.DoerID, audit.StatusFailure, requiredAuditParams.RemoteAddress, auditParams)
			log.Error("DeleteTenant repository.DeleteRepository failed: %v", errDeleteRepository)
			ctx.ServerError("DeleteTenant repository.DeleteRepository failed: %v", err)
			return
		}
	}
	for _, org := range organizations {
		errDeleteOrg := org_service.DeleteOrganization(org)
		if errDeleteOrg != nil {
			auditParams["error"] = "Error has occurred while deleting organization"
			audit.CreateAndSendEvent(audit.TenantDeleteEvent, requiredAuditParams.DoerName, requiredAuditParams.DoerID, audit.StatusFailure, requiredAuditParams.RemoteAddress, auditParams)
			log.Error("DeleteTenant org_service.DeleteOrganization failed: %v", errDeleteOrg)
			ctx.ServerError("DeleteTenant org_service.DeleteOrganization failed: %v", errDeleteOrg)
			return
		}
	}

	err = tenant_service.RemoveTenantByID(ctx, tenantUUID.String(), orgIDs)
	if err != nil {
		auditParams["error"] = "Error has occurred while deleting tenant"
		audit.CreateAndSendEvent(audit.TenantDeleteEvent, requiredAuditParams.DoerName, requiredAuditParams.DoerID, audit.StatusFailure, requiredAuditParams.RemoteAddress, auditParams)
		log.Error("DeleteTenant tenant_service.RemoveTenantByID failed: %v", err)
		ctx.ServerError("DeleteTenant tenant_service.RemoveTenantByID failed: %v", err)
		return
	}

	audit.CreateAndSendEvent(audit.TenantDeleteEvent, requiredAuditParams.DoerName, requiredAuditParams.DoerID, audit.StatusSuccess, requiredAuditParams.RemoteAddress, auditParams)
	ctx.JSON(http.StatusNoContent, nil)
}

// EditTenant редактируем tenant
func EditTenant(ctx *context.Context) {
	requiredAuditParams := auditutils.NewRequiredAuditParams(ctx)
	auditParams := map[string]string{
		"affected_user_id": strconv.FormatInt(ctx.Doer.ID, 10),
		"email":            ctx.Doer.Email,
		"tenant_id":        ctx.Params("tenantid"),
	}
	type auditValue struct {
		ID        string
		Name      string
		Default   bool
		IsActive  bool
		CreatedAt timeutil.TimeStamp
		UpdatedAt timeutil.TimeStamp
	}

	form := ctx.Req.Form
	// если в форме только csrf токен выходим
	if (len(form) == 1 && form.Get("_csrf") != "") || len(form) == 0 {
		auditParams["error"] = "Error has occurred while editing tenant"
		audit.CreateAndSendEvent(audit.TenantEditEvent, requiredAuditParams.DoerName, requiredAuditParams.DoerID, audit.StatusFailure, requiredAuditParams.RemoteAddress, auditParams)
		log.Debug("EditTenant enter required fields")
		ctx.Error(http.StatusBadRequest, "enter required fields")
		return
	}

	oldTenant := &tenat_model.ScTenant{}
	tenantID := ctx.Params("tenantid")
	tenant := &tenat_model.ScTenant{
		ID:        tenantID,
		Default:   false,
		CreatedAt: timeutil.TimeStampNow(),
		UpdatedAt: timeutil.TimeStampNow(),
	}

	if name := form.Get("name"); name != "" {
		if ok := re.MatchString(name); ok {
			olderTenant, err := tenat_model.GetTenantByID(ctx, tenantID)
			if err != nil {
				if errors.As(err, &tenat_model.ErrorTenantDoesntExists{}) {
					log.Debug("EditTenant tenant with name %s does not exist", name)
					ctx.Error(http.StatusNotFound, fmt.Sprintf("Tenant with id %s does not exist: %v", tenantID, err))
					return
				}
				log.Debug("EditTenant with name %s tenant_service.TenantByID tenantIDs: %v", name, err)
				ctx.Error(http.StatusInternalServerError, fmt.Sprintf("Error when try to find tenant with id %s, error: %v", tenantID, err))
			}
			if tenant, has, err := tenat_model.GetTenantByNameWithFlag(ctx, name); err != nil || has && tenant.ID != tenantID {
				log.Debug(fmt.Sprintf("EditTenant tenant with name %s already exists", name))
				ctx.Error(http.StatusBadRequest, ctx.Tr("admin.tenant.already_exists"))
				return
			}
			tenant.Default = oldTenant.Default
			tenant.Name = name
			oldTenant = olderTenant
		} else {
			auditParams["error"] = "Error has occurred while editing the tenant"
			audit.CreateAndSendEvent(audit.TenantEditEvent, requiredAuditParams.DoerName, requiredAuditParams.DoerID, audit.StatusFailure, requiredAuditParams.RemoteAddress, auditParams)
			log.Debug(fmt.Sprintf("EditTenant invalid name: %s", name))
			ctx.Error(http.StatusBadRequest, ctx.Tr("admin.tenant.invalid_tenant_name"))
			return
		}
	}
	if form.Get("is_active") != "" {
		isActive, err := strconv.ParseBool(form.Get("is_active"))
		if err != nil {
			auditParams["error"] = "Error has occurred while editing the tenant"
			audit.CreateAndSendEvent(audit.TenantEditEvent, requiredAuditParams.DoerName, requiredAuditParams.DoerID, audit.StatusFailure, requiredAuditParams.RemoteAddress, auditParams)
			log.Debug("EditTenant strconv.ParseBool failed: %v", err)
			ctx.ServerError("EditTenant strconv.ParseBool EditTenant failed: %v", err)
			return
		}
		tenant.IsActive = isActive
	}

	oldValue := auditValue{
		ID:        tenantID,
		Name:      oldTenant.Name,
		Default:   oldTenant.Default,
		IsActive:  oldTenant.IsActive,
		CreatedAt: oldTenant.CreatedAt,
		UpdatedAt: oldTenant.UpdatedAt,
	}

	oldValueBytes, _ := json.Marshal(oldValue)
	auditParams["old_value"] = string(oldValueBytes)
	newValue := auditValue{
		ID:        tenantID,
		Name:      tenant.Name,
		Default:   tenant.Default,
		IsActive:  tenant.IsActive,
		CreatedAt: tenant.CreatedAt,
		UpdatedAt: timeutil.TimeStampNow(),
	}
	newValueBytes, _ := json.Marshal(newValue)
	auditParams["new_value"] = string(newValueBytes)

	err := tenant_service.UpdateTenant(ctx, tenant)
	if err != nil {
		auditParams["error"] = "Error has occurred while editing the tenant"
		audit.CreateAndSendEvent(audit.TenantEditEvent, requiredAuditParams.DoerName, requiredAuditParams.DoerID, audit.StatusFailure, requiredAuditParams.RemoteAddress, auditParams)
		log.Error("EditTenant tenant_service.UpdateTenant failed: %v", err)
		ctx.ServerError("EditTenant tenant_service.UpdateTenant: %v", err)
		return
	}

	audit.CreateAndSendEvent(audit.TenantEditEvent, requiredAuditParams.DoerName, requiredAuditParams.DoerID, audit.StatusSuccess, requiredAuditParams.RemoteAddress, auditParams)
	ctx.JSON(http.StatusOK, &tenant)
}

// AddTenantOrganization привязка organization_id к tenant_id
func AddTenantOrganization(ctx *context.Context) {
	if !setting.SourceControl.Enabled {
		log.Warn("AddTenantOrganization permission denied")
		ctx.Error(http.StatusForbidden, ctx.Tr("admin.permission_denied"))
		return
	}
	form := ctx.Req.Form
	tenantID := ctx.Params("tenantid")
	tenantOrganization := &tenat_model.ScTenantOrganizations{
		ID:       uuid.NewString(),
		TenantID: tenantID,
	}
	if form.Get("projectid") != "" {
		organizationID, err := strconv.ParseInt(form.Get("projectid"), 10, 64)
		if err != nil {
			log.Error("AddTenantOrganization: strconv.ParseInt failed: %v", err)
			ctx.ServerError("AddTenantOrganization strconv.ParseInt failed AddTenantOrganization: %v", err)
			return
		}
		tenantOrganization.OrganizationID = organizationID
	}
	err := tenant_service.CreateRelationTenantOrganization(ctx, tenantOrganization)
	if err != nil {
		log.Error("AddTenantOrganization tenant_service.CreateRelationTenantOrganization failed: %v", err)
		ctx.ServerError("AddTenantOrganization tenant_service.CreateRelationTenantOrganization: %v", err)
		return
	}
	ctx.JSON(http.StatusCreated, &tenantOrganization)
}

// DeleteTenantOrganization удаляем конкретный organization по organization_id для tenant_id
func DeleteTenantOrganization(ctx *context.Context) {
	if !setting.SourceControl.Enabled {
		log.Warn("DeleteTenantOrganization permission denied")
		ctx.Error(http.StatusForbidden, ctx.Tr("admin.permission_denied"))
		return
	}
	organizationIDParams := ctx.Params("projectid")
	organizationID, err := strconv.Atoi(organizationIDParams)
	if err != nil {
		log.Error("DeleteTenantOrganization strconv.Atoi failed: %v", err)
		ctx.ServerError("DeleteTenantOrganization strconv.Atoi failed: %v", err)
		return
	}
	tenantID := ctx.Params("tenantid")
	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		log.Error("DeleteTenantOrganization tenantID uuid.Parse failed: %v", tenantID)
		ctx.ServerError("DeleteTenantOrganization tenantID uuid.Parse failed: %v", err)
		return
	}
	org, err := organization.GetOrgByID(ctx, int64(organizationID))
	if err != nil {
		log.Error("DeleteTenantOrganization organization.GetOrgByID failed: %v", err)
		ctx.ServerError("DeleteTenantOrganization organization.GetOrgByID failed: %v", err)
		return
	}
	repos, _, err := repo_model.GetUserRepositories(&repo_model.SearchRepoOptions{Actor: ctx.Doer, OwnerIDs: []int64{org.ID}})
	if err != nil {
		log.Error("DeleteTenantOrganization repo_model.GetUserRepositories failed: %v", err)
		ctx.ServerError("DeleteTenantOrganization repo_model.GetUserRepositories failed: %v", err)
		return
	}
	for _, rep := range repos {
		errDeleteRepository := repository.DeleteRepository(ctx, ctx.Doer, rep, true)
		if errDeleteRepository != nil {
			log.Error("DeleteTenantOrganization repository.DeleteRepository failed: %v", errDeleteRepository)
			ctx.ServerError("DeleteTenantOrganization repository.DeleteRepository failed: %v", err)
			return
		}
	}
	err = org_service.DeleteOrganization(org)
	if err != nil {
		log.Error("DeleteTenantOrganization org_service.DeleteOrganization failed: %v", err)
		ctx.ServerError("DeleteTenantOrganization org_service.DeleteOrganization failed: %v", err)
		return
	}
	err = tenant_service.RemoveTenantOrganization(ctx, tenantUUID.String(), int64(organizationID))
	if err != nil {
		log.Error("DeleteTenantOrganization tenant_service.RemoveTenantOrganization failed: %v", err)
		ctx.ServerError("DeleteTenantOrganization tenant_service.RemoveTenantOrganization failed: %v", err)
		return
	}

	ctx.JSON(http.StatusNoContent, nil)
}

// RelationTenantOrganizations получаем все organizations для конкретного tenant
func RelationTenantOrganizations(ctx *context.Context) {
	if !setting.SourceControl.Enabled {
		log.Warn("RelationTenantOrganizations permission denied")
		ctx.Error(http.StatusForbidden, ctx.Tr("admin.permission_denied"))
		return
	}
	tenantID := ctx.Params("tenantid")
	tenant, err := tenant_service.TenantByID(ctx, tenantID)
	if err != nil {
		if tenat_model.IsErrorTenantNotExists(err) {
			log.Debug("RelationTenantOrganizations tenant_service.TenantByID failed to get tenant %s: %v", tenantID, err)
			ctx.Error(http.StatusNotFound, fmt.Sprintf("RelationTenantOrganizations tenant_service.TenantByID: %v", err))
		} else {
			log.Error("RelationTenantOrganizations tenant_service.TenantByID failed to get tenant %s: %v", tenantID, err)
			ctx.Error(http.StatusInternalServerError, fmt.Sprintf("RelationTenantOrganizations tenant_service.TenantByID: %v", err))
		}
		return
	}
	tenantOrganizations, err := tenant_service.TenantOrganizations(ctx, tenantID)
	if err != nil {
		log.Error("RelationTenantOrganizations tenant_service.TenantOrganizations failed: %v", err)
		ctx.ServerError("RelationTenantOrganizations tenant_service.TenantOrganizations: %v", err)
		return
	}
	organizationIDs := make([]int64, len(tenantOrganizations))
	for idx, tenantOrganization := range tenantOrganizations {
		organizationIDs[idx] = tenantOrganization.OrganizationID
	}

	organizations, err := org_service.GetOrganizations(ctx, organizationIDs)
	if err != nil {
		log.Error("RelationTenantOrganizations org_service.GetOrganizations failed: %v", err)
		ctx.ServerError("RelationTenantOrganizations organization.GetOrganizationByIDs: %v", err)
		return
	}
	ctx.JSON(http.StatusOK, map[string][]*organization.Organization{
		tenant.Name: organizations,
	})
}
