package external_counter

import (
	"context"
	"net/http"
	"strconv"

	"code.gitea.io/gitea/models/external_metric_counter"
	"code.gitea.io/gitea/models/external_metric_counter/external_metric_counter_db"
	"code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/tenant"
	api_context "code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/sbt/audit"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/routers/api/v2/models"
	"code.gitea.io/gitea/routers/api/v2/models/metrics"
)

type counterDB interface {
	GetExternalMetricCounter(ctx context.Context, repoID int64) (*external_metric_counter.ExternalMetricCounter, error)
	UpsertCounter(ctx context.Context, repoID int64, counter int, description string) error
	DeleteCounter(ctx context.Context, repoID int64) error
}

type repoKeyDB interface {
	GetRepoByKey(ctx context.Context, key string) (*repo.ScRepoKey, error)
}

type Server struct {
	counterDB
	repoKeyDB
	counterEnabled bool
}

func New(counterDB counterDB, repoKeyDB repoKeyDB, counterEnabled bool) Server {
	return Server{counterDB: counterDB, repoKeyDB: repoKeyDB, counterEnabled: counterEnabled}
}

func (s Server) GetExternalMetricCounter(ctx *api_context.APIContext) {
	// swagger:operation GET /projects/repos/reuse_metric metrics getExternalMetric
	// ---
	// summary: Returns external metric for repo
	// produces:
	// - application/json
	// parameters:
	// - name: repo_key
	//   in: query
	//   type: string
	// - name: tenant_key
	//   in: query
	//   type: string
	// - name: project_key
	//   in: query
	//   type: string
	// responses:
	//   "200":
	//     "$ref": "#/responses/externalMetricGetResponse"
	//   "404":
	//     description: Not found
	//   "500":
	//     description: Internal server error

	s.getExternalMetricCounter(ctx)
}

func (s Server) getExternalMetricCounter(ctx *api_context.APIContext) {
	repoID, err := s.validateAndGetRepoID(ctx, nil, 0)
	if err != nil {
		log.Debug("Input params for creating repository are not valid: %v", err)
		ctx.Error(http.StatusBadRequest, "", "Incorrect params")
		return
	}

	metric, err := s.counterDB.GetExternalMetricCounter(ctx, repoID)
	if err != nil {
		if external_metric_counter_db.IsErrExternalMetricCounterDoesntExists(err) {
			log.Debug("Metric does not exist: %v", err)
			ctx.Error(http.StatusNotFound, "", "Metric does not exist")
		} else {
			log.Error("Error has occurred while getting external metric for repo ID %d: %v", repoID, err)
			ctx.Error(http.StatusInternalServerError, "", "Failed to get metric")
		}
		return
	}

	ctx.JSON(http.StatusOK, metrics.ExternalMetricGetResponse{
		Value: metric.MetricValue,
		Text:  metric.Text,
	})
}

func (s Server) SetExternalMetricCounter(ctx *api_context.APIContext) {
	// swagger:operation POST /projects/repos/reuse_metric metrics setExternalMetric
	// ---
	// summary: Creates/updates external metric for repo
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: repo_key
	//   in: query
	//   type: string
	// - name: tenant_key
	//   in: query
	//   type: string
	// - name: project_key
	//   in: query
	//   type: string
	// - name: body
	//   in: body
	//   description: Details of the repo to be created
	//   required: true
	//   schema:
	//     type: object
	//     required:
	//       - value
	//       - text
	//     properties:
	//       value:
	//         type: int
	//         description: Value of external metric
	//       text:
	//         type: string
	//         description: Description of external metric
	// responses:
	//   "201":
	//     description: Ok
	//   "404":
	//     description: Not found
	//   "500":
	//     description: Internal server error

	s.setExternalMetricCounter(ctx)
}

