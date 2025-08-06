// Copyright 2018 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	issues_model "code.gitea.io/gitea/models/issues"
	pull_model "code.gitea.io/gitea/models/pull"
	"code.gitea.io/gitea/modules/base"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/json"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/sbt/audit"
	auditutils "code.gitea.io/gitea/modules/sbt/audit/utils"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/services/forms"
	pull_service "code.gitea.io/gitea/services/pull"
)

const (
	tplConversation base.TplName = "repo/diff/conversation"
	tplNewComment   base.TplName = "repo/diff/new_comment"
)

// RenderNewCodeCommentForm will render the form for creating a new review comment
func RenderNewCodeCommentForm(ctx *context.Context) {
	issue := GetActionIssue(ctx)
	if !issue.IsPull {
		return
	}
	currentReview, err := issues_model.GetCurrentReview(ctx, ctx.Doer, issue)
	if err != nil && !issues_model.IsErrReviewNotExist(err) {
		ctx.ServerError("GetCurrentReview", err)
		return
	}
	ctx.Data["PageIsPullFiles"] = true
	ctx.Data["Issue"] = issue
	ctx.Data["CurrentReview"] = currentReview
	pullHeadCommitID, err := ctx.Repo.GitRepo.GetRefCommitID(issue.PullRequest.GetGitRefName())
	if err != nil {
		ctx.ServerError("GetRefCommitID", err)
		return
	}
	ctx.Data["AfterCommitID"] = pullHeadCommitID
	ctx.HTML(http.StatusOK, tplNewComment)
}

