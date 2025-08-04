package repo_server

import (
	"errors"
	"fmt"
	"net/http"

	asymkey_model "code.gitea.io/gitea/models/asymkey"
	"code.gitea.io/gitea/models/db"
	git_model "code.gitea.io/gitea/models/git"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/role_model"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/base"
	"code.gitea.io/gitea/modules/charset"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/trace"
	"code.gitea.io/gitea/routers/web/repo"
	"code.gitea.io/gitea/routers/web/user/accesser"
	"code.gitea.io/gitea/services/gitdiff"
)

const (
	tplCommits     base.TplName = "repo/commits"
	tplCommitPage  base.TplName = "repo/commit_page"
	tplPullCommits base.TplName = "repo/pulls/commits"
	tplPullFiles   base.TplName = "repo/pulls/files"

	pullRequestTemplateKey = "PullRequestTemplate"
	timeFormat             = "2006-01-02T15:04:05.000Z"
)

// Commits render branch's commits
func (s *Server) Commits(ctx *context.Context) {
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

	ctx.Data["PageIsCommits"] = true
	if ctx.Repo.Commit == nil {
		log.Debug("Commit not found")
		ctx.NotFound("Commit not found", nil)
		return
	}
	ctx.Data["PageIsViewCode"] = true

	commitsCount, err := ctx.Repo.GetCommitsCount()
	if err != nil {
		log.Error("Error has occurred while getting commits: %v", err)
		ctx.ServerError("Get commit", err)
		return
	}

	page := ctx.FormInt("page")
	if page <= 1 {
		page = 1
	}

	pageSize := ctx.FormInt("limit")
	if pageSize <= 0 {
		pageSize = setting.Git.CommitsRangeSize
	}

	// Both `git log branchName` and `git log commitId` work.
	commits, err := ctx.Repo.Commit.CommitsByRange(page, pageSize, "")
	if err != nil {
		log.Error("Error has occurred while getting commits: %v", err)
		ctx.ServerError("Commits size", err)
		return
	}
	ctx.Data["Commits"] = git_model.ConvertFromGitCommit(ctx, commits, ctx.Repo.Repository)

	ctx.Data["Username"] = ctx.Repo.Owner.Name
	ctx.Data["Reponame"] = ctx.Repo.Repository.Name
	ctx.Data["CommitCount"] = commitsCount
	ctx.Data["RefName"] = ctx.Repo.RefName

	pager := context.NewPagination(int(commitsCount), pageSize, page, 5)
	pager.SetDefaultParams(ctx)
	ctx.Data["Page"] = pager

	if setting.SourceControl.TenantWithRoleModeEnabled {
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
			allow, err := s.repoRequestAccessor.AccessesByCustomPrivileges(ctx, accesser.RepoAccessRequest{
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
				log.Debug("Access denied: user does not have the required role or privilege")
				ctx.ServerError("Error has occurred while checking user's permissions", nil)
				return
			}
		}

	}
	ctx.HTML(http.StatusOK, tplCommits)
}

