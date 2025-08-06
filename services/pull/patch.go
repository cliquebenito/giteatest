// Copyright 2019 The Gitea Authors.
// All rights reserved.
// SPDX-License-Identifier: MIT

package pull

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gitlab.com/gitlab-org/gitaly/v16/proto/go/gitalypb"
	"google.golang.org/grpc/status"

	"code.gitea.io/gitea/models"
	git_model "code.gitea.io/gitea/models/git"
	issues_model "code.gitea.io/gitea/models/issues"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/graceful"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/process"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/util"

	"github.com/gobwas/glob"
)

// DownloadDiffOrPatch will write the patch for the pr to the writer
func DownloadDiffOrPatch(ctx context.Context, pr *issues_model.PullRequest, w io.Writer, patch, binary bool) error {
	if err := pr.LoadBaseRepo(ctx); err != nil {
		log.Error("Unable to load base repository ID %d for pr #%d [%d]", pr.BaseRepoID, pr.Index, pr.ID)
		return err
	}

	gitRepo, closer, err := git.RepositoryFromContextOrOpen(ctx, pr.BaseRepo.OwnerName, pr.BaseRepo.Name, pr.BaseRepo.RepoPath())
	if err != nil {
		return fmt.Errorf("OpenRepository: %w", err)
	}
	defer closer.Close()

	if err := gitRepo.GetDiffOrPatch(pr.MergeBase, pr.GetGitRefName(), w, patch, binary); err != nil {
		log.Error("Unable to get patch file from %s to %s in %s Error: %v", pr.MergeBase, pr.HeadBranch, pr.BaseRepo.FullName(), err)
		return fmt.Errorf("Unable to get patch file from %s to %s in %s Error: %w", pr.MergeBase, pr.HeadBranch, pr.BaseRepo.FullName(), err)
	}
	return nil
}

var patchErrorSuffices = []string{
	": already exists in index",
	": patch does not apply",
	": already exists in working directory",
	"unrecognized input",
	": No such file or directory",
	": does not exist in index",
}

// TestPatch will test whether a simple patch will apply
func TestPatch(pr *issues_model.PullRequest) error {
	ctx, _, finished := process.GetManager().AddContext(graceful.GetManager().HammerContext(), fmt.Sprintf("TestPatch: %s", pr))
	defer finished()

	gitRepo, err := git.OpenRepository(ctx, pr.BaseRepo.OwnerName, pr.BaseRepo.Name, pr.BaseRepo.RepoPath())
	if err != nil {
		return fmt.Errorf("OpenRepository: %w", err)
	}
	defer gitRepo.Close()

	revisions := make([][]byte, 0)
	revisions = append(revisions, []byte(pr.BaseBranch))
	revisions = append(revisions, []byte(pr.HeadBranch))

	// 1. update merge base
	mergeBase, err := gitRepo.RepoClient.FindMergeBase(gitRepo.Ctx, &gitalypb.FindMergeBaseRequest{
		Repository: gitRepo.GitalyRepo,
		Revisions:  revisions,
	})
	if err != nil {
		return err
	}
	pr.MergeBase = mergeBase.GetBase()

	if pr.HeadCommitID == pr.MergeBase {
		pr.Status = issues_model.PullRequestStatusAncestor
		return nil
	}

	// 2. Check for conflicts
	conflicts, err := checkConflicts(ctx, pr, gitRepo, pr.BaseRepo.RepoPath())
	if err != nil {
		log.Error("Error has occurred while checking conflicts pr: %v", err)
		return fmt.Errorf("checking conflicts pr: %w", err)
	}

	// 4. Check for protected files changes
	if err = checkPullFilesProtection(ctx, pr, gitRepo); err != nil {
		log.Error("Error has occurred while checking pull files protection: %v", err)
		return fmt.Errorf("checking pull files protection: %v", err)
	}

	if len(pr.ChangedProtectedFiles) > 0 {
		log.Trace("Found %d protected files changed", len(pr.ChangedProtectedFiles))
	}

	if !conflicts && pr.Status != issues_model.PullRequestStatusEmpty {
		pr.Status = issues_model.PullRequestStatusMergeable
	}

	return nil
}