// CreateCodeComment will create a code comment including an pending review if required
func CreateCodeComment(ctx *context.Context) {
	auditValues := auditutils.NewRequiredAuditParams(ctx)

	repoName := audit.EmptyRequiredField
	repoOwnerName := audit.EmptyRequiredField
	repoID := audit.EmptyRequiredField
	if ctx.Repo != nil && ctx.Repo.Repository != nil {
		repoName = ctx.Repo.Repository.Name
		repoOwnerName = ctx.Repo.Repository.OwnerName
		repoID = strconv.FormatInt(ctx.Repo.Repository.ID, 10)
	}

	auditParams := map[string]string{
		"repository":    repoName,
		"repository_id": repoID,
		"owner":         repoOwnerName,
	}

	form := web.GetForm(ctx).(*forms.CodeCommentForm)
	issue := GetActionIssue(ctx)
	if !issue.IsPull {
		auditParams["error"] = "Error has occurred while getting pull request info"
		audit.CreateAndSendEvent(audit.CommentCreateCodeEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
		return
	}

	newValue := struct {
		Line           int64
		Reply          int64
		Origin         string
		Content        string
		TreePath       string
		LatestCommitID string
		Side           string
		SingleReview   bool
	}{
		Origin:         form.Origin,
		Content:        form.Content,
		Line:           form.Line,
		Side:           form.Side,
		TreePath:       form.TreePath,
		SingleReview:   !form.SingleReview,
		Reply:          form.Reply,
		LatestCommitID: form.LatestCommitID,
	}

	newValueBytes, _ := json.Marshal(newValue)
	auditParams["new_value"] = string(newValueBytes)

	if ctx.Written() {
		auditParams["error"] = "Error occurred while validating form"
		audit.CreateAndSendEvent(audit.CommentCreateCodeEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
		return
	}

	if !ctx.IsUserRepoAdmin() {
		auditParams["error"] = "Error occurred while checking user is repo admin"
		audit.CreateAndSendEvent(audit.CommentCreateCodeEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
		ctx.Error(http.StatusForbidden)
		return
	}

	if ctx.HasError() {
		auditParams["error"] = "Error occurs in form validation"
		audit.CreateAndSendEvent(audit.CommentCreateCodeEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
		ctx.Flash.Error(ctx.Data["ErrorMsg"].(string))
		ctx.Redirect(fmt.Sprintf("%s/pulls/%d/files", ctx.Repo.RepoLink, issue.Index))
		return
	}

	signedLine := form.Line
	if form.Side == "previous" {
		signedLine *= -1
	}

	comment, err := pull_service.CreateCodeComment(ctx,
		ctx.Doer,
		ctx.Repo.GitRepo,
		issue,
		signedLine,
		form.Content,
		form.TreePath,
		!form.SingleReview,
		form.Reply,
		form.LatestCommitID,
	)
	if err != nil {
		auditParams["error"] = "Error occurred while creating code comment"
		audit.CreateAndSendEvent(audit.CommentCreateCodeEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
		ctx.ServerError("CreateCodeComment", err)
		return
	}

	if comment == nil {
		auditParams["error"] = "Error occurred while creating code comment"
		audit.CreateAndSendEvent(audit.CommentCreateCodeEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
		log.Trace("Comment not created: %-v #%d[%d]", ctx.Repo.Repository, issue.Index, issue.ID)
		ctx.Redirect(fmt.Sprintf("%s/pulls/%d/files", ctx.Repo.RepoLink, issue.Index))
		return
	}
	audit.CreateAndSendEvent(audit.CommentCreateCodeEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusSuccess, auditValues.RemoteAddress, auditParams)
	log.Trace("Comment created: %-v #%d[%d] Comment[%d]", ctx.Repo.Repository, issue.Index, issue.ID, comment.ID)

	if form.Origin == "diff" {
		renderConversation(ctx, comment)
		return
	}
	ctx.Redirect(comment.Link())
}

// UpdateResolveConversation add or remove an Conversation resolved mark
func UpdateResolveConversation(ctx *context.Context) {
	origin := ctx.FormString("origin")
	action := ctx.FormString("action")
	commentID := ctx.FormInt64("comment_id")

	comment, err := issues_model.GetCommentByID(ctx, commentID)
	if err != nil {
		ctx.ServerError("GetIssueByID", err)
		return
	}

	if err = comment.LoadIssue(ctx); err != nil {
		ctx.ServerError("comment.LoadIssue", err)
		return
	}

	if comment.Issue.RepoID != ctx.Repo.Repository.ID {
		ctx.NotFound("comment's repoID is incorrect", errors.New("comment's repoID is incorrect"))
		return
	}

	var permResult bool
	if permResult, err = issues_model.CanMarkConversation(comment.Issue, ctx.Doer); err != nil {
		ctx.ServerError("CanMarkConversation", err)
		return
	}
	if !permResult {
		ctx.Error(http.StatusForbidden)
		return
	}

	if !comment.Issue.IsPull {
		ctx.Error(http.StatusBadRequest)
		return
	}

	if action == "Resolve" || action == "UnResolve" {
		err = issues_model.MarkConversation(comment, ctx.Doer, action == "Resolve")
		if err != nil {
			ctx.ServerError("MarkConversation", err)
			return
		}
	} else {
		ctx.Error(http.StatusBadRequest)
		return
	}

	if origin == "diff" {
		renderConversation(ctx, comment)
		return
	}
	ctx.JSON(http.StatusOK, map[string]interface{}{
		"ok": true,
	})
}

func renderConversation(ctx *context.Context, comment *issues_model.Comment) {
	comments, err := issues_model.FetchCodeCommentsByLine(ctx, comment.Issue, ctx.Doer, comment.TreePath, comment.Line)
	if err != nil {
		ctx.ServerError("FetchCodeCommentsByLine", err)
		return
	}
	ctx.Data["PageIsPullFiles"] = true
	ctx.Data["comments"] = comments
	ctx.Data["CanMarkConversation"] = true
	ctx.Data["Issue"] = comment.Issue
	if err = comment.Issue.LoadPullRequest(ctx); err != nil {
		ctx.ServerError("comment.Issue.LoadPullRequest", err)
		return
	}
	pullHeadCommitID, err := ctx.Repo.GitRepo.GetRefCommitID(comment.Issue.PullRequest.GetGitRefName())
	if err != nil {
		ctx.ServerError("GetRefCommitID", err)
		return
	}
	ctx.Data["AfterCommitID"] = pullHeadCommitID
	ctx.HTML(http.StatusOK, tplConversation)
}

// DismissReview dismissing stale review by repo admin
func DismissReview(ctx *context.Context) {
	form := web.GetForm(ctx).(*forms.DismissReviewForm)
	comm, err := pull_service.DismissReview(ctx, form.ReviewID, ctx.Repo.Repository.ID, form.Message, ctx.Doer, true, true)
	if err != nil {
		ctx.ServerError("pull_service.DismissReview", err)
		return
	}

	ctx.Redirect(fmt.Sprintf("%s/pulls/%d#%s", ctx.Repo.RepoLink, comm.Issue.Index, comm.HashTag()))
}

// viewedFilesUpdate Struct to parse the body of a request to update the reviewed files of a PR
// If you want to implement an API to update the review, simply move this struct into modules.
type viewedFilesUpdate struct {
	Files         map[string]bool `json:"files"`
	HeadCommitSHA string          `json:"headCommitSHA"`
}

func UpdateViewedFiles(ctx *context.Context) {
	// Find corresponding PR
	issue := CheckPullInfo(ctx)
	if ctx.Written() {
		return
	}
	pull := issue.PullRequest

	var data *viewedFilesUpdate
	err := json.NewDecoder(ctx.Req.Body).Decode(&data)
	if err != nil {
		log.Warn("Attempted to update a review but could not parse request body: %v", err)
		ctx.Resp.WriteHeader(http.StatusBadRequest)
		return
	}

	// Expect the review to have been now if no head commit was supplied
	if data.HeadCommitSHA == "" {
		data.HeadCommitSHA = pull.HeadCommitID
	}

	updatedFiles := make(map[string]pull_model.ViewedState, len(data.Files))
	for file, viewed := range data.Files {

		// Only unviewed and viewed are possible, has-changed can not be set from the outside
		state := pull_model.Unviewed
		if viewed {
			state = pull_model.Viewed
		}
		updatedFiles[file] = state
	}

	if err := pull_model.UpdateReviewState(ctx, ctx.Doer.ID, pull.ID, data.HeadCommitSHA, updatedFiles); err != nil {
		ctx.ServerError("UpdateReview", err)
	}
}
