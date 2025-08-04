// Copyright 2015 The Gogs Authors. All rights reserved.
// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package git

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"gitlab.com/gitlab-org/gitaly/v16/proto/go/gitalypb"

	"code.gitea.io/gitea/integration/gitaly"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"

	"code.gitea.io/gitea/modules/proxy"
	"code.gitea.io/gitea/modules/util"
)

// GPGSettings represents the default GPG settings for this repository
type GPGSettings struct {
	Sign             bool
	KeyID            string
	Email            string
	Name             string
	PublicKeyContent string
}

const prettyLogFormat = `--pretty=format:%H`

// GetAllCommitsCount returns count of all commits in repository
func (repo *Repository) GetAllCommitsCount() (int64, error) {
	ctx, cancel := context.WithCancel(repo.Ctx)
	defer cancel()
	countCommits, err := repo.CommitClient.CountCommits(ctx, &gitalypb.CountCommitsRequest{
		Repository: repo.GitalyRepo,
		All:        true,
	})
	if err != nil {
		return 0, err
	}

	return int64(countCommits.GetCount()), nil
}

// todo may be one request
func (repo *Repository) parsePrettyFormatLogToList(logs []byte) ([]*Commit, error) {
	var commits []*Commit
	if len(logs) == 0 {
		return commits, nil
	}

	parts := bytes.Split(logs, []byte{'\n'})

	for _, commitID := range parts {
		commit, err := repo.GetCommit(string(commitID))
		if err != nil {
			return nil, err
		}
		commits = append(commits, commit)
	}

	return commits, nil
}

// IsRepoURLAccessible checks if given repository URL is accessible.
func IsRepoURLAccessible(ctx context.Context, url string) bool {
	_, _, err := NewCommand(ctx, "ls-remote", "-q", "-h").AddDynamicArguments(url, "HEAD").RunStdString(nil)
	return err == nil
}

// InitRepository initializes a new Git repository.
func InitRepository(ctx context.Context, repoPath string, bare bool) error {
	err := os.MkdirAll(repoPath, os.ModePerm)
	if err != nil {
		return err
	}

	cmd := NewCommand(ctx, "init")
	if bare {
		cmd.AddArguments("--bare")
	}
	_, _, err = cmd.RunStdString(&RunOpts{Dir: repoPath})
	return err
}

// IsEmpty Check if repository is empty.
func (repo *Repository) IsEmpty() (bool, error) {
	info, err := repo.RepoClient.RepositoryInfo(repo.Ctx, &gitalypb.RepositoryInfoRequest{Repository: repo.GitalyRepo})
	if err != nil {
		log.Error("failed RepositoryExists")
		return false, err
	}
	return info.Objects.Size == 0, nil
}

// CloneRepoOptions options when clone a repository
type CloneRepoOptions struct {
	Timeout       time.Duration
	Mirror        bool
	Bare          bool
	Quiet         bool
	Branch        string
	Shared        bool
	NoCheckout    bool
	Depth         int
	Filter        string
	SkipTLSVerify bool
}

// Clone clones original repository to target path.
func Clone(ctx context.Context, from, to string, opts CloneRepoOptions) error {
	return CloneWithArgs(ctx, globalCommandArgs, from, to, opts)
}

// CloneWithArgs original repository to target path.
func CloneWithArgs(ctx context.Context, args TrustedCmdArgs, from, to string, opts CloneRepoOptions) (err error) {
	toDir := path.Dir(to)
	if err = os.MkdirAll(toDir, os.ModePerm); err != nil {
		return err
	}

	cmd := NewCommandContextNoGlobals(ctx, args...).AddArguments("clone")
	if opts.SkipTLSVerify {
		cmd.AddArguments("-c", "http.sslVerify=false")
	}
	if setting.Mirror.Enabled {
		if opts.Mirror {
			cmd.AddArguments("--mirror")
		}
	}
	if opts.Bare {
		cmd.AddArguments("--bare")
	}
	if opts.Quiet {
		cmd.AddArguments("--quiet")
	}
	if opts.Shared {
		cmd.AddArguments("-s")
	}
	if opts.NoCheckout {
		cmd.AddArguments("--no-checkout")
	}
	if opts.Depth > 0 {
		cmd.AddArguments("--depth").AddDynamicArguments(strconv.Itoa(opts.Depth))
	}
	if opts.Filter != "" {
		cmd.AddArguments("--filter").AddDynamicArguments(opts.Filter)
	}
	if len(opts.Branch) > 0 {
		cmd.AddArguments("-b").AddDynamicArguments(opts.Branch)
	}
	cmd.AddDashesAndList(from, to)

	if strings.Contains(from, "://") && strings.Contains(from, "@") {
		cmd.SetDescription(fmt.Sprintf("clone branch %s from %s to %s (shared: %t, mirror: %t, depth: %d)", opts.Branch, util.SanitizeCredentialURLs(from), to, opts.Shared, opts.Mirror, opts.Depth))
	} else {
		cmd.SetDescription(fmt.Sprintf("clone branch %s from %s to %s (shared: %t, mirror: %t, depth: %d)", opts.Branch, from, to, opts.Shared, opts.Mirror, opts.Depth))
	}

	if opts.Timeout <= 0 {
		opts.Timeout = -1
	}

	envs := os.Environ()
	u, err := url.Parse(from)
	if err == nil {
		envs = proxy.EnvWithProxy(u)
	}

	stderr := new(bytes.Buffer)
	if err = cmd.Run(&RunOpts{
		Timeout: opts.Timeout,
		Env:     envs,
		Stdout:  io.Discard,
		Stderr:  stderr,
	}); err != nil {
		return ConcatenateError(err, stderr.String())
	}

	return nil
}

