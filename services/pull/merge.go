// Copyright 2019 The Gitea Authors.
// All rights reserved.
// SPDX-License-Identifier: MIT

package pull

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"google.golang.org/grpc/status"

	"code.gitea.io/gitea/models"
	"code.gitea.io/gitea/models/db"
	git_model "code.gitea.io/gitea/models/git"
	issues_model "code.gitea.io/gitea/models/issues"
	access_model "code.gitea.io/gitea/models/perm/access"
	pull_model "code.gitea.io/gitea/models/pull"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/unit"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/cache"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/graceful"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/notification"
	"code.gitea.io/gitea/modules/references"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/timeutil"
	issue_service "code.gitea.io/gitea/services/issue"
)

// getMergeMessage composes the message used when merging a pull request.
func getMergeMessage(ctx context.Context, baseGitRepo *git.Repository, pr *issues_model.PullRequest, mergeStyle repo_model.MergeStyle, extraVars map[string]string) (message, body string, err error) {
	if err := pr.LoadBaseRepo(ctx); err != nil {
		return "", "", err
	}
	if err := pr.LoadHeadRepo(ctx); err != nil {
		return "", "", err
	}
	if err := pr.LoadIssue(ctx); err != nil {
		return "", "", err
	}

	isExternalTracker := pr.BaseRepo.UnitEnabled(ctx, unit.TypeExternalTracker)
	issueReference := "#"
	if isExternalTracker {
		issueReference = "!"
	}

	if mergeStyle != "" {
		templateFilepath := fmt.Sprintf(".gitea/default_merge_message/%s_TEMPLATE.md", strings.ToUpper(string(mergeStyle)))
		commit, err := baseGitRepo.GetBranchCommit(pr.BaseRepo.DefaultBranch)
		if err != nil {
			return "", "", err
		}
		templateContent, err := commit.GetFileContent(templateFilepath, setting.Repository.PullRequest.DefaultMergeMessageSize)
		if err != nil {
			if !git.IsErrNotExist(err) {
				return "", "", err
			}
		} else {
			vars := map[string]string{
				"BaseRepoOwnerName":      pr.BaseRepo.OwnerName,
				"BaseRepoName":           pr.BaseRepo.Name,
				"BaseBranch":             pr.BaseBranch,
				"HeadRepoOwnerName":      "",
				"HeadRepoName":           "",
				"HeadBranch":             pr.HeadBranch,
				"PullRequestTitle":       pr.Issue.Title,
				"PullRequestDescription": pr.Issue.Content,
				"PullRequestPosterName":  pr.Issue.Poster.Name,
				"PullRequestIndex":       strconv.FormatInt(pr.Index, 10),
				"PullRequestReference":   fmt.Sprintf("%s%d", issueReference, pr.Index),
			}
			if pr.HeadRepo != nil {
				vars["HeadRepoOwnerName"] = pr.HeadRepo.OwnerName
				vars["HeadRepoName"] = pr.HeadRepo.Name
			}
			for extraKey, extraValue := range extraVars {
				vars[extraKey] = extraValue
			}
			refs, err := pr.ResolveCrossReferences(ctx)
			if err == nil {
				closeIssueIndexes := make([]string, 0, len(refs))
				closeWord := "close"
				if len(setting.Repository.PullRequest.CloseKeywords) > 0 {
					closeWord = setting.Repository.PullRequest.CloseKeywords[0]
				}
				for _, ref := range refs {
					if ref.RefAction == references.XRefActionCloses {
						if err := ref.LoadIssue(ctx); err != nil {
							return "", "", err
						}
						closeIssueIndexes = append(closeIssueIndexes, fmt.Sprintf("%s %s%d", closeWord, issueReference, ref.Issue.Index))
					}
				}
				if len(closeIssueIndexes) > 0 {
					vars["ClosingIssues"] = strings.Join(closeIssueIndexes, ", ")
				} else {
					vars["ClosingIssues"] = ""
				}
			}
			message, body = expandDefaultMergeMessage(templateContent, vars)
			return message, body, nil
		}
	}

	// Squash merge has a different from other styles.
	if mergeStyle == repo_model.MergeStyleSquash {
		return fmt.Sprintf("%s (%s%d)", pr.Issue.Title, issueReference, pr.Issue.Index), "", nil
	}

	if pr.BaseRepoID == pr.HeadRepoID {
		return fmt.Sprintf("Merge pull request '%s' (%s%d) from %s into %s", pr.Issue.Title, issueReference, pr.Issue.Index, pr.HeadBranch, pr.BaseBranch), "", nil
	}

	if pr.HeadRepo == nil {
		return fmt.Sprintf("Merge pull request '%s' (%s%d) from <deleted>:%s into %s", pr.Issue.Title, issueReference, pr.Issue.Index, pr.HeadBranch, pr.BaseBranch), "", nil
	}

	return fmt.Sprintf("Merge pull request '%s' (%s%d) from %s:%s into %s", pr.Issue.Title, issueReference, pr.Issue.Index, pr.HeadRepo.FullName(), pr.HeadBranch, pr.BaseBranch), "", nil
}

