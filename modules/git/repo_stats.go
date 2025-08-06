// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package git

import (
	"context"
	"fmt"
	"io"
	"sort"
	"time"

	"gitlab.com/gitlab-org/gitaly/v16/proto/go/gitalypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"code.gitea.io/gitea/modules/container"
	"code.gitea.io/gitea/modules/log"
)

// CodeActivityStats represents git statistics data
type CodeActivityStats struct {
	AuthorCount              int64
	CommitCount              int64
	ChangedFiles             int64
	Additions                int64
	Deletions                int64
	CommitCountInAllBranches int64
	Authors                  []*CodeActivityAuthor
}

// CodeActivityAuthor represents git statistics data for commit authors
type CodeActivityAuthor struct {
	Name    string
	Email   string
	Commits int64
}

// GetCodeActivityStats returns code statistics for activity page
func (repo *Repository) GetCodeActivityStats(fromTime time.Time, branch string) (*CodeActivityStats, error) {
	stats := &CodeActivityStats{}

	ctx, cancel := context.WithCancel(repo.Ctx)
	defer cancel()
	countCommits, err := repo.CommitClient.CountCommits(ctx, &gitalypb.CountCommitsRequest{
		Repository: repo.GitalyRepo,
		After:      &timestamppb.Timestamp{Seconds: fromTime.Unix()},
		All:        true,
	})
	if err != nil {
		log.Error("Error has occurred while counting commits: %v", err)
		return nil, fmt.Errorf("failed to count commits: %w", err)
	}

	stats.CommitCountInAllBranches = int64(countCommits.Count)

	request := &gitalypb.FindCommitsRequest{
		Repository:       repo.GitalyRepo,
		Limit:            -1,
		After:            &timestamppb.Timestamp{Seconds: fromTime.Unix()},
		IncludeShortstat: true,
	}

	if len(branch) == 0 {
		request.All = true
	} else {
		refNames := make([][]byte, 0)
		refNames = append(refNames, []byte(branch))
		listCommits, err := repo.CommitClient.ListCommitsByRefName(ctx, &gitalypb.ListCommitsByRefNameRequest{
			Repository: repo.GitalyRepo,
			RefNames:   refNames,
		})
		if err != nil {
			log.Error("Error has occurred while requesting list commits: %v", err)
			return nil, fmt.Errorf("failed to request list commits: %w", err)
		}

		recvCommits, err := listCommits.Recv()
		if err != nil {
			log.Error("Error has occurred while receiving list commits: %v", err)
			return nil, fmt.Errorf("failed to receive list commits: %w", err)
		}

		recvCommits.CommitRefs[0].Commit.GetId()
		request.Revision = []byte(recvCommits.CommitRefs[0].Commit.GetId())
	}

	commits := make([]*gitalypb.GitCommit, 0)

	commitsStream, err := repo.CommitClient.FindCommits(ctx, request)
	if err != nil {
		log.Error("Error has occurred while requesting commits: %v", err)
		return nil, fmt.Errorf("failed to request commits: %w", err)
	}
	commitsRecv, err := commitsStream.Recv()
	if err != nil && err != io.EOF {
		log.Error("Error has occurred while receiving commits: %v", err)
		return nil, fmt.Errorf("failed to receive commits: %w", err)
	}
	if err == io.EOF || commitsRecv == nil {
		commits = []*gitalypb.GitCommit{}
	} else {
		commits = commitsRecv.GetCommits()
	}

	stats.CommitCount = 0
	stats.Additions = 0
	stats.Deletions = 0
	authors := make(map[string]*CodeActivityAuthor)
	commitIds := make([]string, 0)
	files := make(container.Set[string])
	for _, commit := range commits {
		stats.CommitCount++
		email := string(commit.GetAuthor().GetEmail())
		if _, ok := authors[email]; !ok {
			authorName := string(commit.GetAuthor().GetName())
			authors[email] = &CodeActivityAuthor{
				Name:    authorName,
				Email:   email,
				Commits: 0,
			}
		}
		authors[email].Commits++
		stats.Additions += int64(commit.GetShortStats().GetAdditions())
		stats.Deletions += int64(commit.GetShortStats().GetDeletions())
		commitIds = append(commitIds, commit.GetId())
	}

	files, err = repo.calculateChangedFilesFromCommits(files, commitIds)
	if err != nil {
		log.Error("Error has occurred while calculating changed files from commits: %v", err)
		return nil, fmt.Errorf("failed to calculate changed files from commits: %w", err)
	}

	a := make([]*CodeActivityAuthor, 0, len(authors))
	for _, v := range authors {
		a = append(a, v)
	}
	// Sort authors descending depending on commit count
	sort.Slice(a, func(i, j int) bool {
		return a[i].Commits > a[j].Commits
	})
	stats.AuthorCount = int64(len(authors))
	stats.ChangedFiles = int64(len(files))
	stats.Authors = a
	return stats, nil
}

func (repo *Repository) calculateChangedFilesFromCommits(files container.Set[string], commitIds []string) (container.Set[string], error) {
	req := make([]*gitalypb.FindChangedPathsRequest_Request, 0, len(commitIds))
	for _, commitId := range commitIds {
		req = append(req,
			&gitalypb.FindChangedPathsRequest_Request{
				Type: &gitalypb.FindChangedPathsRequest_Request_CommitRequest_{
					CommitRequest: &gitalypb.FindChangedPathsRequest_Request_CommitRequest{
						CommitRevision: commitId,
					},
				},
			},
		)
	}

	paths, err := repo.DiffClient.FindChangedPaths(repo.Ctx, &gitalypb.FindChangedPathsRequest{
		Repository: repo.GitalyRepo,
		Commits:    nil,
		Requests:   req,
	})
	if err != nil {
		log.Error("Error has occurred while requesting changed paths: %v", err)
		return files, fmt.Errorf("failed to request changed paths: %w", err)
	}

	recv, err := paths.Recv()
	if err != nil && err != io.EOF {
		log.Error("Error has occurred while receiving changed paths: %v", err)
		return files, fmt.Errorf("failed to receive changed paths: %w", err)
	}
	if err != io.EOF && recv != nil {
		for _, path := range recv.GetPaths() {
			files.Add(string(path.Path))
		}
	}

	return files, nil
}
