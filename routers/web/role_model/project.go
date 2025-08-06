package role_model

import (
	"fmt"
	"net/http"
	"strconv"

	"code.gitea.io/gitea/models/organization"
	tenant_model "code.gitea.io/gitea/models/tenant"
	"code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/json"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/sbt/audit"
	auditutils "code.gitea.io/gitea/modules/sbt/audit/utils"
	"code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/services/forms"
	tenant_service "code.gitea.io/gitea/services/tenant"
)

// OrgResponse структура для выдачи ответа с сокращенной информацией о проекте
type OrgResponse struct {
	ID  int64  `json:"id"`
	URL string `json:"url"`
}

// CreateProject метод для создания проекта
func CreateProject(ctx *context.Context) {
	form := web.GetForm(ctx).(*forms.CreateProjectApiForm)
	auditValues := auditutils.NewRequiredAuditParams(ctx)
	auditParams := make(map[string]string)
	type auditValue struct {
		Name        string
		OrgKey      string
		ProjectKey  string
		Description string
		Visibility  structs.VisibleType
	}

	newValue := auditValue{
		Name:        form.Name,
		OrgKey:      form.OrgKey,
		ProjectKey:  form.ProjectKey,
		Description: form.Description,
		Visibility:  form.Visibility,
	}
	newValueBytes, _ := json.Marshal(newValue)
	auditParams["new_value"] = string(newValueBytes)

	if ctx.Written() {
		auditParams["error"] = "Response already sent or finalization attempt"
		audit.CreateAndSendEvent(audit.ProjectCreateEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
		ctx.JSON(http.StatusInternalServerError, "Internal error")
		return
	}

	// проверка тенанта на существование
	tenant, has, err := tenant_model.GetTenantByOrgKey(ctx, form.OrgKey)
	if err != nil {
		auditParams["error"] = "Error has occurred while getting tenant"
		audit.CreateAndSendEvent(audit.ProjectCreateEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
		log.Error("Error has occurred while getting tenant by orgKey '%s'. Error: %v", form.OrgKey, err)
		ctx.JSON(http.StatusInternalServerError, "Internal error")
		return
	}

	if !tenant.IsActive {
		auditParams["error"] = "Error has occured while tenant is inactive"
		audit.CreateAndSendEvent(audit.ProjectCreateEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
		log.Error("Error while occurred the tenant is inactive")
		ctx.JSON(http.StatusInternalServerError, "Tenant not active")
		return
	}

	if !has {
		auditParams["error"] = "Error has occurred while getting tenant"
		audit.CreateAndSendEvent(audit.ProjectCreateEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
		log.Debug("Tenant not exists by orgKey '%s'. Error: %v", form.OrgKey, err)
		ctx.JSON(http.StatusNotFound, "Tenant not exists")
		return
	}

	// проверка проекта на существование
	_, has, err = tenant_model.GetTenantOrganizationsByProjectKey(ctx, form.ProjectKey)
	if err != nil {
		auditParams["error"] = "Error has occurred while getting tenant organization"
		audit.CreateAndSendEvent(audit.ProjectCreateEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
		log.Error("Error has occurred while getting tenant organization by projectKey '%s'. Error: %v", form.ProjectKey, err)
		ctx.JSON(http.StatusInternalServerError, "Internal error")
		return
	}

	if has {
		auditParams["error"] = "Error has occurred while checking the unique tenant name"
		audit.CreateAndSendEvent(audit.ProjectCreateEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
		log.Error("Project with projectKey '%s' is exists", form.ProjectKey)
		ctx.JSON(http.StatusBadRequest, "Project is exists")
		return
	}

	validateProjectName(ctx, 0, tenant.ID, form.Name)
	if ctx.Written() {
		auditParams["error"] = "Error has occured while validating project name"
		audit.CreateAndSendEvent(audit.ProjectCreateEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
		return
	}

	// проверка уровня видимости проекта
	if form.Visibility == structs.VisibleTypePublic {
		auditParams["error"] = "Error has occurred while checking the visibility"
		audit.CreateAndSendEvent(audit.ProjectCreateEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
		log.Debug("Incorrect visibility while updating project with orgName '%s'", form.Name)
		ctx.JSON(http.StatusBadRequest, "Incorrect visibility")
		return
	}

	// создание проекта под тенантом
	org := &organization.Organization{
		Name:        form.Name,
		Description: form.Description,
		Visibility:  form.Visibility,
		IsActive:    true,
	}

	createdOrg, err := tenant_service.CreateOrg(ctx, org, form.OrgKey, form.ProjectKey)
	if err != nil {
		auditParams["error"] = "Error has occurred while creating organization"
		audit.CreateAndSendEvent(audit.ProjectCreateEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
		log.Error("Error has occurred while creating organization with name '%s' projectKey '%s' and orgKey '%s'. Error: %v", form.Name, form.ProjectKey, form.OrgKey, err)
		ctx.JSON(http.StatusInternalServerError, "Internal error")
		return
	}
	audit.CreateAndSendEvent(audit.ProjectCreateEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusSuccess, auditValues.RemoteAddress, auditParams)

	ctx.JSON(http.StatusOK, OrgResponse{
		ID:  createdOrg.ID,
		URL: createdOrg.HTMLURL(),
	})
}

// EditProject метод для изменения проекта
func EditProject(ctx *context.Context) {
	form := web.GetForm(ctx).(*forms.ModifyProjectApiForm)
	auditValues := auditutils.NewRequiredAuditParams(ctx)
	auditParams := make(map[string]string)
	type auditValue struct {
		Name        string
		ProjectKey  string
		Description string
		Visibility  structs.VisibleType
	}
	newValue := auditValue{
		Name:        form.Name,
		ProjectKey:  form.ProjectKey,
		Description: form.Description,
		Visibility:  form.Visibility,
	}
	newValueBytes, _ := json.Marshal(newValue)
	auditParams["new_value"] = string(newValueBytes)

	if ctx.Written() {
		auditParams["error"] = "Response already sent or finalization attempt"
		audit.CreateAndSendEvent(audit.ProjectEditEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
		ctx.JSON(http.StatusInternalServerError, "Internal error")
		return
	}

	// проверка проекта на существование
	tenantOrganization, has, err := tenant_model.GetTenantOrganizationsByProjectKey(ctx, form.ProjectKey)
	if err != nil {
		auditParams["error"] = "Error has occurred while getting tenant organization"
		audit.CreateAndSendEvent(audit.ProjectEditEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
		log.Error("Error has occurred while getting tenant organization by projectKey '%s'. Error: %v", form.ProjectKey, err)
		ctx.JSON(http.StatusInternalServerError, "Internal error")
		return
	}

	if !has {
		auditParams["error"] = "Error has occurred while getting tenant"
		audit.CreateAndSendEvent(audit.ProjectEditEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
		log.Debug("Organization not exists by projectKey '%s'. Error: %v", form.ProjectKey, err)
		ctx.JSON(http.StatusNotFound, "Project not exists")
		return
	}

	userOrg, err := user.GetUserByID(ctx, tenantOrganization.OrganizationID)
	if err != nil {
		auditParams["error"] = "Error has occurred while getting organization"
		audit.CreateAndSendEvent(audit.ProjectEditEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
		log.Debug("Organization not exists by orgId '%s'. Error: %v", tenantOrganization.OrganizationID, err)
		ctx.JSON(http.StatusNotFound, "Project not exists")
		return
	}
	oldValue := auditValue{
		Name:        userOrg.Name,
		Description: userOrg.Description,
		Visibility:  userOrg.Visibility,
	}
	oldValueBytes, _ := json.Marshal(oldValue)
	auditParams["old_value"] = string(oldValueBytes)

	validateProjectName(ctx, tenantOrganization.OrganizationID, tenantOrganization.TenantID, form.Name)
	if ctx.Written() {
		auditParams["error"] = "Error has occured while validating project name"
		audit.CreateAndSendEvent(audit.ProjectEditEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
		return
	}

	if form.Name != "" {
		userOrg.Name = form.Name
	}

	if form.Visibility != 0 {
		userOrg.Visibility = form.Visibility
	}

	if form.Description != "" {
		userOrg.Description = form.Description
	}

	err = user.UpdateUserSetting(userOrg)
	if err != nil {
		auditParams["error"] = "Error has occurred while updating organization"
		audit.CreateAndSendEvent(audit.ProjectEditEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
		log.Error("Error has occurred while updating organization with name '%s' projectKey '%s' and tenantId '%s'. Error: %v", form.Name, form.ProjectKey, tenantOrganization.TenantID, err)
		ctx.JSON(http.StatusInternalServerError, "Internal error")
		return
	}

	audit.CreateAndSendEvent(audit.ProjectEditEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusSuccess, auditValues.RemoteAddress, auditParams)
	ctx.Status(http.StatusNoContent)
}

// DeleteProject метод для удаления проекта
func DeleteProject(ctx *context.Context) {
	form := web.GetForm(ctx).(*forms.DeleteProjectApiForm)
	auditValues := auditutils.NewRequiredAuditParams(ctx)
	auditParams := make(map[string]string)
	if ctx.Written() {
		auditParams["error"] = "Response already sent or finalization attempt"
		audit.CreateAndSendEvent(audit.ProjectDeleteEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
		ctx.JSON(http.StatusInternalServerError, "Internal error")
		return
	}

	// проверка тенанта на существование
	_, has, err := tenant_model.GetTenantByOrgKey(ctx, form.OrgKey)
	if err != nil {
		auditParams["error"] = "Error has occurred while getting tenant"
		audit.CreateAndSendEvent(audit.ProjectDeleteEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
		log.Error("Error has occurred while getting tenant by orgKey '%s'. Error: %v", form.OrgKey, err)
		ctx.JSON(http.StatusInternalServerError, "Internal error")
		return
	}

	if !has {
		auditParams["error"] = "Error has occurred while getting tenant"
		audit.CreateAndSendEvent(audit.ProjectDeleteEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
		log.Debug("Tenant not exists by orgKey '%s'. Error: %v", form.OrgKey, err)
		ctx.JSON(http.StatusNotFound, "Tenant not exists")
		return
	}

	// проверка проекта на существование
	tenantOrganization, has, err := tenant_model.GetTenantOrganizationsByProjectKey(ctx, form.ProjectKey)
	if err != nil {
		log.Error("Error has occurred while getting tenant organization by projectKey '%s'. Error: %v", form.ProjectKey, err)
		auditParams["error"] = "Error has occurred while getting tenant"
		audit.CreateAndSendEvent(audit.ProjectDeleteEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
		ctx.JSON(http.StatusInternalServerError, "Internal error")
		return
	}

	if !has {
		auditParams["error"] = "Error has occurred while getting tenant"
		audit.CreateAndSendEvent(audit.ProjectDeleteEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
		log.Debug("Organization not exists by projectKey '%s'. Error: %v", form.ProjectKey, err)
		ctx.JSON(http.StatusNotFound, "Project not exists")
		return
	}

	if err = tenant_service.RemoveOrg(ctx, tenantOrganization); err != nil {
		auditParams["error"] = "Error has occurred while deleting organization"
		audit.CreateAndSendEvent(audit.ProjectDeleteEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
		log.Error("Error has occurred while deleting organization by projectKey '%s' and orgKey '%s'. Error: %v", form.ProjectKey, form.OrgKey, err)
		ctx.JSON(http.StatusInternalServerError, fmt.Sprintf("Error has occurred while deleting organization by projectKey '%s' and orgKey '%s'. Error: %v", form.ProjectKey, form.OrgKey, err))
	}

	audit.CreateAndSendEvent(audit.ProjectDeleteEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusSuccess, auditValues.RemoteAddress, auditParams)
}

// validateProjectName метод для валидации имени проекта
func validateProjectName(ctx *context.Context, orgID int64, tenantId, orgName string) {
	if orgName != "" {
		// проверка уникальности имени проекта в тенанте
		organizationsRelationTenant, err := tenant_model.GetTenantOrganizations(ctx, tenantId)
		if err != nil {
			log.Error("Error has occurred while getting organizations by tenantID '%s'. Error: %v", tenantId, err)
			ctx.JSON(http.StatusInternalServerError, "Internal error")
			return
		}
		organizationIDs := make([]int64, len(organizationsRelationTenant))
		for idx, org := range organizationsRelationTenant {
			organizationIDs[idx] = org.OrganizationID
		}
		organizations, err := organization.GetOrganizationByIDs(ctx, organizationIDs)
		if err != nil {
			log.Error("Error has occurred while getting organizations by tenantID '%s'. Error: %v", tenantId, err)
			ctx.JSON(http.StatusInternalServerError, "Internal error")
			return
		}

		for _, org := range organizations {
			if org.Name == orgName && org.ID != orgID {
				log.Debug("Organization with name '%s' is exists in tenantId '%s'", orgName, tenantId)
				ctx.JSON(http.StatusInternalServerError, "Name is duplicated")
				return
			}
		}
	}
}

// GetProjectByID получение проекта по ID
func GetProjectByID(ctx *context.Context) {
	projectID, err := strconv.Atoi(ctx.Params("projectid"))
	if err != nil {
		log.Error("Error has occurred while parsing projectid '%s'. Error: %v", ctx.Params("projectid"), err)
		ctx.JSON(http.StatusInternalServerError, "GetProjectByID failed")
		return
	}

	tenantOrganizations, err := tenant_model.GetTenantOrganizationsByOrgId(ctx, int64(projectID))
	if err != nil {
		log.Error("Error has occurred while getting tenant organizations by projectID '%d'. Error: %v", projectID, err)
		ctx.JSON(http.StatusInternalServerError, "GetProjectByID failed")
		return
	}

	if tenantOrganizations == nil {
		log.Debug("Organization not found by projectID '%d'", projectID)
		ctx.JSON(http.StatusNotFound, "Organization not found")
		return
	}
	organizations, err := organization.GetOrganizationByIDs(ctx, []int64{int64(projectID)})
	if err != nil {
		log.Error("Error has occurred while getting organizations by IDs '%v'. Error: %v", []int64{int64(projectID)}, err)
		ctx.JSON(http.StatusInternalServerError, "Internal error")
		return
	}
	if len(organizations) == 0 {
		log.Debug("Organization not found by projectID '%d'", projectID)
		ctx.JSON(http.StatusNotFound, "Organization not found")
		return
	}
	ctx.JSON(http.StatusOK, OrgResponse{
		ID:  organizations[0].ID,
		URL: organizations[0].HTMLURL(),
	})
}