type errMergeConflict struct {
	filename string
}

func (e *errMergeConflict) Error() string {
	return fmt.Sprintf("conflict detected at: %s", e.filename)
}

func attemptMerge(ctx context.Context, file *unmergedFile, tmpBasePath string, filesToRemove *[]string, filesToAdd *[]git.IndexObjectInfo) error {
	log.Trace("Attempt to merge:\n%v", file)

	switch {
	case file.stage1 != nil && (file.stage2 == nil || file.stage3 == nil):
		// 1. Deleted in one or both:
		//
		// Conflict <==> the stage1 !SameAs to the undeleted one
		if (file.stage2 != nil && !file.stage1.SameAs(file.stage2)) || (file.stage3 != nil && !file.stage1.SameAs(file.stage3)) {
			// Conflict!
			return &errMergeConflict{file.stage1.path}
		}

		// Not a genuine conflict and we can simply remove the file from the index
		*filesToRemove = append(*filesToRemove, file.stage1.path)
		return nil
	case file.stage1 == nil && file.stage2 != nil && (file.stage3 == nil || file.stage2.SameAs(file.stage3)):
		// 2. Added in ours but not in theirs or identical in both
		//
		// Not a genuine conflict just add to the index
		*filesToAdd = append(*filesToAdd, git.IndexObjectInfo{Mode: file.stage2.mode, Object: git.MustIDFromString(file.stage2.sha), Filename: file.stage2.path})
		return nil
	case file.stage1 == nil && file.stage2 != nil && file.stage3 != nil && file.stage2.sha == file.stage3.sha && file.stage2.mode != file.stage3.mode:
		// 3. Added in both with the same sha but the modes are different
		//
		// Conflict! (Not sure that this can actually happen but we should handle)
		return &errMergeConflict{file.stage2.path}
	case file.stage1 == nil && file.stage2 == nil && file.stage3 != nil:
		// 4. Added in theirs but not ours:
		//
		// Not a genuine conflict just add to the index
		*filesToAdd = append(*filesToAdd, git.IndexObjectInfo{Mode: file.stage3.mode, Object: git.MustIDFromString(file.stage3.sha), Filename: file.stage3.path})
		return nil
	case file.stage1 == nil:
		// 5. Created by new in both
		//
		// Conflict!
		return &errMergeConflict{file.stage2.path}
	case file.stage2 != nil && file.stage3 != nil:
		// 5. Modified in both - we should try to merge in the changes but first:
		//
		if file.stage2.mode == "120000" || file.stage3.mode == "120000" {
			// 5a. Conflicting symbolic link change
			return &errMergeConflict{file.stage2.path}
		}
		if file.stage2.mode == "160000" || file.stage3.mode == "160000" {
			// 5b. Conflicting submodule change
			return &errMergeConflict{file.stage2.path}
		}
		if file.stage2.mode != file.stage3.mode {
			// 5c. Conflicting mode change
			return &errMergeConflict{file.stage2.path}
		}

		// Need to get the objects from the object db to attempt to merge
		root, _, err := git.NewCommand(ctx, "unpack-file").AddDynamicArguments(file.stage1.sha).RunStdString(&git.RunOpts{Dir: tmpBasePath})
		if err != nil {
			return fmt.Errorf("unable to get root object: %s at path: %s for merging. Error: %w", file.stage1.sha, file.stage1.path, err)
		}
		root = strings.TrimSpace(root)
		defer func() {
			_ = util.Remove(filepath.Join(tmpBasePath, root))
		}()

		base, _, err := git.NewCommand(ctx, "unpack-file").AddDynamicArguments(file.stage2.sha).RunStdString(&git.RunOpts{Dir: tmpBasePath})
		if err != nil {
			return fmt.Errorf("unable to get base object: %s at path: %s for merging. Error: %w", file.stage2.sha, file.stage2.path, err)
		}
		base = strings.TrimSpace(filepath.Join(tmpBasePath, base))
		defer func() {
			_ = util.Remove(base)
		}()
		head, _, err := git.NewCommand(ctx, "unpack-file").AddDynamicArguments(file.stage3.sha).RunStdString(&git.RunOpts{Dir: tmpBasePath})
		if err != nil {
			return fmt.Errorf("unable to get head object:%s at path: %s for merging. Error: %w", file.stage3.sha, file.stage3.path, err)
		}
		head = strings.TrimSpace(head)
		defer func() {
			_ = util.Remove(filepath.Join(tmpBasePath, head))
		}()

		// now git merge-file annoyingly takes a different order to the merge-tree ...
		_, _, conflictErr := git.NewCommand(ctx, "merge-file").AddDynamicArguments(base, root, head).RunStdString(&git.RunOpts{Dir: tmpBasePath})
		if conflictErr != nil {
			return &errMergeConflict{file.stage2.path}
		}

		// base now contains the merged data
		hash, _, err := git.NewCommand(ctx, "hash-object", "-w", "--path").AddDynamicArguments(file.stage2.path, base).RunStdString(&git.RunOpts{Dir: tmpBasePath})
		if err != nil {
			return err
		}
		hash = strings.TrimSpace(hash)
		*filesToAdd = append(*filesToAdd, git.IndexObjectInfo{Mode: file.stage2.mode, Object: git.MustIDFromString(hash), Filename: file.stage2.path})
		return nil
	default:
		if file.stage1 != nil {
			return &errMergeConflict{file.stage1.path}
		} else if file.stage2 != nil {
			return &errMergeConflict{file.stage2.path}
		} else if file.stage3 != nil {
			return &errMergeConflict{file.stage3.path}
		}
	}
	return nil
}

