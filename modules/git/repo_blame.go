// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package git

import (
	"context"
	"fmt"

	"gitlab.com/gitlab-org/gitaly/v16/proto/go/gitalypb"

	"code.gitea.io/gitea/modules/log"
)

// FileBlame return the Blame object of file
// todo: delete unused
func (repo *Repository) FileBlame(revision, path, file string) ([]byte, error) {
	stdout, _, err := NewCommand(repo.Ctx, "blame", "--root").AddDashesAndList(file).RunStdBytes(&RunOpts{Dir: path})
	return stdout, err
}

// LineBlame returns the latest commit at the given line
func (repo *Repository) LineBlame(revision, path, file string, line uint) (*Commit, error) {
	rangeLines := fmt.Sprintf("%d,%d", line, line)

	ctx, cancel := context.WithCancel(repo.Ctx)
	defer cancel()
	rawBlameStream, err := repo.CommitClient.RawBlame(ctx, &gitalypb.RawBlameRequest{
		Repository: repo.GitalyRepo,
		Revision:   []byte(revision),
		Path:       []byte(file),
		Range:      []byte(rangeLines),
	})
	if err != nil {
		log.Error("Error has occurred while requesting blame: %v", err)
		return nil, fmt.Errorf("failed to request raw blame: %w", err)
	}

	rawBlame, err := rawBlameStream.Recv()
	if err != nil {
		log.Error("Error has occurred while getting blame: %v", err)
		return nil, fmt.Errorf("failed to get raw blame: %w", err)
	}
	res := rawBlame.GetData()

	if len(res) < 40 {
		return nil, fmt.Errorf("invalid result of blame: %s", res)
	}
	return repo.GetCommit(string(res[:40]))
}
