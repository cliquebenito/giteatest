package sonar

import (
	"net/http"
	"strconv"

	"code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/sonar"
	"code.gitea.io/gitea/models/sonar/domain"
	"code.gitea.io/gitea/models/sonar/usecase"
	"code.gitea.io/gitea/models/tenant"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/sbt/audit"
	audit2 "code.gitea.io/gitea/modules/sbt/audit/utils"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/routers/api/v3/models"
)

type Server struct {
	uc usecase.SonarSettingsUsecaser
}

func NewSonarServer(uc usecase.SonarSettingsUsecaser) *Server {
	return &Server{uc: uc}
}

func (s Server) CreateSonarSettings(ctx *context.APIContext) {
	// swagger:operation POST /{tenant}/{project}/{repo}/sonar CreateSonarSettings
	//
	// ---
	// summary: Create or update Sonar settings for a repository
	// description: Creates or updates SonarQube integration settings for the specified repository.
	// produces:
	// - application/json
	// consumes:
	// - application/json
	// parameters:
	// - name: tenant
	//   in: path
	//   required: true
	//   type: string
	//   description: Tenant identifier
	// - name: project
	//   in: path
	//   required: true
	//   type: string
	//   description: Project identifier
	// - name: repo
	//   in: path
	//   required: true
	//   type: string
	//   description: Repository identifier
	// - name: body
	//   in: body
	//   description: Sonar settings details
	//   required: true
	//   schema:
	//     type: object
	//     required:
	//       - sonar_server_url
	//       - sonar_project_key
	//       - sonar_token
	//     properties:
	//       sonar_server_url:
	//         type: string
	//         description: URL of the SonarQube server (must start with http or https)
	//         example: "https://sonarqube.example.com"
	//       sonar_project_key:
	//         type: string
	//         description: Unique project key in SonarQube
	//         example: "my-project-key"
	//       sonar_token:
	//         type: string
	//         description: Token used for authentication with SonarQube
	//         example: "your-secret-token"
	// responses:
	//   200:
	//     description: Sonar settings successfully created or updated
	//     schema:
	//       type: object
	//       properties:
	//         id:
	//           type: integer
	//           format: int64
	//           description: ID of the created or updated sonar settings
	//         project_key:
	//           type: string
	//           description: Sonar project key
	//         url:
	//           type: string
	//           description: URL to Sonar project or settings
	//   400:
	//     description: Bad request
	//     schema:
	//       type: object
	//       properties:
	//         errors:
	//           type: array
	//           items:
	//             type: string
	//           description: List of error messages
	//         message:
	//           type: string
	//           description: Error message
	//         url:
	//           type: string
	//           description: Link to Swagger documentation
	//   500:
	//     description: Internal server error
	//     schema:
	//       type: object
	//       properties:
	//         errors:
	//           type: array
	//           items:
	//             type: string
	//           description: List of error messages
	//         message:
	//           type: string
	//           description: Error message
	//         url:
	//           type: string
	//           description: Link to Swagger documentation
	opt := web.GetForm(ctx).(*models.CreateOrUpdateSonarProjectRequest)
	auditParams := map[string]string{
		"repository":    ctx.Repo.Repository.Name,
		"repository_id": strconv.FormatInt(ctx.Repo.Repository.ID, 10),
		"project":       ctx.Repo.Repository.OwnerName,
		"tenant_id":     ctx.Tenant.TenantID,
	}
	auditInfo := audit2.NewRequiredAuditParamsFromApiContext(ctx)
	if err := opt.Validate(); err != nil {
		log.Warn("Incorrect request params: %v", err)
		auditParams["error"] = "Error has occurred while validating form"
		audit.CreateAndSendEvent(audit.SonarSettingsCreateEvent, auditInfo.DoerName, auditInfo.DoerID, audit.StatusFailure, auditInfo.RemoteAddress, auditParams)
		ctx.Error(http.StatusBadRequest, "Create sonar settings", err)
		return
	}

	if err := s.uc.CreateSonarSettings(ctx,
		domain.CreateOrUpdateSonarProjectRequest{
			SonarServerURL:  opt.SonarServerURL.String(),
			SonarProjectKey: opt.SonarProjectKey.String(),
			SonarToken:      opt.SonarToken.String(),
			RepoId:          ctx.Repo.Repository.ID,
			TenantKey:       ctx.Tenant.OrgKey,
			Project:         ctx.ContextUser,
		}); err != nil {
		switch {
		case sonar.IsSonarSettingsAlreadyExists(err):
			log.Warn("Sonar settings already exists: %v", err)
			auditParams["error"] = "Error has occurred while creating sonar settings"
			audit.CreateAndSendEvent(audit.SonarSettingsCreateEvent, auditInfo.DoerName, auditInfo.DoerID, audit.StatusFailure, auditInfo.RemoteAddress, auditParams)
			ctx.Error(http.StatusConflict, "Sonar settings already exist", err)
		case repo.IsErrRepoNotExist(err):
			log.Warn("Repo not exist: %v", err)
			auditParams["error"] = "Error has occurred while creating sonar settings"
			audit.CreateAndSendEvent(audit.SonarSettingsCreateEvent, auditInfo.DoerName, auditInfo.DoerID, audit.StatusFailure, auditInfo.RemoteAddress, auditParams)
			ctx.Error(http.StatusNotFound, "Repository not found", err)

		case tenant.IsTenantOrganizationsNotExists(err):
			log.Warn("Tenant organizations not exist: %v", err)
			auditParams["error"] = "Error has occurred while creating sonar settings"
			audit.CreateAndSendEvent(audit.SonarSettingsCreateEvent, auditInfo.DoerName, auditInfo.DoerID, audit.StatusFailure, auditInfo.RemoteAddress, auditParams)
			ctx.Error(http.StatusNotFound, "Organization not found", err)

		default:
			log.Error("Error has occurred while creating sonar settings: %v", err)
			auditParams["error"] = "Error has occurred while creating sonar settings"
			audit.CreateAndSendEvent(audit.SonarSettingsCreateEvent, auditInfo.DoerName, auditInfo.DoerID, audit.StatusFailure, auditInfo.RemoteAddress, auditParams)
			ctx.Error(http.StatusInternalServerError, "Internal server error", err)

		}
		return
	}
	audit.CreateAndSendEvent(audit.SonarSettingsCreateEvent, auditInfo.DoerName, auditInfo.DoerID, audit.StatusSuccess, auditInfo.RemoteAddress, auditParams)
	ctx.Status(http.StatusCreated)

	return
}

