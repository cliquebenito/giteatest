// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"bytes"
	"html"
	"net/http"
	"strings"

	"code.gitea.io/gitea/modules/trace"
	"github.com/sergi/go-diff/diffmatchpatch"

	"code.gitea.io/gitea/models/avatars"
	issues_model "code.gitea.io/gitea/models/issues"
	"code.gitea.io/gitea/models/organization"
	"code.gitea.io/gitea/models/role_model"
	"code.gitea.io/gitea/models/unit"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/templates"
	"code.gitea.io/gitea/modules/timeutil"
	"code.gitea.io/gitea/routers/utils"
)

// GetContentHistoryOverview get overview
func GetContentHistoryOverview(ctx *context.Context) {
	issue := GetActionIssue(ctx)
	if issue == nil {
		return
	}

	editedHistoryCountMap, _ := issues_model.QueryIssueContentHistoryEditedCountMap(ctx, issue.ID)
	ctx.JSON(http.StatusOK, map[string]interface{}{
		"i18n": map[string]interface{}{
			"textEdited":                   ctx.Tr("repo.issues.content_history.edited"),
			"textDeleteFromHistory":        ctx.Tr("repo.issues.content_history.delete_from_history"),
			"textDeleteFromHistoryConfirm": ctx.Tr("repo.issues.content_history.delete_from_history_confirm"),
			"textOptions":                  ctx.Tr("repo.issues.content_history.options"),
		},
		"editedHistoryCountMap": editedHistoryCountMap,
	})
}

// GetContentHistoryList  get list
func GetContentHistoryList(ctx *context.Context) {
	issue := GetActionIssue(ctx)
	commentID := ctx.FormInt64("comment_id")
	if issue == nil {
		return
	}

	items, _ := issues_model.FetchIssueContentHistoryList(ctx, issue.ID, commentID)

	// render history list to HTML for frontend dropdown items: (name, value)
	// name is HTML of "avatar + userName + userAction + timeSince"
	// value is historyId
	var results []map[string]interface{}
	for _, item := range items {
		var actionText string
		if item.IsDeleted {
			actionTextDeleted := ctx.Locale.Tr("repo.issues.content_history.deleted")
			actionText = "<i data-history-is-deleted='1'>" + actionTextDeleted + "</i>"
		} else if item.IsFirstCreated {
			actionText = ctx.Locale.Tr("repo.issues.content_history.created")
		} else {
			actionText = ctx.Locale.Tr("repo.issues.content_history.edited")
		}

		username := item.UserName
		if setting.UI.DefaultShowFullName && strings.TrimSpace(item.UserFullName) != "" {
			username = strings.TrimSpace(item.UserFullName)
		}

		src := html.EscapeString(item.UserAvatarLink)
		class := avatars.DefaultAvatarClass + " gt-mr-3"
		name := html.EscapeString(username)
		avatarHTML := string(templates.AvatarHTML(src, 28, class, username))
		timeSinceText := string(timeutil.TimeSinceUnix(item.EditedUnix, ctx.Locale))

		results = append(results, map[string]interface{}{
			"name":  avatarHTML + "<strong>" + name + "</strong> " + actionText + " " + timeSinceText,
			"value": item.HistoryID,
		})
	}

	ctx.JSON(http.StatusOK, map[string]interface{}{
		"results": results,
	})
}

