// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

//go:build !gogit

package git

import (
	"context"
	"strings"

	"gitlab.com/gitlab-org/gitaly/v16/proto/go/gitalypb"
)

// Tree represents a flat directory listing.
type Tree struct {
	ID         SHA1
	ResolvedID SHA1
	repo       *Repository
	path       string

	// parent tree
	ptree *Tree

	entries       Entries
	entriesParsed bool

	entriesRecursive       Entries
	entriesRecursiveParsed bool
}

// ListEntries returns all entries of current tree.
func (t *Tree) ListEntries() (Entries, error) {
	if t.entriesParsed == true {
		return t.entries, nil
	}
	t.entries = make([]*TreeEntry, 0, 10)

	ctx, cancel := context.WithCancel(t.repo.Ctx)
	defer cancel()
	entriesClient, err := t.repo.CommitClient.GetTreeEntries(ctx, &gitalypb.GetTreeEntriesRequest{
		Repository:       t.repo.GitalyRepo,
		Revision:         t.ResolvedID.Byte(),
		Path:             []byte(t.path),
		Sort:             0,
		PaginationParams: nil,
		SkipFlatPaths:    true,
	})
	if err != nil {
		return nil, err
	}
	entries, err := entriesClient.Recv()
	if err != nil {
		return nil, err
	}
	for _, entryResp := range entries.GetEntries() {
		entry := new(TreeEntry)
		entry.Ptree = t
		entry.ID = MustIDFromString(entryResp.Oid)
		entry.name = string(entryResp.Path)
		entry.fullName = strings.TrimPrefix(string(entryResp.Path), t.path+"/")

		switch entryResp.Type {
		case gitalypb.TreeEntry_COMMIT:
			entry.entryMode = EntryModeCommit
		case gitalypb.TreeEntry_BLOB:
			entry.entryMode = EntryModeBlob
		case gitalypb.TreeEntry_TREE:
			entry.entryMode = EntryModeTree
		}

		t.entries = append(t.entries, entry)
		t.entriesParsed = true
	}

	return t.entries, err
}

func (t *Tree) ListPaths() ([][]byte, error) {
	ctx, cancel := context.WithCancel(t.repo.Ctx)
	defer cancel()
	filesClient, err := t.repo.CommitClient.ListFiles(ctx, &gitalypb.ListFilesRequest{
		Repository: t.repo.GitalyRepo,
		Revision:   t.ID.Byte(),
	})
	if err != nil {
		return nil, err
	}
	filesResponse, err := filesClient.Recv()
	if err != nil {
		return nil, err
	}

	return filesResponse.Paths, err
}

// listEntriesRecursive returns all entries of current tree recursively including all subtrees
// extraArgs could be "-l" to get the size, which is slower
func (t *Tree) listEntriesRecursive(extraArgs TrustedCmdArgs) (Entries, error) {
	if t.entriesRecursiveParsed {
		return t.entriesRecursive, nil
	}

	stdout, _, runErr := NewCommand(t.repo.Ctx, "ls-tree", "-t", "-r").
		AddArguments(extraArgs...).
		AddDynamicArguments(t.ID.String()).
		RunStdBytes(&RunOpts{Dir: t.repo.Path})
	if runErr != nil {
		return nil, runErr
	}

	var err error
	t.entriesRecursive, err = parseTreeEntries(stdout, t)
	if err == nil {
		t.entriesRecursiveParsed = true
	}

	return t.entriesRecursive, err
}

// ListEntriesRecursiveFast returns all entries of current tree recursively including all subtrees, no size
func (t *Tree) ListEntriesRecursiveFast() (Entries, error) {
	return t.listEntriesRecursive(nil)
}

// ListEntriesRecursiveWithSize returns all entries of current tree recursively including all subtrees, with size
func (t *Tree) ListEntriesRecursiveWithSize() (Entries, error) {
	if t.entriesParsed == true {
		return t.entries, nil
	}
	t.entries = make([]*TreeEntry, 0, 10)

	ctx, cancel := context.WithCancel(t.repo.Ctx)
	defer cancel()
	entriesClient, err := t.repo.CommitClient.GetTreeEntries(ctx, &gitalypb.GetTreeEntriesRequest{
		Repository:       t.repo.GitalyRepo,
		Revision:         t.ResolvedID.Byte(),
		Path:             []byte(t.path),
		Sort:             0,
		PaginationParams: nil,
		Recursive:        true,
		SkipFlatPaths:    false,
	})
	if err != nil {
		return nil, err
	}
	entries, err := entriesClient.Recv()
	if err != nil {
		return nil, err
	}
	for _, entryResp := range entries.GetEntries() {
		entry := new(TreeEntry)
		entry.Ptree = t
		entry.ID = MustIDFromString(entryResp.Oid)
		entry.name = string(entryResp.Path)
		entry.fullName = strings.TrimPrefix(string(entryResp.Path), t.path+"/")

		switch entryResp.Type {
		case gitalypb.TreeEntry_COMMIT:
			entry.entryMode = EntryModeCommit
		case gitalypb.TreeEntry_BLOB:
			entry.entryMode = EntryModeBlob
		case gitalypb.TreeEntry_TREE:
			entry.entryMode = EntryModeTree
		}

		t.entries = append(t.entries, entry)
		t.entriesParsed = true
	}

	return t.entries, err
}
