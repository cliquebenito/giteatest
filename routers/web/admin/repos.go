// Copyright 2014 The Gogs Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package admin

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"code.gitea.io/gitea/modules/sbt/audit"
	"code.gitea.io/gitea/modules/trace"

	"code.gitea.io/gitea/models/db"
	repo_model "code.gitea.io/gitea/models/repo"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/base"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/log"
	repo_module "code.gitea.io/gitea/modules/repository"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/util"
	"code.gitea.io/gitea/routers/web/explore"
	repo_service "code.gitea.io/gitea/services/repository"
)

const (
	tplRepos          base.TplName = "admin/repo/list"
	tplUnadoptedRepos base.TplName = "admin/repo/unadopted"
)

// Repos show all the repositories
func (s Server) Repos(ctx *context.Context) {
	logTracer := trace.NewLogTracer()
	message := logTracer.CreateTraceMessage(ctx)
	errTrace := logTracer.Trace(message)
	if errTrace != nil {
		log.Error("Error has occurred while creating trace message: %v", errTrace)
	}
	defer func() {
		errTrace = logTracer.TraceTime(message)
		if errTrace != nil {
			log.Error("Error has occurred while creating trace time message: %v", errTrace)
		}
	}()

	ctx.Data["Title"] = ctx.Tr("admin.repositories")
	ctx.Data["PageIsAdminRepositories"] = true

	s.RenderRepoSearch(ctx, &explore.RepoSearchOptions{
		Private:          true,
		PageSize:         setting.UI.Admin.RepoPagingNum,
		TplName:          tplRepos,
		OnlyShowRelevant: false,
	})
}