// canSoftDeleteContentHistory checks whether current user can soft-delete a history revision
// Admins or owners can always delete history revisions. Normal users can only delete own history revisions.
func canSoftDeleteContentHistory(ctx *context.Context, issue *issues_model.Issue, comment *issues_model.Comment,
	history *issues_model.ContentHistory, role string,
) bool {
	isCommentAuthor := ctx.Doer.ID == comment.PosterID
	isNotReader := role != role_model.READER.String()
	isOwner := role == role_model.OWNER.String()
	canSoftDelete := false
	if isOwner || (isCommentAuthor && isNotReader) {
		canSoftDelete = true
	} else if ctx.Repo.CanWrite(unit.TypeIssues) {
		if comment == nil {
			// the issue poster or the history poster can soft-delete
			canSoftDelete = ctx.Doer.ID == issue.PosterID || ctx.Doer.ID == history.PosterID
			canSoftDelete = canSoftDelete && (history.IssueID == issue.ID)
		} else {
			// the comment poster or the history poster can soft-delete
			canSoftDelete = ctx.Doer.ID == comment.PosterID && isCommentAuthor && isNotReader
			canSoftDelete = canSoftDelete && (history.IssueID == issue.ID)
			canSoftDelete = canSoftDelete && (history.CommentID == comment.ID)
		}
	}
	return canSoftDelete
}

// GetContentHistoryDetail get detail
func GetContentHistoryDetail(ctx *context.Context) {
	issue := GetActionIssue(ctx)
	if issue == nil {
		return
	}

	historyID := ctx.FormInt64("history_id")
	history, prevHistory, err := issues_model.GetIssueContentHistoryAndPrev(ctx, historyID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, map[string]interface{}{
			"message": "Can not find the content history",
		})
		return
	}

	// get the related comment if this history revision is for a comment, otherwise the history revision is for an issue.
	var comment *issues_model.Comment
	if history.CommentID != 0 {
		var err error
		if comment, err = issues_model.GetCommentByID(ctx, history.CommentID); err != nil {
			log.Error("can not get comment for issue content history %v. err=%v", historyID, err)
			return
		}
	}

	// get the previous history revision (if exists)
	var prevHistoryID int64
	var prevHistoryContentText string
	if prevHistory != nil {
		prevHistoryID = prevHistory.ID
		prevHistoryContentText = prevHistory.ContentText
	}
	enrichPrivilege, err := utils.GetTenantsPrivilegesByUserID(ctx, ctx.Doer.ID)
	if err != nil {
		log.Error("Error has occurred while getting user's privileges: %v", err)
		ctx.JSON(http.StatusInternalServerError, map[string]interface{}{
			"message": "Error has occurred while getting user's privileges",
		})
		return
	}
	var role string
	for _, v := range enrichPrivilege {
		if v.Org.Name == ctx.Repo.Owner.Name && ctx.Doer.ID == v.User.ID {
			role = v.Role.String()
			break
		}
	}
	//compare the current history revision with the previous one
	dmp := diffmatchpatch.New()
	// `checklines=false` makes better diff result
	diff := dmp.DiffMain(prevHistoryContentText, history.ContentText, false)
	diff = dmp.DiffCleanupEfficiency(diff)

	// use chroma to render the diff html
	diffHTMLBuf := bytes.Buffer{}
	diffHTMLBuf.WriteString("<pre class='chroma' style='tab-size: 4'>")
	for _, it := range diff {
		if it.Type == diffmatchpatch.DiffInsert {
			diffHTMLBuf.WriteString("<span class='gi'>")
			diffHTMLBuf.WriteString(html.EscapeString(it.Text))
			diffHTMLBuf.WriteString("</span>")
		} else if it.Type == diffmatchpatch.DiffDelete {
			diffHTMLBuf.WriteString("<span class='gd'>")
			diffHTMLBuf.WriteString(html.EscapeString(it.Text))
			diffHTMLBuf.WriteString("</span>")
		} else {
			diffHTMLBuf.WriteString(html.EscapeString(it.Text))
		}
	}
	diffHTMLBuf.WriteString("</pre>")

	ctx.JSON(http.StatusOK, map[string]interface{}{
		"canSoftDelete": canSoftDeleteContentHistory(ctx, issue, comment, history, role),
		"historyId":     historyID,
		"prevHistoryId": prevHistoryID,
		"diffHtml":      diffHTMLBuf.String(),
	})
}

