// Copyright 2015 The Gogs Authors. All rights reserved.
// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

//go:build !gogit

package git

import (
	"strings"

	"gitlab.com/gitlab-org/gitaly/v16/proto/go/gitalypb"

	"code.gitea.io/gitea/modules/log"
)

// IsTagExist returns true if given tag exists in the repository.
func (repo *Repository) IsTagExist(name string) bool {
	if repo == nil || name == "" {
		return false
	}

	return repo.IsReferenceExist(TagPrefix + name)
}

// GetTags returns all tags of the repository.
// returning at most limit tags, or all if limit is 0.
func (repo *Repository) GetTags(skip, limit int) (tags []string, err error) {
	tags, _, err = callShowRef(repo.Ctx, repo.Path, TagPrefix, TrustedCmdArgs{TagPrefix, "--sort=-taggerdate"}, skip, limit)
	return tags, err
}

// GetTagType gets the type of the tag, either commit (simple) or tag (annotated)
func (repo *Repository) GetTagType(id SHA1) (string, error) {
	wr, rd, cancel := repo.CatFileBatchCheck(repo.Ctx)
	defer cancel()
	_, err := wr.Write([]byte(id.String() + "\n"))
	if err != nil {
		return "", err
	}
	_, typ, _, err := ReadBatchLine(rd)
	if IsErrNotExist(err) {
		return "", ErrNotExist{ID: id.String()}
	}
	return typ, nil
}

func (repo *Repository) getTag(tagID SHA1, name string) (*Tag, error) {
	t, ok := repo.tagCache.Get(tagID.String())
	if ok {
		log.Debug("Hit cache: %s", tagID)
		tagClone := *t.(*Tag)
		tagClone.Name = name // This is necessary because lightweight tags may have same id
		return &tagClone, nil
	}

	findTagResponse, err := repo.RefClient.FindTag(repo.Ctx, &gitalypb.FindTagRequest{
		Repository: repo.GitalyRepo,
		TagName:    []byte(name),
	})
	if err != nil {
		if (!strings.Contains(err.Error(), "tag does not exist")) || findTagResponse == nil {
			return nil, ErrNotExist{ID: name}
		}
		return nil, err
	}
	gitalyTag := findTagResponse.GetTag()

	commitID, err := NewIDFromString(gitalyTag.GetTargetCommit().GetId())
	if err != nil {
		return nil, err
	}

	tag := &Tag{
		Name:    string(gitalyTag.GetName()),
		ID:      tagID,
		Object:  commitID,
		Tagger:  &Signature{Name: string(gitalyTag.GetTagger().GetName()), Email: string(gitalyTag.GetTagger().GetEmail()), When: gitalyTag.GetTagger().GetDate().AsTime()},
		Message: string(gitalyTag.GetMessage()),
	}

	if gitalyTag.GetMessageSize() != 0 {
		tag.Type = string(ObjectTag.Bytes())
	} else {
		tag.Type = string(ObjectCommit.Bytes())
	}

	repo.tagCache.Set(tagID.String(), tag)
	return tag, nil
}
