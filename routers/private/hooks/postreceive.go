// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package hooks

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"code.gitea.io/gitea/models/db"
	issues_model "code.gitea.io/gitea/models/issues"
	"code.gitea.io/gitea/models/organization/custom"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/role_model"
	"code.gitea.io/gitea/models/user"
	gitea_context "code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/private"
	repo_module "code.gitea.io/gitea/modules/repository"
	"code.gitea.io/gitea/modules/sbt/audit"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/util"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/services"
	repo_service "code.gitea.io/gitea/services/repository"
)

// HookPostReceive updates services and users
func (s Server) HookPostReceive(ctx *gitea_context.PrivateContext) {
	opts := web.GetForm(ctx).(*private.HookOptions)

	// We don't rely on RepoAssignment here because:
	// a) we don't need the git repo in this function
	// b) our update function will likely change the repository in the db so we will need to refresh it
	// c) we don't always need the repo

	ownerName := ctx.Params(":owner")
	repoName := ctx.Params(":repo")

	auditParams := map[string]string{
		"repository": repoName,
		"owner":      ownerName,
	}

	// defer getting the repository at this point - as we should only retrieve it if we're going to call update
	var repo *repo_model.Repository

	updates := make([]*repo_module.PushUpdateOptions, 0, len(opts.OldCommitIDs))
	wasEmpty := false

	for i := range opts.OldCommitIDs {
		refFullName := opts.RefFullNames[i]

		// Only trigger activity updates for changes to branches or
		// tags.  Updates to other refs (eg, refs/notes, refs/changes,
		// or other less-standard refs spaces are ignored since there
		// may be a very large number of them).
		if strings.HasPrefix(refFullName, git.BranchPrefix) || strings.HasPrefix(refFullName, git.TagPrefix) {
			if repo == nil {
				repo = LoadRepository(ctx, ownerName, repoName)
				if ctx.Written() {
					// Error handled in loadRepository
					auditParams["error"] = "Error has occurred while loading repository"
					audit.CreateAndSendEvent(audit.ChangesPushEvent, opts.UserName, strconv.FormatInt(opts.UserID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
					return
				}
				wasEmpty = repo.IsEmpty
			}

			option := &repo_module.PushUpdateOptions{
				RefFullName:  refFullName,
				OldCommitID:  opts.OldCommitIDs[i],
				NewCommitID:  opts.NewCommitIDs[i],
				PusherID:     opts.UserID,
				PusherName:   opts.UserName,
				RepoUserName: ownerName,
				RepoName:     repoName,
			}
			updates = append(updates, option)
			if repo.IsEmpty && option.IsBranch() && (option.BranchName() == "master" || option.BranchName() == "main") {
				// put the master/main branch first
				copy(updates[1:], updates)
				updates[0] = option
			}

			if option.IsTag() {
				usr, err := user.GetUserByID(ctx, opts.UserID)
				if err != nil {
					log.Error("Error has occurred while getting user: %v", err)
					auditParams["error"] = "Error has occurred while getting user"
					audit.CreateAndSendEvent(audit.ChangesPushEvent, opts.UserName, strconv.FormatInt(opts.UserID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
					return
				}

				gitalyRepo, err := git.OpenRepository(ctx, ownerName, repoName, repo.RepoPath())
				if err != nil {
					log.Error("Error has occurred while opening repository: %v", err)
					auditParams["error"] = "Error has occurred while opening repository"
					audit.CreateAndSendEvent(audit.ChangesPushEvent, opts.UserName, strconv.FormatInt(opts.UserID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
					return
				}

				var branchName string
				emptySha := "0000000000000000000000000000000000000000"
				if opts.NewCommitIDs[i] == emptySha {
					branchName = repo.DefaultBranch
				} else {
					commit, err := gitalyRepo.GetCommit(opts.NewCommitIDs[i])
					if err != nil {
						log.Error("Error has occurred while getting commit: %v", err)
						auditParams["error"] = "Error has occurred while getting commit"
						audit.CreateAndSendEvent(audit.ChangesPushEvent, opts.UserName, strconv.FormatInt(opts.UserID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
						return
					}

					branchName, err = commit.GetBranchName()
					if err != nil {
						log.Error("Error has occurred while getting branch name: %v", err)
						auditParams["error"] = "Error has occurred while getting branch name"
						audit.CreateAndSendEvent(audit.ChangesPushEvent, opts.UserName, strconv.FormatInt(opts.UserID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
						return
					}
				}

				newRel := &repo_model.Release{
					RepoID:       repo.ID,
					Repo:         repo,
					PublisherID:  opts.UserID,
					Publisher:    usr,
					TagName:      option.TagName(),
					LowerTagName: strings.ToLower(option.TagName()),
					Target:       branchName,
					Sha1:         option.NewCommitID,
					IsTag:        true,
				}

				if err := repo_model.SaveOrUpdateTag(repo, newRel); err != nil {
					log.Error("Error has occurred while saving tag: %v", err)
					auditParams["error"] = "Error has occurred while updating tags"
					audit.CreateAndSendEvent(audit.ChangesPushEvent, opts.UserName, strconv.FormatInt(opts.UserID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
					return
				}
			}
		}
	}

	if repo != nil && len(updates) > 0 {
		if err := repo_service.PushUpdates(updates); err != nil {
			// Аудирование ошибок при пуше изменений обработано в pushUpdates
			log.Error("Failed to Update: %s/%s Total Updates: %d", ownerName, repoName, len(updates))
			for i, update := range updates {
				log.Error("Failed to Update: %s/%s Update: %d/%d: Branch: %s", ownerName, repoName, i, len(updates), update.BranchName())
			}
			log.Error("Failed to Update: %s/%s Error: %v", ownerName, repoName, err)

			ctx.JSON(http.StatusInternalServerError, private.HookPostReceiveResult{
				Err: fmt.Sprintf("Failed to Update: %s/%s Error: %v", ownerName, repoName, err),
			})
			return
		}
	}

	// Handle Push Options
	if len(opts.GitPushOptions) > 0 {
		// load the repository
		if repo == nil {
			repo = LoadRepository(ctx, ownerName, repoName)
			if ctx.Written() {
				// Error handled in loadRepository
				auditParams["error"] = "Error has occurred while loading repository"
				audit.CreateAndSendEvent(audit.ChangesPushEvent, opts.UserName, strconv.FormatInt(opts.UserID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
				return
			}
			wasEmpty = repo.IsEmpty
		}

		repo.IsPrivate = opts.GitPushOptions.Bool(private.GitPushOptionRepoPrivate, repo.IsPrivate)
		repo.IsTemplate = opts.GitPushOptions.Bool(private.GitPushOptionRepoTemplate, repo.IsTemplate)
		if err := repo_model.UpdateRepositoryCols(ctx, repo, "is_private", "is_template"); err != nil {
			log.Error("Failed to Update: %s/%s Error: %v", ownerName, repoName, err)
			auditParams["error"] = "Error has occurred while updating repository cols"
			audit.CreateAndSendEvent(audit.ChangesPushEvent, opts.UserName, strconv.FormatInt(opts.UserID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			ctx.JSON(http.StatusInternalServerError, private.HookPostReceiveResult{
				Err: fmt.Sprintf("Failed to Update: %s/%s Error: %v", ownerName, repoName, err),
			})
		}
	}

	results := make([]private.HookPostReceiveBranchResult, 0, len(opts.OldCommitIDs))

	// We have to reload the repo in case its state is changed above
	repo = nil
	var baseRepo *repo_model.Repository

	// Now handle the pull request notification trailers
	for i := range opts.OldCommitIDs {
		refFullName := opts.RefFullNames[i]
		newCommitID := opts.NewCommitIDs[i]

		// post update for agit pull request
		if git.SupportProcReceive && strings.HasPrefix(refFullName, git.PullPrefix) {
			if repo == nil {
				repo = LoadRepository(ctx, ownerName, repoName)
				if ctx.Written() {
					auditParams["error"] = "Error has occurred while loading repository"
					audit.CreateAndSendEvent(audit.ChangesPushEvent, opts.UserName, strconv.FormatInt(opts.UserID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
					return
				}
			}

			pullIndexStr := strings.TrimPrefix(refFullName, git.PullPrefix)
			pullIndexStr = strings.Split(pullIndexStr, "/")[0]
			pullIndex, _ := strconv.ParseInt(pullIndexStr, 10, 64)
			if pullIndex <= 0 {
				continue
			}

			pr, err := issues_model.GetPullRequestByIndex(ctx, repo.ID, pullIndex)
			if err != nil && !issues_model.IsErrPullRequestNotExist(err) {
				log.Error("Failed to get PR by index %v Error: %v", pullIndex, err)
				ctx.JSON(http.StatusInternalServerError, private.Response{
					Err: fmt.Sprintf("Failed to get PR by index %v Error: %v", pullIndex, err),
				})
				auditParams["error"] = "Error has occurred while getting pull request by index"
				audit.CreateAndSendEvent(audit.ChangesPushEvent, opts.UserName, strconv.FormatInt(opts.UserID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
				return
			}
			if pr == nil {
				continue
			}

			results = append(results, private.HookPostReceiveBranchResult{
				Message: setting.Git.PullRequestPushMessage && repo.AllowsPulls(),
				Create:  false,
				Branch:  "",
				URL:     fmt.Sprintf("%s/pulls/%d", repo.HTMLURL(), pr.Index),
			})
			continue
		}

		branch := git.RefEndName(opts.RefFullNames[i])

		// If we've pushed a branch (and not deleted it)
		if newCommitID != git.EmptySHA && strings.HasPrefix(refFullName, git.BranchPrefix) {

			// First ensure we have the repository loaded, we're allowed pulls requests and we can get the base repo
			if repo == nil {
				repo = LoadRepository(ctx, ownerName, repoName)
				if ctx.Written() {
					auditParams["error"] = "Error has occurred while loading repository"
					audit.CreateAndSendEvent(audit.ChangesPushEvent, opts.UserName, strconv.FormatInt(opts.UserID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
					return
				}

				baseRepo = repo

				if repo.IsFork {
					if err := repo.GetBaseRepo(ctx); err != nil {
						log.Error("Failed to get Base Repository of Forked repository: %-v Error: %v", repo, err)
						ctx.JSON(http.StatusInternalServerError, private.HookPostReceiveResult{
							Err:          fmt.Sprintf("Failed to get Base Repository of Forked repository: %-v Error: %v", repo, err),
							RepoWasEmpty: wasEmpty,
						})
						auditParams["error"] = "Error has occurred while getting base repository"
						audit.CreateAndSendEvent(audit.ChangesPushEvent, opts.UserName, strconv.FormatInt(opts.UserID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
						return
					}
					if repo.BaseRepo.AllowsPulls() {
						baseRepo = repo.BaseRepo
					}
				}

				if !baseRepo.AllowsPulls() {
					// We can stop there's no need to go any further
					ctx.JSON(http.StatusOK, private.HookPostReceiveResult{
						RepoWasEmpty: wasEmpty,
					})
					audit.CreateAndSendEvent(audit.ChangesPushEvent, opts.UserName, strconv.FormatInt(opts.UserID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)
					return
				}
			}

			// If our branch is the default branch of an unforked repo - there's no PR to create or refer to
			if !repo.IsFork && branch == baseRepo.DefaultBranch {
				results = append(results, private.HookPostReceiveBranchResult{})
				continue
			}

			pr, err := issues_model.GetUnmergedPullRequest(ctx, repo.ID, baseRepo.ID, branch, baseRepo.DefaultBranch, issues_model.PullRequestFlowGithub)
			if err != nil && !issues_model.IsErrPullRequestNotExist(err) {
				log.Error("Failed to get active PR in: %-v Branch: %s to: %-v Branch: %s Error: %v", repo, branch, baseRepo, baseRepo.DefaultBranch, err)
				ctx.JSON(http.StatusInternalServerError, private.HookPostReceiveResult{
					Err: fmt.Sprintf(
						"Failed to get active PR in: %-v Branch: %s to: %-v Branch: %s Error: %v", repo, branch, baseRepo, baseRepo.DefaultBranch, err),
					RepoWasEmpty: wasEmpty,
				})
				auditParams["error"] = "Error has occurred while getting active pull request"
				audit.CreateAndSendEvent(audit.ChangesPushEvent, opts.UserName, strconv.FormatInt(opts.UserID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
				return
			}

			if pr == nil {
				if repo.IsFork {
					branch = fmt.Sprintf("%s:%s", repo.OwnerName, branch)
					auditParams["branch_name"] = branch
				}
				results = append(results, private.HookPostReceiveBranchResult{
					Message: setting.Git.PullRequestPushMessage && baseRepo.AllowsPulls(),
					Create:  true,
					Branch:  branch,
					URL:     fmt.Sprintf("%s/compare/%s...%s", baseRepo.HTMLURL(), util.PathEscapeSegments(baseRepo.DefaultBranch), util.PathEscapeSegments(branch)),
				})
			} else {
				results = append(results, private.HookPostReceiveBranchResult{
					Message: setting.Git.PullRequestPushMessage && baseRepo.AllowsPulls(),
					Create:  false,
					Branch:  branch,
					URL:     fmt.Sprintf("%s/pulls/%d", baseRepo.HTMLURL(), pr.Index),
				})
			}
		}
	}

	if s.taskTrackerEnabled {
		if err := s.linkUnits(ctx, opts); err != nil {
			log.Debug("unit_linker: run: %v", err)
		}
	}

	dbEngine := db.GetEngine(ctx)
	customDb := custom.NewCustomDB(dbEngine)

	teamCustomPrivileges, err := customDb.GetCustomPrivilegesByBranchAndRepoID(ctx, "", repo.ID)
	if err != nil {
		log.Error("Error has occurred while getting custom privileges by branch and repo ID: %v", err)
		ctx.JSON(http.StatusInternalServerError, private.HookPostReceiveResult{
			Err:          fmt.Sprintf("Failed to get custom privileges by branch and repo ID: %v", err),
			RepoWasEmpty: wasEmpty,
		})
		auditParams["error"] = "Error has occurred while getting custom privileges"
		audit.CreateAndSendEvent(audit.ChangesPushEvent, opts.UserName, strconv.FormatInt(opts.UserID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)

		return
	}

	teams := make(map[string]struct{})
	policiesForCustomPrivileges := make([][]string, 0)
	for _, team := range teamCustomPrivileges {
		teams[team.TeamName] = struct{}{}
	}

	teamForUpdate := make([]custom.ScTeamCustomPrivilege, 0)
	for _, teamCustomPrivilege := range teamCustomPrivileges {
		if _, ok := teams[teamCustomPrivilege.TeamName]; !ok {
			teamForUpdate = append(teamForUpdate, custom.ScTeamCustomPrivilege{
				TeamName:         teamCustomPrivilege.TeamName,
				RepositoryID:     teamCustomPrivilege.RepositoryID,
				AllRepositories:  false,
				CustomPrivileges: teamCustomPrivilege.CustomPrivileges,
			})
		}

		policiesForCustomPrivileges = append(policiesForCustomPrivileges,
			[]string{teamCustomPrivilege.TeamName, strconv.FormatInt(repo.OwnerID, 10),
				strconv.FormatInt(repo.ID, 10),
				convertCustomPrivilegeToStringTo(teamCustomPrivilege.CustomPrivileges)})
		teams[teamCustomPrivilege.TeamName] = struct{}{}
	}

	// Подгрузка лицензий в БД SC
	gitalyRepo, err := git.OpenRepository(ctx, ownerName, repoName, repo.RepoPath())
	if err != nil {
		log.Error("Error has occurred while opening repository: %v", err)
		auditParams["error"] = "Error has occurred while opening repository"
		audit.CreateAndSendEvent(audit.ChangesPushEvent, opts.UserName, strconv.FormatInt(opts.UserID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}
	defer gitalyRepo.Close()

	commitID := opts.NewCommitIDs[0]
	fileLicensesInfo, err := services.GetLicensesInfoForRepo(gitalyRepo, commitID, repo.ID, repo.OwnerName, repo.DefaultBranch)
	if err != nil {
		log.Error("Error has occurred while getting licenses info: %v", err)
		auditParams["error"] = "Error has occurred while getting licenses info"
		audit.CreateAndSendEvent(audit.ChangesPushEvent, opts.UserName, strconv.FormatInt(opts.UserID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}

	if err := repo_model.UpsertInfoLicense(repo.ID, commitID, repo.DefaultBranch, fileLicensesInfo); err != nil {
		log.Error("Error has occurred while insert or update information about license for repo with ID '%d': %v", repo.ID, err)
		auditParams["error"] = "Error has occurred while updating licenses info"
		audit.CreateAndSendEvent(audit.ChangesPushEvent, opts.UserName, strconv.FormatInt(opts.UserID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}

	if opts.OldCommitIDs[0] == git.EmptySHA {
		if len(teamForUpdate) > 0 {
			if err := customDb.InsertCustomPrivilegesForTeam(teamForUpdate); err != nil {
				auditParams["error"] = "Error has occurred while inserting custom privileges"
				audit.CreateAndSendEvent(audit.ChangesPushEvent, opts.UserName, strconv.FormatInt(opts.UserID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
				log.Error("Error has occurred while inserting custom privileges for team: %v", err)
				ctx.JSON(http.StatusInternalServerError, private.HookPostReceiveResult{
					Err:          fmt.Sprintf("Failed to insert custom privileges for team: %v", err),
					RepoWasEmpty: wasEmpty,
				})
				return
			}
		}

		if err := s.repoRequestAccessor.UpdateCustomPrivileges(policiesForCustomPrivileges); err != nil {
			auditParams["error"] = "Error has occurred while inserting custom privileges"
			audit.CreateAndSendEvent(audit.ChangesPushEvent, opts.UserName, strconv.FormatInt(opts.UserID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			log.Error("Error has occurred while inserting custom privileges: %v", err)
			ctx.JSON(http.StatusInternalServerError, private.HookPostReceiveResult{
				Err:          fmt.Sprintf("Failed to insert custom privileges: %v", err),
				RepoWasEmpty: wasEmpty,
			})
			return
		}
	} else if opts.NewCommitIDs[0] == git.EmptySHA {
		if err := s.repoRequestAccessor.RemoveCustomPrivilegesByOldPrivileges(policiesForCustomPrivileges); err != nil {
			log.Error("Error has occurred while removing custom privileges: %v", err)
			auditParams["error"] = "Error has occurred while removing custom privileges"
			audit.CreateAndSendEvent(audit.ChangesPushEvent, opts.UserName, strconv.FormatInt(opts.UserID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			ctx.JSON(http.StatusInternalServerError, private.HookPostReceiveResult{
				Err:          fmt.Sprintf("Failed to remove custom privileges: %v", err),
				RepoWasEmpty: wasEmpty,
			})
			return
		}

		for _, t := range teamForUpdate {
			if err := customDb.DeleteCustomPrivilegesByParams(ctx, t); err != nil {
				log.Error("Error has occurred while deleting custom privileges by params: %v", err)
				auditParams["error"] = "Error has occurred while deleting custom privileges"
				audit.CreateAndSendEvent(audit.ChangesPushEvent, opts.UserName, strconv.FormatInt(opts.UserID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
				ctx.JSON(http.StatusInternalServerError, private.HookPostReceiveResult{
					Err:          fmt.Sprintf("Failed to delete custom privileges by params: %v", err),
					RepoWasEmpty: wasEmpty,
				})
				return
			}
		}
	}
	audit.CreateAndSendEvent(audit.ChangesPushEvent, opts.UserName, strconv.FormatInt(opts.UserID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)

	ctx.JSON(http.StatusOK, private.HookPostReceiveResult{
		Results:      results,
		RepoWasEmpty: wasEmpty,
	})
}

// convertCustomPrivilegeToStringTo конвертируем в название кастомной привилегии
func convertCustomPrivilegeToStringTo(customPrivileges string) string {
	arrayOfCustomPrivileges := strings.Split(customPrivileges, ",")
	convertCustomPrivileges := make([]role_model.CustomPrivilege, 0, len(arrayOfCustomPrivileges))
	for _, privilege := range arrayOfCustomPrivileges {
		if privilegeInt, err := strconv.Atoi(privilege); err != nil {
			log.Warn("Error has occurred while trying to convert string to int: %v", err)
			return ""
		} else {
			convertCustomPrivileges = append(convertCustomPrivileges, role_model.CustomPrivilege(privilegeInt))
		}
	}
	return role_model.ConvertCustomPrivilegeToNameOfPolicy(convertCustomPrivileges)
}