// AttemptThreeWayMerge will attempt to three way merge using git read-tree and then follow the git merge-one-file algorithm to attempt to resolve basic conflicts
func AttemptThreeWayMerge(ctx context.Context, gitPath string, gitRepo *git.Repository, base, ours, theirs, description string) (bool, []string, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// First we use read-tree to do a simple three-way merge
	if _, _, err := git.NewCommand(ctx, "read-tree", "-m").AddDynamicArguments(base, ours, theirs).RunStdString(&git.RunOpts{Dir: gitPath}); err != nil {
		log.Error("Unable to run read-tree -m! Error: %v", err)
		return false, nil, fmt.Errorf("unable to run read-tree -m! Error: %w", err)
	}

	var filesToRemove []string
	var filesToAdd []git.IndexObjectInfo

	// Then we use git ls-files -u to list the unmerged files and collate the triples in unmergedfiles
	unmerged := make(chan *unmergedFile)
	go unmergedFiles(ctx, gitPath, unmerged)

	defer func() {
		cancel()
		for range unmerged {
			// empty the unmerged channel
		}
	}()

	numberOfConflicts := 0
	conflict := false
	conflictedFiles := make([]string, 0, 5)

	for file := range unmerged {
		if file == nil {
			break
		}
		if file.err != nil {
			cancel()
			return false, nil, file.err
		}

		// OK now we have the unmerged file triplet attempt to merge it
		if err := attemptMerge(ctx, file, gitPath, &filesToRemove, &filesToAdd); err != nil {
			if conflictErr, ok := err.(*errMergeConflict); ok {
				log.Trace("Conflict: %s in %s", conflictErr.filename, description)
				conflict = true
				if numberOfConflicts < 10 {
					conflictedFiles = append(conflictedFiles, conflictErr.filename)
				}
				numberOfConflicts++
				continue
			}
			return false, nil, err
		}
	}

	// Add and remove files in one command, as this is slow with many files otherwise
	if err := gitRepo.RemoveFilesFromIndex(filesToRemove...); err != nil {
		return false, nil, err
	}
	if err := gitRepo.AddObjectsToIndex(filesToAdd...); err != nil {
		return false, nil, err
	}

	return conflict, conflictedFiles, nil
}