// PushOptions options when push to remote
type PushOptions struct {
	Remote  string
	Branch  string
	Force   bool
	Mirror  bool
	Env     []string
	Timeout time.Duration
}

// Push pushs local commits to given remote branch.
func Push(ctx context.Context, repoPath string, opts PushOptions) error {

	cmd := NewCommand(ctx, "push")
	if opts.Force {
		cmd.AddArguments("-f")
	}
	if setting.Mirror.Enabled {
		if opts.Mirror {
			cmd.AddArguments("--mirror")
		}
	}
	remoteBranchArgs := []string{opts.Remote}
	if len(opts.Branch) > 0 {
		remoteBranchArgs = append(remoteBranchArgs, opts.Branch)
	}
	cmd.AddDashesAndList(remoteBranchArgs...)

	if strings.Contains(opts.Remote, "://") && strings.Contains(opts.Remote, "@") {
		cmd.SetDescription(fmt.Sprintf("push branch %s to %s (force: %t, mirror: %t)", opts.Branch, util.SanitizeCredentialURLs(opts.Remote), opts.Force, opts.Mirror))
	} else {
		cmd.SetDescription(fmt.Sprintf("push branch %s to %s (force: %t, mirror: %t)", opts.Branch, opts.Remote, opts.Force, opts.Mirror))
	}

	stdout, stderr, err := cmd.RunStdString(&RunOpts{Env: opts.Env, Timeout: opts.Timeout, Dir: repoPath})
	if err != nil {
		if strings.Contains(stderr, "non-fast-forward") {
			return &ErrPushOutOfDate{StdOut: stdout, StdErr: stderr, Err: err}
		} else if strings.Contains(stderr, "! [remote rejected]") {
			err := &ErrPushRejected{StdOut: stdout, StdErr: stderr, Err: err}
			err.GenerateMessage()
			return err
		} else if strings.Contains(stderr, "matches more than one") {
			return &ErrMoreThanOne{StdOut: stdout, StdErr: stderr, Err: err}
		}
		return fmt.Errorf("push failed: %w - %s\n%s", err, stderr, stdout)
	}

	return nil
}

// GetLatestCommitTime returns time for latest commit in repository (across all branches)
func GetLatestCommitTime(ctx context.Context, repoPath string) (time.Time, error) {
	cmd := NewCommand(ctx, "for-each-ref", "--sort=-committerdate", BranchPrefix, "--count", "1", "--format=%(committerdate)")
	stdout, _, err := cmd.RunStdString(&RunOpts{Dir: repoPath})
	if err != nil {
		return time.Time{}, err
	}
	commitTime := strings.TrimSpace(stdout)
	return time.Parse(GitTimeLayout, commitTime)
}

// DivergeObject represents commit count diverging commits
type DivergeObject struct {
	Ahead  int
	Behind int
}

// GetDivergingCommits returns the number of commits a targetBranch is ahead or behind a baseBranch
func GetDivergingCommits(ctx context.Context, repoPath, baseBranch, targetBranch string) (do DivergeObject, err error) {
	conn, client, err := gitaly.NewCommitClient(ctx)
	if err != nil {
		return do, fmt.Errorf("NewCommitClient is failed, err: %s", err)
	}

	resp, err := client.CountDivergingCommits(conn, &gitalypb.CountDivergingCommitsRequest{
		Repository: &gitalypb.Repository{
			StorageName:  setting.Gitaly.MainServerName,
			RelativePath: repoPath,
		},
		From:     []byte(baseBranch),
		To:       []byte(targetBranch),
		MaxCount: 100,
	})

	if err != nil {
		return do, fmt.Errorf("CountDivergingCommits is failed, err: %s", err)
	}

	do = DivergeObject{
		Ahead:  int(resp.RightCount),
		Behind: int(resp.LeftCount),
	}

	return do, nil
}

// CreateBundle create bundle content to the target path
func (repo *Repository) CreateBundle(ctx context.Context, commit string, out io.Writer) error {
	bundle, err := repo.RepoClient.CreateBundle(repo.Ctx, &gitalypb.CreateBundleRequest{Repository: repo.GitalyRepo})
	if err != nil {
		return fmt.Errorf("failed to create bundle: %w", err)
	}
	canRead := true
	for canRead {
		bundleResp, err := bundle.Recv()
		if err != nil && err != io.EOF {
			return fmt.Errorf("failed to receive bundle: %w", err)
		}
		if bundleResp == nil {
			canRead = false
			continue
		}

		_, err = out.Write(bundleResp.GetData())
		if err != nil {
			return fmt.Errorf("failed to write bundle: %w", err)
		}
	}
	return err
}