func (s Server) setExternalMetricCounter(ctx *api_context.APIContext) {
	auditParams := map[string]string{}
	opts := models.ParseExternalMetricGetOpts(ctx)

	req := web.GetForm(ctx).(*metrics.SetExternalMetricRequest)
	auditParams = map[string]string{
		"repository_key": opts.RepoKey,
		"project_key":    opts.ProjectKey,
		"tenant_key":     opts.TenantKey,
	}

	repoID, err := s.validateAndGetRepoID(ctx, auditParams, audit.ExternalMetricCounterSetEvent)
	if err != nil {
		return
	}

	if err := s.counterDB.UpsertCounter(ctx, repoID, req.Value, req.Text); err != nil {
		log.Error("Error has occurred while setting counter for repo ID %d: %v", repoID, err)
		ctx.Error(http.StatusInternalServerError, "", "Failed to set counter")
		auditParams["error"] = "Error has occurred while setting counter"
		audit.CreateAndSendEvent(audit.ExternalMetricCounterSetEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}
	audit.CreateAndSendEvent(audit.ExternalMetricCounterSetEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)
	ctx.Status(http.StatusCreated)
}

func (s Server) DeleteExternalMetricCounter(ctx *api_context.APIContext) {
	// swagger:operation DELETE /projects/repos/reuse_metric metrics deleteExternalMetric
	// ---
	// summary: Deletes external metric for repo
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: repo_key
	//   in: query
	//   type: string
	// - name: tenant_key
	//   in: query
	//   type: string
	// - name: project_key
	//   in: query
	//   type: string
	// responses:
	//   "201":
	//     description: Ok
	//   "404":
	//     description: Not found
	//   "500":
	//     description: Internal server error

	s.deleteExternalMetricCounter(ctx)
}

func (s Server) deleteExternalMetricCounter(ctx *api_context.APIContext) {
	auditParams := map[string]string{}
	opts := models.ParseExternalMetricGetOpts(ctx)

	auditParams = map[string]string{
		"repository_key": opts.RepoKey,
		"project_key":    opts.ProjectKey,
		"tenant_key":     opts.TenantKey,
	}
	repoID, err := s.validateAndGetRepoID(ctx, auditParams, audit.ExternalMetricCounterDeleteEvent)
	if err != nil {
		return
	}

	if err := s.counterDB.DeleteCounter(ctx, repoID); err != nil {
		log.Error("Error has occurred while deleting counter for repo ID %d: %v", repoID, err)
		ctx.Error(http.StatusInternalServerError, "", "Failed to delete counter")
		auditParams["error"] = "Error has occurred while deleting counter"
		audit.CreateAndSendEvent(audit.ExternalMetricCounterDeleteEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}
	audit.CreateAndSendEvent(audit.ExternalMetricCounterDeleteEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)
	ctx.Status(http.StatusCreated)
}

// validateAndGetRepoID performs repo/tenant lookup, ID parsing and org check
func (s Server) validateAndGetRepoID(ctx *api_context.APIContext, auditParams map[string]string, event audit.Event) (int64, error) {
	opts := models.ParseExternalMetricGetOpts(ctx)

	scRepoKey, err := s.repoKeyDB.GetRepoByKey(ctx, opts.RepoKey)
	if err != nil {
		if repo.IsErrorRepoKeyDoesntExists(err) {
			log.Debug("Error has occurred while getting repository by key %s: %v", opts.RepoKey, err)
			ctx.Error(http.StatusNotFound, "", "Repository not found")
		} else {
			log.Error("Error has occurred while getting repository by key %s: %v", opts.RepoKey, err)
			ctx.Error(http.StatusInternalServerError, "", "Failed to get repository")
		}
		if auditParams != nil {
			auditParams["error"] = "Error has occurred while getting repo by key"
			audit.CreateAndSendEvent(event, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		}
		return 0, err
	}

	repoID, err := strconv.ParseInt(scRepoKey.RepoID, 10, 64)
	if err != nil {
		log.Error("Error has occurred while parsing repository ID %s: %v", scRepoKey.RepoID, err)
		ctx.Error(http.StatusInternalServerError, "", "Invalid repository ID")
		if auditParams != nil {
			auditParams["error"] = "Error has occurred while parsing repository ID"
			audit.CreateAndSendEvent(event, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		}
		return 0, err
	}

	repository, err := repo.GetRepositoryByID(ctx, repoID)
	if err != nil {
		if repo.IsErrRepoNotExist(err) {
			log.Debug("Error has occurred while getting repository by ID %d: %v", repoID, err)
			ctx.Error(http.StatusNotFound, "", "Repository not found")
		} else {
			log.Error("Error has occurred while getting repository by ID %d: %v", repoID, err)
			ctx.Error(http.StatusInternalServerError, "", "Failed to get repository")
		}
		if auditParams != nil {
			auditParams["error"] = "Error has occurred while getting repository"
			audit.CreateAndSendEvent(event, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		}
		return 0, err
	}

	tenantOrg, err := tenant.GetTenantOrganizationsByKeys(ctx, opts.TenantKey, opts.ProjectKey)
	if err != nil {
		if tenant.IsTenantOrganizationsNotExists(err) {
			log.Debug("Error has occurred while getting tenant by tenant key %s and project key %s: %v", opts.TenantKey, opts.ProjectKey, err)
			ctx.Error(http.StatusNotFound, "", "Tenant not found")
		} else {
			log.Error("Error has occurred while getting tenant by tenant key %s and project key %s: %v", opts.TenantKey, opts.ProjectKey, err)
			ctx.Error(http.StatusInternalServerError, "", "Failed to get tenant")
		}
		if auditParams != nil {
			auditParams["error"] = "Error has occurred while getting tenant"
			audit.CreateAndSendEvent(event, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		}
		return 0, err
	}

	if repository.OwnerID != tenantOrg.OrganizationID {
		log.Debug("Repository ID %d does not correspond to tenant org ID %d", repoID, tenantOrg.OrganizationID)
		ctx.Error(http.StatusNotFound, "", "Repository does not correspond to requested project")
		return 0, err
	}

	return repoID, nil
}
