package repo_server

import (
	stdCtx "context"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"sort"

	activities_model "code.gitea.io/gitea/models/activities"
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/default_reviewers/default_reviewers_db"
	git_model "code.gitea.io/gitea/models/git"
	issues_model "code.gitea.io/gitea/models/issues"
	access_model "code.gitea.io/gitea/models/perm/access"
	project_model "code.gitea.io/gitea/models/project"
	pull_model "code.gitea.io/gitea/models/pull"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/review_settings/review_settings_db"
	"code.gitea.io/gitea/models/role_model"
	"code.gitea.io/gitea/models/unit"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/base"
	"code.gitea.io/gitea/modules/container"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/markup"
	"code.gitea.io/gitea/modules/markup/markdown"
	"code.gitea.io/gitea/modules/setting"
	api "code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/modules/templates/vars"
	"code.gitea.io/gitea/modules/trace"
	"code.gitea.io/gitea/modules/upload"
	"code.gitea.io/gitea/modules/util"
	"code.gitea.io/gitea/routers/sbt/repo"
	repo_web "code.gitea.io/gitea/routers/web/repo"
	"code.gitea.io/gitea/routers/web/user/accesser"
	asymkey_service "code.gitea.io/gitea/services/asymkey"
	issue_service "code.gitea.io/gitea/services/issue"
	pull_service "code.gitea.io/gitea/services/pull"
)

const (
	tplIssueView base.TplName = "repo/issue/view"
)

