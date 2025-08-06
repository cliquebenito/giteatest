// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package agit

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"code.gitea.io/gitea/modules/sbt/audit"

	issues_model "code.gitea.io/gitea/models/issues"
	repo_model "code.gitea.io/gitea/models/repo"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/notification"
	"code.gitea.io/gitea/modules/private"
	pull_service "code.gitea.io/gitea/services/pull"
)

// ProcReceive handle proc receive work
func ProcReceive(ctx context.Context, repo *repo_model.Repository, gitRepo *git.Repository, opts *private.HookOptions) ([]private.HookProcReceiveRefResult, error) {
	// TODO: Add more options?
	var (
		topicBranch string
		title       string
		description string
		forcePush   bool
	)

	results := make([]private.HookProcReceiveRefResult, 0, len(opts.OldCommitIDs))

	ownerName := repo.OwnerName
	repoName := repo.Name

	auditParams := map[string]string{
		"repository":    repoName,
		"repository_id": strconv.FormatInt(repo.ID, 10),
		"owner":         repo.OwnerName,
	}

	topicBranch = opts.GitPushOptions["topic"]
	_, forcePush = opts.GitPushOptions["force-push"]

	for i := range opts.OldCommitIDs {
		if opts.NewCommitIDs[i] == git.EmptySHA {
			results = append(results, private.HookProcReceiveRefResult{
				OriginalRef: opts.RefFullNames[i],
				OldOID:      opts.OldCommitIDs[i],
				NewOID:      opts.NewCommitIDs[i],
				Err:         "Can't delete not exist branch",
			})
			continue
		}

		if !strings.HasPrefix(opts.RefFullNames[i], git.PullRequestPrefix) {
			results = append(results, private.HookProcReceiveRefResult{
				IsNotMatched: true,
				OriginalRef:  opts.RefFullNames[i],
			})
			continue
		}

		baseBranchName := opts.RefFullNames[i][len(git.PullRequestPrefix):]
		curentTopicBranch := ""
		if !gitRepo.IsBranchExist(baseBranchName) {
			// try match refs/for/<target-branch>/<topic-branch>
			for p, v := range baseBranchName {
				if v == '/' && gitRepo.IsBranchExist(baseBranchName[:p]) && p != len(baseBranchName)-1 {
					curentTopicBranch = baseBranchName[p+1:]
					baseBranchName = baseBranchName[:p]
					break
				}
			}
		}

		if len(topicBranch) == 0 && len(curentTopicBranch) == 0 {
			results = append(results, private.HookProcReceiveRefResult{
				OriginalRef: opts.RefFullNames[i],
				OldOID:      opts.OldCommitIDs[i],
				NewOID:      opts.NewCommitIDs[i],
				Err:         "topic-branch is not set",
			})
			continue
		}

		var headBranch string
		userName := strings.ToLower(opts.UserName)

		if len(curentTopicBranch) == 0 {
			curentTopicBranch = topicBranch
		}

		// because different user maybe want to use same topic,
		// So it's better to make sure the topic branch name
		// has user name prefix
		if !strings.HasPrefix(curentTopicBranch, userName+"/") {
			headBranch = userName + "/" + curentTopicBranch
		} else {
			headBranch = curentTopicBranch
		}

		pr, err := issues_model.GetUnmergedPullRequest(ctx, repo.ID, repo.ID, headBranch, baseBranchName, issues_model.PullRequestFlowAGit)
		if err != nil {
			if !issues_model.IsErrPullRequestNotExist(err) {
				auditParams["error"] = "Failed to get unmerged agit flow pull request in repository"
				audit.CreateAndSendEvent(audit.ChangesPushEvent, opts.UserName, strconv.FormatInt(opts.UserID, 10), audit.StatusFailure, audit.EmptyRequiredField, auditParams)
				return nil, fmt.Errorf("Failed to get unmerged agit flow pull request in repository: %s/%s Error: %w", ownerName, repoName, err)
			}

			// create a new pull request
			if len(title) == 0 {
				var has bool
				title, has = opts.GitPushOptions["title"]
				if !has || len(title) == 0 {
					commit, err := gitRepo.GetCommit(opts.NewCommitIDs[i])
					if err != nil {
						auditParams["error"] = "Error has occurred while getting commit"
						audit.CreateAndSendEvent(audit.ChangesPushEvent, opts.UserName, strconv.FormatInt(opts.UserID, 10), audit.StatusFailure, audit.EmptyRequiredField, auditParams)
						return nil, fmt.Errorf("Failed to get commit %s in repository: %s/%s Error: %w", opts.NewCommitIDs[i], ownerName, repoName, err)
					}
					title = strings.Split(commit.CommitMessage, "\n")[0]
				}
				description = opts.GitPushOptions["description"]
			}

			pusher, err := user_model.GetUserByID(ctx, opts.UserID)
			if err != nil {
				auditParams["error"] = "Error has occurred while getting pusher by id"
				audit.CreateAndSendEvent(audit.ChangesPushEvent, opts.UserName, strconv.FormatInt(opts.UserID, 10), audit.StatusFailure, audit.EmptyRequiredField, auditParams)
				return nil, fmt.Errorf("Failed to get user. Error: %w", err)
			}

			prIssue := &issues_model.Issue{
				RepoID:   repo.ID,
				Title:    title,
				PosterID: pusher.ID,
				Poster:   pusher,
				IsPull:   true,
				Content:  description,
			}

			pr := &issues_model.PullRequest{
				HeadRepoID:   repo.ID,
				BaseRepoID:   repo.ID,
				HeadBranch:   headBranch,
				HeadCommitID: opts.NewCommitIDs[i],
				BaseBranch:   baseBranchName,
				HeadRepo:     repo,
				BaseRepo:     repo,
				MergeBase:    "",
				Type:         issues_model.PullRequestGitea,
				Flow:         issues_model.PullRequestFlowAGit,
			}

			if err := pull_service.NewPullRequest(ctx, repo, prIssue, []int64{}, []string{}, pr, []int64{}); err != nil {
				auditParams["error"] = "Error has occurred while creating pull request"
				audit.CreateAndSendEvent(audit.PRCreateEvent, opts.UserName, strconv.FormatInt(opts.UserID, 10), audit.StatusFailure, audit.EmptyRequiredField, auditParams)
				audit.CreateAndSendEvent(audit.ChangesPushEvent, opts.UserName, strconv.FormatInt(opts.UserID, 10), audit.StatusFailure, audit.EmptyRequiredField, auditParams)
				return nil, err
			}
			audit.CreateAndSendEvent(audit.PRCreateEvent, opts.UserName, strconv.FormatInt(opts.UserID, 10), audit.StatusSuccess, audit.EmptyRequiredField, auditParams)

			log.Trace("Pull request created: %d/%d", repo.ID, prIssue.ID)

			results = append(results, private.HookProcReceiveRefResult{
				Ref:         pr.GetGitRefName(),
				OriginalRef: opts.RefFullNames[i],
				OldOID:      git.EmptySHA,
				NewOID:      opts.NewCommitIDs[i],
			})
			continue
		}
		auditParams["pr_number"] = strconv.FormatInt(pr.ID, 10)

		// update exist pull request
		if err := pr.LoadBaseRepo(ctx); err != nil {
			auditParams["error"] = "Error has occurred while loading repository"
			audit.CreateAndSendEvent(audit.ChangesPushEvent, opts.UserName, strconv.FormatInt(opts.UserID, 10), audit.StatusFailure, audit.EmptyRequiredField, auditParams)
			return nil, fmt.Errorf("Unable to load base repository for PR[%d] Error: %w", pr.ID, err)
		}

		oldCommitID, err := gitRepo.GetRefCommitID(pr.GetGitRefName())
		if err != nil {
			auditParams["error"] = "Error has occurred while getting ref commit id in base repository"
			audit.CreateAndSendEvent(audit.ChangesPushEvent, opts.UserName, strconv.FormatInt(opts.UserID, 10), audit.StatusFailure, audit.EmptyRequiredField, auditParams)
			return nil, fmt.Errorf("Unable to get ref commit id in base repository for PR[%d] Error: %w", pr.ID, err)
		}

		if oldCommitID == opts.NewCommitIDs[i] {
			results = append(results, private.HookProcReceiveRefResult{
				OriginalRef: opts.RefFullNames[i],
				OldOID:      opts.OldCommitIDs[i],
				NewOID:      opts.NewCommitIDs[i],
				Err:         "new commit is same with old commit",
			})
			continue
		}

		if !forcePush {
			output, _, err := git.NewCommand(ctx, "rev-list", "--max-count=1").AddDynamicArguments(oldCommitID, "^"+opts.NewCommitIDs[i]).RunStdString(&git.RunOpts{Dir: repo.RepoPath(), Env: os.Environ()})
			if err != nil {
				auditParams["error"] = "Error has occurred while detecting force push"
				audit.CreateAndSendEvent(audit.ChangesPushEvent, opts.UserName, strconv.FormatInt(opts.UserID, 10), audit.StatusFailure, audit.EmptyRequiredField, auditParams)
				return nil, fmt.Errorf("Fail to detect force push: %w", err)
			} else if len(output) > 0 {
				results = append(results, private.HookProcReceiveRefResult{
					OriginalRef: opts.RefFullNames[i],
					OldOID:      opts.OldCommitIDs[i],
					NewOID:      opts.NewCommitIDs[i],
					Err:         "request `force-push` push option",
				})
				continue
			}
		}

		pr.HeadCommitID = opts.NewCommitIDs[i]
		if err = pull_service.UpdateRef(ctx, pr); err != nil {
			auditParams["error"] = "Error has occurred while updating pull ref"
			audit.CreateAndSendEvent(audit.ChangesPushEvent, opts.UserName, strconv.FormatInt(opts.UserID, 10), audit.StatusFailure, audit.EmptyRequiredField, auditParams)
			return nil, fmt.Errorf("Failed to update pull ref. Error: %w", err)
		}

		pull_service.AddToTaskQueue(pr)
		pusher, err := user_model.GetUserByID(ctx, opts.UserID)
		if err != nil {
			auditParams["error"] = "Error has occurred while getting user by id"
			audit.CreateAndSendEvent(audit.ChangesPushEvent, opts.UserName, strconv.FormatInt(opts.UserID, 10), audit.StatusFailure, audit.EmptyRequiredField, auditParams)
			return nil, fmt.Errorf("Failed to get user. Error: %w", err)
		}
		err = pr.LoadIssue(ctx)
		if err != nil {
			auditParams["error"] = "Error has occurred while loading issue"
			audit.CreateAndSendEvent(audit.ChangesPushEvent, opts.UserName, strconv.FormatInt(opts.UserID, 10), audit.StatusFailure, audit.EmptyRequiredField, auditParams)
			return nil, fmt.Errorf("Failed to load pull issue. Error: %w", err)
		}
		comment, err := pull_service.CreatePushPullComment(ctx, pusher, pr, oldCommitID, opts.NewCommitIDs[i])
		if err == nil && comment != nil {
			notification.NotifyPullRequestPushCommits(ctx, pusher, pr, comment)
		}
		err = pr.UpdateReferenceForRequest(ctx, oldCommitID)
		if err != nil {
			log.Error("Error has occurred while updating pull request reference for request: %v", err)
			return nil, fmt.Errorf("failed to update pull request reference for request: %w", err)
		}
		notification.NotifyPullRequestSynchronized(ctx, pusher, pr)
		isForcePush := comment != nil && comment.IsForcePush

		results = append(results, private.HookProcReceiveRefResult{
			OldOID:      oldCommitID,
			NewOID:      opts.NewCommitIDs[i],
			Ref:         pr.GetGitRefName(),
			OriginalRef: opts.RefFullNames[i],
			IsForcePush: isForcePush,
		})
	}

	return results, nil
}

// UserNameChanged handle user name change for agit flow pull
func UserNameChanged(ctx context.Context, user *user_model.User, newName string) error {
	pulls, err := issues_model.GetAllUnmergedAgitPullRequestByPoster(ctx, user.ID)
	if err != nil {
		return err
	}

	newName = strings.ToLower(newName)

	for _, pull := range pulls {
		pull.HeadBranch = strings.TrimPrefix(pull.HeadBranch, user.LowerName+"/")
		pull.HeadBranch = newName + "/" + pull.HeadBranch
		if err = pull.UpdateCols("head_branch"); err != nil {
			return err
		}
	}

	return nil
}