// Diff show different from current commit to previous commit
func (s *Server) Diff(ctx *context.Context) {
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

	ctx.Data["PageIsDiff"] = true

	userName := ctx.Repo.Owner.Name
	repoName := ctx.Repo.Repository.Name
	commitID := ctx.Params(":sha")
	var (
		gitRepo *git.Repository
	)

	if ctx.Data["PageIsWiki"] != nil {
		gitRepo, err = git.OpenRepository(ctx, ctx.Repo.Repository.OwnerName, ctx.Repo.Repository.Name, ctx.Repo.Repository.WikiPath())
		if err != nil {
			log.Error("Error has occurred while opening git repository: %v", err)
			ctx.ServerError("Repo.GitRepo.GetCommit", err)
			return
		}
		defer gitRepo.Close()
	} else {
		gitRepo = ctx.Repo.GitRepo
	}

	commit, err := gitRepo.GetCommit(commitID)
	if err != nil {
		log.Error("Error has occurred while getting commit: %v", err)
		if git.IsErrNotExist(err) {
			ctx.NotFound("Repo.GitRepo.GetCommit", err)
		} else {
			ctx.ServerError("Repo.GitRepo.GetCommit", err)
		}
		return
	}
	if len(commitID) != git.SHAFullLength {
		commitID = commit.ID.String()
	}

	fileOnly := ctx.FormBool("file-only")
	maxLines, maxFiles := setting.Git.MaxGitDiffLines, setting.Git.MaxGitDiffFiles
	files := ctx.FormStrings("files")
	if fileOnly && (len(files) == 2 || len(files) == 1) {
		maxLines, maxFiles = -1, -1
	}

	diff, err := gitdiff.GetDiff(gitRepo, &gitdiff.DiffOptions{
		AfterCommitID:      commitID,
		SkipTo:             ctx.FormString("skip-to"),
		MaxLines:           maxLines,
		MaxLineCharacters:  setting.Git.MaxGitDiffLineCharacters,
		MaxFiles:           maxFiles,
		WhitespaceBehavior: gitdiff.GetWhitespaceFlag(ctx.Data["WhitespaceBehavior"].(string)),
	}, files...)
	if err != nil {
		log.Error("Error has occurred while getting diff: %v", err)
		ctx.NotFound("GetDiff", err)
		return
	}

	parents := make([]string, commit.ParentCount())
	for i := 0; i < commit.ParentCount(); i++ {
		sha, err := commit.ParentID(i)
		if err != nil {
			log.Error("Error has occurred while getting parent ID: %v", err)
			ctx.NotFound("repo.Diff", err)
			return
		}
		parents[i] = sha.String()
	}

	ctx.Data["CommitID"] = commitID
	ctx.Data["AfterCommitID"] = commitID
	ctx.Data["Username"] = userName
	ctx.Data["Reponame"] = repoName

	var parentCommit *git.Commit
	if commit.ParentCount() > 0 {
		parentCommit, err = gitRepo.GetCommit(parents[0])
		if err != nil {
			log.Error("Error has occurred while getting commit: %v", err)
			ctx.NotFound("GetParentCommit", err)
			return
		}
	}
	repo.SetCompareContext(ctx, parentCommit, commit, userName, repoName)
	ctx.Data["Title"] = commit.Summary() + " · " + base.ShortSha(commitID)
	ctx.Data["Commit"] = commit
	ctx.Data["Diff"] = diff

	statuses, _, err := git_model.GetLatestCommitStatus(ctx, ctx.Repo.Repository.ID, commitID, db.ListOptions{})
	if err != nil {
		log.Error("Error has occurred while getting latest commit status: %v", err)
		return
	}

	ctx.Data["CommitStatus"] = git_model.CalcCommitStatus(statuses)
	ctx.Data["CommitStatuses"] = statuses

	verification := asymkey_model.ParseCommitWithSignature(ctx, commit)
	ctx.Data["Verification"] = verification
	ctx.Data["Author"] = user_model.ValidateCommitWithEmail(ctx, commit)
	ctx.Data["Parents"] = parents
	ctx.Data["DiffNotAvailable"] = diff.NumFiles == 0

	if err := asymkey_model.CalculateTrustStatus(verification, ctx.Repo.Repository.GetTrustModel(), func(user *user_model.User) (bool, error) {
		return repo_model.IsOwnerMemberCollaborator(ctx.Repo.Repository, user.ID)
	}, nil); err != nil {
		log.Error("Error has occurred while calculating trust status: %v", err)
		ctx.ServerError("Calculate status", err)
		return
	}

	note := &git.Note{}
	err = git.GetNote(ctx, ctx.Repo.GitRepo, commitID, note)
	if err == nil {
		ctx.Data["Note"] = string(charset.ToUTF8WithFallback(note.Message))
		ctx.Data["NoteCommit"] = note.Commit
		ctx.Data["NoteAuthor"] = user_model.ValidateCommitWithEmail(ctx, note.Commit)
	}

	ctx.Data["BranchName"], err = commit.GetBranchName()
	if err != nil {
		log.Error("Error has occurred while getting branch name: %v", err)
		ctx.ServerError("commit.GetBranchName", err)
		return
	}

	ctx.Data["TagName"], err = commit.GetTagName()
	if err != nil {
		log.Error("Error has occurred while getting tag name: %v", err)
		ctx.ServerError("commit.GetTagName", err)
		return
	}
	var tenantID string
	if ctx.Data != nil && ctx.Data["TenantID"] != "" {
		tenantID = ctx.Data["TenantID"].(string)
	} else {
		tenantID, err = role_model.GetUserTenantId(ctx, ctx.Doer.ID)
		if err != nil {
			log.Error("Error has occurred while getting tenant id by user: %v", err)
			ctx.ServerError("Error has occurred while getting tenant id by user: %v", err)
			return
		}
	}

	action := role_model.READ
	if ctx.Repo.Repository.IsPrivate {
		action = role_model.READ_PRIVATE
	}
	allowed, err := s.orgRequestAccessor.IsAccessGranted(*ctx, accesser.OrgAccessRequest{
		DoerID:         ctx.Doer.ID,
		TargetOrgID:    ctx.Repo.Repository.OwnerID,
		TargetTenantID: tenantID,
		Action:         action,
	})
	if err != nil {
		log.Error("Error has occurred while checking user's permissions: %v", err)
		ctx.ServerError("Error has occurred while checking user's permissions: %v", err)
		return
	}
	if !allowed {
		allow, err := s.repoRequestAccessor.AccessesByCustomPrivileges(ctx, accesser.RepoAccessRequest{
			DoerID:          ctx.Doer.ID,
			OrgID:           ctx.Repo.Repository.OwnerID,
			TargetTenantID:  tenantID,
			RepoID:          ctx.Repo.Repository.ID,
			CustomPrivilege: role_model.ViewBranch.String(),
		})
		if err != nil {
			log.Error("Error has occurred while checking user's permissions: %v", err)
			ctx.ServerError("Error has occurred while checking user's permissions: %v", err)
			return
		}
		if !allow {
			ctx.Error(http.StatusForbidden, "You are not allowed to view diff")
			return
		}
	}

	ctx.HTML(http.StatusOK, tplCommitPage)
}

