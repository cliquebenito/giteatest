// Copyright 2015 The Gogs Authors. All rights reserved.
// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package git

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"gitlab.com/gitlab-org/gitaly/v16/proto/go/gitalypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"code.gitea.io/gitea/modules/cache"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
)

// GetBranchCommitID returns last commit ID string of given branch.
func (repo *Repository) GetBranchCommitID(name string) (string, error) {
	return repo.GetRefCommitID(BranchPrefix + name)
}

// GetTagCommitID returns last commit ID string of given tag.
func (repo *Repository) GetTagCommitID(name string) (string, error) {
	return repo.GetRefCommitID(TagPrefix + name)
}

// GetCommit returns commit object of by ID string.
func (repo *Repository) GetCommit(commitID string) (*Commit, error) {
	return repo.getCommit(commitID)
}

// GetBranchCommit returns the last commit of given branch.
func (repo *Repository) GetBranchCommit(name string) (*Commit, error) {
	commitID, err := repo.GetBranchCommitID(name)
	if err != nil {
		log.Error("Error has occurred while getting last commit ID for branch: %v", err)
		return nil, fmt.Errorf("get last commit ID for branch: %w", err)
	}

	return repo.GetCommit(commitID)
}

// GetTagCommit get the commit of the specific tag via name
func (repo *Repository) GetTagCommit(name string) (*Commit, error) {
	commitID, err := repo.GetTagCommitID(name)
	if err != nil {
		return nil, err
	}
	return repo.GetCommit(commitID)
}

