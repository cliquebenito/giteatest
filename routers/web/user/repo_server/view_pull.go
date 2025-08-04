package repo_server

import (
	"net/http"

	issues_model "code.gitea.io/gitea/models/issues"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/trace"
	"code.gitea.io/gitea/modules/upload"
	"code.gitea.io/gitea/routers/web/repo"
	"code.gitea.io/gitea/services/gitdiff"
)

// ViewPullFiles render pull request changed files list page
func (s *Server) ViewPullFiles(ctx *context.Context) {
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

	ctx.Data["PageIsPullList"] = true
	ctx.Data["PageIsPullFiles"] = true

	issue := repo.CheckPullInfo(ctx)
	if ctx.Written() {
		return
	}
	pull := issue.PullRequest

	var (
		startCommitID string
		endCommitID   string
		gitRepo       = ctx.Repo.GitRepo
	)

	var prInfo *git.CompareInfo
	if pull.HasMerged {
		prInfo = repo.PrepareMergedViewPullInfo(ctx, issue)
	} else {
		prInfo = repo.PrepareViewPullInfo(ctx, issue)
	}

	if ctx.Written() {
		return
	} else if prInfo == nil {
		ctx.NotFound("ViewPullFiles", nil)
		return
	}

	var headCommitID string
	var err error
	if pull.HasMerged {
		headCommitID = prInfo.HeadCommitID
	} else {
		headCommitID, err = gitRepo.GetRefCommitID(pull.GetGitRefName())
		if err != nil {
			ctx.ServerError("GetRefCommitID", err)
			return
		}
	}

	startCommitID = prInfo.MergeBase
	endCommitID = headCommitID

	ctx.Data["Username"] = ctx.Repo.Owner.Name
	ctx.Data["Reponame"] = ctx.Repo.Repository.Name
	ctx.Data["AfterCommitID"] = endCommitID

	fileOnly := ctx.FormBool("file-only")

	maxLines, maxFiles := setting.Git.MaxGitDiffLines, setting.Git.MaxGitDiffFiles
	files := ctx.FormStrings("files")
	if fileOnly && (len(files) == 2 || len(files) == 1) {
		maxLines, maxFiles = -1, -1
	}
	diffOptions := &gitdiff.DiffOptions{
		BeforeCommitID:     startCommitID,
		AfterCommitID:      endCommitID,
		SkipTo:             ctx.FormString("skip-to"),
		MaxLines:           maxLines,
		MaxLineCharacters:  setting.Git.MaxGitDiffLineCharacters,
		MaxFiles:           maxFiles,
		WhitespaceBehavior: gitdiff.GetWhitespaceFlag(ctx.Data["WhitespaceBehavior"].(string)),
	}

	var methodWithError string
	var diff *gitdiff.Diff
	if !ctx.IsSigned {
		diff, err = gitdiff.GetDiff(gitRepo, diffOptions, files...)
		methodWithError = "GetDiff"
	} else {
		diff, err = gitdiff.SyncAndGetUserSpecificDiff(ctx, ctx.Doer.ID, pull, gitRepo, diffOptions, files...)
		methodWithError = "SyncAndGetUserSpecificDiff"
	}
	if err != nil {
		ctx.ServerError(methodWithError, err)
		return
	}

	ctx.PageData["prReview"] = map[string]interface{}{
		"numberOfFiles":       diff.NumFiles,
		"numberOfViewedFiles": diff.NumViewedFiles,
	}

	if err = diff.LoadComments(ctx, issue, ctx.Doer); err != nil {
		ctx.ServerError("LoadComments", err)
		return
	}

	pb, err := s.protectedBranchManager.GetMergeMatchProtectedBranchRule(ctx, pull.BaseRepoID, pull.BaseBranch)
	if err != nil {
		log.Error("Error has occured while get merge match branch with repo id - %d, branch name: %v", pull.BaseRepoID, pull.BaseBranch, err)
		ctx.ServerError("LoadProtectedBranch", err)
		return
	}

	if pb != nil {
		glob := s.protectedBranchManager.GetProtectedFilePatterns(ctx, *pb)
		if len(glob) != 0 {
			for _, file := range diff.Files {
				file.IsProtected = s.protectedBranchManager.IsProtectedFile(ctx, *pb, glob, file.Name)
			}
		}
	}

	ctx.Data["Diff"] = diff
	ctx.Data["DiffNotAvailable"] = diff.NumFiles == 0

	baseCommit, err := ctx.Repo.GitRepo.GetCommit(startCommitID)
	if err != nil {
		ctx.ServerError("GetCommit", err)
		return
	}
	commit, err := gitRepo.GetCommit(endCommitID)
	if err != nil {
		ctx.ServerError("GetCommit", err)
		return
	}

	if ctx.IsSigned && ctx.Doer != nil {
		if ctx.Data["CanMarkConversation"], err = issues_model.CanMarkConversation(issue, ctx.Doer); err != nil {
			ctx.ServerError("CanMarkConversation", err)
			return
		}
	}

	repo.SetCompareContext(ctx, baseCommit, commit, ctx.Repo.Owner.Name, ctx.Repo.Repository.Name)

	assigneeUsers, err := repo_model.GetRepoAssignees(ctx, ctx.Repo.Repository)
	if err != nil {
		ctx.ServerError("GetRepoAssignees", err)
		return
	}
	ctx.Data["Assignees"] = repo.MakeSelfOnTop(ctx, assigneeUsers)

	repo.HandleTeamMentions(ctx)
	if ctx.Written() {
		return
	}

	currentReview, err := issues_model.GetCurrentReview(ctx, ctx.Doer, issue)
	if err != nil && !issues_model.IsErrReviewNotExist(err) {
		ctx.ServerError("GetCurrentReview", err)
		return
	}
	numPendingCodeComments := int64(0)
	if currentReview != nil {
		numPendingCodeComments, err = issues_model.CountComments(&issues_model.FindCommentsOptions{
			Type:     issues_model.CommentTypeCode,
			ReviewID: currentReview.ID,
			IssueID:  issue.ID,
		})
		if err != nil {
			ctx.ServerError("CountComments", err)
			return
		}
	}
	ctx.Data["CurrentReview"] = currentReview
	ctx.Data["PendingCodeCommentNumber"] = numPendingCodeComments

	s.getBranchData(ctx, issue)
	ctx.Data["IsIssuePoster"] = ctx.IsSigned && issue.IsPoster(ctx.Doer.ID)
	ctx.Data["HasIssuesOrPullsWritePermission"] = ctx.Repo.CanWriteIssuesOrPulls(issue.IsPull)

	ctx.Data["IsAttachmentEnabled"] = setting.Attachment.Enabled
	upload.AddUploadContext(ctx, "comment")

	ctx.HTML(http.StatusOK, tplPullFiles)
}
