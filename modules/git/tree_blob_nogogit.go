// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

//go:build !gogit

package git

import (
	"context"
	"path"
	"strings"

	"gitlab.com/gitlab-org/gitaly/v16/proto/go/gitalypb"
)

// GetTreeEntryByPath перебирает все элементы дерева гита и ищет тот, что находится по пути relpath
func (t *Tree) GetTreeEntryByPath(relpath string) (*TreeEntry, error) {
	if len(relpath) == 0 {
		return &TreeEntry{
			Ptree:     t,
			ID:        t.ID,
			name:      "",
			fullName:  "",
			entryMode: EntryModeTree,
		}, nil
	}

	ctx, cancel := context.WithCancel(t.repo.Ctx)
	defer cancel()

	// FIXME: This should probably use git cat-file --batch to be a bit more efficient
	relpath = path.Clean(relpath)
	treeEntries, err := t.repo.CommitClient.TreeEntry(ctx, &gitalypb.TreeEntryRequest{
		Repository: t.repo.GitalyRepo,
		Revision:   t.ID.Byte(),
		Path:       []byte(relpath),
	})
	if err != nil {
		return nil, err
	}
	entriesResponse, err := treeEntries.Recv()
	if err != nil {
		if strings.Contains(err.Error(), "tree entry not found") {
			return nil, ErrNotExist{
				ID:      "",
				RelPath: relpath,
			}
		}
		return nil, err
	}

	entry := new(TreeEntry)
	entry.Ptree = t
	entry.ID = MustIDFromString(entriesResponse.Oid)
	entry.name = relpath
	entry.fullName = relpath

	switch entriesResponse.Type {
	case gitalypb.TreeEntryResponse_COMMIT:
		entry.entryMode = EntryModeCommit
	case gitalypb.TreeEntryResponse_BLOB:
		entry.entryMode = EntryModeBlob
	case gitalypb.TreeEntryResponse_TREE:
		entry.entryMode = EntryModeTree
		entry.ID = MustIDFromString(t.ID.String())
	}
	entry.size = entriesResponse.Size
	entry.sized = true

	t.entries = append(t.entries, entry)
	return entry, nil
}