func (s Server) SonarSettings(ctx *context.APIContext) {
	sonarSettings, err := s.uc.SonarSettings(ctx, ctx.Repo.Repository.ID)
	if err != nil {
		if sonar.IsSonarSettingsNotFound(err) {
			log.Warn("Sonar settings not found: %v", err)
			ctx.Error(http.StatusNotFound, "Internal server error", err)
			return
		}
		log.Error("Error has occurred while getting sonar settings: %v", err)
		ctx.Error(http.StatusInternalServerError, "Internal server error", err)
		return
	}
	ctx.JSON(http.StatusOK, sonarSettings)
	return
}

func (s Server) UpdateSonarSettings(ctx *context.APIContext) {
	opt := web.GetForm(ctx).(*models.CreateOrUpdateSonarProjectRequest)
	auditParams := map[string]string{
		"repository":    ctx.Repo.Repository.Name,
		"repository_id": strconv.FormatInt(ctx.Repo.Repository.ID, 10),
		"project":       ctx.Repo.Repository.OwnerName,
		"tenant_id":     ctx.Tenant.TenantID,
	}
	auditInfo := audit2.NewRequiredAuditParamsFromApiContext(ctx)

	if err := opt.Validate(); err != nil {
		log.Warn("Incorrect request params: %v", err)
		auditParams["error"] = "Error has occurred while validating form"
		audit.CreateAndSendEvent(audit.SonarSettingsUpdateEvent, auditInfo.DoerName, auditInfo.DoerID, audit.StatusFailure, auditInfo.RemoteAddress, auditParams)
		ctx.Error(http.StatusBadRequest, "Update sonar settings", err)
		return
	}
	if err := s.uc.UpdateSonarSettings(ctx,
		domain.CreateOrUpdateSonarProjectRequest{
			SonarServerURL:  opt.SonarServerURL.String(),
			SonarProjectKey: opt.SonarProjectKey.String(),
			SonarToken:      opt.SonarToken.String(),
			RepoId:          ctx.Repo.Repository.ID,
			TenantKey:       ctx.Tenant.OrgKey,
		}); err != nil {
		switch {
		case sonar.IsSonarSettingsNotFound(err):
			log.Warn("Sonar settings not found: %v", err)
			auditParams["error"] = "Error has occurred while creating sonar settings"
			audit.CreateAndSendEvent(audit.SonarSettingsUpdateEvent, auditInfo.DoerName, auditInfo.DoerID, audit.StatusFailure, auditInfo.RemoteAddress, auditParams)
			ctx.Error(http.StatusNotFound, "Sonar settings already exist", err)

		case repo.IsErrRepoNotExist(err):
			log.Warn("Repo not exist: %v", err)
			auditParams["error"] = "Error has occurred while creating sonar settings"
			audit.CreateAndSendEvent(audit.SonarSettingsUpdateEvent, auditInfo.DoerName, auditInfo.DoerID, audit.StatusFailure, auditInfo.RemoteAddress, auditParams)
			ctx.Error(http.StatusNotFound, "Repository not found", err)

		case tenant.IsTenantOrganizationsNotExists(err):
			log.Warn("Tenant organizations not exist: %v", err)
			auditParams["error"] = "Error has occurred while creating sonar settings"
			audit.CreateAndSendEvent(audit.SonarSettingsUpdateEvent, auditInfo.DoerName, auditInfo.DoerID, audit.StatusFailure, auditInfo.RemoteAddress, auditParams)
			ctx.Error(http.StatusNotFound, "Organization not found", err)

		default:
			log.Error("Error has occurred while creating sonar settings: %v", err)
			auditParams["error"] = "Error has occurred while creating sonar settings"
			audit.CreateAndSendEvent(audit.SonarSettingsUpdateEvent, auditInfo.DoerName, auditInfo.DoerID, audit.StatusFailure, auditInfo.RemoteAddress, auditParams)
			ctx.Error(http.StatusInternalServerError, "Internal server error", err)

		}
		return
	}
	audit.CreateAndSendEvent(audit.SonarSettingsUpdateEvent, auditInfo.DoerName, auditInfo.DoerID, audit.StatusSuccess, auditInfo.RemoteAddress, auditParams)
	ctx.Status(http.StatusOK)

	return
}

