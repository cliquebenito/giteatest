package repo

import (
	gocontext "context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"code.gitea.io/gitea/models/organization"
	"code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/repo_marks"
	"code.gitea.io/gitea/models/repo_marks/repo_marks_db"
	"code.gitea.io/gitea/models/role_model"
	"code.gitea.io/gitea/models/tenant"
	trace_model "code.gitea.io/gitea/models/trace"
	"code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/repository"
	"code.gitea.io/gitea/modules/sbt/audit"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/routers/api/v2/models"
	apirepo "code.gitea.io/gitea/routers/api/v2/models/repo"
	"code.gitea.io/gitea/routers/private/repo_mark"
	"code.gitea.io/gitea/services/project"
)

// Server структура для DI при работе с репозиторием
type Server struct {
	checkUserPermissionFn role_model.CheckUserPermissionFnType
	repoKeyDB             repoKeyDB
	repoMarksEditor       repo_mark.RepoMarksEditor
	codeHubMark           repo_marks.RepoMark
}

// NewRepoServer получить Server
func NewRepoServer(checkPermFn role_model.CheckUserPermissionFnType, db repoKeyDB, repoMark repo_mark.RepoMarksEditor, mark repo_marks.RepoMark) Server {
	return Server{
		checkUserPermissionFn: checkPermFn,
		repoKeyDB:             db,
		repoMarksEditor:       repoMark,
		codeHubMark:           mark,
	}
}

type repoKeyDB interface {
	GetRepoByKey(ctx gocontext.Context, key string) (*repo.ScRepoKey, error)
	GetRepoByRepoID(ctx gocontext.Context, repoId string) (*repo.ScRepoKey, error)
	UpdateRepoKey(ctx gocontext.Context, repoKey *repo.ScRepoKey) error
}

// getOrgRepo returns repo according to tenant and project
func (s Server) getOrgRepo(ctx *context.APIContext) {
	getOpts := models.ParseRepoGetOpts(ctx)

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
			log.Error("Error has occurred while get repository by key %s: %v", getOpts.RepoKey, err)
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
			log.Error("Error has occurred while getting repository id %d: %v", repoId, err)
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
			log.Error("Error has occurred while getting tenant by tenant key: %s, project key %s. Error: %v", getOpts.TenantKey, getOpts.ProjectKey, err)
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

	action := role_model.READ
	if repository.IsPrivate {
		action = role_model.READ_PRIVATE
	}

	ctxTrace := gocontext.WithValue(ctx, trace_model.Key, "v2")
	ctxTrace = gocontext.WithValue(ctxTrace, trace_model.EndpointKey, ctx.Req.RequestURI)
	ctxTrace = gocontext.WithValue(ctxTrace, trace_model.FrontedKey, false)

	allow, err := s.checkUserPermissionFn(ctxTrace, ctx.Doer, tenantOrg.TenantID, &organization.Organization{ID: tenantOrg.OrganizationID}, action)
	if err != nil {
		log.Error("Error has occurred while checking user permission to organization: %v", err)
		ctx.Error(http.StatusInternalServerError, "", "Fail to check user permission to organization")
		return
	}

	if !allow {
		log.Debug("User does not have permission to organization %d", tenantOrg.OrganizationID)
		ctx.Error(http.StatusNotFound, "", "User does not have permission to organization")
		return
	}

	repoResponse := &apirepo.RepositoryGetResponse{
		ID:            strconv.FormatInt(repository.ID, 10),
		TenantKey:     tenantOrg.OrgKey,
		ProjectKey:    tenantOrg.ProjectKey,
		RepositoryKey: scRepoKey.RepoKey,
		DefaultBranch: repository.DefaultBranch,
		Name:          repository.Name,
		Private:       repository.IsPrivate,
		URI:           fmt.Sprintf("/%s/%s", strings.ToLower(repository.OwnerName), repository.LowerName),
	}
	ctx.JSON(http.StatusOK, repoResponse)
}

