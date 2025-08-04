// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

//go:build !gogit

package git

import (
	"context"

	"gitlab.com/gitlab-org/gitaly/v16/proto/go/gitalypb"
)

// GetRefsFiltered returns all references of the repository that matches patterm exactly or starting with.
func (repo *Repository) GetRefsFiltered(pattern string) ([]*Reference, error) {
	ctx, cancel := context.WithCancel(repo.Ctx)
	defer cancel()

	if pattern == "" {
		pattern = "*"
	}
	patterns := make([][]byte, 0, 1)
	patterns = append(patterns, []byte(pattern))

	listRefs, err := repo.RefClient.ListRefs(ctx, &gitalypb.ListRefsRequest{
		Repository: repo.GitalyRepo,
		Patterns:   patterns,
		Head:       true,
	})
	if err != nil {
		return nil, err
	}
	recv, err := listRefs.Recv()
	if err != nil {
		return nil, err
	}
	refs := make([]*Reference, 0)
	for _, v := range recv.References {
		refs = append(refs, &Reference{
			Name:   string(v.Name),
			repo:   repo,
			Object: MustIDFromString(v.Target),
			Type:   "",
		})
	}

	return refs, nil
}