func expandDefaultMergeMessage(template string, vars map[string]string) (message, body string) {
	message = strings.TrimSpace(template)
	if splits := strings.SplitN(message, "\n", 2); len(splits) == 2 {
		message = splits[0]
		body = strings.TrimSpace(splits[1])
	}
	mapping := func(s string) string { return vars[s] }
	return os.Expand(message, mapping), os.Expand(body, mapping)
}

// GetDefaultMergeMessage returns default message used when merging pull request
func GetDefaultMergeMessage(ctx context.Context, baseGitRepo *git.Repository, pr *issues_model.PullRequest, mergeStyle repo_model.MergeStyle) (message, body string, err error) {
	return getMergeMessage(ctx, baseGitRepo, pr, mergeStyle, nil)
}

// Merge merges pull request to base repository.
// Caller should check PR is ready to be merged (review and status checks)
func Merge(ctx context.Context, pr *issues_model.PullRequest, doer *user_model.User, baseGitRepo *git.Repository, mergeStyle repo_model.MergeStyle, expectedHeadCommitID, message string, wasAutoMerged bool) error {
	if err := pr.LoadBaseRepo(ctx); err != nil {
		log.Error("Unable to load base repo: %v", err)
		return fmt.Errorf("unable to load base repo: %w", err)
	} else if err := pr.LoadHeadRepo(ctx); err != nil {
		log.Error("Unable to load head repo: %v", err)
		return fmt.Errorf("unable to load head repo: %w", err)
	}

	pullWorkingPool.CheckIn(fmt.Sprint(pr.ID))
	defer pullWorkingPool.CheckOut(fmt.Sprint(pr.ID))

	// Removing an auto merge pull and ignore if not exist
	// FIXME: is this the correct point to do this? Shouldn't this be after IsMergeStyleAllowed?
	if err := pull_model.DeleteScheduledAutoMerge(ctx, pr.ID); err != nil && !db.IsErrNotExist(err) {
		return err
	}

	prUnit, err := pr.BaseRepo.GetUnit(ctx, unit.TypePullRequests)
	if err != nil {
		log.Error("pr.BaseRepo.GetUnit(unit.TypePullRequests): %v", err)
		return err
	}
	prConfig := prUnit.PullRequestsConfig()

	// Check if merge style is correct and allowed
	if !prConfig.IsMergeStyleAllowed(mergeStyle) {
		return models.ErrInvalidMergeStyle{ID: pr.BaseRepo.ID, Style: mergeStyle}
	}

	defer func() {
		go AddTestPullRequestTask(doer, pr.BaseRepo.ID, pr.BaseBranch, false, "", "")
	}()

	// Run the merge in the hammer context to prevent cancellation
	hammerCtx := graceful.GetManager().HammerContext()

	pr.MergedCommitID, err = doMergeAndPush(hammerCtx, pr, doer, mergeStyle, expectedHeadCommitID, message)
	if err != nil {
		return err
	}

	pr.MergedUnix = timeutil.TimeStampNow()
	pr.Merger = doer
	pr.MergerID = doer.ID

	if _, err := pr.SetMerged(hammerCtx); err != nil {
		log.Error("SetMerged %-v: %v", pr, err)
	}

	if err := pr.LoadIssue(hammerCtx); err != nil {
		log.Error("LoadIssue %-v: %v", pr, err)
	}

	if err := pr.Issue.LoadRepo(hammerCtx); err != nil {
		log.Error("pr.Issue.LoadRepo %-v: %v", pr, err)
	}
	if err := pr.Issue.Repo.LoadOwner(hammerCtx); err != nil {
		log.Error("LoadOwner for %-v: %v", pr, err)
	}

	if wasAutoMerged {
		notification.NotifyAutoMergePullRequest(hammerCtx, doer, pr)
	} else {
		notification.NotifyMergePullRequest(hammerCtx, doer, pr)
	}

	// Reset cached commit count
	cache.Remove(pr.Issue.Repo.GetCommitsCountCacheKey(pr.BaseBranch, true))

	// Resolve cross references
	refs, err := pr.ResolveCrossReferences(hammerCtx)
	if err != nil {
		log.Error("ResolveCrossReferences: %v", err)
		return nil
	}

	for _, ref := range refs {
		if err = ref.LoadIssue(hammerCtx); err != nil {
			return err
		}
		if err = ref.Issue.LoadRepo(hammerCtx); err != nil {
			return err
		}
		close := ref.RefAction == references.XRefActionCloses
		if close != ref.Issue.IsClosed {
			if err = issue_service.ChangeStatus(ref.Issue, doer, pr.MergedCommitID, close); err != nil {
				// Allow ErrDependenciesLeft
				if !issues_model.IsErrDependenciesLeft(err) {
					return err
				}
			}
		}
	}
	return nil
}

