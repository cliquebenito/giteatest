package role_model

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"

	tenant_model "code.gitea.io/gitea/models/tenant"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/json"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/sbt/audit"
	auditutils "code.gitea.io/gitea/modules/sbt/audit/utils"
	"code.gitea.io/gitea/modules/timeutil"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/services/forms"
)

// ScTenantResponse структура для выдачи ответа с сокращенной информацией о тенанте
type ScTenantResponse struct {
	ID        string             `json:"id"`
	Name      string             `json:"name"`
	IsActive  bool               `json:"is_active"`
	IsDefault bool               `json:"is_default"`
	CreatedAt timeutil.TimeStamp `json:"created_at"`
	UpdatedAt timeutil.TimeStamp `json:"updated_at"`
	OrgKey    string             `json:"org_key"`
}

// GetDefaultTenant получение дефотного тенанта
func GetDefaultTenant(ctx *context.Context) {
	tenant, err := tenant_model.GetDefaultTenant(ctx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, "DefaultTenant not Exists")
		return
	}
	ctx.JSON(http.StatusOK, ScTenantResponse{
		ID:        tenant.ID,
		Name:      tenant.Name,
		IsActive:  tenant.IsActive,
		IsDefault: tenant.Default,
		OrgKey:    fmt.Sprintf("default_%s", tenant.Name),
		CreatedAt: tenant.CreatedAt,
		UpdatedAt: tenant.UpdatedAt,
	})
}