// DeleteRepo delete one repository
func DeleteRepo(ctx *context.Context) {
	auditParams := map[string]string{
		"repository_id": strconv.FormatInt(ctx.FormInt64("id"), 10),
	}

	repo, err := repo_model.GetRepositoryByID(ctx, ctx.FormInt64("id"))
	if err != nil {
		ctx.ServerError("GetRepositoryByID", err)
		auditParams["error"] = "Error has occurred while getting repository by id"
		audit.CreateAndSendEvent(audit.RepositoryDeleteEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}

	auditParams["repository"] = repo.Name
	auditParams["owner"] = repo.OwnerName

	if ctx.Repo != nil && ctx.Repo.GitRepo != nil && ctx.Repo.Repository != nil && ctx.Repo.Repository.ID == repo.ID {
		ctx.Repo.GitRepo.Close()
	}

	if err := repo_service.DeleteRepository(ctx, ctx.Doer, repo, true); err != nil {
		ctx.ServerError("DeleteRepository", err)
		auditParams["error"] = "Error has occurred while deleting repository"
		audit.CreateAndSendEvent(audit.RepositoryDeleteEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}
	log.Trace("Repository deleted: %s", repo.FullName())

	audit.CreateAndSendEvent(audit.RepositoryDeleteEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)
	ctx.Flash.Success(ctx.Tr("repo.settings.deletion_success"))
	ctx.JSON(http.StatusOK, map[string]interface{}{
		"redirect": setting.AppSubURL + "/admin/repos?page=" + url.QueryEscape(ctx.FormString("page")) + "&sort=" + url.QueryEscape(ctx.FormString("sort")),
	})
}

// UnadoptedRepos lists the unadopted repositories
func UnadoptedRepos(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("admin.repositories")
	ctx.Data["PageIsAdminRepositories"] = true

	opts := db.ListOptions{
		PageSize: setting.UI.Admin.UserPagingNum,
		Page:     ctx.FormInt("page"),
	}

	if opts.Page <= 0 {
		opts.Page = 1
	}

	ctx.Data["CurrentPage"] = opts.Page

	doSearch := ctx.FormBool("search")

	ctx.Data["search"] = doSearch
	q := ctx.FormString("q")

	if !doSearch {
		pager := context.NewPagination(0, opts.PageSize, opts.Page, 5)
		pager.SetDefaultParams(ctx)
		pager.AddParam(ctx, "search", "search")
		ctx.Data["Page"] = pager
		ctx.HTML(http.StatusOK, tplUnadoptedRepos)
		return
	}

	ctx.Data["Keyword"] = q
	repoNames, count, err := repo_service.ListUnadoptedRepositories(ctx, q, &opts)
	if err != nil {
		ctx.ServerError("ListUnadoptedRepositories", err)
	}
	ctx.Data["Dirs"] = repoNames
	pager := context.NewPagination(count, opts.PageSize, opts.Page, 5)
	pager.SetDefaultParams(ctx)
	pager.AddParam(ctx, "search", "search")
	ctx.Data["Page"] = pager
	ctx.HTML(http.StatusOK, tplUnadoptedRepos)
}

// AdoptOrDeleteRepository adopts or deletes a repository
func AdoptOrDeleteRepository(ctx *context.Context) {
	dir := ctx.FormString("id")
	action := ctx.FormString("action")
	page := ctx.FormString("page")
	q := ctx.FormString("q")

	var event audit.Event
	if action == "adopt" {
		event = audit.RepositoryAdoptEvent
	} else if action == "delete" {
		event = audit.RepositoryDeleteEvent
	}

	dirSplit := strings.SplitN(dir, "/", 2)
	if len(dirSplit) != 2 {
		ctx.Redirect(setting.AppSubURL + "/admin/repos")
		auditParams := map[string]string{
			"error": "Incorrect identifier of repository",
		}
		audit.CreateAndSendEvent(event, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}

	auditParams := map[string]string{
		"owner":      dirSplit[0],
		"repository": dirSplit[1],
	}

	ctxUser, err := user_model.GetUserByName(ctx, dirSplit[0])
	if err != nil {
		if user_model.IsErrUserNotExist(err) {
			log.Debug("User does not exist: %s", dirSplit[0])
			ctx.Redirect(setting.AppSubURL + "/admin/repos")
			auditParams["error"] = "User does not exist"
			audit.CreateAndSendEvent(event, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}
		ctx.ServerError("GetUserByName", err)
		auditParams["error"] = "Error has occurred while getting user by name"
		audit.CreateAndSendEvent(event, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}

	repoName := dirSplit[1]

	// check not a repo
	has, err := repo_model.IsRepositoryModelExist(ctx, ctxUser, repoName)
	if err != nil {
		ctx.ServerError("IsRepositoryExist", err)
		auditParams["error"] = "Error has occurred while checking repository model exist"
		audit.CreateAndSendEvent(event, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}
	isDir, err := util.IsDir(repo_model.RepoPath(ctxUser.Name, repoName))
	if err != nil {
		ctx.ServerError("IsDir", err)
		auditParams["error"] = "Error has occurred while checking repository directory"
		audit.CreateAndSendEvent(event, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}
	if has || !isDir {
		// Fallthrough to failure mode
	} else if action == "adopt" {
		if _, err := repo_service.AdoptRepository(ctx, ctx.Doer, ctxUser, repo_module.CreateRepoOptions{
			Name:      dirSplit[1],
			IsPrivate: true,
		}); err != nil {
			ctx.ServerError("repository.AdoptRepository", err)
			auditParams["error"] = "Error has occurred while adopting repository"
			audit.CreateAndSendEvent(event, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}
		audit.CreateAndSendEvent(event, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)
		ctx.Flash.Success(ctx.Tr("repo.adopt_preexisting_success", dir))
	} else if action == "delete" {
		if err := repo_service.DeleteUnadoptedRepository(ctx, ctx.Doer, ctxUser, dirSplit[1]); err != nil {
			ctx.ServerError("repository.AdoptRepository", err)
			auditParams["error"] = "Error has occurred while deleting repository"
			audit.CreateAndSendEvent(event, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}
		audit.CreateAndSendEvent(event, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)
		ctx.Flash.Success(ctx.Tr("repo.delete_preexisting_success", dir))
	}
	ctx.Redirect(setting.AppSubURL + "/admin/repos/unadopted?search=true&q=" + url.QueryEscape(q) + "&page=" + url.QueryEscape(page))
}