func (repo *Repository) getCommitByPathWithID(id SHA1, relpath string) (*Commit, error) {
	// File name starts with ':' must be escaped.
	if relpath[0] == ':' {
		relpath = `\` + relpath
	}

	stdout, _, runErr := NewCommand(repo.Ctx, "log", "-1", prettyLogFormat).AddDynamicArguments(id.String()).AddDashesAndList(relpath).RunStdString(&RunOpts{Dir: repo.Path})
	if runErr != nil {
		return nil, runErr
	}

	id, err := NewIDFromString(stdout)
	if err != nil {
		return nil, err
	}

	return repo.getCommit(id.String())
}

// GetCommitByPath returns the last commit of relative path.
func (repo *Repository) GetCommitByPath(relpath string) (*Commit, error) {
	stdout, _, runErr := NewCommand(repo.Ctx, "log", "-1", prettyLogFormat).AddDashesAndList(relpath).RunStdBytes(&RunOpts{Dir: repo.Path})
	if runErr != nil {
		return nil, runErr
	}

	commits, err := repo.parsePrettyFormatLogToList(stdout)
	if err != nil {
		return nil, err
	}
	if len(commits) == 0 {
		return nil, ErrNotExist{ID: relpath}
	}
	return commits[0], nil
}

func (repo *Repository) commitsByRange(id SHA1, page, pageSize int, not string) ([]*Commit, error) {
	ctx, cancel := context.WithCancel(repo.Ctx)
	defer cancel()

	commits := make([]*Commit, 0)
	commitsStream, err1 := repo.CommitClient.FindAllCommits(ctx, &gitalypb.FindAllCommitsRequest{
		Repository: repo.GitalyRepo,
		Revision:   id.Byte(),
		MaxCount:   int32(pageSize),
		Skip:       int32((page - 1) * pageSize),
	})
	if err1 != nil {
		return nil, err1
	}

	canRead := true
	for canRead {
		commitsResp, err := commitsStream.Recv()
		if err != nil && err != io.EOF {
			return nil, err
		}
		if commitsResp == nil {
			canRead = false
		} else {
			comm, err := repo.ParseGitCommitsToCommit(commitsResp.Commits)
			if err != nil {
				return nil, err
			}
			commits = append(commits, comm...)
		}
	}

	return commits, nil
}

func (repo *Repository) SearchCommits(id string, opts SearchCommitsOptions) ([]*Commit, error) {
	request := &gitalypb.FindCommitsRequest{
		Repository: repo.GitalyRepo,
		Revision:   []byte(id),
		Limit:      -1,
		All:        opts.All,
	}

	var afterDate, beforeDate time.Time
	var err error
	if opts.After != "" {
		afterDate, err = time.Parse("2006-01-02", opts.After)
		if err != nil {
			return nil, err
		}
		request.After = timestamppb.New(afterDate)
	}
	if opts.Before != "" {
		beforeDate, err = time.Parse("2006-01-02", opts.Before)
		if err != nil {
			return nil, err
		}
		request.Before = timestamppb.New(beforeDate)
	}

	authors := make([]string, 0)
	authors = append(authors, opts.Authors...)
	authors = append(authors, opts.Committers...)
	gitCommits := make([]*gitalypb.GitCommit, 0)

	ctx, cancel := context.WithCancel(repo.Ctx)
	defer cancel()
	if len(authors) > 0 {
		for _, author := range authors {
			request.Author = []byte(author)
			findCommitsClient, err := repo.CommitClient.FindCommits(ctx, request)
			if err != nil {
				continue
			}
			commitsResp, err := findCommitsClient.Recv()
			if err != nil {
				continue
			}
			gitCommits = append(gitCommits, commitsResp.Commits...)
		}
	} else {
		if findCommitsClient, err := repo.CommitClient.FindCommits(ctx, request); err == nil {
			if commitsResp, err := findCommitsClient.Recv(); err == nil {
				gitCommits = append(gitCommits, commitsResp.Commits...)
			}
		}
	}

	if opts.Sha != "" {
		var filteredCommits []*gitalypb.GitCommit
		for _, c := range gitCommits {
			if strings.HasPrefix(c.Id, opts.Sha) {
				filteredCommits = append(filteredCommits, c)
			}
		}
		gitCommits = filteredCommits
	}

	commits, err := repo.ParseGitCommitsToCommit(gitCommits)
	if err != nil {
		return nil, err
	}
	return commits, nil
}

// FileChangedBetweenCommits Returns true if the file changed between commit IDs id1 and id2
// You must ensure that id1 and id2 are valid commit ids.
func (repo *Repository) FileChangedBetweenCommits(filename, id1, id2 string) (bool, error) {
	stdout, _, err := NewCommand(repo.Ctx, "diff", "--name-only", "-z").AddDynamicArguments(id1, id2).AddDashesAndList(filename).RunStdBytes(&RunOpts{Dir: repo.Path})
	if err != nil {
		return false, err
	}
	return len(strings.TrimSpace(string(stdout))) > 0, nil
}

// FileCommitsCount return the number of files at a revision
func (repo *Repository) FileCommitsCount(revision, file string) (int64, error) {
	return CommitsCount(repo.Ctx,
		CommitsCountOptions{
			RepoPath:     repo.Path,
			Revision:     []string{revision},
			RelPath:      []string{file},
			GitalyRepo:   repo.GitalyRepo,
			Ctx:          repo.Ctx,
			CommitClient: repo.CommitClient,
		})
}

type CommitsByFileAndRangeOptions struct {
	Revision string
	File     string
	Not      string
	Page     int
}

// CommitsByFileAndRange return the commits according revision file and the page
func (repo *Repository) CommitsByFileAndRange(opts CommitsByFileAndRangeOptions) ([]*Commit, error) {
	paths := make([][]byte, 0)
	paths = append(paths, []byte(opts.File))

	ctx, cancel := context.WithCancel(repo.Ctx)
	defer cancel()
	findCommitsClient, err := repo.CommitClient.FindCommits(ctx, &gitalypb.FindCommitsRequest{
		Repository: repo.GitalyRepo,
		Revision:   []byte(opts.Revision),
		Limit:      int32(setting.Git.CommitsRangeSize),
		Paths:      paths,
	})
	if err != nil {
		return nil, err
	}
	commitsResp, err := findCommitsClient.Recv()
	if err != nil && err != io.EOF {
		return nil, err
	}

	commits, err := repo.ParseGitCommitsToCommit(commitsResp.Commits)
	if err != nil {
		return nil, err
	}
	return commits, nil
}

// FilesCountBetween return the number of files changed between two commits
func (repo *Repository) FilesCountBetween(startCommitID, endCommitID string) (int, error) {
	stdout, _, err := NewCommand(repo.Ctx, "diff", "--name-only").AddDynamicArguments(startCommitID + "..." + endCommitID).RunStdString(&RunOpts{Dir: repo.Path})
	if err != nil && strings.Contains(err.Error(), "no merge base") {
		// git >= 2.28 now returns an error if startCommitID and endCommitID have become unrelated.
		// previously it would return the results of git diff --name-only startCommitID endCommitID so let's try that...
		stdout, _, err = NewCommand(repo.Ctx, "diff", "--name-only").AddDynamicArguments(startCommitID, endCommitID).RunStdString(&RunOpts{Dir: repo.Path})
	}
	if err != nil {
		return 0, err
	}
	return len(strings.Split(stdout, "\n")) - 1, nil
}

// CommitsBetween returns a list that contains commits between [before, last).
// If before is detached (removed by reset + push) it is not included.
func (repo *Repository) CommitsBetween(last, before *Commit) ([]*Commit, error) {
	revisions := make([]string, 0)
	if before != nil {
		revisions = append(revisions, "^"+before.ID.String())
	}

	revisions = append(revisions, last.ID.String())
	commits := make([]*Commit, 0)

	ctx, cancel := context.WithCancel(repo.Ctx)
	defer cancel()
	commitsClient, err := repo.CommitClient.ListCommits(ctx, &gitalypb.ListCommitsRequest{
		Repository: repo.GitalyRepo,
		Revisions:  revisions,
	})
	if err != nil {
		return nil, err
	}

	canRead := true
	for canRead {
		commitsResp, err := commitsClient.Recv()
		if err != nil && err != io.EOF {
			return nil, err
		}
		if commitsResp == nil {
			canRead = false
		} else {
			comm, err := repo.ParseGitCommitsToCommit(commitsResp.Commits)
			if err != nil {
				return nil, err
			}
			commits = append(commits, comm...)
		}
	}

	return commits, nil
}

// CommitsBetweenLimit returns a list that contains at most limit commits skipping the first skip commits between [before, last)
func (repo *Repository) CommitsBetweenLimit(last, before *Commit, limit, skip int) ([]*Commit, error) {
	revision := []byte("")
	if before != nil {
		revision = before.ID.Byte()
	}

	ctx, cancel := context.WithCancel(repo.Ctx)
	defer cancel()

	commits := make([]*Commit, 0)
	commitsClient, err := repo.CommitClient.FindAllCommits(ctx, &gitalypb.FindAllCommitsRequest{
		Repository: repo.GitalyRepo,
		Revision:   revision,
		MaxCount:   int32(limit),
		Skip:       int32(skip),
	})
	if err != nil {
		return nil, err
	}

	canRead := true
	for canRead {
		commitsResp, err := commitsClient.Recv()
		if err != nil && err != io.EOF {
			return nil, err
		}
		if commitsResp == nil {
			canRead = false
		} else {
			comm, err := repo.ParseGitCommitsToCommit(commitsResp.Commits)
			if err != nil {
				return nil, err
			}
			commits = append(commits, comm...)
		}
	}

	return commits, nil
}

// CommitsBetweenNotBase returns a list that contains commits between [before, last), excluding commits in baseBranch.
// If before is detached (removed by reset + push) it is not included.
func (repo *Repository) CommitsBetweenNotBase(last, before *Commit, baseBranch string) ([]*Commit, error) {
	var stdout []byte
	var err error
	if before == nil {
		stdout, _, err = NewCommand(repo.Ctx, "rev-list").AddDynamicArguments(last.ID.String()).AddOptionValues("--not", baseBranch).RunStdBytes(&RunOpts{Dir: repo.Path})
	} else {
		stdout, _, err = NewCommand(repo.Ctx, "rev-list").AddDynamicArguments(before.ID.String()+".."+last.ID.String()).AddOptionValues("--not", baseBranch).RunStdBytes(&RunOpts{Dir: repo.Path})
		if err != nil && strings.Contains(err.Error(), "no merge base") {
			// future versions of git >= 2.28 are likely to return an error if before and last have become unrelated.
			// previously it would return the results of git rev-list before last so let's try that...
			stdout, _, err = NewCommand(repo.Ctx, "rev-list").AddDynamicArguments(before.ID.String(), last.ID.String()).AddOptionValues("--not", baseBranch).RunStdBytes(&RunOpts{Dir: repo.Path})
		}
	}
	if err != nil {
		return nil, err
	}
	return repo.parsePrettyFormatLogToList(bytes.TrimSpace(stdout))
}

// CommitsBetweenIDs return commits between twoe commits
func (repo *Repository) CommitsBetweenIDs(last, before string) ([]*Commit, error) {
	lastCommit, err := repo.GetCommit(last)
	if err != nil {
		return nil, err
	}
	if before == "" {
		return repo.CommitsBetween(lastCommit, nil)
	}
	beforeCommit, err := repo.GetCommit(before)
	if err != nil {
		return nil, err
	}
	return repo.CommitsBetween(lastCommit, beforeCommit)
}

// CommitsCountBetween return numbers of commits between two commits
func (repo *Repository) CommitsCountBetween(start, end string) (int64, error) {
	count, err := CommitsCount(repo.Ctx, CommitsCountOptions{
		RepoPath:     repo.Path,
		Revision:     []string{start + ".." + end},
		GitalyRepo:   repo.GitalyRepo,
		Ctx:          repo.Ctx,
		CommitClient: repo.CommitClient,
	})

	if err != nil && strings.Contains(err.Error(), "no merge base") {
		// future versions of git >= 2.28 are likely to return an error if before and last have become unrelated.
		// previously it would return the results of git rev-list before last so let's try that...
		return CommitsCount(repo.Ctx, CommitsCountOptions{
			RepoPath:     repo.Path,
			Revision:     []string{start, end},
			GitalyRepo:   repo.GitalyRepo,
			Ctx:          repo.Ctx,
			CommitClient: repo.CommitClient,
		})
	}

	return count, err
}

// commitsBefore the limit is depth, not total number of returned commits.
func (repo *Repository) commitsBefore(id SHA1, limit int) ([]*Commit, error) {
	revision := id.Byte()

	ctx, cancel := context.WithCancel(repo.Ctx)
	defer cancel()

	commits := make([]*Commit, 0)
	commitsClient, err := repo.CommitClient.FindAllCommits(ctx, &gitalypb.FindAllCommitsRequest{
		Repository: repo.GitalyRepo,
		Revision:   revision,
		MaxCount:   int32(limit),
	})
	if err != nil {
		return nil, err
	}

	canRead := true
	for canRead {
		commitsResp, err := commitsClient.Recv()
		if err != nil && err != io.EOF {
			return nil, err
		}
		if commitsResp == nil {
			canRead = false
		} else {
			comm, err := repo.ParseGitCommitsToCommit(commitsResp.Commits)
			if err != nil {
				return nil, err
			}
			commits = append(commits, comm...)
		}
	}

	return commits, nil
}

func (repo *Repository) getCommitsBefore(id SHA1) ([]*Commit, error) {
	return repo.commitsBefore(id, 0)
}

func (repo *Repository) getCommitsBeforeLimit(id SHA1, num int) ([]*Commit, error) {
	return repo.commitsBefore(id, num)
}

func (repo *Repository) getBranches(commit *Commit, limit int) ([]string, error) {
	if CheckGitVersionAtLeast("2.7.0") == nil {
		stdout, _, err := NewCommand(repo.Ctx, "for-each-ref", "--format=%(refname:strip=2)").
			AddOptionFormat("--count=%d", limit).
			AddOptionValues("--contains", commit.ID.String(), BranchPrefix).
			RunStdString(&RunOpts{Dir: repo.Path})
		if err != nil {
			return nil, err
		}

		branches := strings.Fields(stdout)
		return branches, nil
	}

	stdout, _, err := NewCommand(repo.Ctx, "branch").AddOptionValues("--contains", commit.ID.String()).RunStdString(&RunOpts{Dir: repo.Path})
	if err != nil {
		return nil, err
	}

	refs := strings.Split(stdout, "\n")

	var max int
	if len(refs) > limit {
		max = limit
	} else {
		max = len(refs) - 1
	}

	branches := make([]string, max)
	for i, ref := range refs[:max] {
		parts := strings.Fields(ref)

		branches[i] = parts[len(parts)-1]
	}
	return branches, nil
}

// GetCommitsFromIDs get commits from commit IDs
func (repo *Repository) GetCommitsFromIDs(commitIDs []string) []*Commit {
	commits := make([]*Commit, 0, len(commitIDs))

	for _, commitID := range commitIDs {
		commit, err := repo.GetCommit(commitID)
		if err == nil && commit != nil {
			commits = append(commits, commit)
		}
	}

	return commits
}

// IsCommitInBranch check if the commit is on the branch
func (repo *Repository) IsCommitInBranch(commitID, branch string) (r bool, err error) {
	stdout, _, err := NewCommand(repo.Ctx, "branch", "--contains").AddDynamicArguments(commitID, branch).RunStdString(&RunOpts{Dir: repo.Path})
	if err != nil {
		return false, err
	}
	return len(stdout) > 0, err
}

func (repo *Repository) AddLastCommitCache(cacheKey, fullName, sha string) error {
	if repo.LastCommitCache == nil {
		commitsCount, err := cache.GetInt64(cacheKey, func() (int64, error) {
			commit, err := repo.GetCommit(sha)
			if err != nil {
				return 0, err
			}
			return commit.CommitsCount()
		})
		if err != nil {
			return err
		}
		repo.LastCommitCache = NewLastCommitCache(commitsCount, fullName, repo, cache.GetCache())
	}
	return nil
}

func (repo *Repository) ParseGitCommitsToCommit(gitCommits []*gitalypb.GitCommit) ([]*Commit, error) {
	commits := make([]*Commit, 0)
	for _, commitResp := range gitCommits {
		parents := make([]SHA1, 0, len(commitResp.ParentIds))
		for _, parentId := range commitResp.ParentIds {
			id, err := repo.ConvertToSHA1(parentId)
			if err != nil {
				return nil, err
			}

			parents = append(parents, id)
		}
		commit := &Commit{
			ID:            MustIDFromString(commitResp.Id),
			Author:        &Signature{Name: string(commitResp.Author.Name), Email: string(commitResp.Author.Email), When: commitResp.Author.Date.AsTime()},
			Committer:     &Signature{Name: string(commitResp.Committer.Name), Email: string(commitResp.Committer.Email), When: commitResp.Committer.Date.AsTime()},
			CommitMessage: string(commitResp.Body),
			Parents:       parents,
			Tree:          *NewTree(repo, MustIDFromString(commitResp.TreeId), "."),
		}
		commit.ResolvedID = commit.ID

		commits = append(commits, commit)
	}
	return commits, nil
}