// doMergeAndPush performs the merge operation without changing any pull information in database and pushes it up to the base repository
func doMergeAndPush(ctx context.Context, pr *issues_model.PullRequest, doer *user_model.User, mergeStyle repo_model.MergeStyle, expectedHeadCommitID, message string) (string, error) {
	gitRepo, err := git.OpenRepository(ctx, pr.BaseRepo.OwnerName, pr.BaseRepo.Name, pr.BaseRepo.RepoPath())
	if err != nil {
		return "", fmt.Errorf("OpenRepository: %w", err)
	}
	defer gitRepo.Close()

	if pr.HeadCommitID == "" {
		pr.HeadCommitID, err = gitRepo.GetRefCommitID(pr.GetGitHeadBranchRefName())
		if err != nil {
			return "", fmt.Errorf("GetRefCommitID: %w", err)
		}
	}

	mergeCtx := &mergeContext{
		prContext: &prContext{
			Context:     ctx,
			tmpBasePath: pr.BaseRepo.RepoPath(),
			pr:          pr,
			outbuf:      &strings.Builder{},
			errbuf:      &strings.Builder{},
		},
		doer:    doer,
		gitRepo: gitRepo,
	}
	mergeCtx.sig = doer.NewGitSig()
	mergeCtx.committer = mergeCtx.sig

	mergeProcess := &pull_model.SCMergeProcess{
		RepoId:     pr.BaseRepoID,
		UserId:     doer.ID,
		PrId:       pr.ID,
		BaseBranch: pr.BaseBranch,
	}

	err = pull_model.InsertMergeProcess(ctx, mergeProcess)
	if err != nil {
		log.Error("Error has occurred while merging: %v", err)
		return "", fmt.Errorf("failed to insert merge process while merging: %w", err)
	}
	defer func() {
		err = pull_model.DeleteMergeProcess(ctx, mergeProcess)
		if err != nil {
			log.Error("Error has occurred while finishing merge: %v", err)
		}
	}()

	// Merge commits.
	switch mergeStyle {
	case repo_model.MergeStyleMerge:
		if err := doMergeStyleMerge(mergeCtx, message); err != nil {
			mergeStatusError, _ := status.FromError(err)
			if strings.Contains(mergeStatusError.Message(), "conflicting files") {
				return "", models.ErrMergeConflicts{
					Style:  mergeStyle,
					StdErr: mergeStatusError.Message(),
					Err:    err,
				}
			}
			return "", err
		}
	case repo_model.MergeStyleRebase, repo_model.MergeStyleRebaseMerge:
		if err := doMergeStyleRebase(mergeCtx, mergeStyle, message); err != nil {
			mergeStatusError, _ := status.FromError(err)
			if strings.Contains(mergeStatusError.Message(), "conflicting files") {
				return "", models.ErrRebaseConflicts{
					Style:     mergeStyle,
					CommitSHA: expectedHeadCommitID,
					StdErr:    mergeStatusError.Message(),
					Err:       err,
				}
			}
			return "", err
		}
	case repo_model.MergeStyleSquash:
		if err := doMergeStyleSquash(mergeCtx, message); err != nil {
			mergeStatusError, _ := status.FromError(err)
			if strings.Contains(mergeStatusError.Message(), "conflicting files") {
				return "", models.ErrMergeConflicts{
					Style:  mergeStyle,
					StdErr: mergeStatusError.Message(),
					Err:    err,
				}
			}
			return "", err
		}
	default:
		return "", models.ErrInvalidMergeStyle{ID: pr.BaseRepo.ID, Style: mergeStyle}
	}

	return pr.MergedCommitID, nil
}

