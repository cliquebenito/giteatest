package webhook

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"code.gitea.io/gitea/models/repo"
	webhook2 "code.gitea.io/gitea/models/webhook"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/sbt/audit"
	auditAssign "code.gitea.io/gitea/modules/sbt/audit/utils"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/util"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/routers/api/v2/models"
	"code.gitea.io/gitea/services/webhook"
)

type Server struct {
	service webhook.HookProcessor
	repo    repo.RepoKeyDB
}

func NewServer(service webhook.HookProcessor, repo repo.RepoKeyDB) *Server {
	return &Server{
		service: service,
		repo:    repo,
	}
}

// CreateHook create a hook for a repository
func (s Server) CreateHook(ctx *context.APIContext) {
	// swagger:operation POST /repos/webhooks repository repoCreateHook
	// ---
	// summary: Create a hook
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: project_key
	//   in: query
	//   description: Owner of the repo
	//   required: true
	//   type: string
	// - name: repo_key
	//   in: query
	//   description: Repository name
	//   required: true
	//   type: string
	// - name: tenant_key
	//   in: query
	//   description: Tenant identifier (UUID)
	//   required: true
	//   type: string
	// - name: body
	//   in: body
	//   description: Webhook creation options
	//   required: true
	//   schema:
	//     $ref: "#/definitions/CreateHookOption"
	// responses:
	//   "201":
	//     description: Created
	//     schema:
	//       $ref: "#/responses/Hook"
	form := web.GetForm(ctx).(*models.CreateHookOption)
	auditReqParams := auditAssign.NewRequiredAuditParamsFromApiContext(ctx)
	auditParams := map[string]string{
		"project_key": ctx.FormString("project_key"),
		"repo_key":    ctx.FormString("repo_key"),
		"email":       ctx.Doer.Email,
		"tenant_key":  ctx.FormString("tenant_key"),
	}

	if err := form.Validate(); err != nil {
		if models.IsErrInvalidHookContentType(err) {
			auditParams["error"] = "Error has occurred while validating form, invalid hook content type"
			audit.CreateAndSendEvent(audit.HookInRepositoryAddEvent, auditReqParams.DoerName, auditReqParams.DoerID, audit.StatusFailure, auditReqParams.RemoteAddress, auditParams)
			ctx.Error(http.StatusUnprocessableEntity, "", "Invalid content type")
			return
		}
		auditParams["error"] = "Error has occurred while validating form"
		audit.CreateAndSendEvent(audit.HookInRepositoryAddEvent, auditReqParams.DoerName, auditReqParams.DoerID, audit.StatusFailure, auditReqParams.RemoteAddress, auditParams)
		log.Debug("invalid form")
		ctx.Error(http.StatusBadRequest, "validation error", err)
		return
	}
	auditParams["hook_type"] = form.Type
	auditParams["config_url"] = form.Config.Url
	auditParams["content_type"] = form.Config.ContentType

	form.RepoID = ctx.Repo.Repository.ID
	form.OwnerID = ctx.Repo.Owner.ID
	apiHook, err := s.service.AddRepoHook(ctx, form)
	if err != nil {
		if models.IsErrInvalidHookType(err) {
			auditParams["error"] = "Error has occurred while adding hook in repo, hook type is invalid"
			audit.CreateAndSendEvent(audit.HookInRepositoryAddEvent, auditReqParams.DoerName, auditReqParams.DoerID, audit.StatusFailure, auditReqParams.RemoteAddress, auditParams)
			ctx.Error(http.StatusUnprocessableEntity, "", err)
			return
		}
		if webhook2.IsErrWebHookLimit(err) {
			auditParams["error"] = "Error has occurred while adding hook, the maximum number of hooks for this repository has been reached"
			audit.CreateAndSendEvent(audit.HookInRepositoryAddEvent, auditReqParams.DoerName, auditReqParams.DoerID, audit.StatusFailure, auditReqParams.RemoteAddress, auditParams)
			ctx.Error(http.StatusConflict, "", err)
			return
		}
		auditParams["error"] = "Error has occurred while adding hook in repo"
		audit.CreateAndSendEvent(audit.HookInRepositoryAddEvent, auditReqParams.DoerName, auditReqParams.DoerID, audit.StatusFailure, auditReqParams.RemoteAddress, auditParams)
		ctx.Error(http.StatusInternalServerError, "AddRepoHook", err)
		return
	}
	audit.CreateAndSendEvent(audit.HookInRepositoryAddEvent, auditReqParams.DoerName, auditReqParams.DoerID, audit.StatusSuccess, auditReqParams.RemoteAddress, auditParams)
	rootUrl := setting.CfgProvider.Section("server").Key("ROOT_URL")

	ctx.Resp.Header().Set("Location", fmt.Sprintf("%s%s/%s/settings/hooks/%d", rootUrl, ctx.Repo.Owner.Name,
		ctx.Repo.Repository.Name, apiHook.ID))

	ctx.JSON(http.StatusCreated, apiHook)
}

