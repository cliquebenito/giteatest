package tenant

import (
	"net/http"
	"strconv"

	"code.gitea.io/gitea/modules/sbt/audit"
	"github.com/google/uuid"

	tenant_model "code.gitea.io/gitea/models/tenant"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/routers/api/v2/models"
)

type Server struct{}

func NewTenantServer() Server {
	return Server{}
}

// getTenantByKey returns tenant by id
func (s Server) getTenantByKey(ctx *context.APIContext) {
	tenantKey := ctx.FormString("tenant_key")

	var tenant *tenant_model.ScTenant
	var has bool
	var err error

	// Check if tenant exists
	tenant, has, err = tenant_model.GetTenantByOrgKey(ctx, tenantKey)
	if err != nil {
		log.Error("Error has occurred while getting tenant by tenant key '%s'. Error: %v", tenantKey, err)
		ctx.Error(http.StatusInternalServerError, "", "Fail to get tenant by tenant key")
		return
	}

	if !has {
		log.Debug("Tenant not exists by tenant key '%s'. Error: %v", tenantKey, err)
		ctx.JSON(http.StatusNotFound, context.APIError{
			Message: "Tenant does not exist",
			URL:     setting.API.SwaggerURL,
		})
		return
	}

	ctx.JSON(http.StatusOK, models.TenantGetResponse{
		ID:        tenant.ID,
		Name:      tenant.Name,
		IsActive:  tenant.IsActive,
		TenantKey: tenantKey,
	})
}

// createTenant create tenant by tenant key and name
func (s Server) createTenant(ctx *context.APIContext) {
	form := web.GetForm(ctx).(*models.CreateTenantOptions)
	if form == nil {
		log.Error("Error has occurred while parsing form")
		ctx.Error(http.StatusInternalServerError, "", "Fail to parse form")
		return
	}

	if err := form.Validate(); err != nil {
		log.Debug("Input params for creating tenant are not valid: %v", err)
		ctx.Error(http.StatusBadRequest, "", "Incorrect params")
		return
	}

	auditParams := map[string]string{
		"tenant_key": form.Name,
	}

	// Check if tenant with same name or orgKey already exists
	tenants, err := tenant_model.GetTenantsByNameOrOrgKey(ctx, form.Name, form.TenantKey)
	if err != nil {
		log.Error("Error has occurred while getting tenants by name '%s' or tenant key '%s'. Error: %v", form.Name, form.TenantKey, err)
		auditParams["error"] = "Error has occurred while getting tenants"
		audit.CreateAndSendEvent(audit.TenantCreateEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		ctx.Error(http.StatusInternalServerError, "", "Fail to get tenants by name or tenant key")
		return
	}
	if len(tenants) > 0 {
		log.Debug("Tenant with name '%s' or tenant key '%s' exists", form.Name, form.TenantKey)
		auditParams["error"] = "Error has occurred the tenant already exists"
		audit.CreateAndSendEvent(audit.TenantCreateEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		ctx.JSON(http.StatusConflict, context.APIError{
			Message: "Name or organization key already used",
			URL:     setting.API.SwaggerURL,
		})
		return
	}

	tenant := &tenant_model.ScTenant{
		ID:       uuid.NewString(),
		Default:  false,
		IsActive: true,
		Name:     form.Name,
		OrgKey:   form.TenantKey,
	}

	tenantDB, err := tenant_model.InsertTenant(ctx, tenant)
	if err != nil {
		log.Error("Error has occurred while creating tenant with name '%s' and tenant key '%s'. Error: %v", form.Name, form.TenantKey, err)
		auditParams["error"] = "Error has occurred while inserting tenant"
		audit.CreateAndSendEvent(audit.TenantCreateEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		ctx.Error(http.StatusInternalServerError, "", "Fail to create tenant")
		return
	}

	audit.CreateAndSendEvent(audit.TenantCreateEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)
	ctx.JSON(http.StatusCreated, models.TenantPostResponse{
		ID:        tenantDB.ID,
		Name:      tenant.Name,
		TenantKey: tenantDB.OrgKey,
	})
}

// GetTenantByKey returns tenant by id
func (s Server) GetTenantByKey(ctx *context.APIContext) {
	// swagger:operation GET /tenants tenant getTenantByKey
	// ---
	// summary: Returns the tenant by key
	// produces:
	// - application/json
	// parameters:
	// - name: tenant_key
	//   in: query
	//   description: key of tenant
	//   type: string
	//   required: true
	// responses:
	//   "200":
	//     "$ref": "#/responses/tenantGetResponse"
	//   "404":
	//     description: Not found
	//   "500":
	//     description: Internal server error

	s.getTenantByKey(ctx)
}

// CreateTenant create tenant by tenant key and name
func (s Server) CreateTenant(ctx *context.APIContext) {
	// swagger:operation POST /tenants tenant createTenant
	// ---
	// summary: Creates tenant by tenant key and name
	// produces:
	// - application/json
	// parameters:
	// - name: body
	//   in: body
	//   description: Details of the repo to be created
	//   required: true
	//   schema:
	//     type: object
	//     required:
	//       - tenant_key
	//       - name
	//     properties:
	//       tenant_key:
	//         type: string
	//         description: External key of tenant
	//       name:
	//         type: string
	//         description: Name of tenant
	// responses:
	//   "201":
	//     "$ref": "#/responses/tenantPostResponse"
	//   "409":
	//     description: Conflict
	//   "500":
	//     description: Internal server error

	s.createTenant(ctx)
}