func findConflicts(ctx context.Context, pr *issues_model.PullRequest, gitRepo *git.Repository) (bool, []string, error) {
	conflictedFiles := make([]string, 0, 5)

	ctxWithCancel, cancel := context.WithCancel(gitRepo.Ctx)
	defer cancel()
	if pr.HeadCommitID == "" {
		listAllCommit, err := gitRepo.CommitClient.ListCommits(ctxWithCancel, &gitalypb.ListCommitsRequest{
			Repository: gitRepo.GitalyRepo,
			Revisions:  []string{pr.HeadBranch},
		})

		if err != nil {
			return false, nil, fmt.Errorf("last commits not found for branch: '%s' error: %v", pr.HeadBranch, err)
		}

		for {
			listCommit, err := listAllCommit.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				return false, nil, fmt.Errorf("error last commits not found for branch: %v", err)
			}
			if len(listCommit.Commits) == 0 {
				return false, nil, fmt.Errorf("commit not found for branch: '%s'", pr.HeadBranch)
			}
			pr.HeadCommitID = listCommit.Commits[0].Id
			break
		}
	}

	mergeClient, err := gitRepo.OperationClient.UserMergeBranch(ctxWithCancel)
	if err != nil {
		return false, conflictedFiles, err
	}
	req := &gitalypb.UserMergeBranchRequest{
		Repository: gitRepo.GitalyRepo,
		User: &gitalypb.User{
			GlId:       strconv.FormatInt(0, 10),
			Name:       []byte("find_conflict"),
			Email:      []byte("find_conflict"),
			GlUsername: "find_conflict",
		},
		CommitId: pr.HeadCommitID,
		Branch:   []byte(pr.BaseBranch),
		Message:  []byte("find_conflict"),
		Apply:    false,
	}

	err = mergeClient.Send(req)
	if err != nil {
		return false, conflictedFiles, err
	}

	_, err = mergeClient.Recv()
	if err != nil {
		if mergeStatusError, ok := status.FromError(err); ok && strings.Contains(mergeStatusError.Message(), "conflicting files") {
			conflictFilesBytes := make([][]byte, 0)
			for _, details := range mergeStatusError.Details() {
				mergeErr := details.(*gitalypb.UserMergeBranchError)
				if mergeErr.GetMergeConflict() != nil {
					conflictFilesBytes = append(conflictFilesBytes, mergeErr.GetMergeConflict().GetConflictingFiles()...)
				}
			}

			for _, file := range conflictFilesBytes {
				conflictedFiles = append(conflictedFiles, string(file))
			}
			return true, conflictedFiles, nil
		}
		return false, conflictedFiles, err
	}

	return false, conflictedFiles, nil
}

