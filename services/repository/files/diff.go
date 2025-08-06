// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package files

import (
	contextDefault "context"
	"strings"

	"gitlab.com/gitlab-org/gitaly/v16/proto/go/gitalypb"

	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/setting"

	"code.gitea.io/gitea/services/gitdiff"
)

// GetDiffPreview создает и возвращает результат сравнения файла, который еще не зафиксирован.
// todo fix test
func GetDiffPreview(ctx *context.Context, repo *context.Repository, branch, treePath, content string, entry *git.TreeEntry) (*gitdiff.Diff, error) {
	if branch == "" {
		branch = repo.Repository.DefaultBranch
	}

	// Создаем временную репу в которой будем выполнять сравнение
	t, err := NewTemporaryUploadRepository(ctx, repo.Repository)
	if err != nil {
		return nil, err
	}
	defer t.Close()

	t.gitRepo = repo.GitRepo

	ctxWithCancel, cancel := contextDefault.WithCancel(repo.GitRepo.Ctx)
	defer cancel()
	// получаем первоначальное содержимое, которое редактировали
	treeEntry, err := repo.GitRepo.CommitClient.TreeEntry(ctxWithCancel, &gitalypb.TreeEntryRequest{
		Repository: &gitalypb.Repository{
			StorageName:   setting.Gitaly.MainServerName,
			RelativePath:  repo.Repository.RepoPath(),
			GlRepository:  repo.Repository.Name,
			GlProjectPath: repo.Repository.OwnerName,
		},
		Limit:    0,
		MaxSize:  0,
		Path:     []byte(treePath),
		Revision: entry.Ptree.ID.Byte(),
	})
	if err != nil {
		return nil, err
	}
	recv, err := treeEntry.Recv()
	if err != nil {
		return nil, err
	}

	// записываем первоначальное содержимое в индексацию временного репозитория
	// 1. Получаем хэш объекта
	// 2. Добавляем хэш объекта в индексацию
	oldObjectHash, err := t.HashObject(strings.NewReader(string(recv.Data)))
	if err != nil {
		return nil, err
	}
	if err := t.AddObjectToIndex("100644", oldObjectHash, treePath); err != nil {
		return nil, err
	}
	treeHash, err := t.WriteTree()
	if err != nil {
		return nil, err
	}

	// записываем новое содержимое в индексацию временного репозитория
	newObjectHash, err := t.HashObject(strings.NewReader(content))
	if err != nil {
		return nil, err
	}
	if err := t.AddObjectToIndex("100644", newObjectHash, treePath); err != nil {
		return nil, err
	}

	return t.DiffIndex(treeHash)
}
