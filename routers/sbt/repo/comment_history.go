package repo

import (
	issuesModel "code.gitea.io/gitea/models/issues"
	"code.gitea.io/gitea/models/unit"
	"code.gitea.io/gitea/modules/context"
	apiError "code.gitea.io/gitea/routers/sbt/apierror"
	"code.gitea.io/gitea/routers/sbt/logger"
	"code.gitea.io/gitea/routers/sbt/response"
	"net/http"

	"github.com/sergi/go-diff/diffmatchpatch"
)

// GetCommentHistory  получает историю изменений комментария
func GetCommentHistory(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	index := ctx.ParamsInt64(":index")

	issue := getIssueByIndex(ctx, index)
	commentId := ctx.ParamsInt64("id")

	items, err := issuesModel.FetchIssueContentHistoryList(ctx, issue.ID, commentId)
	if err != nil {
		log.Error("Not able to get comment: %d history, error: %v", commentId, err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		return
	}

	results := make([]*response.CommentHistory, len(items))
	for i, item := range items {
		var actionText string
		if item.IsDeleted {
			actionText = "deleted"
		} else if item.IsFirstCreated {
			actionText = "created"
		} else {
			actionText = "edited"
		}
		results[i] = &response.CommentHistory{
			HistoryId:    item.HistoryID,
			Action:       actionText,
			UserId:       item.UserID,
			UserName:     item.UserName,
			UserFullName: item.UserFullName,
			Updated:      item.EditedUnix.AsTime(),
		}
	}

	ctx.JSON(http.StatusOK, results)
}

// canSoftDeleteContentHistory проверить что пользователь может затереть пункт истории
// Админы и владельцы могут удалять любую историю. Обычные пользователи могут удалить только свою историю.
func canSoftDeleteContentHistory(ctx *context.Context, issue *issuesModel.Issue, comment *issuesModel.Comment,
	history *issuesModel.ContentHistory,
) bool {
	canSoftDelete := false
	if ctx.Repo.IsOwner() {
		canSoftDelete = true
	} else if ctx.Repo.CanWrite(unit.TypeIssues) {
		if comment == nil {
			// the issue poster or the history poster can soft-delete
			canSoftDelete = ctx.Doer.ID == issue.PosterID || ctx.Doer.ID == history.PosterID
			canSoftDelete = canSoftDelete && (history.IssueID == issue.ID)
		} else {
			// the comment poster or the history poster can soft-delete
			canSoftDelete = ctx.Doer.ID == comment.PosterID || ctx.Doer.ID == history.PosterID
			canSoftDelete = canSoftDelete && (history.IssueID == issue.ID)
			canSoftDelete = canSoftDelete && (history.CommentID == comment.ID)
		}
	}
	return canSoftDelete
}

// GetCommentHistoryDetail получает детали изменений
func GetCommentHistoryDetail(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	issueId := ctx.ParamsInt64(":index")
	issue := getIssueByIndex(ctx, issueId)

	historyID := ctx.FormInt64("history_id")
	history, prevHistory, err := issuesModel.GetIssueContentHistoryAndPrev(ctx, historyID)
	if err != nil {
		log.Debug("Comment history id: %d not found", historyID)
		ctx.JSON(http.StatusBadRequest, apiError.CommentHistoryDetailNotFound())
		return
	}

	var comment *issuesModel.Comment
	if history.CommentID != 0 {
		var err error
		if comment, err = issuesModel.GetCommentByID(ctx, history.CommentID); err != nil {
			log.Error("Not able to get comment id: %d for history id: %d, error: %v", history.CommentID, historyID, err)
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
			return
		}
	}
	var prevHistoryID int64
	var prevHistoryContentText string
	if prevHistory != nil {
		prevHistoryID = prevHistory.ID
		prevHistoryContentText = prevHistory.ContentText
	}

	// сравниваем изменения
	dmp := diffmatchpatch.New()
	diff := dmp.DiffMain(prevHistoryContentText, history.ContentText, false)
	diff = dmp.DiffCleanupEfficiency(diff)

	diffs := make([]response.Diff, len(diff))
	for i, it := range diff {
		diffs[i].Text = it.Text
		diffs[i].Type = it.Type.String()
	}

	result := &response.CommentHistoryDetail{
		Current:       history.ContentText,
		Previous:      prevHistoryContentText,
		PreviousId:    prevHistoryID,
		Diff:          diffs,
		CanSoftDelete: canSoftDeleteContentHistory(ctx, issue, comment, history),
	}

	ctx.JSON(http.StatusOK, result)
}

// SoftDeleteCommentHistory удалить историю, удаляется только содержимое, пометка о факте изменений остается
func SoftDeleteCommentHistory(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	issueId := ctx.ParamsInt64(":index")
	issue := getIssueByIndex(ctx, issueId)
	if issue == nil {
		return
	}

	commentID := ctx.ParamsInt64(":id")
	historyID := ctx.FormInt64("history_id")

	var comment *issuesModel.Comment
	var history *issuesModel.ContentHistory
	var err error
	if commentID != 0 {
		if comment, err = issuesModel.GetCommentByID(ctx, commentID); err != nil {
			log.Error("Not able to get comment id: %d, error: %v", commentID, err)
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
			return
		}
	}
	if history, err = issuesModel.GetIssueContentHistoryByID(ctx, historyID); err != nil {
		log.Debug("Comment history id: %d not found", historyID)
		ctx.JSON(http.StatusBadRequest, apiError.CommentHistoryDetailNotFound())
		return
	}

	canSoftDelete := canSoftDeleteContentHistory(ctx, issue, comment, history)
	if !canSoftDelete {
		log.Debug("User is not authorized to delete comment history id: %d not found", historyID)
		ctx.JSON(http.StatusForbidden, apiError.UserUnauthorized())
		return
	}

	err = issuesModel.SoftDeleteIssueContentHistory(ctx, historyID)
	if err != nil {
		log.Error("Not able to delete comment history id: %d, error: %v", historyID, err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		return
	}

	ctx.Status(http.StatusOK)
}