// CreateTenant метод для создания тенанта в системе по публичному api
func CreateTenant(ctx *context.Context) {
	form := web.GetForm(ctx).(*forms.CreateTenantApiForm)
	auditValues := auditutils.NewRequiredAuditParams(ctx)
	auditParams := make(map[string]string)

	if ctx.Written() {
		auditParams["error"] = "Response already sent or finalization attempt"
		audit.CreateAndSendEvent(audit.TenantCreateEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
		ctx.JSON(http.StatusInternalServerError, "Internal error")
		return
	}

	// проверка уникальности имени и ключа orgKey
	tenants, err := tenant_model.GetTenantsByNameOrOrgKey(ctx, form.Name, form.OrgKey)
	if err != nil {
		auditParams["error"] = "Error has occurred while getting tenants"
		audit.CreateAndSendEvent(audit.TenantCreateEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
		log.Error("Error has occurred while getting tenants by name '%s' or orgKey '%s'. Error: %v", form.Name, form.OrgKey, err)
		ctx.JSON(http.StatusInternalServerError, "Internal error")
		return
	}
	if len(tenants) > 0 {
		auditParams["error"] = "Error has occurred while creating tenant"
		audit.CreateAndSendEvent(audit.TenantCreateEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
		log.Debug("Tenant with name '%s' or orgKey '%s' is exists", form.Name, form.OrgKey)
		ctx.JSON(http.StatusInternalServerError, "Name is duplicated")
		return
	}

	tenant := &tenant_model.ScTenant{
		ID:       uuid.NewString(),
		Default:  false,
		IsActive: true,
		Name:     form.Name,
		OrgKey:   form.OrgKey,
	}

	tenantDB, err := tenant_model.InsertTenant(ctx, tenant)
	if err != nil {
		auditParams["error"] = "Error has occurred while inserting tenant"
		audit.CreateAndSendEvent(audit.TenantCreateEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
		log.Error("Error has occurred while inserting tenant with name '%s' and orgKey '%s'. Error: %v", form.Name, form.OrgKey, err)
		ctx.JSON(http.StatusInternalServerError, "Internal error")
		return
	}

	type auditValue struct {
		ID       string
		Name     string
		OrgKey   string
		IsActive bool
	}
	newValue := auditValue{
		ID:       tenantDB.ID,
		Name:     tenantDB.Name,
		OrgKey:   tenantDB.OrgKey,
		IsActive: tenantDB.IsActive,
	}
	newValueBytes, _ := json.Marshal(newValue)
	auditParams["new_value"] = string(newValueBytes)
	audit.CreateAndSendEvent(audit.TenantCreateEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusSuccess, auditValues.RemoteAddress, auditParams)

	ctx.JSON(http.StatusCreated, ScTenantResponse{
		ID:       tenantDB.ID,
		Name:     tenantDB.Name,
		IsActive: tenantDB.IsActive,
	})
}

// EditTenant метод для редактирования тенанта в системе по публичному api
func EditTenant(ctx *context.Context) {
	tenantID := ctx.Params("tenantid")
	form := web.GetForm(ctx).(*forms.EditTenantApiForm)
	auditValues := auditutils.NewRequiredAuditParams(ctx)
	auditParams := make(map[string]string)

	type auditValue struct {
		ID       string
		Name     string
		OrgKey   string
		IsActive bool
	}
	newValue := auditValue{
		ID:   tenantID,
		Name: form.Name,
	}
	newValueBytes, _ := json.Marshal(newValue)
	auditParams["new_value"] = string(newValueBytes)

	if ctx.Written() {
		auditParams["error"] = "Response already sent or finalization attempt"
		audit.CreateAndSendEvent(audit.TenantEditEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
		ctx.JSON(http.StatusInternalServerError, "Internal error")
		return
	}

	// проверка существования тенанта
	tenant, err := tenant_model.GetTenantByID(ctx, tenantID)
	if err != nil {
		if tenant_model.IsErrorTenantNotExists(err) {
			auditParams["error"] = "Error has occurred while getting tenant"
			audit.CreateAndSendEvent(audit.TenantEditEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
			log.Debug("Error has occurred while getting tenant by id '%s'. Error: %v", tenantID, err)
			ctx.JSON(http.StatusNotFound, "Tenant not exists")
			return
		}
		auditParams["error"] = "Error has occurred while getting tenant"
		audit.CreateAndSendEvent(audit.TenantEditEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
		log.Error("Error has occurred while getting tenant by id '%s'. Error: %v", tenantID, err)
		ctx.JSON(http.StatusInternalServerError, "Internal error")
		return
	}

	oldValue := auditValue{
		ID:       tenant.ID,
		Name:     tenant.Name,
		OrgKey:   tenant.OrgKey,
		IsActive: tenant.IsActive,
	}
	oldValueBytes, _ := json.Marshal(oldValue)
	auditParams["old_value"] = string(oldValueBytes)
	// проверка включенности тенанта
	if !tenant.IsActive {
		auditParams["error"] = "Error has occurred while checking the tenant's activity."
		audit.CreateAndSendEvent(audit.TenantEditEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
		log.Debug("Tenant with tenantID '%s' not active", tenant.ID)
		ctx.JSON(http.StatusBadRequest, "Tenant not active")
		return
	}

	// проверка уникальности имени
	_, has, err := tenant_model.GetTenantByNameWithFlag(ctx, form.Name)
	if err != nil {
		auditParams["error"] = "Error has occurred while the receipt of the tenant"
		audit.CreateAndSendEvent(audit.TenantEditEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
		log.Error("Error has occurred while getting tenant by name '%s'. Error: %v", form.Name, err)
		ctx.JSON(http.StatusInternalServerError, "Internal error")
		return
	}

	if has {
		auditParams["error"] = "Error has occurred while checking the unique tenant name"
		audit.CreateAndSendEvent(audit.TenantEditEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
		log.Debug("Tenant with name '%s' is exists", form.Name)
		ctx.JSON(http.StatusInternalServerError, "Name is duplicated")
		return
	}

	tenant.Name = form.Name
	err = tenant_model.UpdateTenant(ctx, tenant)
	if err != nil {
		auditParams["error"] = "Error has occurred while updating tenant"
		audit.CreateAndSendEvent(audit.TenantEditEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
		log.Error("Error has occurred while updating tenant with tenantID '%s'. Error: %v", tenant.ID, err)
		ctx.JSON(http.StatusInternalServerError, "Internal error")
		return
	}
	audit.CreateAndSendEvent(audit.TenantEditEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusSuccess, auditValues.RemoteAddress, auditParams)

	ctx.JSON(http.StatusOK, ScTenantResponse{
		ID:       tenant.ID,
		Name:     tenant.Name,
		IsActive: tenant.IsActive,
	})
}

// ActivateTenant метод для активации тенанта в системе по публичному api
func ActivateTenant(ctx *context.Context) {
	tenantID := ctx.Params("tenantid")
	auditValues := auditutils.NewRequiredAuditParams(ctx)
	auditParams := make(map[string]string)

	// проверка существования тенанта
	tenant, err := tenant_model.GetTenantByID(ctx, tenantID)
	if err != nil {
		if tenant_model.IsErrorTenantNotExists(err) {
			auditParams["error"] = "Error has occurred while getting tenant"
			audit.CreateAndSendEvent(audit.TenantActivateEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
			log.Debug("Error has occurred while getting tenant by id '%s'. Error: %v", tenantID, err)
			ctx.JSON(http.StatusNotFound, "Tenant not exists")
			return
		}
		auditParams["error"] = "Error has occurred while getting tenant"
		audit.CreateAndSendEvent(audit.TenantActivateEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
		log.Error("Error has occurred while getting tenant by id '%s'. Error: %v", tenantID, err)
		ctx.JSON(http.StatusInternalServerError, "Internal error")
		return
	}

	if tenant.IsActive {
		auditParams["error"] = "Error has occurred while activating tenant"
		audit.CreateAndSendEvent(audit.TenantActivateEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
		log.Error("Tenant with tenantID '%s' already activated", tenant.ID)
		ctx.JSON(http.StatusInternalServerError, "Tenant already activated")
		return
	}

	tenant.IsActive = true
	err = tenant_model.UpdateTenant(ctx, tenant)
	if err != nil {
		auditParams["error"] = "Error has occurred while updating tenant"
		audit.CreateAndSendEvent(audit.TenantActivateEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
		log.Error("Error has occurred while updating tenant with tenantID '%s'. Error: %v", tenant.ID, err)
		ctx.JSON(http.StatusInternalServerError, "Internal error")
		return
	}
	audit.CreateAndSendEvent(audit.TenantActivateEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusSuccess, auditValues.RemoteAddress, auditParams)

	ctx.JSON(http.StatusOK, ScTenantResponse{
		ID:       tenant.ID,
		Name:     tenant.Name,
		IsActive: tenant.IsActive,
	})
}

// DeactivateTenant метод для деактивации тенанта в системе по публичному api
func DeactivateTenant(ctx *context.Context) {
	tenantID := ctx.Params("tenantid")
	auditValues := auditutils.NewRequiredAuditParams(ctx)
	auditParams := make(map[string]string)
	// проверка существования тенанта
	tenant, err := tenant_model.GetTenantByID(ctx, tenantID)
	if err != nil {
		if tenant_model.IsErrorTenantNotExists(err) {
			auditParams["error"] = "Error has occurred while getting tenant"
			audit.CreateAndSendEvent(audit.TenantDeactivateEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
			log.Debug("Error has occurred while getting tenant by id '%s'. Error: %v", tenantID, err)
			ctx.JSON(http.StatusNotFound, "Tenant not exists")
			return
		}
		auditParams["error"] = "Error has occurred while getting tenant"
		audit.CreateAndSendEvent(audit.TenantDeactivateEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
		log.Error("Error has occurred while getting tenant by id '%s'. Error: %v", tenantID, err)
		ctx.JSON(http.StatusInternalServerError, "Internal error")
		return
	}

	// проверка на дефолтность тенанта
	if tenant.Default == true {
		auditParams["error"] = "Error has occurred while deactivating tenant"
		audit.CreateAndSendEvent(audit.TenantDeactivateEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
		log.Debug("Tenant with tenantID '%s' is default", tenant.ID)
		ctx.JSON(http.StatusBadRequest, "Tenant is default")
		return
	}

	if !tenant.IsActive {
		auditParams["error"] = "Error has occurred while deactivating tenant"
		audit.CreateAndSendEvent(audit.TenantDeactivateEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
		log.Error("Tenant with tenantID '%s' already deactivated", tenant.ID)
		ctx.JSON(http.StatusInternalServerError, "Tenant already deactivated")
		return
	}

	tenant.IsActive = false
	err = tenant_model.UpdateTenant(ctx, tenant)
	if err != nil {
		auditParams["error"] = "Error has occurred while updating tenant"
		audit.CreateAndSendEvent(audit.TenantDeactivateEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
		log.Error("Error has occurred while updating tenant with tenantID '%s'. Error: %v", tenant.ID, err)
		ctx.JSON(http.StatusInternalServerError, "Internal error")
		return
	}
	audit.CreateAndSendEvent(audit.TenantDeactivateEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusSuccess, auditValues.RemoteAddress, auditParams)
	ctx.JSON(http.StatusOK, ScTenantResponse{
		ID:       tenant.ID,
		Name:     tenant.Name,
		IsActive: tenant.IsActive,
	})
}

// DeleteTenant метод для удаления тенанта в системе по публичному api
func DeleteTenant(ctx *context.Context) {
	tenantID := ctx.Params("tenantid")
	auditValues := auditutils.NewRequiredAuditParams(ctx)
	auditParams := make(map[string]string)

	// проверка существования тенанта
	tenant, err := tenant_model.GetTenantByID(ctx, tenantID)
	if err != nil {
		if tenant_model.IsErrorTenantNotExists(err) {
			auditParams["error"] = "Error has occurred while getting tenant"
			audit.CreateAndSendEvent(audit.TenantDeleteEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
			log.Debug("Error has occurred while getting tenant by id '%s'. Error: %v", tenantID, err)
			ctx.JSON(http.StatusNotFound, "Tenant not exists")
			return
		}
		auditParams["error"] = "Error has occurred while getting tenant"
		audit.CreateAndSendEvent(audit.TenantDeleteEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
		log.Error("Error has occurred while getting tenant by id '%s'. Error: %v", tenantID, err)
		ctx.JSON(http.StatusInternalServerError, "Internal error")
		return
	}

	type auditValue struct {
		Name      string
		ID        string
		OrgKey    string
		Default   bool
		IsActive  bool
		CreatedAt timeutil.TimeStamp
		UpdatedAt timeutil.TimeStamp
	}
	oldValue := auditValue{
		Name:      tenant.Name,
		ID:        tenant.ID,
		OrgKey:    tenant.OrgKey,
		Default:   tenant.Default,
		IsActive:  tenant.IsActive,
		CreatedAt: tenant.CreatedAt,
		UpdatedAt: tenant.UpdatedAt,
	}
	oldValueBytes, _ := json.Marshal(oldValue)
	auditParams["old_value"] = string(oldValueBytes)

	// проверка на дефолтность тенанта
	if tenant.Default == true {
		auditParams["error"] = "Error has occurred while deleting tenant"
		audit.CreateAndSendEvent(audit.TenantDeleteEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
		log.Debug("Tenant with tenantID '%s' is default", tenant.ID)
		ctx.JSON(http.StatusBadRequest, "Tenant is default")
		return
	}
	organizations, err := tenant_model.GetTenantOrganizations(ctx, tenantID)
	if err != nil {
		auditParams["error"] = "Error has occurred while getting tenant organizations"
		audit.CreateAndSendEvent(audit.TenantDeleteEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
		ctx.Error(http.StatusInternalServerError)
		return
	}
	organizationIDs := make([]int64, 0, len(organizations))
	for _, organization := range organizations {
		organizationIDs = append(organizationIDs, organization.OrganizationID)
	}
	err = tenant_model.DeleteTenant(ctx, tenantID, organizationIDs)
	if err != nil {
		auditParams["error"] = "Error has occurred while deleting tenant"
		audit.CreateAndSendEvent(audit.TenantDeleteEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
		log.Error("Error has occurred while deleting tenant by id '%s'. Error: %v", tenantID, err)
		ctx.JSON(http.StatusInternalServerError, "Internal error")
		return
	}

	audit.CreateAndSendEvent(audit.TenantDeleteEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusSuccess, auditValues.RemoteAddress, auditParams)
	ctx.Status(http.StatusNoContent)
}

// GetTenantByID получение тенанта по id в системе по публичному api
func GetTenantByID(ctx *context.Context) {
	tenantID := ctx.Params("tenantid")

	// проверка существования тенанта
	tenant, err := tenant_model.GetTenantByID(ctx, tenantID)
	if err != nil {
		if tenant_model.IsErrorTenantNotExists(err) {
			log.Debug("Error has occurred while getting tenant by id '%s'. Error: %v", tenantID, err)
			ctx.JSON(http.StatusNotFound, "Tenant not exists")
			return
		}

		log.Error("Error has occurred while getting tenant by id '%s'. Error: %v", tenantID, err)
		ctx.JSON(http.StatusInternalServerError, "Internal error")
		return
	}

	ctx.JSON(http.StatusOK, ScTenantResponse{
		ID:        tenant.ID,
		Name:      tenant.Name,
		IsActive:  tenant.IsActive,
		IsDefault: tenant.Default,
		CreatedAt: tenant.CreatedAt,
		UpdatedAt: tenant.UpdatedAt,
		OrgKey:    tenant.OrgKey,
	})
}