// ViewIssue render issue view page
func (s *Server) ViewIssue(ctx *context.Context) {
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

	if ctx.Params(":type") == "issues" {
		// If issue was requested we check if repo has external tracker and redirect
		extIssueUnit, err := ctx.Repo.Repository.GetUnit(ctx, unit.TypeExternalTracker)
		if err == nil && extIssueUnit != nil {
			if extIssueUnit.ExternalTrackerConfig().ExternalTrackerStyle == markup.IssueNameStyleNumeric || extIssueUnit.ExternalTrackerConfig().ExternalTrackerStyle == "" {
				metas := ctx.Repo.Repository.ComposeMetas()
				metas["index"] = ctx.Params(":index")
				res, err := vars.Expand(extIssueUnit.ExternalTrackerConfig().ExternalTrackerFormat, metas)
				if err != nil {
					log.Error("unable to expand template vars for issue url. issue: %s, err: %v", metas["index"], err)
					ctx.ServerError("Expand", err)
					return
				}
				ctx.Redirect(res)
				return
			}
		} else if err != nil && !repo_model.IsErrUnitTypeNotExist(err) {
			ctx.ServerError("GetUnit", err)
			return
		}
	}

	issue, err := issues_model.GetIssueByIndex(ctx.Repo.Repository.ID, ctx.ParamsInt64(":index"))
	if err != nil {
		if issues_model.IsErrIssueNotExist(err) {
			ctx.NotFound("GetIssueByIndex", err)
		} else {
			ctx.ServerError("GetIssueByIndex", err)
		}
		return
	}
	if issue.Repo == nil {
		issue.Repo = ctx.Repo.Repository
	}

	// Make sure type and URL matches.
	if ctx.Params(":type") == "issues" && issue.IsPull {
		ctx.Redirect(issue.Link())
		return
	} else if ctx.Params(":type") == "pulls" && !issue.IsPull {
		ctx.Redirect(issue.Link())
		return
	}

	if issue.IsPull {
		repo.MustAllowPulls(ctx)

		allowed, err := s.orgRequestAccessor.IsAccessGranted(*ctx, accesser.OrgAccessRequest{
			DoerID:         ctx.Doer.ID,
			TargetOrgID:    ctx.Repo.Repository.OwnerID,
			TargetTenantID: ctx.Data["TenantID"].(string),
			Action:         role_model.READ,
		})
		if err != nil {
			log.Error("Error has occurred while getting user's permissions: %v", err)
			ctx.Error(http.StatusForbidden, "Error has occurred while getting user's permissions")
			return
		}
		if !allowed {
			allow, err := s.repoRequestAccessor.AccessesByCustomPrivileges(*ctx, accesser.RepoAccessRequest{
				DoerID:          ctx.Doer.ID,
				OrgID:           ctx.Repo.Repository.OwnerID,
				TargetTenantID:  ctx.Data["TenantID"].(string),
				RepoID:          ctx.Repo.Repository.ID,
				CustomPrivilege: role_model.ViewBranch.String(),
			})
			if err != nil || !allow {
				log.Error("Error has occurred while getting user's permissions: %v", err)
				ctx.Error(http.StatusForbidden, "Error has occurred while getting user's permissions")
				return
			}
		}

		if ctx.Written() {
			return
		}
		ctx.Data["PageIsPullList"] = true
		ctx.Data["PageIsPullConversation"] = true
	} else {
		repo_web.MustEnableIssues(ctx)
		if ctx.Written() {
			return
		}
		ctx.Data["PageIsIssueList"] = true
		ctx.Data["NewIssueChooseTemplate"] = issue_service.HasTemplatesOrContactLinks(ctx.Repo.Repository, ctx.Repo.GitRepo)
	}

	if issue.IsPull && !ctx.Repo.CanRead(unit.TypeIssues) {
		ctx.Data["IssueType"] = "pulls"
	} else if !issue.IsPull && !ctx.Repo.CanRead(unit.TypePullRequests) {
		ctx.Data["IssueType"] = "issues"
	} else {
		ctx.Data["IssueType"] = "all"
	}

	ctx.Data["IsProjectsEnabled"] = ctx.Repo.CanRead(unit.TypeProjects)
	ctx.Data["IsAttachmentEnabled"] = setting.Attachment.Enabled
	upload.AddUploadContext(ctx, "comment")

	if err = issue.LoadAttributes(ctx); err != nil {
		ctx.ServerError("LoadAttributes", err)
		return
	}

	if err = filterXRefComments(ctx, issue); err != nil {
		ctx.ServerError("filterXRefComments", err)
		return
	}

	ctx.Data["Title"] = fmt.Sprintf("#%d - %s", issue.Index, issue.Title)

	iw := new(issues_model.IssueWatch)
	if ctx.Doer != nil {
		iw.UserID = ctx.Doer.ID
		iw.IssueID = issue.ID
		iw.IsWatching, err = issues_model.CheckIssueWatch(ctx.Doer, issue)
		if err != nil {
			ctx.ServerError("CheckIssueWatch", err)
			return
		}
	}
	ctx.Data["IssueWatch"] = iw

	issue.RenderedContent, err = markdown.RenderString(&markup.RenderContext{
		URLPrefix: ctx.Repo.RepoLink,
		Metas:     ctx.Repo.Repository.ComposeMetas(),
		GitRepo:   ctx.Repo.GitRepo,
		Ctx:       ctx,
	}, issue.Content)
	if err != nil {
		ctx.ServerError("RenderString", err)
		return
	}

	repo := ctx.Repo.Repository

	// Get more information if it's a pull request.
	if issue.IsPull {
		if issue.PullRequest.HasMerged {
			ctx.Data["DisableStatusChange"] = issue.PullRequest.HasMerged
			repo_web.PrepareMergedViewPullInfo(ctx, issue)
		} else {
			repo_web.PrepareViewPullInfo(ctx, issue)
			ctx.Data["DisableStatusChange"] = ctx.Data["IsPullRequestBroken"] == true && issue.IsClosed
		}
		if ctx.Written() {
			return
		}
	}

	// Metas.
	// Check labels.
	labelIDMark := make(container.Set[int64])
	for _, label := range issue.Labels {
		labelIDMark.Add(label.ID)
	}
	labels, err := issues_model.GetLabelsByRepoID(ctx, repo.ID, "", db.ListOptions{})
	if err != nil {
		ctx.ServerError("GetLabelsByRepoID", err)
		return
	}
	ctx.Data["Labels"] = labels

	if repo.Owner.IsOrganization() {
		orgLabels, err := issues_model.GetLabelsByOrgID(ctx, repo.Owner.ID, ctx.FormString("sort"), db.ListOptions{})
		if err != nil {
			ctx.ServerError("GetLabelsByOrgID", err)
			return
		}
		ctx.Data["OrgLabels"] = orgLabels

		labels = append(labels, orgLabels...)
	}

	hasSelected := false
	for i := range labels {
		if labelIDMark.Contains(labels[i].ID) {
			labels[i].IsChecked = true
			hasSelected = true
		}
	}
	ctx.Data["HasSelectedLabel"] = hasSelected

	// Check milestone and assignee.
	if ctx.Repo.CanWriteIssuesOrPulls(issue.IsPull) {
		repo_web.RetrieveRepoMilestonesAndAssignees(ctx, repo)
		repo_web.RetrieveProjects(ctx, repo)

		if ctx.Written() {
			return
		}
	}

	if issue.IsPull {
		canChooseReviewer := ctx.Repo.CanWrite(unit.TypePullRequests)
		if ctx.Doer != nil && ctx.IsSigned {
			if !canChooseReviewer {
				canChooseReviewer = ctx.Doer.ID == issue.PosterID
			}
			if !canChooseReviewer {
				canChooseReviewer, err = issues_model.IsOfficialReviewer(ctx, issue, ctx.Doer)
				if err != nil {
					ctx.ServerError("IsOfficialReviewer", err)
					return
				}
			}
		}

		repo_web.RetrieveRepoReviewers(ctx, repo, issue, canChooseReviewer)
		if ctx.Written() {
			return
		}
	}

	if ctx.IsSigned {
		// Update issue-user.
		if err = activities_model.SetIssueReadBy(ctx, issue.ID, ctx.Doer.ID); err != nil {
			ctx.ServerError("ReadBy", err)
			return
		}
	}

	var (
		role                 issues_model.RoleDescriptor
		ok                   bool
		marked               = make(map[int64]issues_model.RoleDescriptor)
		comment              *issues_model.Comment
		participants         = make([]*user_model.User, 1, 10)
		latestCloseCommentID int64
	)
	if ctx.Repo.Repository.IsTimetrackerEnabled(ctx) {
		if ctx.IsSigned {
			// Deal with the stopwatch
			ctx.Data["IsStopwatchRunning"] = issues_model.StopwatchExists(ctx.Doer.ID, issue.ID)
			if !ctx.Data["IsStopwatchRunning"].(bool) {
				var exists bool
				var swIssue *issues_model.Issue
				if exists, _, swIssue, err = issues_model.HasUserStopwatch(ctx, ctx.Doer.ID); err != nil {
					ctx.ServerError("HasUserStopwatch", err)
					return
				}
				ctx.Data["HasUserStopwatch"] = exists
				if exists {
					// Add warning if the user has already a stopwatch
					// Add link to the issue of the already running stopwatch
					ctx.Data["OtherStopwatchURL"] = swIssue.Link()
				}
			}
			ctx.Data["CanUseTimetracker"] = ctx.Repo.CanUseTimetracker(issue, ctx.Doer)
		} else {
			ctx.Data["CanUseTimetracker"] = false
		}
		if ctx.Data["WorkingUsers"], err = issues_model.TotalTimes(&issues_model.FindTrackedTimesOptions{IssueID: issue.ID}); err != nil {
			ctx.ServerError("TotalTimes", err)
			return
		}
	}

	// Check if the user can use the dependencies
	ctx.Data["CanCreateIssueDependencies"] = ctx.Repo.CanCreateIssueDependencies(ctx.Doer, issue.IsPull)

	// check if dependencies can be created across repositories
	ctx.Data["AllowCrossRepositoryDependencies"] = setting.Service.AllowCrossRepositoryDependencies

	if issue.ShowRole, err = roleDescriptor(ctx, repo, issue.Poster, issue, issue.HasOriginalAuthor()); err != nil {
		ctx.ServerError("roleDescriptor", err)
		return
	}
	marked[issue.PosterID] = issue.ShowRole

	// Render comments and and fetch participants.
	participants[0] = issue.Poster
	for _, comment = range issue.Comments {
		comment.Issue = issue

		if err := comment.LoadPoster(ctx); err != nil {
			ctx.ServerError("LoadPoster", err)
			return
		}

		if comment.Type == issues_model.CommentTypeComment || comment.Type == issues_model.CommentTypeReview {
			if err := comment.LoadAttachments(ctx); err != nil {
				ctx.ServerError("LoadAttachments", err)
				return
			}

			comment.RenderedContent, err = markdown.RenderString(&markup.RenderContext{
				URLPrefix: ctx.Repo.RepoLink,
				Metas:     ctx.Repo.Repository.ComposeMetas(),
				GitRepo:   ctx.Repo.GitRepo,
				Ctx:       ctx,
			}, comment.Content)
			if err != nil {
				ctx.ServerError("RenderString", err)
				return
			}
			// Check tag.
			role, ok = marked[comment.PosterID]
			if ok {
				comment.ShowRole = role
				continue
			}

			comment.ShowRole, err = roleDescriptor(ctx, repo, comment.Poster, issue, comment.HasOriginalAuthor())
			if err != nil {
				ctx.ServerError("roleDescriptor", err)
				return
			}
			marked[comment.PosterID] = comment.ShowRole
			participants = addParticipant(comment.Poster, participants)
		} else if comment.Type == issues_model.CommentTypeLabel {
			if err = comment.LoadLabel(); err != nil {
				ctx.ServerError("LoadLabel", err)
				return
			}
		} else if comment.Type == issues_model.CommentTypeMilestone {
			if err = comment.LoadMilestone(ctx); err != nil {
				ctx.ServerError("LoadMilestone", err)
				return
			}
			ghostMilestone := &issues_model.Milestone{
				ID:   -1,
				Name: ctx.Tr("repo.issues.deleted_milestone"),
			}
			if comment.OldMilestoneID > 0 && comment.OldMilestone == nil {
				comment.OldMilestone = ghostMilestone
			}
			if comment.MilestoneID > 0 && comment.Milestone == nil {
				comment.Milestone = ghostMilestone
			}
		} else if comment.Type == issues_model.CommentTypeProject {

			if err = comment.LoadProject(); err != nil {
				ctx.ServerError("LoadProject", err)
				return
			}

			ghostProject := &project_model.Project{
				ID:    -1,
				Title: ctx.Tr("repo.issues.deleted_project"),
			}

			if comment.OldProjectID > 0 && comment.OldProject == nil {
				comment.OldProject = ghostProject
			}

			if comment.ProjectID > 0 && comment.Project == nil {
				comment.Project = ghostProject
			}

		} else if comment.Type == issues_model.CommentTypeAssignees || comment.Type == issues_model.CommentTypeReviewRequest {
			if err = comment.LoadAssigneeUserAndTeam(); err != nil {
				ctx.ServerError("LoadAssigneeUserAndTeam", err)
				return
			}
		} else if comment.Type == issues_model.CommentTypeRemoveDependency || comment.Type == issues_model.CommentTypeAddDependency {
			if err = comment.LoadDepIssueDetails(); err != nil {
				if !issues_model.IsErrIssueNotExist(err) {
					ctx.ServerError("LoadDepIssueDetails", err)
					return
				}
			}
		} else if comment.Type.HasContentSupport() {
			comment.RenderedContent, err = markdown.RenderString(&markup.RenderContext{
				URLPrefix: ctx.Repo.RepoLink,
				Metas:     ctx.Repo.Repository.ComposeMetas(),
				GitRepo:   ctx.Repo.GitRepo,
				Ctx:       ctx,
			}, comment.Content)
			if err != nil {
				ctx.ServerError("RenderString", err)
				return
			}
			if err = comment.LoadReview(); err != nil && !issues_model.IsErrReviewNotExist(err) {
				ctx.ServerError("LoadReview", err)
				return
			}
			participants = addParticipant(comment.Poster, participants)
			if comment.Review == nil {
				continue
			}
			if err = comment.Review.LoadAttributes(ctx); err != nil {
				if !user_model.IsErrUserNotExist(err) {
					ctx.ServerError("Review.LoadAttributes", err)
					return
				}
				comment.Review.Reviewer = user_model.NewGhostUser()
			}
			if err = comment.Review.LoadCodeComments(ctx); err != nil {
				ctx.ServerError("Review.LoadCodeComments", err)
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
				ctx.ServerError("LoadResolveDoer", err)
				return
			}
		} else if comment.Type == issues_model.CommentTypePullRequestPush {
			participants = addParticipant(comment.Poster, participants)
			if err = comment.LoadPushCommits(ctx); err != nil {
				ctx.ServerError("LoadPushCommits", err)
				return
			}
		} else if comment.Type == issues_model.CommentTypeAddTimeManual ||
			comment.Type == issues_model.CommentTypeStopTracking {
			// drop error since times could be pruned from DB..
			_ = comment.LoadTime()
		}

		if comment.Type == issues_model.CommentTypeClose || comment.Type == issues_model.CommentTypeMergePull {
			// record ID of the latest closed/merged comment.
			// if PR is closed, the comments whose type is CommentTypePullRequestPush(29) after latestCloseCommentID won't be rendered.
			latestCloseCommentID = comment.ID
		}
	}

	ctx.Data["LatestCloseCommentID"] = latestCloseCommentID

	// Combine multiple label assignments into a single comment
	combineLabelComments(issue)

	s.getBranchData(ctx, issue)
	if issue.IsPull {
		pull := issue.PullRequest
		pull.Issue = issue
		canDelete := false
		ctx.Data["AllowMerge"] = false

		if ctx.IsSigned {
			if err := pull.LoadHeadRepo(ctx); err != nil {
				log.Error("LoadHeadRepo: %v", err)
			} else if pull.HeadRepo != nil {
				perm, err := access_model.GetUserRepoPermission(ctx, pull.HeadRepo, ctx.Doer)
				if err != nil {
					ctx.ServerError("GetUserRepoPermission", err)
					return
				}
				if perm.CanWrite(unit.TypeCode) {
					// Check if branch is not protected
					if pull.HeadBranch != pull.HeadRepo.DefaultBranch {
						if protected, err := git_model.IsBranchProtected(ctx, pull.HeadRepo.ID, pull.HeadBranch); err != nil {
							log.Error("IsProtectedBranch: %v", err)
						} else if !protected {
							canDelete = true
							ctx.Data["DeleteBranchLink"] = issue.Link() + "/cleanup"
						}
					}
					ctx.Data["CanWriteToHeadRepo"] = true
				}
			}

			if err := pull.LoadBaseRepo(ctx); err != nil {
				log.Error("LoadBaseRepo: %v", err)
			}
			perm, err := access_model.GetUserRepoPermission(ctx, pull.BaseRepo, ctx.Doer)
			if err != nil {
				ctx.ServerError("GetUserRepoPermission", err)
				return
			}
			ctx.Data["AllowMerge"], err = pull_service.IsUserAllowedToMerge(ctx, pull, perm, ctx.Doer)
			if err != nil {
				ctx.ServerError("IsUserAllowedToMerge", err)
				return
			}

			if ctx.Data["CanMarkConversation"], err = issues_model.CanMarkConversation(issue, ctx.Doer); err != nil {
				ctx.ServerError("CanMarkConversation", err)
				return
			}
		}

		prUnit, err := repo.GetUnit(ctx, unit.TypePullRequests)
		if err != nil {
			ctx.ServerError("GetUnit", err)
			return
		}
		prConfig := prUnit.PullRequestsConfig()

		var mergeStyle repo_model.MergeStyle
		// Check correct values and select default
		if ms, ok := ctx.Data["MergeStyle"].(repo_model.MergeStyle); !ok ||
			!prConfig.IsMergeStyleAllowed(ms) {
			defaultMergeStyle := prConfig.GetDefaultMergeStyle()
			if prConfig.IsMergeStyleAllowed(defaultMergeStyle) && !ok {
				mergeStyle = defaultMergeStyle
			} else if prConfig.AllowMerge {
				mergeStyle = repo_model.MergeStyleMerge
			} else if prConfig.AllowRebase {
				mergeStyle = repo_model.MergeStyleRebase
			} else if prConfig.AllowRebaseMerge {
				mergeStyle = repo_model.MergeStyleRebaseMerge
			} else if prConfig.AllowSquash {
				mergeStyle = repo_model.MergeStyleSquash
			} else if prConfig.AllowManualMerge {
				mergeStyle = repo_model.MergeStyleManuallyMerged
			}
		}

		ctx.Data["MergeStyle"] = mergeStyle

		defaultMergeMessage, defaultMergeBody, err := pull_service.GetDefaultMergeMessage(ctx, ctx.Repo.GitRepo, pull, mergeStyle)
		if err != nil {
			ctx.ServerError("GetDefaultMergeMessage", err)
			return
		}
		ctx.Data["DefaultMergeMessage"] = defaultMergeMessage
		ctx.Data["DefaultMergeBody"] = defaultMergeBody

		defaultSquashMergeMessage, defaultSquashMergeBody, err := pull_service.GetDefaultMergeMessage(ctx, ctx.Repo.GitRepo, pull, repo_model.MergeStyleSquash)
		if err != nil {
			ctx.ServerError("GetDefaultSquashMergeMessage", err)
			return
		}
		ctx.Data["DefaultSquashMergeMessage"] = defaultSquashMergeMessage
		ctx.Data["DefaultSquashMergeBody"] = defaultSquashMergeBody
		repo_web.PrepareValuesMergeCheck(ctx, ctx.Repo.Repository.ID, pull)

		engine := db.GetEngine(ctx)
		defaultReviewersdb := default_reviewers_db.New(engine)
		reviewSettingsdb := review_settings_db.New(engine)
		reviewSetting := pull_service.NewReviewSettings(defaultReviewersdb, reviewSettingsdb)

		reviewSettings, err := reviewSetting.GetMatchedReviewSetting(ctx, pull.BaseRepoID, pull.BaseBranch)
		if err != nil {
			ctx.ServerError("LoadReviewSetting", err)
			return
		}

		pb, err := s.protectedBranchManager.GetMergeMatchProtectedBranchRule(ctx, pull.BaseRepoID, pull.BaseBranch)
		if err != nil {
			log.Error("Error has occured while get merge match branch with repo id - %d, branch name: %v", pull.BaseRepoID, pull.BaseBranch, err)
			ctx.ServerError("LoadProtectedBranch", err)
			return
		}
		ctx.Data["ShowMergeInstructions"] = true
		if pb != nil {
			pb.Repo = pull.BaseRepo
			var showMergeInstructions bool
			if ctx.Doer != nil {
				showMergeInstructions = s.protectedBranchManager.CheckUserCanPush(ctx, *pb, ctx.Doer)
			}
			var (
				isBlockedByApprovals              bool
				isBlockedByRejection              bool
				isBlockedByOfficialReviewRequests bool
				isBlockedByOutdatedBranch         bool
			)
			for _, rs := range reviewSettings {
				defaultReviewers, err := reviewSetting.GetDefaultReviewers(ctx, rs.ID)
				if err != nil {
					log.Error("Error has occurred while getting default reviewers. Error: %v", err)
					continue
				}
				if !issues_model.HasEnoughApprovals(ctx, rs, defaultReviewers, pull) {
					isBlockedByApprovals = true
				}
				if issues_model.MergeBlockedByRejectedReview(ctx, rs, pull) {
					isBlockedByRejection = true
				}
				if issues_model.MergeBlockedByOfficialReviewRequests(ctx, rs, pull) {
					isBlockedByOfficialReviewRequests = true
				}
				if issues_model.MergeBlockedByOutdatedBranch(rs, pull) {
					isBlockedByOutdatedBranch = true
				}
			}
			ctx.Data["ProtectedBranch"] = pb
			ctx.Data["IsBlockedByApprovals"] = isBlockedByApprovals
			ctx.Data["IsBlockedByRejection"] = isBlockedByRejection
			ctx.Data["IsBlockedByOfficialReviewRequests"] = isBlockedByOfficialReviewRequests
			ctx.Data["IsBlockedByOutdatedBranch"] = isBlockedByOutdatedBranch
			ctx.Data["GrantedApprovals"] = issues_model.GetGrantedApprovalsCount(ctx, pb, pull)
			ctx.Data["RequireSigned"] = pb.RequireSignedCommits
			ctx.Data["ChangedProtectedFiles"] = pull.ChangedProtectedFiles
			ctx.Data["IsBlockedByChangedProtectedFiles"] = len(pull.ChangedProtectedFiles) != 0
			ctx.Data["ChangedProtectedFilesNum"] = len(pull.ChangedProtectedFiles)
			ctx.Data["ShowMergeInstructions"] = showMergeInstructions
		}
		conditions, _ := reviewSetting.GetRequiredReviewConditions(ctx, pull.BaseRepoID, pull)
		ctx.Data["DefaultReviewersRulesCheck"] = conditions
		ctx.Data["WillSign"] = false
		if ctx.Doer != nil {
			sign, key, _, err := asymkey_service.SignMerge(ctx, pull, ctx.Doer, pull.BaseRepo.RepoPath(), pull.BaseBranch, pull.GetGitRefName())
			ctx.Data["WillSign"] = sign
			ctx.Data["SigningKey"] = key
			if err != nil {
				if asymkey_service.IsErrWontSign(err) {
					ctx.Data["WontSignReason"] = err.(*asymkey_service.ErrWontSign).Reason
				} else {
					ctx.Data["WontSignReason"] = "error"
					log.Error("Error whilst checking if could sign pr %d in repo %s. Error: %v", pull.ID, pull.BaseRepo.FullName(), err)
				}
			}
		} else {
			ctx.Data["WontSignReason"] = "not_signed_in"
		}

		isPullBranchDeletable := canDelete &&
			pull.HeadRepo != nil &&
			git.IsBranchExist(ctx, pull.HeadRepo.OwnerName, pull.HeadRepo.Name, pull.HeadRepo.RepoPath(), pull.HeadBranch) &&
			(!pull.HasMerged || ctx.Data["HeadBranchCommitID"] == ctx.Data["PullHeadCommitID"])

		if isPullBranchDeletable && pull.HasMerged {
			exist, err := issues_model.HasUnmergedPullRequestsByHeadInfo(ctx, pull.HeadRepoID, pull.HeadBranch)
			if err != nil {
				ctx.ServerError("HasUnmergedPullRequestsByHeadInfo", err)
				return
			}

			isPullBranchDeletable = !exist
		}
		ctx.Data["IsPullBranchDeletable"] = isPullBranchDeletable

		// Признак того разрешено ли пользователю с правами администратора игнорировать отсутствие успешных сборок для слияние ПРа
		ctx.Data["IsAdminCanMergeWithoutChecks"] = prUnit.PullRequestsConfig().AdminCanMergeWithoutChecks

		stillCanManualMerge := func() bool {
			if pull.HasMerged || issue.IsClosed || !ctx.IsSigned {
				return false
			}
			if pull.CanAutoMerge() || pull.IsWorkInProgress() || pull.IsChecking() {
				return false
			}
			if (ctx.Doer.IsAdmin || ctx.Repo.IsAdmin()) && prConfig.AllowManualMerge {
				return true
			}

			return false
		}

		ctx.Data["StillCanManualMerge"] = stillCanManualMerge()

		// Check if there is a pending pr merge
		ctx.Data["HasPendingPullRequestMerge"], ctx.Data["PendingPullRequestMerge"], err = pull_model.GetScheduledMergeByPullID(ctx, pull.ID)
		if err != nil {
			ctx.ServerError("GetScheduledMergeByPullID", err)
			return
		}
	}

	// Get Dependencies
	blockedBy, err := issue.BlockedByDependencies(ctx, db.ListOptions{})
	if err != nil {
		ctx.ServerError("BlockedByDependencies", err)
		return
	}
	ctx.Data["BlockedByDependencies"], ctx.Data["BlockedByDependenciesNotPermitted"] = checkBlockedByIssues(ctx, blockedBy)
	if ctx.Written() {
		return
	}

	blocking, err := issue.BlockingDependencies(ctx)
	if err != nil {
		ctx.ServerError("BlockingDependencies", err)
		return
	}

	ctx.Data["BlockingDependencies"], ctx.Data["BlockingByDependenciesNotPermitted"] = checkBlockedByIssues(ctx, blocking)
	if ctx.Written() {
		return
	}

	ctx.Data["Participants"] = participants
	ctx.Data["NumParticipants"] = len(participants)
	ctx.Data["Issue"] = issue
	ctx.Data["Reference"] = issue.Ref
	ctx.Data["SignInLink"] = setting.AppSubURL + "/user/login?redirect_to=" + url.QueryEscape(ctx.Data["Link"].(string))
	ctx.Data["IsIssuePoster"] = ctx.IsSigned && issue.IsPoster(ctx.Doer.ID)
	ctx.Data["HasIssuesOrPullsWritePermission"] = ctx.Repo.CanWriteIssuesOrPulls(issue.IsPull)
	ctx.Data["HasProjectsWritePermission"] = ctx.Repo.CanWrite(unit.TypeProjects)
	ctx.Data["IsRepoAdmin"] = ctx.IsSigned && (ctx.Repo.IsAdmin() || ctx.Doer.IsAdmin)
	ctx.Data["LockReasons"] = setting.Repository.Issue.LockReasons
	ctx.Data["RefEndName"] = git.RefEndName(issue.Ref)

	var hiddenCommentTypes *big.Int
	if ctx.IsSigned {
		val, err := user_model.GetUserSetting(ctx.Doer.ID, user_model.SettingsKeyHiddenCommentTypes)
		if err != nil {
			ctx.ServerError("GetUserSetting", err)
			return
		}
		hiddenCommentTypes, _ = new(big.Int).SetString(val, 10) // we can safely ignore the failed conversion here
	}
	ctx.Data["ShouldShowCommentType"] = func(commentType issues_model.CommentType) bool {
		return hiddenCommentTypes == nil || hiddenCommentTypes.Bit(int(commentType)) == 0
	}
	// для проверки настроек для подключения к sonarqube у репозитория
	sonarSettings, err := repo_model.GetSonarSettings(ctx.Repo.Repository.ID)
	if err != nil {
		log.Error(fmt.Sprintf("ViewUssue repo_model.GetSonarSettings failed while getting sonar settings for repository_id %v: %v", ctx.Repo.Repository.ID, err))
		ctx.ServerError("Failed getting sonar settings", err)
		return
	}
	if sonarSettings != nil {
		ctx.Data["SonarEnabled"] = true
	} else {
		ctx.Data["SonarEnabled"] = false
	}
	// получаем параметры для защиты веток
	rules, err := git_model.FindRepoProtectedBranchRules(ctx, ctx.Repo.Repository.ID)
	if err != nil {
		log.Error("ViewIssue git_model.FindRepoProtectedBranchRules failed while getting params for branches rules for repository_id %v: %v", ctx.Repo.Repository.ID, err)
		ctx.ServerError(fmt.Sprintf("Failed while getting params for branches rules for repository_id %v", ctx.Repo.Repository.ID), err)
		return
	}
	if len(rules) == 0 {
		// Признак того является ли правило защиты веток пустым
		ctx.Data["IsEmptyProtectedBranchRule"] = true
	}

	for _, rule := range rules {
		if rule.EnableSonarQube {
			ctx.Data["EnableSonarQube"] = true
			break
		}
	}

	action := role_model.READ
	if repo.IsPrivate {
		action = role_model.READ_PRIVATE
	}

	allowed, err := s.orgRequestAccessor.IsAccessGranted(*ctx, accesser.OrgAccessRequest{
		DoerID:         ctx.Doer.ID,
		TargetOrgID:    ctx.Repo.Repository.OwnerID,
		TargetTenantID: ctx.Data["TenantID"].(string),
		Action:         action,
	})
	if err != nil {
		log.Error("Error has occurred while checking user's permissions: %v", err)
		ctx.ServerError("Error has occurred while checking user's permissions: %v", err)
		return
	}
	if !allowed {
		allow, err := s.repoRequestAccessor.AccessesByCustomPrivileges(*ctx, accesser.RepoAccessRequest{
			DoerID:          ctx.Doer.ID,
			OrgID:           ctx.Repo.Repository.OwnerID,
			TargetTenantID:  ctx.Data["TenantID"].(string),
			RepoID:          ctx.Repo.Repository.ID,
			CustomPrivilege: role_model.ViewBranch.String(),
		})
		if err != nil {
			log.Error("Error has occurred while checking user's permissions: %v", err)
			ctx.ServerError("Error has occurred while checking user's permissions: %v", err)
			return
		}
		if !allow {
			log.Warn("Access denied: user does not have the required role or privilege")
			ctx.Error(http.StatusForbidden, "User is not allowed to access")
			return
		}
	}

	ctx.HTML(http.StatusOK, tplIssueView)
}