func commitAndSignNoAuthor(ctx *mergeContext, message string) error {
	cmdCommit := git.NewCommand(ctx, "commit").AddOptionFormat("--message=%s", message)
	if ctx.signKeyID == "" {
		cmdCommit.AddArguments("--no-gpg-sign")
	} else {
		cmdCommit.AddOptionFormat("-S%s", ctx.signKeyID)
	}
	if err := cmdCommit.Run(ctx.RunOpts()); err != nil {
		log.Error("git commit %-v: %v\n%s\n%s", ctx.pr, err, ctx.outbuf.String(), ctx.errbuf.String())
		return fmt.Errorf("git commit %v: %w\n%s\n%s", ctx.pr, err, ctx.outbuf.String(), ctx.errbuf.String())
	}
	return nil
}

func runMergeCommand(ctx *mergeContext, mergeStyle repo_model.MergeStyle, cmd *git.Command) error {
	if err := cmd.Run(ctx.RunOpts()); err != nil {
		// Merge will leave a MERGE_HEAD file in the .git folder if there is a conflict
		if _, statErr := os.Stat(filepath.Join(ctx.tmpBasePath, ".git", "MERGE_HEAD")); statErr == nil {
			// We have a merge conflict error
			log.Debug("MergeConflict %-v: %v\n%s\n%s", ctx.pr, err, ctx.outbuf.String(), ctx.errbuf.String())
			return models.ErrMergeConflicts{
				Style:  mergeStyle,
				StdOut: ctx.outbuf.String(),
				StdErr: ctx.errbuf.String(),
				Err:    err,
			}
		} else if strings.Contains(ctx.errbuf.String(), "refusing to merge unrelated histories") {
			log.Debug("MergeUnrelatedHistories %-v: %v\n%s\n%s", ctx.pr, err, ctx.outbuf.String(), ctx.errbuf.String())
			return models.ErrMergeUnrelatedHistories{
				Style:  mergeStyle,
				StdOut: ctx.outbuf.String(),
				StdErr: ctx.errbuf.String(),
				Err:    err,
			}
		}
		log.Error("git merge %-v: %v\n%s\n%s", ctx.pr, err, ctx.outbuf.String(), ctx.errbuf.String())
		return fmt.Errorf("git merge %v: %w\n%s\n%s", ctx.pr, err, ctx.outbuf.String(), ctx.errbuf.String())
	}
	ctx.outbuf.Reset()
	ctx.errbuf.Reset()

	return nil
}