// RawDiff dumps diff results of repository in given commit ID to io.Writer
func RawDiff(ctx *context.Context) {
	var gitRepo *git.Repository
	if ctx.Data["PageIsWiki"] != nil {
		wikiRepo, err := git.OpenRepository(ctx, ctx.Repo.Repository.OwnerName, ctx.Repo.Repository.Name, ctx.Repo.Repository.WikiPath())
		if err != nil {
			ctx.ServerError("OpenRepository", err)
			return
		}
		defer wikiRepo.Close()
		gitRepo = wikiRepo
	} else {
		gitRepo = ctx.Repo.GitRepo
		if gitRepo == nil {
			ctx.ServerError("GitRepo not open", fmt.Errorf("no open git repo for '%s'", ctx.Repo.Repository.FullName()))
			return
		}
	}
	if err := git.GetRawDiff(
		gitRepo,
		ctx.Params(":sha"),
		git.RawDiffType(ctx.Params(":ext")),
		ctx.Resp,
	); err != nil {
		if git.IsErrNotExist(err) {
			ctx.NotFound("GetRawDiff",
				errors.New("commit "+ctx.Params(":sha")+" does not exist."))
			return
		}
		ctx.ServerError("GetRawDiff", err)
		return
	}
}

// FileHistory show a file's reversions
func (s *Server) FileHistory(ctx *context.Context) {
	ctx.Data["IsRepoToolbarCommits"] = true
	fileName := ctx.Repo.TreePath
	if len(fileName) == 0 {
		s.Commits(ctx)
		return
	}

	commitsCount, err := ctx.Repo.GitRepo.FileCommitsCount(ctx.Repo.RefName, fileName)
	if err != nil {
		ctx.ServerError("FileCommitsCount", err)
		return
	} else if commitsCount == 0 {
		ctx.NotFound("FileCommitsCount", nil)
		return
	}

	page := ctx.FormInt("page")
	if page <= 1 {
		page = 1
	}

	commits, err := ctx.Repo.GitRepo.CommitsByFileAndRange(
		git.CommitsByFileAndRangeOptions{
			Revision: ctx.Repo.RefName,
			File:     fileName,
			Page:     page,
		})
	if err != nil {
		ctx.ServerError("CommitsByFileAndRange", err)
		return
	}
	ctx.Data["Commits"] = git_model.ConvertFromGitCommit(ctx, commits, ctx.Repo.Repository)

	ctx.Data["Username"] = ctx.Repo.Owner.Name
	ctx.Data["Reponame"] = ctx.Repo.Repository.Name
	ctx.Data["FileName"] = fileName
	ctx.Data["CommitCount"] = commitsCount
	ctx.Data["RefName"] = ctx.Repo.RefName

	pager := context.NewPagination(int(commitsCount), setting.Git.CommitsRangeSize, page, 5)
	pager.SetDefaultParams(ctx)
	ctx.Data["Page"] = pager

	ctx.HTML(http.StatusOK, tplCommits)
}

// RefCommits render commits page
func (s *Server) RefCommits(ctx *context.Context) {
	switch {
	case len(ctx.Repo.TreePath) == 0:
		s.Commits(ctx)
	case ctx.Repo.TreePath == "search":
		s.SearchCommits(ctx)
	default:
		s.FileHistory(ctx)
	}
}

// SearchCommits render commits filtered by keyword
func (s *Server) SearchCommits(ctx *context.Context) {
	ctx.Data["PageIsCommits"] = true
	ctx.Data["PageIsViewCode"] = true

	query := ctx.FormTrim("q")
	if len(query) == 0 {
		ctx.Redirect(ctx.Repo.RepoLink + "/commits/" + ctx.Repo.BranchNameSubURL())
		return
	}

	all := ctx.FormBool("all")
	opts := git.NewSearchCommitsOptions(query, all)
	commits, err := ctx.Repo.GitRepo.SearchCommits(ctx.Repo.CommitID, opts)
	if err != nil {
		ctx.ServerError("SearchCommits", err)
		return
	}
	ctx.Data["CommitCount"] = len(commits)
	ctx.Data["Commits"] = git_model.ConvertFromGitCommit(ctx, commits, ctx.Repo.Repository)

	ctx.Data["Keyword"] = query
	if all {
		ctx.Data["All"] = "checked"
	}
	ctx.Data["Username"] = ctx.Repo.Owner.Name
	ctx.Data["Reponame"] = ctx.Repo.Repository.Name
	ctx.Data["RefName"] = ctx.Repo.RefName
	ctx.HTML(http.StatusOK, tplCommits)
}