func addParticipant(poster *user_model.User, participants []*user_model.User) []*user_model.User {
	for _, part := range participants {
		if poster.ID == part.ID {
			return participants
		}
	}
	return append(participants, poster)
}

func filterXRefComments(ctx *context.Context, issue *issues_model.Issue) error {
	// Remove comments that the user has no permissions to see
	for i := 0; i < len(issue.Comments); {
		c := issue.Comments[i]
		if issues_model.CommentTypeIsRef(c.Type) && c.RefRepoID != issue.RepoID && c.RefRepoID != 0 {
			var err error
			// Set RefRepo for description in template
			c.RefRepo, err = repo_model.GetRepositoryByID(ctx, c.RefRepoID)
			if err != nil {
				return err
			}
			perm, err := access_model.GetUserRepoPermission(ctx, c.RefRepo, ctx.Doer)
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

// roleDescriptor returns the Role Descriptor for a comment in/with the given repo, poster and issue
func roleDescriptor(ctx stdCtx.Context, repo *repo_model.Repository, poster *user_model.User, issue *issues_model.Issue, hasOriginalAuthor bool) (issues_model.RoleDescriptor, error) {
	if hasOriginalAuthor {
		return issues_model.RoleDescriptorNone, nil
	}

	perm, err := access_model.GetUserRepoPermission(ctx, repo, poster)
	if err != nil {
		return issues_model.RoleDescriptorNone, err
	}

	// By default the poster has no roles on the comment.
	roleDescriptor := issues_model.RoleDescriptorNone

	// Check if the poster is owner of the repo.
	if perm.IsOwner() {
		// If the poster isn't a admin, enable the owner role.
		if !poster.IsAdmin {
			roleDescriptor = roleDescriptor.WithRole(issues_model.RoleDescriptorOwner)
		} else {

			// Otherwise check if poster is the real repo admin.
			ok, err := access_model.IsUserRealRepoAdmin(repo, poster)
			if err != nil {
				return issues_model.RoleDescriptorNone, err
			}
			if ok {
				roleDescriptor = roleDescriptor.WithRole(issues_model.RoleDescriptorOwner)
			}
		}
	}

	// Is the poster can write issues or pulls to the repo, enable the Writer role.
	// Only enable this if the poster doesn't have the owner role already.
	if !roleDescriptor.HasRole("Owner") && perm.CanWriteIssuesOrPulls(issue.IsPull) {
		roleDescriptor = roleDescriptor.WithRole(issues_model.RoleDescriptorWriter)
	}

	// If the poster is the actual poster of the issue, enable Poster role.
	if issue.IsPoster(poster.ID) {
		roleDescriptor = roleDescriptor.WithRole(issues_model.RoleDescriptorPoster)
	}

	return roleDescriptor, nil
}

func checkBlockedByIssues(ctx *context.Context, blockers []*issues_model.DependencyInfo) (canRead, notPermitted []*issues_model.DependencyInfo) {
	var lastRepoID int64
	var lastPerm access_model.Permission
	for i, blocker := range blockers {
		// Get the permissions for this repository
		perm := lastPerm
		if lastRepoID != blocker.Repository.ID {
			if blocker.Repository.ID == ctx.Repo.Repository.ID {
				perm = ctx.Repo.Permission
			} else {
				var err error
				perm, err = access_model.GetUserRepoPermission(ctx, &blocker.Repository, ctx.Doer)
				if err != nil {
					ctx.ServerError("GetUserRepoPermission", err)
					return
				}
			}
			lastRepoID = blocker.Repository.ID
		}

		// check permission
		if !perm.CanReadIssuesOrPulls(blocker.Issue.IsPull) {
			blockers[len(notPermitted)], blockers[i] = blocker, blockers[len(notPermitted)]
			notPermitted = blockers[:len(notPermitted)+1]
		}
	}
	blockers = blockers[len(notPermitted):]
	sortDependencyInfo(blockers)
	sortDependencyInfo(notPermitted)

	return blockers, notPermitted
}

func sortDependencyInfo(blockers []*issues_model.DependencyInfo) {
	sort.Slice(blockers, func(i, j int) bool {
		if blockers[i].RepoID == blockers[j].RepoID {
			return blockers[i].Issue.CreatedUnix < blockers[j].Issue.CreatedUnix
		}
		return blockers[i].RepoID < blockers[j].RepoID
	})
}

// combineLabelComments combine the nearby label comments as one.
func combineLabelComments(issue *issues_model.Issue) {
	var prev, cur *issues_model.Comment
	for i := 0; i < len(issue.Comments); i++ {
		cur = issue.Comments[i]
		if i > 0 {
			prev = issue.Comments[i-1]
		}
		if i == 0 || cur.Type != issues_model.CommentTypeLabel ||
			(prev != nil && prev.PosterID != cur.PosterID) ||
			(prev != nil && cur.CreatedUnix-prev.CreatedUnix >= 60) {
			if cur.Type == issues_model.CommentTypeLabel && cur.Label != nil {
				if cur.Content != "1" {
					cur.RemovedLabels = append(cur.RemovedLabels, cur.Label)
				} else {
					cur.AddedLabels = append(cur.AddedLabels, cur.Label)
				}
			}
			continue
		}

		if cur.Label != nil { // now cur MUST be label comment
			if prev.Type == issues_model.CommentTypeLabel { // we can combine them only prev is a label comment
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

// IssuesOrPull render issues page
func (s *Server) IssuesOrPull(ctx *context.Context) {
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

	var isPull bool

	isPullList := ctx.Params(":type")
	switch isPullList {
	case "pulls":
		isPull = true
		repo_web.MustAllowPulls(ctx)
		if ctx.Written() {
			return
		}
		ctx.Data["Title"] = ctx.Tr("repo.pulls")
		ctx.Data["PageIsPullList"] = true
	case "issues":
		if setting.OneWork.Enabled {
			ctx.NotFound("OneWork", nil)
			return
		}
		repo_web.MustEnableIssues(ctx)
		if ctx.Written() {
			return
		}
		ctx.Data["Title"] = ctx.Tr("repo.issues")
		ctx.Data["PageIsIssueList"] = true
		ctx.Data["NewIssueChooseTemplate"] = issue_service.HasTemplatesOrContactLinks(ctx.Repo.Repository, ctx.Repo.GitRepo)

	}

	repo_web.Issues(ctx, ctx.FormInt64("milestone"), ctx.FormInt64("project"), util.OptionalBoolOf(isPull))

	action := role_model.READ
	if ctx.Repo.Repository.IsPrivate {
		action = role_model.READ_PRIVATE
	}

	allowed, err := s.orgRequestAccessor.IsAccessGranted(*ctx, accesser.OrgAccessRequest{
		DoerID:         ctx.Doer.ID,
		TargetOrgID:    ctx.Repo.Repository.OwnerID,
		TargetTenantID: ctx.Data["TenantID"].(string),
		Action:         action,
	})
	if err != nil {
		log.Error("Error has occurred while checking user's permissions: %v", err)
		ctx.ServerError("Error has occurred while checking user's permissions: %v", err)
		return
	}
	if !allowed {
		allow, err := s.repoRequestAccessor.AccessesByCustomPrivileges(*ctx, accesser.RepoAccessRequest{
			DoerID:          ctx.Doer.ID,
			OrgID:           ctx.Repo.Repository.OwnerID,
			TargetTenantID:  ctx.Data["TenantID"].(string),
			RepoID:          ctx.Repo.Repository.ID,
			CustomPrivilege: role_model.ViewBranch.String(),
		})
		if err != nil || !allow {
			log.Error("Error has occurred while checking user's permissions: %v", err)
			ctx.ServerError("Error has occurred while checking user's permissions: %v", err)
			return
		}
	}
	if ctx.Written() {
		return
	}
	renderMilestones(ctx)
	if ctx.Written() {
		return
	}
	ctx.Data["CanWriteIssuesOrPulls"] = ctx.Repo.CanWriteIssuesOrPulls(isPull)

	ctx.HTML(http.StatusOK, tplIssues)
}

func renderMilestones(ctx *context.Context) {
	// Get milestones
	milestones, _, err := issues_model.GetMilestones(issues_model.GetMilestonesOption{
		RepoID: ctx.Repo.Repository.ID,
		State:  api.StateAll,
	})
	if err != nil {
		ctx.ServerError("GetAllRepoMilestones", err)
		return
	}

	openMilestones, closedMilestones := issues_model.MilestoneList{}, issues_model.MilestoneList{}
	for _, milestone := range milestones {
		if milestone.IsClosed {
			closedMilestones = append(closedMilestones, milestone)
		} else {
			openMilestones = append(openMilestones, milestone)
		}
	}
	ctx.Data["OpenMilestones"] = openMilestones
	ctx.Data["ClosedMilestones"] = closedMilestones
}