var escapedSymbols = regexp.MustCompile(`([*[?! \\])`)

// IsUserAllowedToMerge check if user is allowed to merge PR with given permissions and branch protections
func IsUserAllowedToMerge(ctx context.Context, pr *issues_model.PullRequest, p access_model.Permission, user *user_model.User) (bool, error) {
	if user == nil {
		return false, nil
	}

	pb, err := git_model.GetMergeMatchProtectedBranchRule(ctx, pr.BaseRepoID, pr.BaseBranch)
	if err != nil {
		log.Error("Err: get merge protected branch rule with base repo id - %d, base branch %s", pr.BaseRepoID, pr.BaseBranch)
		return false, fmt.Errorf("Err: get merge protected branch rule with base repo id - %d, base branch %s: %w", pr.BaseRepoID, pr.BaseBranch, err)
	}

	if (p.CanWrite(unit.TypeCode) && pb == nil) || (pb != nil && git_model.IsUserMergeWhitelisted(ctx, *pb, user.ID, p)) {
		return true, nil
	}

	return false, nil
}

// isApprovalConditionsMet проверяет, выполнены ли условия для одобрения PR
func isApprovalConditionsMet(amountApprovedStatus int64, amountUsers int64, amountUsersSettings int64, amountUsersCodeOwners int) bool {
	return amountApprovedStatus >= amountUsersSettings &&
		amountUsers >= amountUsersSettings &&
		int(amountUsers) >= amountUsersCodeOwners
}

// canAdminMergeWithoutChecks проверяет, может ли администратор объединить PR без проверок
func canAdminMergeWithoutChecks(ctx context.Context, pr *issues_model.PullRequest, user *user_model.User) (bool, error) {
	prUnit, err := pr.BaseRepo.GetUnit(ctx, unit.TypePullRequests)
	if err != nil {
		return false, fmt.Errorf("get pull request unit: %w", err)
	}
	return prUnit.PullRequestsConfig().AdminCanMergeWithoutChecks && user.IsAdmin, nil
}

// getApprovedStatusCount считает количество Code Owners, которые одобрили PR
func getApprovedStatusCount(usersCodeOwners []*repo_model.CodeOwners) int64 {
	var count int64
	for _, v := range usersCodeOwners {
		if v.ApprovalStatus == repo_model.ReviewTypeApprove {
			count++
		}
	}
	return count
}

// IsCodeOwnersAllowedToMerge проверяет, может ли пользователь объединить PR, учитывая Code Owners и защиту веток
func IsCodeOwnersAllowedToMerge(ctx context.Context, pr *issues_model.PullRequest, user *user_model.User) (bool, error) {
	// Получаем количество Code Owners из файла CODEOWNERS
	amountUsers, _, err := issues_model.GetAmountCodeOwners(ctx, pr)
	if err != nil {
		return false, fmt.Errorf("get amount code owners: %w", err)
	}
	if amountUsers == 0 {
		return true, nil
	}

	// Получаем количество пользователей из таблицы code_owners_settings
	amountUsersSettings, err := repo_model.GetCodeOwnersSettings(ctx, pr.BaseRepo.ID)
	if err != nil {
		return false, fmt.Errorf("get amount code owners settings: %w", err)
	}
	if amountUsersSettings.AmountUsers == 0 {
		return true, nil
	}

	// Получаем список пользователей из таблицы code_owners
	amountUsersCodeOwners, err := repo_model.GetCodeOwners(ctx, pr.BaseRepo.ID, pr.ID)
	if err != nil {
		return false, fmt.Errorf("get amount code owners: %w", err)
	}
	if len(amountUsersCodeOwners) == 0 {
		return true, nil
	}

	// Подсчитываем количество одобрений
	amountApprovedStatus := getApprovedStatusCount(amountUsersCodeOwners)

	// Проверяем, выполнены ли условия для одобрения PR
	isApproved := isApprovalConditionsMet(amountApprovedStatus, amountUsers, amountUsersSettings.AmountUsers, len(amountUsersCodeOwners))

	// Проверяем, может ли администратор объединить PR без проверок
	canAdminMerge, err := canAdminMergeWithoutChecks(ctx, pr, user)
	if err != nil {
		return false, fmt.Errorf("can admin merge without checks: %w", err)
	}

	return isApproved || canAdminMerge, nil
}

