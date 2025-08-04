// Copyright 2015 The Gogs Authors. All rights reserved.
// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package git

import (
	"bytes"
)

// NewTree create a new tree according the repository and tree id
func NewTree(repo *Repository, id SHA1, path string) *Tree {
	return &Tree{
		ID:   id,
		repo: repo,
		path: path,
	}
}

// SubTree get a sub tree by the sub dir path
func (t *Tree) SubTree(rpath string) (*Tree, error) {
	if len(rpath) == 0 {
		return t, nil
	}
	te, err := t.GetTreeEntryByPath(rpath)
	if err != nil {
		return nil, err
	}

	g, err := t.repo.getTree(te.ID, te.name, te.entryMode)
	if err != nil {
		return nil, err
	}
	g.ptree = t
	return g, nil
}

// LsTree checks if the given filenames are in the tree
func (repo *Repository) LsTree(ref string, filenames ...string) ([]string, error) {
	cmd := NewCommand(repo.Ctx, "ls-tree", "-z", "--name-only").
		AddDashesAndList(append([]string{ref}, filenames...)...)

	res, _, err := cmd.RunStdBytes(&RunOpts{Dir: repo.Path})
	if err != nil {
		return nil, err
	}
	filelist := make([]string, 0, len(filenames))
	for _, line := range bytes.Split(res, []byte{'\000'}) {
		filelist = append(filelist, string(line))
	}

	return filelist, err
}
