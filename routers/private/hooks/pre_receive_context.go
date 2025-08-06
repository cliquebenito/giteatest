package hooks

import (
	"fmt"
	"net/http"
	"strconv"

	asymkey_model "code.gitea.io/gitea/models/asymkey"
	git_model "code.gitea.io/gitea/models/git"
	issues_model "code.gitea.io/gitea/models/issues"
	"code.gitea.io/gitea/models/organization"
	perm_model "code.gitea.io/gitea/models/perm"
	access_model "code.gitea.io/gitea/models/perm/access"
	"code.gitea.io/gitea/models/role_model"
	"code.gitea.io/gitea/models/tenant"
	"code.gitea.io/gitea/models/unit"
	user_model "code.gitea.io/gitea/models/user"
	gitea_context "code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/private"
	"code.gitea.io/gitea/modules/sbt/audit"
	"code.gitea.io/gitea/modules/setting"
)

type preReceiveContext struct {
	*gitea_context.PrivateContext

	// loadedPusher indicates that where the following information are loaded
	loadedPusher        bool
	user                *user_model.User // it's the org user if a DeployKey is used
	userPerm            access_model.Permission
	deployKeyAccessMode perm_model.AccessMode

	canCreatePullRequest        bool
	checkedCanCreatePullRequest bool

	canWriteCode        bool
	checkedCanWriteCode bool

	protectedTags    []*git_model.ProtectedTag
	gotProtectedTags bool

	env []string

	opts *private.HookOptions

	branchName string
}

// CanWriteCode returns true if pusher can write code
func (ctx *preReceiveContext) CanWriteCode() bool {
	if !ctx.checkedCanWriteCode {
		if !ctx.loadPusherAndPermission() {
			return false
		}
		// если у нас включена ролевая модель SourceControl, то запускается проверка привилегий на запись в репозиторий
		if setting.SourceControl.TenantWithRoleModeEnabled {
			if ctx.user != nil && ctx.Repo.Repository.Owner.IsOrganization() {
				tenantId, err := tenant.GetTenantByOrgIdOrDefault(ctx, ctx.Repo.Repository.OwnerID)
				if err != nil {
					ctx.canWriteCode = false
				} else {
					allowed, err := role_model.CheckUserPermissionToOrganization(ctx, ctx.user, tenantId, &organization.Organization{ID: ctx.Repo.Repository.OwnerID}, role_model.WRITE)
					if err != nil {
						ctx.canCreatePullRequest = false
					}
					if !allowed {
						allowCustomPrivilege, errGetCustomPrivilege := role_model.CheckPermissionForUserOfTeam(ctx, ctx.user.ID, ctx.Repo.Repository.OwnerID, ctx.Repo.Repository.ID, role_model.CreatePR.String())
						if errGetCustomPrivilege == nil && allowCustomPrivilege {
							allowed = true
						}
					}
					ctx.canWriteCode = allowed
				}
			}
		} else {
			ctx.canWriteCode = issues_model.CanMaintainerWriteToBranch(ctx.userPerm, ctx.branchName, ctx.user) || ctx.deployKeyAccessMode >= perm_model.AccessModeWrite
		}
	}
	return ctx.canWriteCode
}

// AssertCanWriteCode returns true if pusher can write code
func (ctx *preReceiveContext) AssertCanWriteCode() bool {
	if !ctx.CanWriteCode() {
		auditParams := map[string]string{
			"repository":    ctx.Repo.Repository.Name,
			"repository_id": strconv.FormatInt(ctx.Repo.Repository.ID, 10),
			"owner":         ctx.Repo.Repository.OwnerName,
		}
		if ctx.Written() {
			auditParams["error"] = "Error has occurred while checking ability to write code"
			audit.CreateAndSendEvent(audit.ChangesPushEvent, ctx.opts.UserName, strconv.FormatInt(ctx.opts.UserID, 10), audit.StatusFailure, ctx.PrivateContext.Req.RemoteAddr, auditParams)
			return false
		}

		auditParams["error"] = "User permission denied for writing"
		audit.CreateAndSendEvent(audit.ChangesPushEvent, ctx.user.Name, strconv.FormatInt(ctx.user.ID, 10), audit.StatusFailure, ctx.PrivateContext.Req.RemoteAddr, auditParams)
		ctx.JSON(http.StatusForbidden, private.Response{
			UserMsg: "User permission denied for writing.",
		})
		return false
	}
	return true
}