func (s Server) DeleteSonarSettings(ctx *context.APIContext) {
	auditParams := map[string]string{
		"repository":    ctx.Repo.Repository.Name,
		"repository_id": strconv.FormatInt(ctx.Repo.Repository.ID, 10),
		"project":       ctx.Repo.Repository.OwnerName,
		"tenant_id":     ctx.Tenant.TenantID,
	}
	auditInfo := audit2.NewRequiredAuditParamsFromApiContext(ctx)

	if err := s.uc.DeleteSonarSettings(ctx, ctx.Repo.Repository.ID); err != nil {
		if sonar.IsSonarSettingsNotExist(err) {
			log.Warn("Sonar settings not exist: %v", err)
			auditParams["error"] = "Error has occurred while deleting sonar settings"
			audit.CreateAndSendEvent(audit.SonarSettingsDeleteEvent, auditInfo.DoerName, auditInfo.DoerID, audit.StatusFailure, auditInfo.RemoteAddress, auditParams)
			ctx.Error(http.StatusNotFound, "Delete sonar settings", err)
			return
		}
		log.Warn("Delete sonar settings: %v", err)
		auditParams["error"] = "Error has occurred while deleting sonar settings"
		audit.CreateAndSendEvent(audit.SonarSettingsDeleteEvent, auditInfo.DoerName, auditInfo.DoerID, audit.StatusFailure, auditInfo.RemoteAddress, auditParams)
		ctx.Error(http.StatusBadRequest, "Delete sonar settings", err)
		return
	}
	audit.CreateAndSendEvent(audit.SonarSettingsDeleteEvent, auditInfo.DoerName, auditInfo.DoerID, audit.StatusSuccess, auditInfo.RemoteAddress, auditParams)

	ctx.Status(http.StatusNoContent)
	return
}
