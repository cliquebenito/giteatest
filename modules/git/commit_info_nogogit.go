// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

//go:build !gogit

package git

import (
	"context"
	"fmt"
	"io"
	"path"
	"sort"
	"strings"

	"gitlab.com/gitlab-org/gitaly/v16/proto/go/gitalypb"

	"code.gitea.io/gitea/modules/log"
)

// GetCommitsInfo gets information of all commits that are corresponding to these entries
func (tes Entries) GetCommitsInfo(ctx context.Context, commit *Commit, treePath string) ([]CommitInfo, *Commit, error) {
	entryPaths := make([]string, len(tes)+1)
	// Get the commit for the treePath itself
	entryPaths[0] = ""
	for i, entry := range tes {
		entryPaths[i+1] = entry.Name()
	}

	var err error

	var revs map[string]*Commit
	if commit.repo.LastCommitCache != nil {
		var unHitPaths []string
		revs, unHitPaths, err = getLastCommitForPathsByCache(ctx, commit.ID.String(), treePath, entryPaths, commit.repo.LastCommitCache)
		if err != nil {
			return nil, nil, err
		}
		if len(unHitPaths) > 0 {
			sort.Strings(unHitPaths)
			commits, err := GetLastCommitForPaths(ctx, commit, treePath, unHitPaths)
			if err != nil {
				return nil, nil, err
			}

			for pth, found := range commits {
				revs[pth] = found
			}
		}
	} else {
		sort.Strings(entryPaths)
		revs, err = GetLastCommitForPaths(ctx, commit, treePath, entryPaths)
	}
	if err != nil {
		return nil, nil, err
	}

	commitsInfo := make([]CommitInfo, len(tes))
	for i, entry := range tes {
		commitsInfo[i] = CommitInfo{
			Entry: entry,
		}

		entryRevName := strings.TrimPrefix(entry.Name(), fmt.Sprintf("%s/", treePath))

		// Check if we have found a commit for this entry in time
		if entryCommit, ok := revs[entryRevName]; ok {
			commitsInfo[i].Commit = entryCommit
		} else {
			log.Debug("missing commit for %s", entry.Name())
		}

		// If the entry if a submodule add a submodule file for this
		if entry.IsSubModule() {
			subModuleURL := ""
			var fullPath string
			if len(treePath) > 0 {
				fullPath = treePath + "/" + entry.Name()
			} else {
				fullPath = entry.Name()
			}
			if subModule, err := commit.GetSubModule(fullPath); err != nil {
				return nil, nil, err
			} else if subModule != nil {
				subModuleURL = subModule.URL
			}
			subModuleFile := NewSubModuleFile(commitsInfo[i].Commit, subModuleURL, entry.ID.String())
			commitsInfo[i].SubModuleFile = subModuleFile
		}
	}

	// Retrieve the commit for the treePath itself (see above). We basically
	// get it for free during the tree traversal and it's used for listing
	// pages to display information about newest commit for a given path.
	var treeCommit *Commit
	var ok bool
	if treePath == "" {
		treeCommit = commit
	} else if treeCommit, ok = revs[""]; ok {
		treeCommit.repo = commit.repo
	}

	return commitsInfo, treeCommit, nil
}

func getLastCommitForPathsByCache(ctx context.Context, commitID, treePath string, paths []string, cache *LastCommitCache) (map[string]*Commit, []string, error) {
	var unHitEntryPaths []string
	results := make(map[string]*Commit)
	for _, p := range paths {
		lastCommit, err := cache.Get(commitID, path.Join(treePath, p))
		if err != nil {
			return nil, nil, err
		}
		if lastCommit != nil {
			results[p] = lastCommit
			continue
		}

		unHitEntryPaths = append(unHitEntryPaths, p)
	}

	return results, unHitEntryPaths, nil
}

// GetLastCommitForPaths returns last commit information
func GetLastCommitForPaths(ctx context.Context, commit *Commit, treePath string, paths []string) (map[string]*Commit, error) {
	commitCommits := map[string]*Commit{}
	var bytePath []byte
	if treePath != "" && treePath != "." {
		bytePath = []byte(treePath + "/.")
	}
	ctxWithCancel, cancel := context.WithCancel(commit.repo.Ctx)
	defer cancel()
	lastCommitsForTree, err := commit.repo.CommitClient.ListLastCommitsForTree(ctxWithCancel, &gitalypb.ListLastCommitsForTreeRequest{
		Repository: commit.repo.GitalyRepo,
		Revision:   commit.ID.String(),
		Path:       bytePath,
		Limit:      int32(^uint32(0) >> 1),
	})
	if err != nil {
		log.Error("Error has occurred while requesting last commit for paths: %v", err)
		return nil, fmt.Errorf("error request last commit for paths: %w", err)
	}

	listLastCommits := make([]*gitalypb.ListLastCommitsForTreeResponse_CommitForTree, 0, len(paths))
	for {
		recv, err := lastCommitsForTree.Recv()
		if err != nil && err != io.EOF {
			log.Error("Error has occurred while receiving last commit for paths: %v", err)
			return nil, fmt.Errorf("error receive last commit for paths: %w", err)
		}
		if recv == nil {
			break
		}

		listLastCommits = append(listLastCommits, recv.GetCommits()...)
	}

	for _, lastCommitInfo := range listLastCommits {
		parents := make([]SHA1, len(lastCommitInfo.Commit.ParentIds))
		for _, parentId := range lastCommitInfo.Commit.ParentIds {
			id, err := NewIDFromString(parentId)
			if err != nil {
				log.Error("Error has occurred while converting parent id: %v", err)
				return commitCommits, fmt.Errorf("error convert parent id: %w", err)
			}
			parents = append(parents, id)
		}
		commitRes := &Commit{
			ID:            MustIDFromString(lastCommitInfo.Commit.Id),
			Author:        &Signature{Name: string(lastCommitInfo.Commit.Author.Name), Email: string(lastCommitInfo.Commit.Author.Email), When: lastCommitInfo.Commit.Author.Date.AsTime()},
			Committer:     &Signature{Name: string(lastCommitInfo.Commit.Committer.Name), Email: string(lastCommitInfo.Commit.Committer.Email), When: lastCommitInfo.Commit.Committer.Date.AsTime()},
			CommitMessage: string(lastCommitInfo.Commit.Body),
			Parents:       parents,
			Tree:          *NewTree(commit.repo, MustIDFromString(lastCommitInfo.Commit.TreeId), treePath),
		}
		commitRes.ResolvedID = commitRes.ID
		relPath := strings.TrimPrefix(string(lastCommitInfo.GetPathBytes()), fmt.Sprintf("%s/", treePath))
		commitCommits[relPath] = commitRes
	}

	return commitCommits, nil
}
