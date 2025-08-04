// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

//go:build !gogit

package git

func (repo *Repository) getTree(id SHA1, path string, entryMode EntryMode) (*Tree, error) {
	// todo: tag
	switch entryMode {
	case EntryModeTree:
		tree := NewTree(repo, id, path)
		tree.ResolvedID = id

		_, err := tree.ListEntries()
		if err != nil {
			return nil, err
		}
		tree.entriesParsed = true
		return tree, err
	case EntryModeCommit:
		commit, err := repo.getCommit(id.String())
		if err != nil {
			return nil, err
		}
		return &commit.Tree, err
	default:
		return nil, ErrNotExist{
			ID: id.String(),
		}
	}
}

// GetTree find the tree object in the repository.
func (repo *Repository) GetTree(idStr string) (*Tree, error) {
	if len(idStr) != SHAFullLength {
		res, err := repo.GetRefCommitID(idStr)
		if err != nil {
			return nil, err
		}
		if len(res) > 0 {
			idStr = res
		}
	}
	id, err := NewIDFromString(idStr)
	if err != nil {
		return nil, err
	}

	return repo.getTree(id, ".", EntryModeTree)
}
