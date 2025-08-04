package repo

import (
	issuesModel "code.gitea.io/gitea/models/issues"
	accessModel "code.gitea.io/gitea/models/perm/access"
	projectModel "code.gitea.io/gitea/models/project"
	repoModel "code.gitea.io/gitea/models/repo"
	userModel "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/git"
	module "code.gitea.io/gitea/modules/repository"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/util"
	"code.gitea.io/gitea/modules/web"
	apiError "code.gitea.io/gitea/routers/sbt/apierror"
	sbtConvert "code.gitea.io/gitea/routers/sbt/convert"
	"code.gitea.io/gitea/routers/sbt/logger"
	"code.gitea.io/gitea/routers/sbt/request"
	"code.gitea.io/gitea/routers/sbt/response"
	issueService "code.gitea.io/gitea/services/issue"
	pullService "code.gitea.io/gitea/services/pull"
	stdCtx "context"
	"fmt"
	"net/http"
)

// CreateComment добавить обычный комментарий к пулл реквесту
func CreateComment(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	req := web.GetForm(ctx).(*request.CreateComment)

	index := ctx.ParamsInt64(":index")
	issue := getIssueByIndex(ctx, index)
	if ctx.Written() {
		return
	}

	if issue.IsLocked && !ctx.Repo.CanWriteIssuesOrPulls(issue.IsPull) && !ctx.Doer.IsAdmin {
		log.Debug("Pull request with id: %d is locked in repository: /%s", issue.ID, ctx.Repo.Repository.FullName())
		ctx.JSON(http.StatusUnauthorized, apiError.PullRequestIsLocked())
		return
	}
	if ctx.Written() {
		return
	}

	var attachments []string
	if setting.Attachment.Enabled {
		attachments = req.Files
	}

	if ctx.HasError() {
		log.Error("Not able to create comment for pull request id: %d in repository: /%s, error: %v", issue.ID, ctx.Repo.Repository.FullName(), ctx.Err())
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		return
	}

	defer changePullRequestStatus(ctx, issue, req.Status)

	if len(req.Content) == 0 && len(attachments) == 0 {
		return
	}

	comment, err := issueService.CreateIssueComment(ctx, ctx.Doer, ctx.Repo.Repository, issue, req.Content, attachments)
	if err != nil {
		log.Error("Not able to create comment for pull request id: %d in repository: /%s, error: %v", index, ctx.Repo.Repository.FullName(), err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		return
	}

	ctx.JSON(http.StatusCreated, sbtConvert.ToComment(ctx, comment))
}

// changePullRequestStatus меняет статус пулл реквеста (закрыт, переоткрыт)
func changePullRequestStatus(ctx *context.Context, issue *issuesModel.Issue, status string) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	if (status == "reopen" || status == "close") && !(issue.IsPull && issue.PullRequest.HasMerged) {

		// Если происходит переоткрытие пулл реквеста по перепроверяем на конфликты и дубликаты
		var pr *issuesModel.PullRequest

		if status == "reopen" && issue.IsPull {
			pull := issue.PullRequest
			var err error
			pr, err = issuesModel.GetUnmergedPullRequest(ctx, pull.HeadRepoID, pull.BaseRepoID, pull.HeadBranch, pull.BaseBranch, pull.Flow)
			if err != nil {
				if !issuesModel.IsErrPullRequestNotExist(err) {
					log.Debug("Unmerged pull request id: %d not found for reopen", pull.ID)
					ctx.JSON(http.StatusBadRequest, apiError.PullRequestNotFound(pr.Index))
					return
				}
			}

			// Проверяем на конфликты
			if pr == nil {
				issue.PullRequest.HeadCommitID = ""
				pullService.AddToTaskQueue(issue.PullRequest)
			}

			prHeadRef := pull.GetGitRefName()
			if err := pull.LoadBaseRepo(ctx); err != nil {
				log.Error("Unable to load base repo in pull request id: %d n repository: /%s, error: %v", pull.ID, ctx.Repo.Repository.FullName(), err)
				ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
				return
			}
			prHeadCommitID, err := git.GetFullCommitID(ctx, pull.BaseRepo.RepoPath(), prHeadRef)
			if err != nil {
				log.Error("Get head commit id of pull request id: %d failed in repository: /%s path: %s ref, error: %v", pull.ID, ctx.Repo.Repository.FullName(), pull.BaseRepo.RepoPath(), prHeadRef, err)
				ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
				return
			}

			// get head commit of branch in the head repo
			if err := pull.LoadHeadRepo(ctx); err != nil {
				log.Error("Unable to load head repo in pull request id: %d in repository: /%s, error: %v", pull.ID, ctx.Repo.Repository.FullName(), err)
				ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
				return
			}
			if ok := git.IsBranchExist(ctx, pull.HeadRepo.OwnerName, pull.HeadRepo.Name, pull.HeadRepo.RepoPath(), pull.BaseBranch); !ok {
				log.Debug("Not able to reopen PR, branch name: %s not exists in repository: /%s", pull.BaseBranch, ctx.Repo.Repository.FullName())
				ctx.JSON(http.StatusBadRequest, apiError.BranchNotExist(pull.BaseBranch))
				return
			}
			headBranchRef := pull.GetGitHeadBranchRefName()
			headBranchCommitID, err := git.GetFullCommitID(ctx, pull.HeadRepo.RepoPath(), headBranchRef)
			if err != nil {
				log.Error("Get head commit id of pull request id: %d failed in repository: /%s path: %s ref: %s, error: %v", pull.ID, ctx.Repo.Repository.FullName(), pull.HeadRepo.RepoPath(), headBranchRef, err)
				ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
				return
			}

			err = pull.LoadIssue(ctx)
			if err != nil {
				log.Error("Not able to load pull request id: %d information from DB, error: %v", pull.ID, err)
				ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
				return
			}

			if prHeadCommitID != headBranchCommitID {
				// force push to base repo
				pushOptions := git.PushOptions{
					Remote: pull.BaseRepo.RepoPath(),
					Branch: pull.HeadBranch + ":" + prHeadRef,
					Force:  true,
					Env:    module.InternalPushingEnvironment(pull.Issue.Poster, pull.BaseRepo),
				}

				err := git.Push(ctx, pull.HeadRepo.RepoPath(), pushOptions)
				if err != nil {
					log.Error("Not able to push with options: %v, error: %v", pushOptions, err)
					ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
					return
				}
			}
		}

		if pr != nil {
			log.Debug("Pull request for branches %s and %s repository: /%s already exists - existing pr index %d", pr.BaseBranch, pr.HeadBranch, ctx.Repo.Repository.FullName(), pr.Index)
			ctx.JSON(http.StatusBadRequest, apiError.PullRequestAlreadyExist(pr.Index))
			return
		} else {
			isClosed := status == "close"
			if err := issueService.ChangeStatus(issue, ctx.Doer, "", isClosed); err != nil {
				log.Debug("Not able to close pull request, error: %v", err)
				ctx.JSON(http.StatusBadRequest, apiError.PullRequestAlreadyClosed(issue.ID))
				return
			} else {
				if err := stopTimerIfAvailable(ctx.Doer, issue); err != nil {
					log.Error("Not able to change pull request status id: %d , error: %v", pr.ID, err)
					ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
					return
				}
			}
		}
	}
}

