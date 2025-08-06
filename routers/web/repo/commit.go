// Copyright 2014 The Gogs Authors. All rights reserved.
// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	files_service "code.gitea.io/gitea/services/repository/files"

	"code.gitea.io/gitea/modules/base"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
)

const (
	tplCommits  base.TplName = "repo/commits"
	tplGraph    base.TplName = "repo/graph"
	tplGraphDiv base.TplName = "repo/graph/div"
)

// Graph render commit graph - show commits from all branches.
func Graph(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("repo.commit_graph")
	ctx.Data["PageIsCommits"] = true
	ctx.Data["PageIsViewCode"] = true
	mode := strings.ToLower(ctx.FormTrim("mode"))
	if mode != "monochrome" {
		mode = "color"
	}
	ctx.Data["Mode"] = mode
	hidePRRefs := ctx.FormBool("hide-pr-refs")
	ctx.Data["HidePRRefs"] = hidePRRefs
	branches := ctx.FormStrings("branch")
	realBranches := make([]string, len(branches))
	copy(realBranches, branches)
	for i, branch := range realBranches {
		if strings.HasPrefix(branch, "--") {
			realBranches[i] = git.BranchPrefix + branch
		}
	}
	ctx.Data["SelectedBranches"] = realBranches
	files := ctx.FormStrings("file")

	commitsCount, err := ctx.Repo.GetCommitsCount()
	if err != nil {
		ctx.ServerError("GetCommitsCount", err)
		return
	}

	graphCommitsCount, err := ctx.Repo.GetCommitGraphsCount(ctx, hidePRRefs, realBranches, files)
	if err != nil {
		log.Warn("GetCommitGraphsCount error for generate graph exclude prs: %t branches: %s in %-v, Will Ignore branches and try again. Underlying Error: %v", hidePRRefs, branches, ctx.Repo.Repository, err)
		realBranches = []string{}
		branches = []string{}
		graphCommitsCount, err = ctx.Repo.GetCommitGraphsCount(ctx, hidePRRefs, realBranches, files)
		if err != nil {
			ctx.ServerError("GetCommitGraphsCount", err)
			return
		}
	}

	page := ctx.FormInt("page")

	// todo graph commit
	// Создаем временную репу в которой будем выполнять сравнение
	t, err := files_service.NewTemporaryUploadRepository(ctx, ctx.Repo.Repository)
	if err != nil {
		return
	}
	defer t.Close()

	err = t.Clone(ctx.Repo.Repository.DefaultBranch)
	if err != nil {
		return
	}

	gitRefs, err := ctx.Repo.GitRepo.GetRefs()
	if err != nil {
		ctx.ServerError("GitRepo.GetRefs", err)
		return
	}

	ctx.Data["AllRefs"] = gitRefs

	ctx.Data["Username"] = ctx.Repo.Owner.Name
	ctx.Data["Reponame"] = ctx.Repo.Repository.Name
	ctx.Data["CommitCount"] = commitsCount
	ctx.Data["RefName"] = ctx.Repo.RefName
	paginator := context.NewPagination(int(graphCommitsCount), setting.UI.GraphMaxCommitNum, page, 5)
	paginator.AddParam(ctx, "mode", "Mode")
	paginator.AddParam(ctx, "hide-pr-refs", "HidePRRefs")
	for _, branch := range branches {
		paginator.AddParamString("branch", branch)
	}
	for _, file := range files {
		paginator.AddParamString("file", file)
	}
	ctx.Data["Page"] = paginator
	if ctx.FormBool("div-only") {
		ctx.HTML(http.StatusOK, tplGraphDiv)
		return
	}

	ctx.HTML(http.StatusOK, tplGraph)
}

// RawDiff dumps diff results of repository in given commit ID to io.Writer
func RawDiff(ctx *context.Context) {
	var gitRepo *git.Repository
	if ctx.Data["PageIsWiki"] != nil {
		wikiRepo, err := git.OpenRepository(ctx, ctx.Repo.Repository.OwnerName, ctx.Repo.Repository.Name, ctx.Repo.Repository.WikiPath())
		if err != nil {
			ctx.ServerError("OpenRepository", err)
			return
		}
		defer wikiRepo.Close()
		gitRepo = wikiRepo
	} else {
		gitRepo = ctx.Repo.GitRepo
		if gitRepo == nil {
			ctx.ServerError("GitRepo not open", fmt.Errorf("no open git repo for '%s'", ctx.Repo.Repository.FullName()))
			return
		}
	}
	if err := git.GetRawDiff(
		gitRepo,
		ctx.Params(":sha"),
		git.RawDiffType(ctx.Params(":ext")),
		ctx.Resp,
	); err != nil {
		if git.IsErrNotExist(err) {
			ctx.NotFound("GetRawDiff",
				errors.New("commit "+ctx.Params(":sha")+" does not exist."))
			return
		}
		ctx.ServerError("GetRawDiff", err)
		return
	}
}