// CheckPullBranchProtections checks whether the PR is ready to be merged (reviews and status checks)
func CheckPullBranchProtections(ctx context.Context, pr *issues_model.PullRequest, skipProtectedFilesCheck bool) (err error) {
	if err = pr.LoadBaseRepo(ctx); err != nil {
		return fmt.Errorf("LoadBaseRepo: %w", err)
	}

	pb, err := git_model.GetMergeMatchProtectedBranchRule(ctx, pr.BaseRepoID, pr.BaseBranch)
	if err != nil {
		return fmt.Errorf("LoadProtectedBranch: %w", err)
	}
	if pb == nil {
		return nil
	}

	isPass, err := IsPullCommitStatusPass(ctx, pr)
	if err != nil {
		return err
	}
	if !isPass {
		return models.ErrDisallowedToMerge{
			Reason: "Not all required status checks successful",
		}
	}

	if skipProtectedFilesCheck {
		return nil
	}

	if git_model.MergeBlockedByProtectedFiles(*pb, pr.ChangedProtectedFiles) {
		return models.ErrDisallowedToMerge{
			Reason: "Changed protected files",
		}
	}

	return nil
}

// MergedManually mark pr as merged manually
func MergedManually(pr *issues_model.PullRequest, doer *user_model.User, baseGitRepo *git.Repository, commitID string) error {
	pullWorkingPool.CheckIn(fmt.Sprint(pr.ID))
	defer pullWorkingPool.CheckOut(fmt.Sprint(pr.ID))

	if err := db.WithTx(db.DefaultContext, func(ctx context.Context) error {
		if err := pr.LoadBaseRepo(ctx); err != nil {
			return err
		}
		prUnit, err := pr.BaseRepo.GetUnit(ctx, unit.TypePullRequests)
		if err != nil {
			return err
		}
		prConfig := prUnit.PullRequestsConfig()

		// Check if merge style is correct and allowed
		if !prConfig.IsMergeStyleAllowed(repo_model.MergeStyleManuallyMerged) {
			return models.ErrInvalidMergeStyle{ID: pr.BaseRepo.ID, Style: repo_model.MergeStyleManuallyMerged}
		}

		if len(commitID) < git.SHAFullLength {
			return fmt.Errorf("Wrong commit ID")
		}

		commit, err := baseGitRepo.GetCommit(commitID)
		if err != nil {
			if git.IsErrNotExist(err) {
				return fmt.Errorf("Wrong commit ID")
			}
			return err
		}
		commitID = commit.ID.String()

		ok, err := baseGitRepo.IsCommitInBranch(commitID, pr.BaseBranch)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("Wrong commit ID")
		}

		pr.MergedCommitID = commitID
		pr.MergedUnix = timeutil.TimeStamp(commit.Author.When.Unix())
		pr.Status = issues_model.PullRequestStatusManuallyMerged
		pr.Merger = doer
		pr.MergerID = doer.ID

		var merged bool
		if merged, err = pr.SetMerged(ctx); err != nil {
			return err
		} else if !merged {
			return fmt.Errorf("SetMerged failed")
		}
		return nil
	}); err != nil {
		return err
	}

	notification.NotifyMergePullRequest(baseGitRepo.Ctx, doer, pr)
	log.Info("manuallyMerged[%d]: Marked as manually merged into %s/%s by commit id: %s", pr.ID, pr.BaseRepo.Name, pr.BaseBranch, commitID)
	return nil
}
