// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

//go:build !gogit

package git

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"gitlab.com/gitlab-org/gitaly/v16/proto/go/gitalypb"

	"code.gitea.io/gitea/modules/log"
)

// ResolveReference resolves a name to a reference
func (repo *Repository) ResolveReference(name string) (string, error) {
	stdout, _, err := NewCommand(repo.Ctx, "show-ref", "--hash").AddDynamicArguments(name).RunStdString(&RunOpts{Dir: repo.Path})
	if err != nil {
		if strings.Contains(err.Error(), "not a valid ref") {
			return "", ErrNotExist{name, ""}
		}
		return "", err
	}
	stdout = strings.TrimSpace(stdout)
	if stdout == "" {
		return "", ErrNotExist{name, ""}
	}

	return stdout, nil
}

// GetRefCommitID returns the last commit ID string of given reference (branch or tag).
func (repo *Repository) GetRefCommitID(name string) (string, error) {
	refName := []byte(name)
	refNames := make([][]byte, 0)
	refNames = append(refNames, refName)

	ctx, cancel := context.WithCancel(repo.Ctx)
	defer cancel()
	listCommitsResp, err := repo.CommitClient.ListCommitsByRefName(ctx, &gitalypb.ListCommitsByRefNameRequest{RefNames: refNames, Repository: repo.GitalyRepo})
	if err != nil {
		log.Error("Error has occurred while getting list of commits by ref name from gitaly: %v", err)
		return "", fmt.Errorf("get list of commits by ref name from gitaly: %w", err)
	}
	recv, err := listCommitsResp.Recv()
	if err != nil && err != io.EOF {
		log.Error("Error has occurred while receiving commits by ref name from gitaly: %v", err)
		return "", fmt.Errorf("receive commits by ref name from gitaly: %w", err)
	}
	if recv == nil {
		log.Error("Error has occurred while getting commits by ref name from gitaly: no commits found")
		return "", fmt.Errorf("received nil from gitaly")
	}

	lastCom := recv.CommitRefs[len(recv.CommitRefs)-1]

	return lastCom.Commit.Id, nil
}

// SetReference sets the commit ID string of given reference (e.g. branch or tag).
func (repo *Repository) SetReference(name, commitID string) error {
	_, _, err := NewCommand(repo.Ctx, "update-ref").AddDynamicArguments(name, commitID).RunStdString(&RunOpts{Dir: repo.Path})
	return err
}

// RemoveReference removes the given reference (e.g. branch or tag).
func (repo *Repository) RemoveReference(name string) error {
	ctx, cancel := context.WithCancel(repo.Ctx)
	defer cancel()

	refs := make([][]byte, 0)
	refs = append(refs, []byte(name))
	_, err := repo.RefClient.DeleteRefs(ctx, &gitalypb.DeleteRefsRequest{
		Repository: repo.GitalyRepo,
		Refs:       refs,
	})

	return err
}

// IsCommitExist returns true if given commit exists in current repository.
func (repo *Repository) IsCommitExist(name string) bool {
	_, _, err := NewCommand(repo.Ctx, "cat-file", "-e").AddDynamicArguments(name).RunStdString(&RunOpts{Dir: repo.Path})
	return err == nil
}

func (repo *Repository) getCommit(id string) (*Commit, error) {
	ctx, cancel := context.WithCancel(repo.Ctx)
	defer cancel()
	commitResponse, err := repo.CommitClient.FindCommit(ctx, &gitalypb.FindCommitRequest{Repository: repo.GitalyRepo, Revision: []byte(id)})
	if err != nil {
		return nil, err
	}
	if commitResponse.Commit == nil {
		return nil, &ErrNotExist{
			ID:      id,
			RelPath: repo.Path,
		}
	}

	parents := make([]SHA1, 0, len(commitResponse.Commit.ParentIds))
	for _, parentId := range commitResponse.Commit.ParentIds {
		id, err := repo.ConvertToSHA1(parentId)
		if err != nil {
			return nil, err
		}
		parents = append(parents, id)
	}
	commit := &Commit{
		ID:            MustIDFromString(commitResponse.Commit.Id),
		Author:        &Signature{Name: string(commitResponse.Commit.Author.Name), Email: string(commitResponse.Commit.Author.Email), When: commitResponse.Commit.Author.Date.AsTime()},
		Committer:     &Signature{Name: string(commitResponse.Commit.Committer.Name), Email: string(commitResponse.Commit.Committer.Email), When: commitResponse.Commit.Committer.Date.AsTime()},
		CommitMessage: string(commitResponse.Commit.Body),
		Parents:       parents,
		Tree:          *NewTree(repo, MustIDFromString(commitResponse.Commit.TreeId), "."),
	}
	commit.ResolvedID = commit.ID

	return commit, err
}

func (repo *Repository) getCommitFromBatchReader(rd *bufio.Reader, id SHA1) (*Commit, error) {
	_, typ, size, err := ReadBatchLine(rd)
	if err != nil {
		if errors.Is(err, io.EOF) || IsErrNotExist(err) {
			return nil, ErrNotExist{ID: id.String()}
		}
		return nil, err
	}

	switch typ {
	case "missing":
		return nil, ErrNotExist{ID: id.String()}
	case "tag":
		// then we need to parse the tag
		// and load the commit
		data, err := io.ReadAll(io.LimitReader(rd, size))
		if err != nil {
			return nil, err
		}
		_, err = rd.Discard(1)
		if err != nil {
			return nil, err
		}
		tag, err := parseTagData(data)
		if err != nil {
			return nil, err
		}

		commit, err := tag.Commit(repo)
		if err != nil {
			return nil, err
		}

		return commit, nil
	case "commit":
		commit, err := CommitFromReader(repo, id, io.LimitReader(rd, size))
		if err != nil {
			return nil, err
		}
		_, err = rd.Discard(1)
		if err != nil {
			return nil, err
		}

		return commit, nil
	default:
		log.Debug("Unknown typ: %s", typ)
		_, err = rd.Discard(int(size) + 1)
		if err != nil {
			return nil, err
		}
		return nil, ErrNotExist{
			ID: id.String(),
		}
	}
}

// ConvertToSHA1 returns a Hash object from a potential ID string
func (repo *Repository) ConvertToSHA1(commitID string) (SHA1, error) {
	if len(commitID) == SHAFullLength && IsValidSHAPattern(commitID) {
		sha1, err := NewIDFromString(commitID)
		if err == nil {
			return sha1, nil
		}
	}
	return MustIDFromString(commitID), nil
}