func checkConflicts(ctx context.Context, pr *issues_model.PullRequest, gitRepo *git.Repository, tmpBasePath string) (bool, error) {
	// 1. checkConflicts resets the conflict status - therefore - reset the conflict status
	pr.ConflictedFiles = nil

	conflict, conflictFiles, err := findConflicts(ctx, pr, gitRepo)
	if err != nil {
		return false, err
	}

	if !conflict {
		// No conflicts detected so we need to check if the patch is empty...

		rawPatchClient, err := gitRepo.DiffClient.RawRangeDiff(gitRepo.Ctx, &gitalypb.RawRangeDiffRequest{
			Repository: gitRepo.GitalyRepo,
			RangeSpec: &gitalypb.RawRangeDiffRequest_RevisionRange{
				RevisionRange: &gitalypb.RevisionRange{
					Rev1: pr.BaseBranch,
					Rev2: pr.HeadBranch,
				},
			},
		})
		if err != nil {
			return false, err
		}
		patchResponse, err := rawPatchClient.Recv()
		if err != nil && err != io.EOF {
			return false, err
		}

		if err == io.EOF || patchResponse == nil || patchResponse.Data == nil {
			log.Debug("PullRequest[%d]: Patch is empty - ignoring", pr.ID)
			pr.Status = issues_model.PullRequestStatusEmpty
		}

		return false, nil
	}

	// 3. OK the three-way merge method has detected conflicts
	// 3a. Are still testing with GitApply? If not set the conflict status and move on
	if !setting.Repository.PullRequest.TestConflictingPatchesWithGitApply {
		pr.Status = issues_model.PullRequestStatusConflict
		pr.ConflictedFiles = conflictFiles

		log.Trace("Found %d files conflicted: %v", len(pr.ConflictedFiles), pr.ConflictedFiles)
		return true, nil
	}
	return false, nil
}

// CheckFileProtection check file Protection
func CheckFileProtection(repo *git.Repository, oldCommitID, newCommitID string, patterns []glob.Glob, limit int, env []string) ([]string, error) {
	if len(patterns) == 0 {
		return nil, nil
	}
	affectedFiles, err := git.GetAffectedFiles(repo, oldCommitID, newCommitID, env)
	if err != nil {
		return nil, err
	}
	changedProtectedFiles := make([]string, 0, limit)
	for _, affectedFile := range affectedFiles {
		lpath := strings.ToLower(affectedFile)
		for _, pat := range patterns {
			if pat.Match(lpath) {
				changedProtectedFiles = append(changedProtectedFiles, lpath)
				break
			}
		}
		if len(changedProtectedFiles) >= limit {
			break
		}
	}
	if len(changedProtectedFiles) > 0 {
		err = models.ErrFilePathProtected{
			Path: changedProtectedFiles[0],
		}
	}
	return changedProtectedFiles, err
}

// CheckUnprotectedFiles check if the commit only touches unprotected files
func CheckUnprotectedFiles(repo *git.Repository, oldCommitID, newCommitID string, patterns []glob.Glob, env []string) (bool, error) {
	if len(patterns) == 0 {
		return false, nil
	}
	affectedFiles, err := git.GetAffectedFiles(repo, oldCommitID, newCommitID, env)
	if err != nil {
		return false, err
	}
	for _, affectedFile := range affectedFiles {
		lpath := strings.ToLower(affectedFile)
		unprotected := false
		for _, pat := range patterns {
			if pat.Match(lpath) {
				unprotected = true
				break
			}
		}
		if !unprotected {
			return false, nil
		}
	}
	return true, nil
}

// checkPullFilesProtection check if pr changed protected files and save results
func checkPullFilesProtection(ctx context.Context, pr *issues_model.PullRequest, gitRepo *git.Repository) error {
	if pr.Status == issues_model.PullRequestStatusEmpty {
		pr.ChangedProtectedFiles = nil
		return nil
	}

	pb, err := git_model.GetMergeMatchProtectedBranchRule(ctx, pr.BaseRepoID, pr.BaseBranch)
	if err != nil {
		log.Error("Error has occured while get merge match protected branch with repoID - %d, branchName - %s", pr.BaseRepoID, pr.BaseBranch)
		return fmt.Errorf("Err: get merge protected branch: %w", err)
	}

	if pb == nil {
		pr.ChangedProtectedFiles = nil
		return nil
	}

	pr.ChangedProtectedFiles, err = CheckFileProtection(gitRepo, pr.MergeBase, "tracking", git_model.GetProtectedFilePatterns(*pb), 10, os.Environ())
	if err != nil && !models.IsErrFilePathProtected(err) {
		return err
	}
	return nil
}
