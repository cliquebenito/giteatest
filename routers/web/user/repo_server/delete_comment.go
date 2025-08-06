package repo_server

import (
	"encoding/json"
	"net/http"
	"strconv"

	issues_model "code.gitea.io/gitea/models/issues"
	"code.gitea.io/gitea/models/organization"
	"code.gitea.io/gitea/models/role_model"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/sbt/audit"
	auditutils "code.gitea.io/gitea/modules/sbt/audit/utils"
	"code.gitea.io/gitea/modules/trace"
	issue_service "code.gitea.io/gitea/services/issue"
)

// DeleteComment delete comment of issue
func (s *Server) DeleteComment(ctx *context.Context) {
	logTracer := trace.NewLogTracer()
	message := logTracer.CreateTraceMessage(ctx)
	err := logTracer.Trace(message)
	if err != nil {
		log.Error("Error has occurred while creating trace message: %v", err)
	}
	defer func() {
		err = logTracer.TraceTime(message)
		if err != nil {
			log.Error("Error has occurred while creating trace time message: %v", err)
		}
	}()

	requiredAuditParams := auditutils.NewRequiredAuditParams(ctx)
	commentID := strconv.FormatInt(ctx.ParamsInt64(":id"), 10)
	repoOwnerName := audit.EmptyRequiredField
	repoName := audit.EmptyRequiredField
	if ctx.Repo != nil && ctx.Repo.Repository != nil {
		repoOwnerName = ctx.Repo.Repository.OwnerName
		repoName = ctx.Repo.Repository.Name
	}
	auditParams := map[string]string{
		"owner":      repoOwnerName,
		"repository": repoName,
		"comment_id": commentID,
	}

	tenantID, err := role_model.GetUserTenantId(ctx, ctx.Doer.ID)
	if err != nil {
		auditParams["error"] = "Error has occurred while getting tenant id"
		audit.CreateAndSendEvent(audit.CommentDeleteEvent, requiredAuditParams.DoerName, requiredAuditParams.DoerID, audit.StatusFailure, requiredAuditParams.RemoteAddress, auditParams)
		ctx.ServerError("Getting tenant", err)
		return
	}

	comment, err := issues_model.GetCommentByID(ctx, ctx.ParamsInt64(":id"))
	if err != nil {
		auditParams["error"] = "Error has occurred while getting comment info"
		audit.CreateAndSendEvent(audit.CommentDeleteEvent, requiredAuditParams.DoerName, requiredAuditParams.DoerID, audit.StatusFailure, requiredAuditParams.RemoteAddress, auditParams)
		ctx.NotFoundOrServerError("GetCommentByID", issues_model.IsErrCommentNotExist, err)
		return
	}

	type auditValue struct {
		PosterID       string
		CommentContent string
	}

	oldValue := auditValue{
		PosterID:       comment.PosterIDString(),
		CommentContent: comment.Content,
	}

	oldValueBytes, _ := json.Marshal(oldValue)
	auditParams["old_value"] = string(oldValueBytes)
	isNotCommentAuthor := ctx.Doer.ID != comment.PosterID
	// если запросил удаление не тот, кто создал комментарий,
	// то надо проверить есть ли у него права на удаление комментария
	if isNotCommentAuthor {
		allowed, err := role_model.CheckUserPermissionToOrganization(ctx, ctx.Doer, tenantID, &organization.Organization{ID: ctx.Repo.Repository.OwnerID}, role_model.MANAGE_COMMENTS)
		if err != nil {
			auditParams["error"] = "Error has occurred while deleting comment"
			audit.CreateAndSendEvent(audit.CommentDeleteEvent, requiredAuditParams.DoerName, requiredAuditParams.DoerID, audit.StatusFailure, requiredAuditParams.RemoteAddress, auditParams)
			ctx.NotFound("CheckUserPermissionToOrganization", err)
			return
		}
		if !allowed {
			auditParams["error"] = "Error has occurred while checking access to the project under the tenant"
			audit.CreateAndSendEvent(audit.CommentDeleteEvent, requiredAuditParams.DoerName, requiredAuditParams.DoerID, audit.StatusFailure, requiredAuditParams.RemoteAddress, auditParams)
			ctx.Error(http.StatusForbidden)
			return
		}
		auditParams["role"] = role_model.OWNER.String()
	}

	if err := comment.LoadIssue(ctx); err != nil {
		auditParams["error"] = "Error has occurred while loading issue"
		audit.CreateAndSendEvent(audit.CommentDeleteEvent, requiredAuditParams.DoerName, requiredAuditParams.DoerID, audit.StatusFailure, requiredAuditParams.RemoteAddress, auditParams)
		ctx.NotFoundOrServerError("LoadIssue", issues_model.IsErrIssueNotExist, err)
		return
	}

	if !ctx.IsSigned || (isNotCommentAuthor && !ctx.Repo.CanWriteIssuesOrPulls(comment.Issue.IsPull)) {
		auditParams["error"] = "Error has occurred while checking permissions"
		audit.CreateAndSendEvent(audit.CommentDeleteEvent, requiredAuditParams.DoerName, requiredAuditParams.DoerID, audit.StatusFailure, requiredAuditParams.RemoteAddress, auditParams)
		ctx.Error(http.StatusForbidden)
		return
	}
	if !comment.Type.HasContentSupport() {
		auditParams["error"] = "Error has occurred while checking comment type"
		audit.CreateAndSendEvent(audit.CommentDeleteEvent, requiredAuditParams.DoerName, requiredAuditParams.DoerID, audit.StatusFailure, requiredAuditParams.RemoteAddress, auditParams)
		ctx.Error(http.StatusNoContent)
		return
	}

	if err = issue_service.DeleteComment(ctx, ctx.Doer, comment); err != nil {
		auditParams["error"] = "Error has occurred while deleting comment"
		audit.CreateAndSendEvent(audit.CommentDeleteEvent, requiredAuditParams.DoerName, requiredAuditParams.DoerID, audit.StatusFailure, requiredAuditParams.RemoteAddress, auditParams)
		ctx.ServerError("DeleteComment", err)
		return
	}

	audit.CreateAndSendEvent(audit.CommentDeleteEvent, requiredAuditParams.DoerName, requiredAuditParams.DoerID, audit.StatusSuccess, requiredAuditParams.RemoteAddress, auditParams)
	ctx.Status(http.StatusOK)
}