func (s Server) createTenantOrgRepo(ctx *context.APIContext) {
	opt := web.GetForm(ctx).(*apirepo.CreateRepoOptions)

	if err := opt.Validate(); err != nil {
		log.Debug("Input params for creating repository are not valid: %v", err)
		ctx.Error(http.StatusBadRequest, "", "Incorrect params")
		return
	}

	var (
		tenantOrg  *tenant.ScTenantOrganizations
		org        *organization.Organization
		repoKey    *repo.ScRepoKey
		repoTenant *tenant.ScTenant
		err        error
	)

	auditParams := map[string]string{
		"repository": opt.Name,
		"owner":      ctx.Doer.Name,
	}

	// Check if tenant exists
	if tenantOrg, err = tenant.GetTenantOrganizationsByKeys(ctx, opt.TenantKey, opt.ProjectKey); err != nil {
		if tenant.IsTenantOrganizationsNotExists(err) {
			log.Error("Error has occurred while getting tenant by tenant key: %s, project key %s. Error: %v", opt.TenantKey, opt.ProjectKey, err)
			auditParams["error"] = "Error has occurred while creating repository - options aren't valid"
			audit.CreateAndSendEvent(audit.RepositoryCreateEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			ctx.Error(http.StatusNotFound, "", "Tenant not found by given tenant key and project key")
			return
		} else {
			log.Error("Error has occurred while getting tenant by org key %s and project key %s: %v", opt.TenantKey, opt.ProjectKey, err)
			auditParams["error"] = "Error has occurred while creating repository"
			audit.CreateAndSendEvent(audit.RepositoryCreateEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			ctx.Error(http.StatusInternalServerError, "", "Fail to get tenant by tenant key and project key")
			return
		}
	}

	// Check if repo key already used
	if _, err = s.repoKeyDB.GetRepoByKey(ctx, opt.RepositoryKey); err == nil {
		log.Error("Error has occurred while getting repository by key %s. Repo with key already exists.", opt.RepositoryKey)
		auditParams["error"] = "Error has occurred while creating repository - repository name been taken"
		audit.CreateAndSendEvent(audit.RepositoryCreateEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		ctx.Error(http.StatusConflict, "", "Fail to create repo by key, key already exists")
		return
	} else if !repo.IsErrorRepoKeyDoesntExists(err) {
		log.Error("Error has occurred while getting repository by key %s: %v", opt.RepositoryKey, err)
		auditParams["error"] = "Error has occurred while creating repository"
		audit.CreateAndSendEvent(audit.RepositoryCreateEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		ctx.Error(http.StatusInternalServerError, "", "Fail to get repository by key")
		return
	}

	if repoTenant, err = tenant.GetTenantByID(ctx, tenantOrg.TenantID); err != nil {
		log.Error("Error has occurred while getting tenant by id %s: %v", tenantOrg.TenantID, err)
		auditParams["error"] = "Error has occurred while creating repository"
		audit.CreateAndSendEvent(audit.RepositoryCreateEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		ctx.Error(http.StatusInternalServerError, "", "Fail to get tenant")
		return
	}
	if !repoTenant.IsActive {
		log.Debug("Tenant with id %s is not active", repoTenant.ID)
		auditParams["error"] = "Error has occurred while creating repository"
		audit.CreateAndSendEvent(audit.RepositoryCreateEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		ctx.Error(http.StatusNotFound, "", "Tenant is not active")
		return
	}

	if org, err = organization.GetOrgByID(ctx, tenantOrg.OrganizationID); err != nil {
		if user.IsErrUserNotExist(err) {
			log.Error("Error has occurred while getting project by id %d: %v", tenantOrg.OrganizationID, err)
			auditParams["error"] = "Error has occurred while creating repository - options aren't valid"
			audit.CreateAndSendEvent(audit.RepositoryCreateEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			ctx.Error(http.StatusNotFound, "", "Project not found")
			return
		} else {
			log.Error("Error has occurred while getting project by id %d: %v", tenantOrg.OrganizationID, err)
			auditParams["error"] = "Error has occurred while creating repository"
			audit.CreateAndSendEvent(audit.RepositoryCreateEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			ctx.Error(http.StatusInternalServerError, "", "Fail to get Project")
			return
		}
	}

	ctxTrace := gocontext.WithValue(ctx, trace_model.Key, "v2")
	ctxTrace = gocontext.WithValue(ctxTrace, trace_model.EndpointKey, ctx.Req.RequestURI)
	ctxTrace = gocontext.WithValue(ctxTrace, trace_model.FrontedKey, false)

	allow, err := s.checkUserPermissionFn(ctxTrace, ctx.Doer, tenantOrg.TenantID, &organization.Organization{ID: tenantOrg.OrganizationID}, role_model.CREATE)
	if err != nil {
		log.Error("Error has occurred while checking user permission to organization: %v", err)
		auditParams["error"] = "Error has occurred while creating repository"
		audit.CreateAndSendEvent(audit.RepositoryCreateEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		ctx.Error(http.StatusInternalServerError, "", "Fail to check user permission to organization")
		return
	}

	if !allow {
		log.Debug("User does not have permission to organization %d", tenantOrg.OrganizationID)
		auditParams["error"] = "Error has occurred while creating repository - options aren't valid"
		audit.CreateAndSendEvent(audit.RepositoryCreateEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		ctx.Error(http.StatusNotFound, "", "User does not have permission to organization")
		return
	}

	createOptions := repository.CreateRepoOptions{
		DefaultBranch: opt.DefaultBranch,
		Description:   opt.Description,
		Name:          opt.Name,
		IsPrivate:     *opt.Private,
		Readme:        "Default",
	}

	createRepository, err := repository.CreateRepository(ctx.Doer, org.AsUser(), createOptions)
	if err != nil {
		if repo.IsErrRepoAlreadyExist(err) {
			log.Error("The repository with the same name already exists")
			auditParams["error"] = "Error has occurred while creating repository - repository name been taken"
			audit.CreateAndSendEvent(audit.RepositoryCreateEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			ctx.Error(http.StatusConflict, "", "The repository with the same name already exists.")
		} else if repo.IsErrCreateUserRepo(err) {
			log.Error("Creating a repository outside the project is prohibited")
			auditParams["error"] = "Error has occurred while creating repository - creating a repository outside the project is prohibited"
			audit.CreateAndSendEvent(audit.RepositoryCreateEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			ctx.Error(http.StatusBadRequest, "", "Creating a repository outside the project is prohibited")
		} else {
			log.Error("Fail to create repository %v", err)
			auditParams["error"] = "Error has occurred while creating repository"
			audit.CreateAndSendEvent(audit.RepositoryCreateEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			ctx.Error(http.StatusInternalServerError, "", "Fail to create repository")
		}
		return
	}

	// Update repokey key
	if repoKey, err = s.repoKeyDB.GetRepoByRepoID(ctx, strconv.FormatInt(createRepository.ID, 10)); err != nil {
		log.Error("Error has occurred while getting repository by key %s: %v", opt.RepositoryKey, err)
		auditParams["error"] = "Error has occurred while creating repository"
		audit.CreateAndSendEvent(audit.RepositoryCreateEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		ctx.Error(http.StatusInternalServerError, "", "Fail to get repository by key")
		return
	}
	repoKey.RepoKey = opt.RepositoryKey
	if err = s.repoKeyDB.UpdateRepoKey(ctx, repoKey); err != nil {
		log.Error("Error has occurred while inserting repository key %d: %v", repoKey.ID, err)
		auditParams["error"] = "Error has occurred while creating repository"
		audit.CreateAndSendEvent(audit.RepositoryCreateEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		ctx.Error(http.StatusInternalServerError, "", "Fail to insert repository key")
		return
	}

	repositoryResp := apirepo.RepositoryPostResponse{
		ID:            strconv.FormatInt(createRepository.ID, 10),
		TenantKey:     opt.TenantKey,
		ProjectKey:    opt.ProjectKey,
		RepositoryKey: opt.RepositoryKey,
		DefaultBranch: createRepository.DefaultBranch,
		Name:          createRepository.Name,
		Private:       createRepository.IsPrivate,
		URI:           fmt.Sprintf("/%s/%s", org.LowerName, createRepository.LowerName),
	}

	audit.CreateAndSendEvent(audit.RepositoryCreateEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)
	ctx.JSON(http.StatusCreated, repositoryResp)
}

// GetOrgRepo returns repo according to tenant and project
func (s Server) GetOrgRepo(ctx *context.APIContext) {
	// swagger:operation GET /projects/repos repo getOrgRepo
	// ---
	// summary: Returns the repo by repo external key
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
	//     "$ref": "#/responses/repositoryGetResponse"
	//   "404":
	//     description: Not found
	//   "500":
	//     description: Internal server error

	s.getOrgRepo(ctx)
}

// CreateTenantOrgRepo creates repo for tenant and project
func (s Server) CreateTenantOrgRepo(ctx *context.APIContext) {
	// swagger:operation POST /projects/repos repo createTenantOrgRepo
	// ---
	// summary: Creates the repo owned by the given tenant and project
	// consumes:
	// - application/json
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
	//       - project_key
	//       - repository_key
	//       - default_branch
	//       - description
	//       - name
	//       - private
	//     properties:
	//       tenant_key:
	//         type: string
	//         description: External key of tenant
	//       project_key:
	//         type: string
	//         description: External key of project
	//       repository_key:
	//         type: string
	//         description: External key of repository
	//       default_branch:
	//         type: string
	//         description: Name of branch
	//       description:
	//         type: integer
	//         description: Description of repository
	//       name:
	//         type: string
	//         description: Name of repository
	//       private:
	//         type: boolean
	//         description: Is repo private
	// responses:
	//   "201":
	//     "$ref": "#/responses/repositoryPostResponse"
	//   "400":
	//     description: Bad request
	//   "404":
	//     description: Not found
	//   "409":
	//     description: Conflict
	//   "500":
	//     description: Internal server error

	s.createTenantOrgRepo(ctx)
}

// SetMark - установка метки для репозитория проекта
func (s Server) SetMark(ctx *context.APIContext) {
	// swagger:operation POST /projects/repos/marks/codehub repo setMark
	// ---
	// summary: Set mark code hub for repo
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: body
	//   in: body
	//   description: Details of the mark to be set
	//   required: true
	//   schema:
	//     type: object
	//     required:
	//       - tenant_key
	//       - project_key
	//       - repo_key
	//     properties:
	//       tenant_key:
	//         type: string
	//         description: External key of tenant
	//       project_key:
	//         type: string
	//         description: External key of project
	//       repo_key:
	//         type: string
	//         description: External key of repository
	// responses:
	//   "201":
	//     description: Created
	//   "400":
	//     description: Bad request
	//   "404":
	//     description: Not found
	//   "409":
	//     description: Conflict
	//   "500":
	//     description: Internal server error
	s.setMark(ctx)
}

func (s Server) setMark(ctx *context.APIContext) {
	opt := web.GetForm(ctx).(*apirepo.SetMarkRequest)

	auditParams := map[string]string{}
	if err := opt.Validate(); err != nil {
		auditParams["error"] = "Error has occurred while validating repository - options aren't valid"
		audit.CreateAndSendEvent(audit.CodeHubMarkSetEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		log.Debug("Input params for creating repository are not valid: %v", err)
		ctx.Error(http.StatusBadRequest, "", "Incorrect params")
		return
	}

	var (
		tenantOrg  *tenant.ScTenantOrganizations
		scRepoKey  *repo.ScRepoKey
		repoTenant *tenant.ScTenant
		repos      *repo.Repository
		repoId     int64
		err        error
	)

	auditParams = map[string]string{
		"repository_key": opt.RepoKey,
		"project_key":    opt.ProjectKey,
		"tenant_key":     opt.TenantKey,
	}

	// Check if tenant exists
	if tenantOrg, err = tenant.GetTenantOrganizationsByKeys(ctx, opt.TenantKey, opt.ProjectKey); err != nil {
		if tenant.IsTenantOrganizationsNotExists(err) {
			log.Error("Error has occurred while getting tenant by tenant key: %s, project key %s. Error: %v", opt.TenantKey, opt.ProjectKey, err)
			auditParams["error"] = "Error has occurred while getting tenant by tenant_key and project_key"
			audit.CreateAndSendEvent(audit.CodeHubMarkSetEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			ctx.Error(http.StatusNotFound, "", repo.ErrorOrgDoestExist{TenantKey: opt.TenantKey, ProjectKey: opt.ProjectKey}.Error())
			return
		} else {
			log.Error("Error has occurred while getting tenant by org key %s and project key %s: %v", opt.TenantKey, opt.ProjectKey, err)
			auditParams["error"] = "Error has occurred while getting tenant by tenant_key and project_key"
			audit.CreateAndSendEvent(audit.CodeHubMarkSetEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			ctx.Error(http.StatusInternalServerError, "", "Err: fail to get tenant by tenant_key and project_key")
			return
		}
	}

	if repoTenant, err = tenant.GetTenantByID(ctx, tenantOrg.TenantID); err != nil {
		log.Error("Error has occurred while getting tenant by id %s: %v", tenantOrg.TenantID, err)
		auditParams["error"] = "Error has occurred while getting tenant by id"
		audit.CreateAndSendEvent(audit.CodeHubMarkSetEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		ctx.Error(http.StatusInternalServerError, "", "Err: fail to get tenant")
		return
	}

	if !repoTenant.IsActive {
		log.Debug("Tenant with id %s is not active", repoTenant.ID)
		auditParams["error"] = "Error has occurred while checking if tenant is active"
		audit.CreateAndSendEvent(audit.CodeHubMarkSetEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		ctx.Error(http.StatusNotFound, "", "Err: tenant is not active")
		return
	}

	isPublic, err := project.GetProject(*ctx, opt.TenantKey, opt.ProjectKey)
	if err != nil {
		log.Error("Error has occurred while getting tenant by id %s: %v", tenantOrg.TenantID, err)
		auditParams["error"] = "Error has occurred while getting tenant by id"
		audit.CreateAndSendEvent(audit.CodeHubMarkSetEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		ctx.Error(http.StatusBadRequest, "", "Err: fail to get project")
		return
	}
	if isPublic.Visibility.IsPrivate() {
		log.Debug("Project is not public: %s", opt.ProjectKey)
		auditParams["error"] = "Error has occurred while checking if project is public"
		audit.CreateAndSendEvent(audit.CodeHubMarkSetEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		ctx.Error(http.StatusBadRequest, "", "Err: project is not public")
		return
	}
	// Get repo by external key
	if scRepoKey, err = s.repoKeyDB.GetRepoByKey(ctx, opt.RepoKey); err != nil {
		if repo.IsErrorRepoKeyDoesntExists(err) {
			auditParams["error"] = "Error has occurred while getting repository by repo_key"
			audit.CreateAndSendEvent(audit.CodeHubMarkSetEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			log.Error("Error has occurred while get repository by key %s: %v", opt.RepoKey, err)
			ctx.Error(http.StatusNotFound, "", "Err: repo_key not exists")
			return
		} else {
			auditParams["error"] = "Error has occuredd while getting repository by repo_key"
			audit.CreateAndSendEvent(audit.CodeHubMarkSetEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			log.Error("Error has occurred while getting repository by key %s: %v", opt.RepoKey, err)
			ctx.Error(http.StatusInternalServerError, "", "Err: fail to get repository")
			return
		}
	}

	repoId, err = strconv.ParseInt(scRepoKey.RepoID, 10, 64)
	if err != nil {
		auditParams["error"] = "Error has occurred while parsing repository by id"
		audit.CreateAndSendEvent(audit.CodeHubMarkSetEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		log.Error("Error has occurred while parsing repository by id %s: %v", scRepoKey.RepoID, err)
		ctx.Error(http.StatusInternalServerError, "", "Fail to parse repository id")
		return
	}

	// Check if repository exists
	if repos, err = repo.GetRepositoryByID(ctx, repoId); err != nil {
		if repo.IsErrRepoNotExist(err) {
			auditParams["error"] = "Error has occurred while getting repository by id"
			audit.CreateAndSendEvent(audit.CodeHubMarkSetEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			log.Error("Error has occurred while getting repository id %d: %v", repoId, err)
			ctx.Error(http.StatusNotFound, "", "Err: repo not exists")
			return
		} else {
			auditParams["error"] = "Error has occurred while getting repository by id"
			audit.CreateAndSendEvent(audit.CodeHubMarkSetEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			log.Error("Error has occurred while getting repository by id %d: %v", repoId, err)
			ctx.Error(http.StatusInternalServerError, "", "Err: fail to get repository")
			return
		}
	}

	if repos.IsPrivate {
		log.Debug("Repo is not public: %s", opt.ProjectKey)
		auditParams["error"] = "Error has occurred while set marks - repository is private"
		audit.CreateAndSendEvent(audit.CodeHubMarkSetEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		ctx.Error(http.StatusBadRequest, "", "Err: repo is not public")
		return
	}

	// Set repository labels
	if err = s.repoMarksEditor.InsertRepoMark(ctx, scRepoKey.RepoKey, ctx.Doer.ID, s.codeHubMark); err != nil {
		if errors.As(err, &repo_marks_db.ErrMarkAlreadyExists{}) {
			auditParams["error"] = "Error has occurred while inserting repository mark"
			audit.CreateAndSendEvent(audit.CodeHubMarkSetEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			log.Error("Error has occurred while setting repository mark: %v", err)
			ctx.Error(http.StatusConflict, "", "Err: repository mark already exists")
			return
		}
		auditParams["error"] = "Error has occurred while inserting repository mark"
		audit.CreateAndSendEvent(audit.CodeHubMarkSetEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		log.Error("Error has occurred while setting repository mark: %v", err)
		ctx.Error(http.StatusInternalServerError, "", "Err: fail to set repository mark")
		return
	}
	audit.CreateAndSendEvent(audit.CodeHubMarkSetEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)

	ctx.Status(http.StatusCreated)
}

// DeleteMark - удаление метки для репозитория в проекте
func (s Server) DeleteMark(ctx *context.APIContext) {
	// swagger:operation DELETE /projects/repos/marks/codehub repo mark
	// ---
	// summary: Delete mark code hub for repo
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: body
	//   in: body
	//   description: Details of the mark to be set
	//   required: true
	//   schema:
	//     type: object
	//     required:
	//       - tenant_key
	//       - project_key
	//       - repo_key
	//     properties:
	//       tenant_key:
	//         type: string
	//         description: External key of tenant
	//       project_key:
	//         type: string
	//         description: External key of project
	//       repo_key:
	//         type: string
	//         description: External key of repository
	// responses:
	//   "200":
	//     description: OK
	//   "400":
	//     description: Bad request
	//   "404":
	//     description: Not found
	//   "500":
	//     description: Internal server error
	s.deleteMark(ctx)
}

func (s Server) deleteMark(ctx *context.APIContext) {
	opt := web.GetForm(ctx).(*apirepo.DeleteMarkRequest)
	auditParams := map[string]string{}
	if err := opt.Validate(); err != nil {
		auditParams["error"] = "Error has occurred while validating repository - options aren't valid"
		audit.CreateAndSendEvent(audit.CodeHubMarkDeleteEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		log.Debug("Input params for creating repository are not valid: %v", err)
		ctx.Error(http.StatusBadRequest, "", "Incorrect params")
		return
	}

	var (
		tenantOrg  *tenant.ScTenantOrganizations
		scRepoKey  *repo.ScRepoKey
		repoTenant *tenant.ScTenant
		repoId     int64
		err        error
	)

	auditParams = map[string]string{
		"repository_key": opt.RepoKey,
		"project_key":    opt.ProjectKey,
		"tenant_key":     opt.TenantKey,
	}

	// Check if tenant exists
	if tenantOrg, err = tenant.GetTenantOrganizationsByKeys(ctx, opt.TenantKey, opt.ProjectKey); err != nil {
		if tenant.IsTenantOrganizationsNotExists(err) {
			log.Error("Error has occurred while getting tenant by tenant key: %s, project key %s. Error: %v", opt.TenantKey, opt.ProjectKey, err)
			auditParams["error"] = "Error has occurred while getting tenant by tenant_key and project_key"
			audit.CreateAndSendEvent(audit.CodeHubMarkDeleteEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			ctx.Error(http.StatusNotFound, "", repo.ErrorOrgDoestExist{TenantKey: opt.TenantKey, ProjectKey: opt.ProjectKey}.Error())
			return
		} else {
			log.Error("Error has occurred while getting tenant by org key %s and project key %s: %v", opt.TenantKey, opt.ProjectKey, err)
			auditParams["error"] = "Error has occurred while getting tenant by tenant_key and project_key"
			audit.CreateAndSendEvent(audit.CodeHubMarkDeleteEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			ctx.Error(http.StatusInternalServerError, "", "Err: fail to get tenant by tenant_key and project_key")
			return
		}
	}

	if repoTenant, err = tenant.GetTenantByID(ctx, tenantOrg.TenantID); err != nil {
		log.Error("Error has occurred while getting tenant by id %s: %v", tenantOrg.TenantID, err)
		auditParams["error"] = "Error has occurred while getting tenant by id"
		audit.CreateAndSendEvent(audit.CodeHubMarkDeleteEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		ctx.Error(http.StatusInternalServerError, "", "Err: fail to get tenant")
		return
	}

	if !repoTenant.IsActive {
		log.Debug("Tenant with id %s is not active", repoTenant.ID)
		auditParams["error"] = "Error has occurred while checking if tenant is active"
		audit.CreateAndSendEvent(audit.CodeHubMarkDeleteEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		ctx.Error(http.StatusNotFound, "", "Err: tenant is not active")
		return
	}

	// Get repo by external key
	if scRepoKey, err = s.repoKeyDB.GetRepoByKey(ctx, opt.RepoKey); err != nil {
		if repo.IsErrorRepoKeyDoesntExists(err) {
			auditParams["error"] = "Error has occurred while getting repository by repo_key"
			audit.CreateAndSendEvent(audit.CodeHubMarkDeleteEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			log.Error("Error has occurred while get repository by key %s: %v", opt.RepoKey, err)
			ctx.Error(http.StatusNotFound, "", "Err: repo_key not exists")
			return
		} else {
			auditParams["error"] = "Error has occuredd while getting repository by repo_key"
			audit.CreateAndSendEvent(audit.CodeHubMarkDeleteEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			log.Error("Error has occurred while getting repository by key %s: %v", opt.RepoKey, err)
			ctx.Error(http.StatusInternalServerError, "", "Err: fail to get repository")
			return
		}
	}

	repoId, err = strconv.ParseInt(scRepoKey.RepoID, 10, 64)
	if err != nil {
		auditParams["error"] = "Error has occurred while parsing repository by id"
		audit.CreateAndSendEvent(audit.CodeHubMarkDeleteEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		log.Error("Error has occurred while parsing repository by id %s: %v", scRepoKey.RepoID, err)
		ctx.Error(http.StatusInternalServerError, "", "Fail to parse repository id")
		return
	}

	// Check if repository exists
	if _, err = repo.GetRepositoryByID(ctx, repoId); err != nil {
		if repo.IsErrRepoNotExist(err) {
			auditParams["error"] = "Error has occurred while getting repository by id"
			audit.CreateAndSendEvent(audit.CodeHubMarkDeleteEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			log.Error("Error has occurred while getting repository id %d: %v", repoId, err)
			ctx.Error(http.StatusNotFound, "", "Err: repository not exists")
			return
		} else {
			auditParams["error"] = "Error has occurred while getting repository by id"
			audit.CreateAndSendEvent(audit.CodeHubMarkDeleteEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			log.Error("Error has occurred while getting repository by id %d: %v", repoId, err)
			ctx.Error(http.StatusInternalServerError, "", "Err: fail to get repository")
			return
		}
	}

	// Delete repository labels
	if err = s.repoMarksEditor.DeleteRepoMark(ctx, scRepoKey.RepoKey, s.codeHubMark); err != nil {
		auditParams["error"] = "Error has occurred while deleting repository mark"
		audit.CreateAndSendEvent(audit.CodeHubMarkDeleteEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		log.Error("Error has occurred while setting repository mark: %v", err)
		ctx.Error(http.StatusInternalServerError, "", "Err: fail to delete repository mark")
		return
	}
	audit.CreateAndSendEvent(audit.CodeHubMarkDeleteEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)

	ctx.Status(http.StatusOK)
}