// GetHook get an organization's hook by id
func (s Server) GetHook(ctx *context.APIContext) {
	// swagger:operation GET /repos/webhooks repository repoGetHook
	// ---
	// summary: Get a hook
	// produces:
	// - application/json
	// parameters:
	// - name: project_key
	//   in: query
	//   description: Owner of the repo
	//   required: true
	//   type: string
	// - name: repo_key
	//   in: query
	//   description: Name of the repository
	//   required: true
	//   type: string
	// - name: tenant_key
	//   in: query
	//   description: Tenant identifier (UUID)
	//   required: true
	//   type: string
	// - name: id
	//   in: query
	//   description: ID of the hook to get
	//   required: true
	//   type: integer
	//   format: int64
	// responses:
	//   "200":
	//     description: successful operation
	//     schema:
	//       $ref: "#/responses/Hook"

	hookID, err := getHookID(ctx)
	if err != nil {
		if IsErrIDRequired(err) {
			ctx.Error(http.StatusBadRequest, "get hook ID", err)
			return
		}
		ctx.Error(http.StatusNotFound, "get hook ID", err)
		return
	}

	hook, err := webhook2.GetSystemOrDefaultWebhookWithParams(ctx, hookID, ctx.Repo.Repository.ID, ctx.Repo.Owner.ID)
	if err != nil {
		if errors.Is(err, util.ErrNotExist) {
			ctx.Error(http.StatusNotFound, "GetSystemOrDefaultWebhook", err)
			return
		} else {
			ctx.Error(http.StatusInternalServerError, "GetSystemOrDefaultWebhook", err)
			return
		}
	}
	h, err := webhook.ToHook(ctx.Repo.RepoLink, hook)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "convert.ToHook", err)
		return
	}
	ctx.JSON(http.StatusOK, h)
}

// DeleteHook delete a system hook
func (s Server) DeleteHook(ctx *context.APIContext) {
	// swagger:operation DELETE /repos/webhooks repository DeleteHook
	// ---
	// summary: Delete a hook
	// produces:
	// - application/json
	// parameters:
	// - name: project_key
	//   in: query
	//   description: Owner of the repo
	//   required: true
	//   type: string
	// - name: repo_key
	//   in: query
	//   description: Name of the repository
	//   required: true
	//   type: string
	// - name: tenant_key
	//   in: query
	//   description: Tenant identifier (UUID)
	//   required: true
	//   type: string
	// - name: id
	//   in: query
	//   description: ID of the hook to delete
	//   required: true
	//   type: integer
	//   format: int64
	// responses:
	//   "204":
	//     description: Successfully deleted

	auditReqParams := auditAssign.NewRequiredAuditParamsFromApiContext(ctx)
	auditParams := map[string]string{
		"project_key": ctx.FormString("project_key"),
		"repo_key":    ctx.FormString("repo_key"),
		"email":       ctx.Doer.Email,
		"tenant_key":  ctx.FormString("tenant_key"),
	}

	hookID, err := getHookID(ctx)
	if err != nil {
		if IsErrIDRequired(err) {
			ctx.Error(http.StatusBadRequest, "get hook ID", err)
			return
		}
		auditParams["error"] = "Error has occurred while getting hook ID"
		audit.CreateAndSendEvent(audit.HookInRepositoryRemoveEvent, auditReqParams.DoerName, auditReqParams.DoerID, audit.StatusFailure, auditReqParams.RemoteAddress, auditParams)
		ctx.Error(http.StatusNotFound, "get hook ID", err)
		return
	}

	auditParams["hook_id"] = strconv.FormatInt(hookID, 10)

	if err := webhook2.DeleteDefaultSystemWebhookWithParams(ctx, hookID, ctx.Repo.Repository.ID, ctx.Repo.Owner.ID); err != nil {
		if errors.Is(err, util.ErrNotExist) {
			auditParams["error"] = "Error has occurred while deleting default system webhook, hook is not exist"
			audit.CreateAndSendEvent(audit.HookInRepositoryRemoveEvent, auditReqParams.DoerName, auditReqParams.DoerID, audit.StatusFailure, auditReqParams.RemoteAddress, auditParams)
			ctx.Error(http.StatusNotFound, "DeleteDefaultSystemWebhook", err)
			return
		}
		auditParams["error"] = "Error has occurred while deleting default system webhook"
		audit.CreateAndSendEvent(audit.HookInRepositoryRemoveEvent, auditReqParams.DoerName, auditReqParams.DoerID, audit.StatusFailure, auditReqParams.RemoteAddress, auditParams)
		ctx.Error(http.StatusInternalServerError, "DeleteDefaultSystemWebhook", err)
		return
	}
	audit.CreateAndSendEvent(audit.HookInRepositoryRemoveEvent, auditReqParams.DoerName, auditReqParams.DoerID, audit.StatusSuccess, auditReqParams.RemoteAddress, auditParams)

	ctx.Status(http.StatusNoContent)
}
