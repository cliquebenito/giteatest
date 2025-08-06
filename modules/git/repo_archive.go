// Copyright 2015 The Gogs Authors. All rights reserved.
// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package git

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"gitlab.com/gitlab-org/gitaly/v16/proto/go/gitalypb"
)

// ArchiveType archive types
type ArchiveType int

const (
	// ZIP zip archive type
	ZIP ArchiveType = iota + 1
	// TARGZ tar gz archive type
	TARGZ
	// BUNDLE bundle archive type
	BUNDLE
)

// String converts an ArchiveType to string
func (a ArchiveType) String() string {
	switch a {
	case ZIP:
		return "zip"
	case TARGZ:
		return "tar.gz"
	case BUNDLE:
		return "bundle"
	}
	return "unknown"
}

func ToArchiveType(s string) ArchiveType {
	switch s {
	case "zip":
		return ZIP
	case "tar.gz":
		return TARGZ
	case "bundle":
		return BUNDLE
	}
	return 0
}

// CreateArchive create archive content to the target path
func (repo *Repository) CreateArchive(ctx context.Context, format ArchiveType, target io.Writer, usePrefix bool, commitID string) error {
	request := &gitalypb.GetArchiveRequest{
		Repository:      repo.GitalyRepo,
		CommitId:        commitID,
		Format:          castArchiveType(format),
		Path:            []byte("."),
		ElidePath:       true,
		IncludeLfsBlobs: true,
	}
	if usePrefix {
		request.Prefix = filepath.Base(strings.TrimSuffix(repo.Path, ".git")) + "/"
	}

	archive, err := repo.RepoClient.GetArchive(repo.Ctx, request)
	if err != nil {
		return fmt.Errorf("failed to get archive: %w", err)
	}
	canRead := true
	for canRead {
		archiveResp, err := archive.Recv()
		if err != nil && err != io.EOF {
			return fmt.Errorf("failed to receive archive: %w", err)
		}
		if archiveResp == nil {
			canRead = false
			continue
		}

		_, err = target.Write(archiveResp.GetData())
		if err != nil {
			return fmt.Errorf("failed to write archive: %w", err)
		}
	}
	return nil
}

func castArchiveType(archType ArchiveType) gitalypb.GetArchiveRequest_Format {
	switch archType {
	case ZIP:
		return gitalypb.GetArchiveRequest_ZIP
	case TARGZ:
		return gitalypb.GetArchiveRequest_TAR_GZ
	default:
		return gitalypb.GetArchiveRequest_ZIP
	}
}
