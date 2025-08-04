// Copyright 2015 The Gogs Authors. All rights reserved.
// Copyright 2018 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

//go:build !gogit

package git

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"gitlab.com/gitlab-org/gitaly/v16/proto/go/gitalypb"

	"code.gitea.io/gitea/modules/log"
)

// IsObjectExist returns true if given reference exists in the repository.
func (repo *Repository) IsObjectExist(name string) bool {
	if name == "" {
		return false
	}

	wr, rd, cancel := repo.CatFileBatchCheck(repo.Ctx)
	defer cancel()
	_, err := wr.Write([]byte(name + "\n"))
	if err != nil {
		log.Debug("Error writing to CatFileBatchCheck %v", err)
		return false
	}
	sha, _, _, err := ReadBatchLine(rd)
	return err == nil && bytes.HasPrefix(sha, []byte(strings.TrimSpace(name)))
}

// IsReferenceExist returns true if given reference exists in the repository.
func (repo *Repository) IsReferenceExist(name string) bool {
	if name == "" {
		return false
	}

	ctx, cancel := context.WithCancel(repo.Ctx)
	defer cancel()
	exist, err := repo.RefClient.RefExists(ctx, &gitalypb.RefExistsRequest{
		Repository: repo.GitalyRepo,
		Ref:        []byte(name),
	})
	if err != nil {
		return false
	}
	return exist.GetValue()
}

// IsBranchExist returns true if given branch exists in current repository.
func (repo *Repository) IsBranchExist(name string) bool {
	if repo == nil || name == "" {
		return false
	}
	ctx, cancel := context.WithCancel(repo.Ctx)
	defer cancel()
	exists, err := repo.RefClient.RefExists(ctx, &gitalypb.RefExistsRequest{
		Repository: repo.GitalyRepo,
		Ref:        []byte(BranchPrefix + name),
	})
	if err != nil {
		return false
	}

	return exists.GetValue()
}

// GetBranchNames returns branches from the repository, skipping "skip" initial branches and
// returning at most "limit" branches, or all branches if "limit" is 0.
func (repo *Repository) GetBranchNames(skip, limit int) ([]string, int, error) {
	ctx, cancel := context.WithCancel(repo.Ctx)
	defer cancel()

	client, err := repo.RefClient.FindAllBranches(ctx, &gitalypb.FindAllBranchesRequest{Repository: repo.GitalyRepo})
	if err != nil {
		return nil, 0, err
	}

	resp := make([]*gitalypb.FindAllBranchesResponse_Branch, 0, limit)
	canRead := true
	for canRead {
		branchResponse, err := client.Recv()
		if err != nil && err != io.EOF {
			log.Error("Error has occurred while receiving branch names: %v", err)
			return nil, 0, fmt.Errorf("receive branch names: %w", err)
		}
		if branchResponse == nil {
			canRead = false
		} else {
			resp = append(resp, branchResponse.GetBranches()...)
		}
	}

	branchNames := make([]string, 0, len(resp))

	for _, branch := range resp {
		branchNames = append(branchNames, strings.TrimPrefix(string(branch.GetName()), BranchPrefix))
	}

	branchesCount := len(branchNames)

	if limit != 0 {
		limit = limit + skip // вычисляем смещение относительно конца, так как пагинация происходит на пост обработке

		if len(branchNames) > limit {
			return branchNames[skip:limit], branchesCount, nil
		}
	}

	return branchNames[skip:], branchesCount, nil
}

// WalkReferences walks all the references from the repository
func WalkReferences(ctx context.Context, repoPath string, walkfn func(sha1, refname string) error) (int, error) {
	return walkShowRef(ctx, repoPath, nil, 0, 0, walkfn)
}

// WalkReferences walks all the references from the repository
// refType should be empty, ObjectTag or ObjectBranch. All other values are equivalent to empty.
func (repo *Repository) WalkReferences(refType ObjectType, skip, limit int, walkfn func(sha1, refname string) error) (int, error) {
	var args TrustedCmdArgs
	switch refType {
	case ObjectTag:
		args = TrustedCmdArgs{TagPrefix, "--sort=-taggerdate"}
	case ObjectBranch:
		args = TrustedCmdArgs{BranchPrefix, "--sort=-committerdate"}
	}

	return walkShowRef(repo.Ctx, repo.Path, args, skip, limit, walkfn)
}

// callShowRef return refs, if limit = 0 it will not limit
func callShowRef(ctx context.Context, repoPath, trimPrefix string, extraArgs TrustedCmdArgs, skip, limit int) (branchNames []string, countAll int, err error) {
	countAll, err = walkShowRef(ctx, repoPath, extraArgs, skip, limit, func(_, branchName string) error {
		branchName = strings.TrimPrefix(branchName, trimPrefix)
		branchNames = append(branchNames, branchName)

		return nil
	})
	return branchNames, countAll, err
}

func walkShowRef(ctx context.Context, repoPath string, extraArgs TrustedCmdArgs, skip, limit int, walkfn func(sha1, refname string) error) (countAll int, err error) {
	stdoutReader, stdoutWriter := io.Pipe()
	defer func() {
		_ = stdoutReader.Close()
		_ = stdoutWriter.Close()
	}()

	go func() {
		stderrBuilder := &strings.Builder{}
		args := TrustedCmdArgs{"for-each-ref", "--format=%(objectname) %(refname)"}
		args = append(args, extraArgs...)
		err := NewCommand(ctx, args...).Run(&RunOpts{
			Dir:    repoPath,
			Stdout: stdoutWriter,
			Stderr: stderrBuilder,
		})
		if err != nil {
			if stderrBuilder.Len() == 0 {
				_ = stdoutWriter.Close()
				return
			}
			_ = stdoutWriter.CloseWithError(ConcatenateError(err, stderrBuilder.String()))
		} else {
			_ = stdoutWriter.Close()
		}
	}()

	i := 0
	bufReader := bufio.NewReader(stdoutReader)
	for i < skip {
		_, isPrefix, err := bufReader.ReadLine()
		if err == io.EOF {
			return i, nil
		}
		if err != nil {
			return 0, err
		}
		if !isPrefix {
			i++
		}
	}
	for limit == 0 || i < skip+limit {
		// The output of show-ref is simply a list:
		// <sha> SP <ref> LF
		sha, err := bufReader.ReadString(' ')
		if err == io.EOF {
			return i, nil
		}
		if err != nil {
			return 0, err
		}

		branchName, err := bufReader.ReadString('\n')
		if err == io.EOF {
			// This shouldn't happen... but we'll tolerate it for the sake of peace
			return i, nil
		}
		if err != nil {
			return i, err
		}

		if len(branchName) > 0 {
			branchName = branchName[:len(branchName)-1]
		}

		if len(sha) > 0 {
			sha = sha[:len(sha)-1]
		}

		err = walkfn(sha, branchName)
		if err != nil {
			return i, err
		}
		i++
	}
	// count all refs
	for limit != 0 {
		_, isPrefix, err := bufReader.ReadLine()
		if err == io.EOF {
			return i, nil
		}
		if err != nil {
			return 0, err
		}
		if !isPrefix {
			i++
		}
	}
	return i, nil
}

// GetRefsBySha returns all references filtered with prefix that belong to a sha commit hash
func (repo *Repository) GetRefsBySha(sha, prefix string) ([]string, error) {
	var revList []string
	_, err := walkShowRef(repo.Ctx, repo.Path, nil, 0, 0, func(walkSha, refname string) error {
		if walkSha == sha && strings.HasPrefix(refname, prefix) {
			revList = append(revList, refname)
		}
		return nil
	})
	return revList, err
}