// stopTimerIfAvailable отключает отслеживание по таймеру (применимо с milestone)
func stopTimerIfAvailable(user *userModel.User, issue *issuesModel.Issue) error {
	if issuesModel.StopwatchExists(user.ID, issue.ID) {
		if err := issuesModel.CreateOrStopIssueStopwatch(user, issue); err != nil {
			return err
		}
	}
	return nil
}

// DeleteComment удалить комментарий
func DeleteComment(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	id := ctx.ParamsInt64(":id")

	comment, err := issuesModel.GetCommentByID(ctx, id)
	if err != nil {
		if issuesModel.IsErrCommentNotExist(err) {
			log.Debug("Comment: %d not found", id)
			ctx.JSON(http.StatusBadRequest, apiError.CommentNotFound())
			return
		}

		log.Error("Not able to find issue comment id: %d, error: %v", id, err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		return
	}

	if err := comment.LoadIssue(ctx); err != nil {
		if issuesModel.IsErrPullRequestNotExist(err) || issuesModel.IsErrIssueNotExist(err) {
			log.Debug("Pull request for comment id: %d not found", id)
			ctx.JSON(http.StatusBadRequest, apiError.PullRequestForCommentNotFound(id))
		} else {
			log.Error("Not able to load issue|pull request for comment id: %d, error: %v", id, err)
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		}
		return
	}

	checkIssueWriteRights(ctx, comment.Issue)
	if ctx.Written() {
		return
	}

	if !comment.Type.HasContentSupport() {
		log.Debug("Comment: %d has not content", id)
		ctx.JSON(http.StatusBadRequest, apiError.CommentHasNotContent())
		return
	}

	if err = issueService.DeleteComment(ctx, ctx.Doer, comment); err != nil {
		log.Error("Not able to delete comment id: %d, error: %v", id, err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
	}

	ctx.Status(http.StatusOK)
}

// UpdateComment изменить комментарий
func UpdateComment(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	req := web.GetForm(ctx).(*request.UpdateComment)
	commentId := ctx.ParamsInt64(":id")

	comment, err := issuesModel.GetCommentByID(ctx, commentId)
	if err != nil {
		if issuesModel.IsErrCommentNotExist(err) {
			log.Debug("Comment id: %d not found", commentId)
			ctx.JSON(http.StatusBadRequest, apiError.CommentNotFound())
			return
		}

		log.Error("Not able to find issue comment id: %d, error: %v", commentId, err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		return
	}

	if err := comment.LoadIssue(ctx); err != nil {
		if issuesModel.IsErrPullRequestNotExist(err) || issuesModel.IsErrIssueNotExist(err) {
			log.Debug("Pull request for comment id: %d not found", commentId)
			ctx.JSON(http.StatusBadRequest, apiError.PullRequestForCommentNotFound(commentId))
		} else {
			log.Error("Not able to load issue|pull request for comment id: %d, error: %v", commentId, err)
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		}
		return
	}

	checkIssueWriteRights(ctx, comment.Issue)
	if ctx.Written() {
		return
	}

	if !comment.Type.HasContentSupport() {
		log.Debug("Comment: %d has not content", commentId)
		ctx.JSON(http.StatusBadRequest, apiError.CommentHasNotContent())
		return
	}

	oldContent := comment.Content
	comment.Content = req.Content

	if err = issueService.UpdateComment(ctx, comment, ctx.Doer, oldContent); err != nil {
		log.Error("Not able to update comment id: %d content, error: %v", commentId, err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		return
	}

	if err := comment.LoadAttachments(ctx); err != nil {
		log.Error("Not able to load comment id: %d attachments, error: %v", commentId, err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		return
	}

	if err := updateAttachments(ctx, comment, req.Files); err != nil {
		log.Error("Not able to update comment id: %d attachments, error: %v", commentId, err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		return
	}

	ctx.Status(http.StatusOK)
}

// updateAttachments обновляет (после редактирования) список вложенных в коммент файлов
func updateAttachments(ctx *context.Context, item interface{}, files []string) error {
	var attachments []*repoModel.Attachment
	switch content := item.(type) {
	case *issuesModel.Issue:
		attachments = content.Attachments
	case *issuesModel.Comment:
		attachments = content.Attachments
	default:
		return fmt.Errorf("unknown content type: %T ", content)
	}
	for i := 0; i < len(attachments); i++ {
		if util.SliceContainsString(files, attachments[i].UUID) {
			continue
		}
		if err := repoModel.DeleteAttachment(attachments[i], true); err != nil {
			return err
		}
	}
	var err error
	if len(files) > 0 {
		switch content := item.(type) {
		case *issuesModel.Issue:
			err = issuesModel.UpdateIssueAttachments(content.ID, files)
		case *issuesModel.Comment:
			err = content.UpdateAttachments(files)
		default:
			return fmt.Errorf("unknown content type: %T", content)
		}
		if err != nil {
			return err
		}
	}
	switch content := item.(type) {
	case *issuesModel.Issue:
		content.Attachments, err = repoModel.GetAttachmentsByIssueID(ctx, content.ID)
	case *issuesModel.Comment:
		content.Attachments, err = repoModel.GetAttachmentsByCommentID(ctx, content.ID)
	default:
		return fmt.Errorf("uunknown content type: %T", content)
	}
	return err
}

// ChangeCommentReaction добавить реакцию к комментарию
func ChangeCommentReaction(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	req := web.GetForm(ctx).(*request.Reaction)

	commentId := ctx.ParamsInt64(":id")

	comment, err := issuesModel.GetCommentByID(ctx, commentId)
	if err != nil {
		if !issuesModel.IsErrCommentNotExist(err) {
			log.Debug("Comment id: %d not found", commentId)
			ctx.JSON(http.StatusBadRequest, apiError.CommentNotFound())
			return
		}

		log.Error("Not able to find issue comment id: %d, error: %v", commentId, err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		return
	}

	if err := comment.LoadIssue(ctx); err != nil {
		if issuesModel.IsErrPullRequestNotExist(err) || issuesModel.IsErrIssueNotExist(err) {
			log.Debug("Pull request for comment id: %d not found", commentId)
			ctx.JSON(http.StatusBadRequest, apiError.PullRequestForCommentNotFound(commentId))
		} else {
			log.Error("Not able to load issue|pull request for comment id: %d, error: %v", commentId, err)
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		}
		return
	}

	checkIssueReadRights(ctx, comment.Issue)
	if ctx.Written() {
		return
	}

	if !comment.Type.HasContentSupport() {
		log.Debug("Comment: %d has not content", commentId)
		ctx.JSON(http.StatusBadRequest, apiError.CommentHasNotContent())
		return
	}

	switch ctx.Params(":action") {
	case "react":
		_, err := issuesModel.CreateCommentReaction(ctx.Doer.ID, comment.Issue.ID, comment.ID, req.Content)
		if err != nil {
			if issuesModel.IsErrForbiddenIssueReaction(err) {
				log.Debug("Reaction: %s is forbidden to use, error: %v", req.Content, err)
				ctx.JSON(http.StatusBadRequest, apiError.ReactionNotFound())
				return
			}
			log.Error("Not able to add reaction: %s for comment id: , error: %v", req.Content, commentId, err)
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
			break
		}
	case "unreact":
		if err := issuesModel.DeleteCommentReaction(ctx.Doer.ID, comment.Issue.ID, comment.ID, req.Content); err != nil {
			log.Error("Not able to deete reaction: %s for comment id: , error: %v", req.Content, commentId, err)
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
			return
		}
	default:
		log.Debug("Reaction action type: %s not found", ctx.Params(":action"))
		ctx.JSON(http.StatusBadRequest, apiError.ReactionActionUnknown())
		return
	}

	reactionList, _, _ := issuesModel.FindCommentReactions(0, commentId)

	responseReactions := make([]*response.Reaction, len(reactionList))

	for i, reaction := range reactionList {
		reaction.LoadUser()
		responseReactions[i] = sbtConvert.ToReaction(reaction)
	}

	ctx.JSON(http.StatusOK, responseReactions)
}

// GetComments возвращает список комментариев
func GetComments(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	index := ctx.ParamsInt64(":index")

	issue := getIssueByIndex(ctx, index)
	if ctx.Written() {
		return
	}

	MustAllowPulls(ctx)
	if ctx.Written() {
		return
	}

	loadCommentDetails(ctx, issue)
	if ctx.Written() {
		return
	}

	filterXRefComments(ctx, issue)
	if ctx.Written() {
		return
	}

	combineLabelComments(issue)
	if ctx.Written() {
		return
	}

	responseComments := sbtConvert.IssueToComments(ctx, issue)

	ctx.JSON(http.StatusOK, responseComments)
}

// filterXRefComments Удаляем комментарии к которым не должно быть доступа
func filterXRefComments(ctx *context.Context, issue *issuesModel.Issue) error {
	for i := 0; i < len(issue.Comments); {
		c := issue.Comments[i]
		if issuesModel.CommentTypeIsRef(c.Type) && c.RefRepoID != issue.RepoID && c.RefRepoID != 0 {
			var err error
			// Set RefRepo for description in template
			c.RefRepo, err = repoModel.GetRepositoryByID(ctx, c.RefRepoID)
			if err != nil {
				return err
			}
			perm, err := accessModel.GetUserRepoPermission(ctx, c.RefRepo, ctx.Doer)
			if err != nil {
				return err
			}
			if !perm.CanReadIssuesOrPulls(c.RefIsPull) {
				issue.Comments = append(issue.Comments[:i], issue.Comments[i+1:]...)
				continue
			}
		}
		i++
	}
	return nil
}

// loadCommentDetails добавляем к комментариям детали в зависимости от их типа
func loadCommentDetails(ctx *context.Context, issue *issuesModel.Issue) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	var (
		role         issuesModel.RoleDescriptor
		ok           bool
		marked       = make(map[int64]issuesModel.RoleDescriptor)
		comment      *issuesModel.Comment
		participants = make([]*userModel.User, 1, 10)
		err          error
	)

	repo := ctx.Repo.Repository
	participants[0] = issue.Poster

	for _, comment = range issue.Comments {
		comment.Issue = issue

		if err := comment.LoadPoster(ctx); err != nil {
			log.Error("Not able to load poster for comment: %d, error: %v", comment.ID, err)
			ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
			return
		}

		if comment.Type == issuesModel.CommentTypeComment || comment.Type == issuesModel.CommentTypeReview {
			if err := comment.LoadAttachments(ctx); err != nil {
				log.Error("Not able to load attachments for comment: %d, error: %v", comment.ID, err)
				ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
				return
			}

			role, ok = marked[comment.PosterID]
			if ok {
				comment.ShowRole = role
				continue
			}

			comment.ShowRole, err = roleDescriptor(ctx, repo, comment.Poster, issue, comment.HasOriginalAuthor())
			if err != nil {
				log.Error("Not able to load poster role for comment: %d, error: %v", comment.ID, err)
				ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
				return
			}
			marked[comment.PosterID] = comment.ShowRole
			participants = addParticipant(comment.Poster, participants)
		} else if comment.Type == issuesModel.CommentTypeLabel {
			if err = comment.LoadLabel(); err != nil {
				log.Error("Not able to load labels for comment: %d, error: %v", comment.ID, err)
				ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
				return
			}
		} else if comment.Type == issuesModel.CommentTypeMilestone {
			if err = comment.LoadMilestone(ctx); err != nil {
				log.Error("Not able to load milestones for comment: %d, error: %v", comment.ID, err)
				ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
				return
			}
			ghostMilestone := &issuesModel.Milestone{
				ID:   -1,
				Name: "deleted",
			}
			if comment.OldMilestoneID > 0 && comment.OldMilestone == nil {
				comment.OldMilestone = ghostMilestone
			}
			if comment.MilestoneID > 0 && comment.Milestone == nil {
				comment.Milestone = ghostMilestone
			}
		} else if comment.Type == issuesModel.CommentTypeProject {

			if err = comment.LoadProject(); err != nil {
				log.Error("Not able to load projects for comment: %d, error: %v", comment.ID, err)
				ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
				return
			}

			ghostProject := &projectModel.Project{
				ID:    -1,
				Title: "deleted",
			}

			if comment.OldProjectID > 0 && comment.OldProject == nil {
				comment.OldProject = ghostProject
			}

			if comment.ProjectID > 0 && comment.Project == nil {
				comment.Project = ghostProject
			}

		} else if comment.Type == issuesModel.CommentTypeAssignees || comment.Type == issuesModel.CommentTypeReviewRequest {
			if err = comment.LoadAssigneeUserAndTeam(); err != nil {
				log.Error("Not able to load assignees for comment: %d, error: %v", comment.ID, err)
				ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
				return
			}
		} else if comment.Type == issuesModel.CommentTypeRemoveDependency || comment.Type == issuesModel.CommentTypeAddDependency {
			if err = comment.LoadDepIssueDetails(); err != nil {
				if !issuesModel.IsErrIssueNotExist(err) {
					log.Error("Not able to load issue dependencies for comment: %d, error: %v", comment.ID, err)
					ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
					return
				}
			}
		} else if comment.Type.HasContentSupport() {
			if err = comment.LoadReview(); err != nil && !issuesModel.IsErrReviewNotExist(err) {
				log.Error("Not able to load review for comment: %d, error: %v", comment.ID, err)
				ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
				return
			}
			participants = addParticipant(comment.Poster, participants)
			if comment.Review == nil {
				continue
			}
			if err = comment.Review.LoadAttributes(ctx); err != nil {
				if !userModel.IsErrUserNotExist(err) {
					log.Error("Not able to load attributes for comment: %d, error: %v", comment.ID, err)
					ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
					return
				}
				comment.Review.Reviewer = userModel.NewGhostUser()
			}
			if err = comment.Review.LoadCodeComments(ctx); err != nil {
				log.Error("Not able to load code details for comment: %d, error: %v", comment.ID, err)
				ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
				return
			}
			for _, codeComments := range comment.Review.CodeComments {
				for _, lineComments := range codeComments {
					for _, c := range lineComments {
						// Check tag.
						role, ok = marked[c.PosterID]
						if ok {
							c.ShowRole = role
							continue
						}

						c.ShowRole, err = roleDescriptor(ctx, repo, c.Poster, issue, c.HasOriginalAuthor())
						if err != nil {
							ctx.ServerError("roleDescriptor", err)
							return
						}
						marked[c.PosterID] = c.ShowRole
						participants = addParticipant(c.Poster, participants)
					}
				}
			}
			if err = comment.LoadResolveDoer(); err != nil {
				log.Error("Not able to load resolve doer for comment: %d, error: %v", comment.ID, err)
				ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
				return
			}
		} else if comment.Type == issuesModel.CommentTypePullRequestPush {
			participants = addParticipant(comment.Poster, participants)
			if err = comment.LoadPushCommits(ctx); err != nil {
				log.Error("Not able to load push commits for comment: %d, error: %v", comment.ID, err)
				ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
				return
			}
		} else if comment.Type == issuesModel.CommentTypeAddTimeManual ||
			comment.Type == issuesModel.CommentTypeStopTracking {
			_ = comment.LoadTime()
		}
	}
}

// roleDescriptor возвращает описание роли комментатора
func roleDescriptor(ctx stdCtx.Context, repo *repoModel.Repository, poster *userModel.User, issue *issuesModel.Issue, hasOriginalAuthor bool) (issuesModel.RoleDescriptor, error) {
	if hasOriginalAuthor {
		return issuesModel.RoleDescriptorNone, nil
	}

	perm, err := accessModel.GetUserRepoPermission(ctx, repo, poster)
	if err != nil {
		return issuesModel.RoleDescriptorNone, err
	}

	// По умолчанию нет никакой роли у автора комментария
	roleDescriptor := issuesModel.RoleDescriptorNone

	// Проверка, что комментатор владелец репы
	if perm.IsOwner() {
		if !poster.IsAdmin {
			roleDescriptor = roleDescriptor.WithRole(issuesModel.RoleDescriptorOwner)
		} else {

			ok, err := accessModel.IsUserRealRepoAdmin(repo, poster)
			if err != nil {
				return issuesModel.RoleDescriptorNone, err
			}
			if ok {
				roleDescriptor = roleDescriptor.WithRole(issuesModel.RoleDescriptorOwner)
			}
		}
	}

	// Проверяем на соавтора
	if !roleDescriptor.HasRole("Owner") && perm.CanWriteIssuesOrPulls(issue.IsPull) {
		roleDescriptor = roleDescriptor.WithRole(issuesModel.RoleDescriptorWriter)
	}

	// Автор пулл реквеста
	if issue.IsPoster(poster.ID) {
		roleDescriptor = roleDescriptor.WithRole(issuesModel.RoleDescriptorPoster)
	}

	return roleDescriptor, nil
}

// addParticipant добавляем участника в список
func addParticipant(poster *userModel.User, participants []*userModel.User) []*userModel.User {
	for _, part := range participants {
		if poster.ID == part.ID {
			return participants
		}
	}
	return append(participants, poster)
}

// combineLabelComments объединяем одинаковые комментарии-метки в один.
func combineLabelComments(issue *issuesModel.Issue) {
	var prev, cur *issuesModel.Comment
	for i := 0; i < len(issue.Comments); i++ {
		cur = issue.Comments[i]
		if i > 0 {
			prev = issue.Comments[i-1]
		}
		if i == 0 || cur.Type != issuesModel.CommentTypeLabel ||
			(prev != nil && prev.PosterID != cur.PosterID) ||
			(prev != nil && cur.CreatedUnix-prev.CreatedUnix >= 60) {
			if cur.Type == issuesModel.CommentTypeLabel && cur.Label != nil {
				if cur.Content != "1" {
					cur.RemovedLabels = append(cur.RemovedLabels, cur.Label)
				} else {
					cur.AddedLabels = append(cur.AddedLabels, cur.Label)
				}
			}
			continue
		}

		if cur.Label != nil { // now cur MUST be label comment
			if prev.Type == issuesModel.CommentTypeLabel { // we can combine them only prev is a label comment
				if cur.Content != "1" {
					// remove labels from the AddedLabels list if the label that was removed is already
					// in this list, and if it's not in this list, add the label to RemovedLabels
					addedAndRemoved := false
					for i, label := range prev.AddedLabels {
						if cur.Label.ID == label.ID {
							prev.AddedLabels = append(prev.AddedLabels[:i], prev.AddedLabels[i+1:]...)
							addedAndRemoved = true
							break
						}
					}
					if !addedAndRemoved {
						prev.RemovedLabels = append(prev.RemovedLabels, cur.Label)
					}
				} else {
					// remove labels from the RemovedLabels list if the label that was added is already
					// in this list, and if it's not in this list, add the label to AddedLabels
					removedAndAdded := false
					for i, label := range prev.RemovedLabels {
						if cur.Label.ID == label.ID {
							prev.RemovedLabels = append(prev.RemovedLabels[:i], prev.RemovedLabels[i+1:]...)
							removedAndAdded = true
							break
						}
					}
					if !removedAndAdded {
						prev.AddedLabels = append(prev.AddedLabels, cur.Label)
					}
				}
				prev.CreatedUnix = cur.CreatedUnix
				// remove the current comment since it has been combined to prev comment
				issue.Comments = append(issue.Comments[:i], issue.Comments[i+1:]...)
				i--
			} else { // if prev is not a label comment, start a new group
				if cur.Content != "1" {
					cur.RemovedLabels = append(cur.RemovedLabels, cur.Label)
				} else {
					cur.AddedLabels = append(cur.AddedLabels, cur.Label)
				}
			}
		}
	}
}

// LockIssue ограничиваем комментирование только для соавторов, иные юзеры не могут комментировать
func LockIssue(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	req := web.GetForm(ctx).(*request.IssueLock)
	index := ctx.ParamsInt64(":index")

	issue := getIssueByIndex(ctx, index)
	if ctx.Written() {
		return
	}

	if issue.IsLocked {
		log.Debug("Comments for issue id: %s are already locked, error: %v", issue.ID)
		ctx.JSON(http.StatusBadRequest, apiError.CommentsAlreadyLocked())
		return
	}

	if !req.HasValidReason() {
		log.Debug("Reason to lock issue is invalid, error: %v", issue.ID)
		ctx.JSON(http.StatusBadRequest, apiError.InvalidCommentLockReason())
		return
	}

	if err := issuesModel.LockIssue(&issuesModel.IssueLockOptions{
		Doer:   ctx.Doer,
		Issue:  issue,
		Reason: req.Reason,
	}); err != nil {
		log.Error("Not able lock issue|pull request id: %d, error: %v", issue.ID, err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		return
	}

	ctx.Status(http.StatusOK)
}

// UnlockIssue разблокирует предыдущую блокировку комментариев
func UnlockIssue(ctx *context.Context) {
	log := logger.Logger{}
	log.SetTraceId(ctx)

	index := ctx.ParamsInt64(":index")

	issue := getIssueByIndex(ctx, index)
	if ctx.Written() {
		return
	}

	if !issue.IsLocked {
		log.Debug("Comments for issue id: %s are not locked", issue.ID)
		ctx.JSON(http.StatusBadRequest, apiError.CommentsNotLocked())
		return
	}

	if err := issuesModel.UnlockIssue(&issuesModel.IssueLockOptions{
		Doer:  ctx.Doer,
		Issue: issue,
	}); err != nil {
		log.Error("Not able unlock issue|pull request id: %d, error: %v", issue.ID, err)
		ctx.JSON(http.StatusInternalServerError, apiError.InternalServerError())
		return
	}

	ctx.Status(http.StatusOK)
}
