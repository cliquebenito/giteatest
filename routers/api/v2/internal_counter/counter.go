package internal_counter

import (
	"context"
	"net/http"
	"strconv"

	"code.gitea.io/gitea/models/internal_metric_counter"
	"code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/tenant"
	api_context "code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/routers/api/v2/models"
	"code.gitea.io/gitea/routers/api/v2/models/metrics"
)

type counterDB interface {
	GetInternalMetricCounter(_ context.Context, repoID int64, metricKey string) (*internal_metric_counter.InternalMetricCounter, error)
}

type repoKeyDB interface {
	GetRepoByKey(ctx context.Context, key string) (*repo.ScRepoKey, error)
}

type Server struct {
	counterDB
	repoKeyDB
	metricsList    []string
	counterEnabled bool
}

func New(internalMetricCounterDB counterDB, repoKeyDB repoKeyDB, metricsList []string, counterEnabled bool) Server {
	return Server{counterDB: internalMetricCounterDB, repoKeyDB: repoKeyDB, metricsList: metricsList, counterEnabled: counterEnabled}
}

func (s Server) GetInternalMetricCounter(ctx *api_context.APIContext) {
	// swagger:operation GET /projects/repos/metrics metrics getInternalMetric
	// ---
	// summary: Returns internal metric for repo
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
	// - name: metric
	//   in: query
	//   type: string
	// responses:
	//   "200":
	//     "$ref": "#/responses/internalMetricGetResponse"
	//   "400":
	//     description: Bad request
	//   "404":
	//     description: Not found
	//   "500":
	//     description: Internal server error

	s.getInternalMetricCounter(ctx)
}

func (s Server) getInternalMetricCounter(ctx *api_context.APIContext) {
	getOpts := models.ParseInternalMetricGetOpts(ctx)

	var (
		repository *repo.Repository
		tenantOrg  *tenant.ScTenantOrganizations
		scRepoKey  *repo.ScRepoKey
		repoId     int64
		err        error
	)

	// Get repo by external key
	if scRepoKey, err = s.repoKeyDB.GetRepoByKey(ctx, getOpts.RepoKey); err != nil {
		if repo.IsErrorRepoKeyDoesntExists(err) {
			log.Debug("Error has occurred while get repository by key %s: %v", getOpts.RepoKey, err)
			ctx.Error(http.StatusNotFound, "", "Repository not found")
			return
		} else {
			log.Error("Error has occurred while getting repository by key %s: %v", getOpts.RepoKey, err)
			ctx.Error(http.StatusInternalServerError, "", "Fail to get repository")
			return
		}
	}
	repoId, err = strconv.ParseInt(scRepoKey.RepoID, 10, 64)
	if err != nil {
		log.Error("Error has occurred while parsing repository by id %s: %v", scRepoKey.RepoID, err)
		ctx.Error(http.StatusInternalServerError, "", "Fail to parse repository id")
		return
	}

	// Check if repository exists
	if repository, err = repo.GetRepositoryByID(ctx, repoId); err != nil {
		if repo.IsErrRepoNotExist(err) {
			log.Debug("Error has occurred while getting repository id %d: %v", repoId, err)
			ctx.Error(http.StatusNotFound, "", "Repository not found")
			return
		} else {
			log.Error("Error has occurred while getting repository by id %d: %v", repoId, err)
			ctx.Error(http.StatusInternalServerError, "", "Fail to get repository")
			return
		}
	}

	if tenantOrg, err = tenant.GetTenantOrganizationsByKeys(ctx, getOpts.TenantKey, getOpts.ProjectKey); err != nil {
		if tenant.IsTenantOrganizationsNotExists(err) {
			log.Debug("Error has occurred while getting tenant by tenant key: %s, project key %s. Error: %v", getOpts.TenantKey, getOpts.ProjectKey, err)
			ctx.Error(http.StatusNotFound, "", "Tenant not found by given org key and project key")
			return
		} else {
			log.Error("Error has occurred while getting tenant by tenant key %s and project key %s: %v", getOpts.TenantKey, getOpts.ProjectKey, err)
			ctx.Error(http.StatusInternalServerError, "", "Fail to get tenant by tenant key and project key")
			return
		}
	}

	// Check that requested repo corresponds to requested project and tenant
	if repository.OwnerID != tenantOrg.OrganizationID {
		log.Debug("Repository %d does not correspond to requested project: %v", repoId, err)
		ctx.Error(http.StatusNotFound, "", "Repository does not correspond to requested project")
		return
	}

	metricsMap := make(map[string]struct{})
	for _, metric := range s.metricsList {
		metricsMap[metric] = struct{}{}
	}

	if _, ok := metricsMap[getOpts.Metric]; !ok {
		log.Debug("Metric with such key %s not declared: %v", getOpts.Metric, err)
		ctx.Error(http.StatusBadRequest, "", "Fail to find metric")
		return
	}

	metric, err := s.counterDB.GetInternalMetricCounter(ctx, repoId, getOpts.Metric)
	if err != nil {
		log.Error("Error has occurred while getting internal metric by key %s: %v", getOpts.Metric, err)
		ctx.Error(http.StatusInternalServerError, "", "Fail to get metric")
		return
	}

	ctx.JSON(http.StatusOK, metrics.InternalMetricGetResponse{Value: metric.MetricValue})
}