// CanCreatePullRequest returns true if pusher can create pull requests
func (ctx *preReceiveContext) CanCreatePullRequest() bool {
	if !ctx.checkedCanCreatePullRequest {
		if !ctx.loadPusherAndPermission() {
			return false
		}
		// если у нас включена ролевая модель SourceControl, то запускается проверка привилегий на запись в репозиторий
		if setting.SourceControl.TenantWithRoleModeEnabled {
			if ctx.user != nil && ctx.Repo.Repository.Owner.IsOrganization() {
				tenantId, err := tenant.GetTenantByOrgIdOrDefault(ctx, ctx.Repo.Repository.OwnerID)
				if err != nil {
					ctx.canCreatePullRequest = false
				} else {
					allowed, err := role_model.CheckUserPermissionToOrganization(ctx, ctx.user, tenantId, &organization.Organization{ID: ctx.Repo.Repository.OwnerID}, role_model.WRITE)
					if err != nil {
						ctx.canCreatePullRequest = false
					}
					if !allowed {
						allowCustomPrivilege, errGetCustomPrivilege := role_model.CheckPermissionForUserOfTeam(ctx, ctx.user.ID, ctx.Repo.Repository.OwnerID, ctx.Repo.Repository.ID, role_model.CreatePR.String())
						if errGetCustomPrivilege == nil && allowCustomPrivilege {
							allowed = true
						}
					}
					ctx.canCreatePullRequest = allowed
				}
			} else {
				ctx.canCreatePullRequest = false
			}
		} else {
			ctx.canCreatePullRequest = ctx.userPerm.CanRead(unit.TypePullRequests)
		}
		ctx.checkedCanCreatePullRequest = true
	}
	return ctx.canCreatePullRequest
}

// AssertCreatePullRequest returns true if can create pull requests
func (ctx *preReceiveContext) AssertCreatePullRequest() bool {
	if !ctx.CanCreatePullRequest() {
		auditParams := map[string]string{
			"repository":    ctx.Repo.Repository.Name,
			"repository_id": strconv.FormatInt(ctx.Repo.Repository.ID, 10),
			"owner":         ctx.Repo.Repository.OwnerName,
		}
		if ctx.Written() {
			auditParams["error"] = "Error has occurred while checking ability create pull request"
			audit.CreateAndSendEvent(audit.ChangesPushEvent, ctx.opts.UserName, strconv.FormatInt(ctx.opts.UserID, 10), audit.StatusFailure, ctx.PrivateContext.Req.RemoteAddr, auditParams)
			return false
		}
		auditParams["error"] = "User permission denied for creating pull-request"
		audit.CreateAndSendEvent(audit.ChangesPushEvent, ctx.user.Name, strconv.FormatInt(ctx.user.ID, 10), audit.StatusFailure, ctx.PrivateContext.Req.RemoteAddr, auditParams)
		ctx.JSON(http.StatusForbidden, private.Response{
			UserMsg: "User permission denied for creating pull-request.",
		})
		return false
	}
	return true
}

// loadPusherAndPermission returns false if an error occurs, and it writes the error response
func (ctx *preReceiveContext) loadPusherAndPermission() bool {
	if ctx.loadedPusher {
		return true
	}

	if ctx.opts.UserID == user_model.ActionsUserID {
		ctx.user = user_model.NewActionsUser()
		ctx.userPerm.AccessMode = perm_model.AccessMode(ctx.opts.ActionPerm)
		if err := ctx.Repo.Repository.LoadUnits(ctx); err != nil {
			log.Error("Unable to get User id %d Error: %v", ctx.opts.UserID, err)
			ctx.JSON(http.StatusInternalServerError, private.Response{
				Err: fmt.Sprintf("Unable to get User id %d Error: %v", ctx.opts.UserID, err),
			})
			return false
		}
		ctx.userPerm.Units = ctx.Repo.Repository.Units
		ctx.userPerm.UnitsMode = make(map[unit.Type]perm_model.AccessMode)
		for _, u := range ctx.Repo.Repository.Units {
			ctx.userPerm.UnitsMode[u.Type] = ctx.userPerm.AccessMode
		}
	} else {
		user, err := user_model.GetUserByID(ctx, ctx.opts.UserID)
		if err != nil {
			log.Error("Unable to get User id %d Error: %v", ctx.opts.UserID, err)
			ctx.JSON(http.StatusInternalServerError, private.Response{
				Err: fmt.Sprintf("Unable to get User id %d Error: %v", ctx.opts.UserID, err),
			})
			return false
		}
		ctx.user = user
		userPerm, err := access_model.GetUserRepoPermission(ctx, ctx.Repo.Repository, user)
		if err != nil {
			log.Error("Unable to get Repo permission of repo %s/%s of User %s: %v", ctx.Repo.Repository.OwnerName, ctx.Repo.Repository.Name, user.Name, err)
			ctx.JSON(http.StatusInternalServerError, private.Response{
				Err: fmt.Sprintf("Unable to get Repo permission of repo %s/%s of User %s: %v", ctx.Repo.Repository.OwnerName, ctx.Repo.Repository.Name, user.Name, err),
			})
			return false
		}
		ctx.userPerm = userPerm
	}

	if ctx.opts.DeployKeyID != 0 {
		deployKey, err := asymkey_model.GetDeployKeyByID(ctx, ctx.opts.DeployKeyID)
		if err != nil {
			log.Error("Unable to get DeployKey id %d Error: %v", ctx.opts.DeployKeyID, err)
			ctx.JSON(http.StatusInternalServerError, private.Response{
				Err: fmt.Sprintf("Unable to get DeployKey id %d Error: %v", ctx.opts.DeployKeyID, err),
			})
			return false
		}
		ctx.deployKeyAccessMode = deployKey.Mode
	}

	ctx.loadedPusher = true
	return true
}