// SoftDeleteContentHistory soft delete
func SoftDeleteContentHistory(ctx *context.Context) {
	logTracer := trace.NewLogTracer()
	message := logTracer.CreateTraceMessage(ctx)
	errTrace := logTracer.Trace(message)
	if errTrace != nil {
		log.Error("Error has occurred while creating trace message: %v", errTrace)
	}
	defer func() {
		errTrace = logTracer.TraceTime(message)
		if errTrace != nil {
			log.Error("Error has occurred while creating trace time message: %v", errTrace)
		}
	}()

	issue := GetActionIssue(ctx)
	if issue == nil {
		return
	}

	commentID := ctx.FormInt64("comment_id")
	historyID := ctx.FormInt64("history_id")
	var history *issues_model.ContentHistory
	comment, err := issues_model.GetCommentByID(ctx, commentID)
	if err != nil {
		log.Debug("Error has occurred while getting comment info for issue %v. err=%v", commentID, err)
		ctx.JSON(http.StatusNotFound, map[string]interface{}{
			"message": "Error has occurred while getting information about a comment",
		})
		return
	}

	tenantID, err := role_model.GetUserTenantId(ctx, ctx.Doer.ID)
	if err != nil {
		log.Debug("Error has occurred while getting user tenant id for user %v. err=%v", ctx.Doer.ID, err)
		ctx.JSON(http.StatusNotFound, map[string]interface{}{
			"message": "Error has occurred while getting user tenant id",
		})
		return
	}

	enrichPrivilege, err := utils.GetTenantsPrivilegesByUserID(ctx, ctx.Doer.ID)
	if err != nil {
		log.Error("Error has occurred while getting user's privileges: %v", err)
		ctx.JSON(http.StatusInternalServerError, map[string]interface{}{
			"message": "Error has occurred while getting user's privileges",
		})
		return
	}
	var role string
	for _, v := range enrichPrivilege {
		if v.Org.Name == ctx.Repo.Owner.Name && ctx.Doer.ID == v.User.ID {
			role = v.Role.String()
			break
		}
	}

	isNotCommentAuthor := ctx.Doer.ID != comment.PosterID
	if isNotCommentAuthor {
		allowed, err := role_model.CheckUserPermissionToOrganization(ctx, ctx.Doer, tenantID, &organization.Organization{ID: ctx.Repo.Repository.OwnerID}, role_model.MANAGE_COMMENTS)
		if err != nil {
			log.Error("Error has occurred while checking user's permissions: %v", err)
			ctx.JSON(http.StatusInternalServerError, map[string]interface{}{
				"message": "Error has occurred while checking user's permissions",
			})
			return
		}
		if !allowed {
			log.Error("Error has occurred while checking permission %v. err=%v", tenantID, err)
			ctx.JSON(http.StatusInternalServerError, map[string]interface{}{
				"message": "Error has occurred while checking access to the project under the tenant",
			})
			return
		}
	}

	if commentID != 0 {
		if comment, err = issues_model.GetCommentByID(ctx, commentID); err != nil {
			log.Error("can not get comment for issue content history %v. err=%v", historyID, err)
			return
		}
	}
	if history, err = issues_model.GetIssueContentHistoryByID(ctx, historyID); err != nil {
		log.Error("can not get issue content history %v. err=%v", historyID, err)
		return
	}

	canSoftDelete := canSoftDeleteContentHistory(ctx, issue, comment, history, role)
	if !canSoftDelete {
		ctx.JSON(http.StatusForbidden, map[string]interface{}{
			"message": "Can not delete the content history",
		})
		return
	}

	err = issues_model.SoftDeleteIssueContentHistory(ctx, historyID)
	log.Debug("soft delete issue content history. issue=%d, comment=%d, history=%d", issue.ID, commentID, historyID)
	ctx.JSON(http.StatusOK, map[string]interface{}{
		"ok": err == nil,
	})
}
