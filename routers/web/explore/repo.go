// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package explore

import (
	"fmt"
	"net/http"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/external_metric_counter"
	"code.gitea.io/gitea/models/internal_metric_counter"
	"code.gitea.io/gitea/models/organization"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/role_model"
	"code.gitea.io/gitea/models/tenant"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/base"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/sitemap"
	"code.gitea.io/gitea/modules/trace"
	"code.gitea.io/gitea/routers/utils"
)

const (
	// tplExploreRepos explore repositories page template
	tplExploreRepos        base.TplName = "explore/repos"
	relevantReposOnlyParam string       = "only_show_relevant"
	codeHubParam           string       = "code_hub"
)

// RepoSearchOptions when calling search repositories
type RepoSearchOptions struct {
	OwnerID          int64
	OwnerIDs         []int64
	Private          bool
	Restricted       bool
	PageSize         int
	OnlyShowRelevant bool
	OnlyShowCodeHub  bool
	TplName          base.TplName
}

// RenderRepoSearch render repositories search page
// This function is also used to render the Admin Repository Management page.
func (s Server) RenderRepoSearch(ctx *context.Context, opts *RepoSearchOptions) {
	// Sitemap index for sitemap paths
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

	page := int(ctx.ParamsInt64("idx"))
	isSitemap := ctx.Params("idx") != ""
	if page <= 1 {
		page = ctx.FormInt("page")
	}

	if page <= 0 {
		page = 1
	}

	if isSitemap {
		opts.PageSize = setting.UI.SitemapPagingNum
	}

	var (
		repos   []*repo_model.Repository
		count   int64
		err     error
		orderBy db.SearchOrderBy
	)

	ctx.Data["SortType"] = ctx.FormString("sort")
	switch ctx.FormString("sort") {
	case "newest":
		orderBy = db.SearchOrderByNewest
	case "oldest":
		orderBy = db.SearchOrderByOldest
	case "leastupdate":
		orderBy = db.SearchOrderByLeastUpdated
	case "reversealphabetically":
		orderBy = db.SearchOrderByAlphabeticallyReverse
	case "alphabetically":
		orderBy = db.SearchOrderByAlphabetically
	case "reversesize":
		orderBy = db.SearchOrderBySizeReverse
	case "size":
		orderBy = db.SearchOrderBySize
	case "moststars":
		orderBy = db.SearchOrderByStarsReverse
	case "feweststars":
		orderBy = db.SearchOrderByStars
	case "mostforks":
		orderBy = db.SearchOrderByForksReverse
	case "fewestforks":
		orderBy = db.SearchOrderByForks
	default:
		ctx.Data["SortType"] = "recentupdate"
		orderBy = db.SearchOrderByRecentUpdated
	}
	if opts.OwnerID != 0 && ctx.Data["TenantID"] == "" {
		tenantID, err := tenant.GetTenantByOrgIdOrDefault(ctx, opts.OwnerID)
		if err != nil {
			ctx.Error(http.StatusInternalServerError, err.Error())
			return
		}

		ctx.Data["TenantID"] = tenantID
	}

	keyword := ctx.FormTrim("q")

	ctx.Data["OnlyShowRelevant"] = opts.OnlyShowRelevant

	topicOnly := ctx.FormBool("topic")
	ctx.Data["TopicOnly"] = topicOnly

	language := ctx.FormTrim("language")
	ctx.Data["Language"] = language

	// TenantWithRoleModeEnabled = true получаем проекты по тенатнам
	allowRepoIDs := make([]int64, 0)
	if setting.SourceControl.TenantWithRoleModeEnabled {
		tenantID, errGetTenantID := role_model.GetUserTenantId(ctx, ctx.Doer.ID)
		if errGetTenantID != nil {
			log.Error("Error has occurred while getting tenant id: %v", errGetTenantID)
			ctx.Error(http.StatusNotFound, fmt.Sprintf("Error has occurred while getting tenant id: %v", errGetTenantID))
			return
		}
		privileges, errGetTenantPrivilege := utils.GetTenantsPrivilegesByUserID(ctx, ctx.Doer.ID)
		if errGetTenantPrivilege != nil {
			log.Error("Error has occurred while getting user's permissions: %v", errGetTenantPrivilege)
			ctx.Error(http.StatusNotFound, fmt.Sprintf("Error has occurred while getting user's permissions: %v", errGetTenantPrivilege))
			return
		}
		privilegeOrganizations := utils.ConvertPrivilegesTenantFromOrganizationsOrUsers(privileges, user_model.UserTypeOrganization)
		ownerIDs := make([]int64, 0)
		for orgPrivilegeID := range privilegeOrganizations {
			allowed, errCheckPermission := role_model.CheckUserPermissionToOrganization(ctx, ctx.Doer, tenantID, &organization.Organization{ID: orgPrivilegeID}, role_model.READ)
			if errCheckPermission != nil {
				log.Error("Error has occurred while checking user's permissions: %v", errCheckPermission)
				ctx.Error(http.StatusNotFound, fmt.Sprintf("Error has occurred while checking user's permissions: %v", errCheckPermission))
				return
			}
			repositories, err := organization.GetOrgRepositories(ctx, orgPrivilegeID)
			if err != nil {
				log.Error("Error has occurred while getting repositories: %v", err)
				ctx.Error(http.StatusNotFound, fmt.Sprintf("Error has occurred while getting repositories: %v", err))
				return
			}

			for _, repo := range repositories {
				action := role_model.READ
				if repo.IsPrivate {
					action = role_model.READ_PRIVATE
				}
				allowRoleModel, err := role_model.CheckUserPermissionToOrganization(ctx, &user_model.User{ID: ctx.Doer.ID}, tenantID, &organization.Organization{ID: orgPrivilegeID}, action)
				if err != nil {
					log.Error("Error has occurred while checking user's permissions: %v", err)
					ctx.Error(http.StatusNotFound, fmt.Sprintf("Error has occurred while checking user's permissions: %v", err))
					return
				}
				if !allowRoleModel {
					allow, err := role_model.CheckUserPermissionToTeam(ctx, &user_model.User{ID: ctx.Doer.ID}, tenantID, &organization.Organization{ID: orgPrivilegeID}, &repo_model.Repository{ID: repo.ID}, role_model.ViewBranch.String())
					if err != nil {
						log.Error("Error has occurred while checking user's permissions: %v", err)
						ctx.Error(http.StatusForbidden, fmt.Sprintf("Error has occurred while checking user's permissions: %v", err))
						return
					}
					if !allow {
						continue
					}
				}
				allowRepoIDs = append(allowRepoIDs, repo.ID)
			}
			if allowed {
				ownerIDs = append(ownerIDs, orgPrivilegeID)
			}
		}
		opts.OwnerIDs = ownerIDs
	}
	var isAdminPanel bool
	if setting.SourceControl.TenantWithRoleModeEnabled && ctx.Doer.IsAdmin {
		if ctx.Link == "/admin/repos" || ctx.Link == "/admin/orgs" || ctx.Link == "/admin/users" {
			isAdminPanel = true
		}
	}
	repos, count, err = repo_model.SearchRepository(ctx, &repo_model.SearchRepoOptions{
		ListOptions: db.ListOptions{
			Page:     page,
			PageSize: opts.PageSize,
		},
		Actor:              ctx.Doer,
		OrderBy:            orderBy,
		Private:            opts.Private,
		Keyword:            keyword,
		OwnerID:            opts.OwnerID,
		OwnerIDs:           opts.OwnerIDs,
		AllPublic:          true,
		AllLimited:         true,
		TopicOnly:          topicOnly,
		Language:           language,
		IncludeDescription: setting.UI.SearchRepoDescription,
		OnlyShowRelevant:   opts.OnlyShowRelevant,
		OnlyShowCodeHub:    opts.OnlyShowCodeHub,
		AdminPanel:         isAdminPanel,
		AllowedRepoIDs:     allowRepoIDs,
	})
	if err != nil {
		ctx.ServerError("SearchRepository", err)
		return
	}

	repoIDs := make([]int64, 0)
	if s.counterEnabled || s.marksEnabled {
		for _, repo := range repos {
			repoIDs = append(repoIDs, repo.ID)
		}
	}

	// Enriching with internal counter
	if s.counterEnabled {
		internalMetrics, err := s.GetInternalMetricCountersByRepoIDs(ctx, repoIDs)
		if err != nil {
			log.Error("Error has occurred while getting internal metrics counter: %v", err)
			ctx.ServerError("Fail to get internal metrics counter", err)
			return
		}
		internalResult := make(map[int64][]*internal_metric_counter.InternalMetricCounter)
		for _, metric := range internalMetrics {
			internalResult[metric.RepoID] = append(internalResult[metric.RepoID], metric)
		}

		externalMetrics, err := s.GetExternalMetricCountersByRepoIDs(ctx, repoIDs)
		if err != nil {
			log.Error("Error has occurred while getting external metrics counter: %v", err)
			ctx.ServerError("Fail to get external metrics counter", err)
			return
		}
		externalResult := make(map[int64]*external_metric_counter.ExternalMetricCounter, len(externalMetrics))
		for _, metric := range externalMetrics {
			externalResult[metric.RepoID] = metric
		}

		for _, repo := range repos {
			repo.InternalMetrics = internalResult[repo.ID]
			repo.ExternalMetric = externalResult[repo.ID]
		}
	}

	if s.marksEnabled {
		marksDef := make(map[string]string)

		for _, mark := range s.processedMarks {
			marksDef[mark.Key()] = mark.Label()
		}
		marks, err := s.GetRepoMarksByRepoIDs(ctx, repoIDs)
		if err != nil {
			log.Error("Error has occurred while getting repo marks: %v", err)
			ctx.ServerError("Fail to get marks", err)
			return
		}
		marksResult := make(map[int64][]*repo_model.Mark)
		for _, mark := range marks {
			if val, ok := marksDef[mark.MarkKey]; ok {
				marksResult[mark.RepoID] = append(marksResult[mark.RepoID], &repo_model.Mark{Label: val, ExpertID: mark.ExpertID})
			}
		}
		for _, repo := range repos {
			repo.RepoMarks = marksResult[repo.ID]
		}
	}

	if isSitemap {
		m := sitemap.NewSitemap()
		for _, item := range repos {
			m.Add(sitemap.URL{URL: item.HTMLURL(), LastMod: item.UpdatedUnix.AsTimePtr()})
		}
		ctx.Resp.Header().Set("Content-Type", "text/xml")
		if _, err := m.WriteTo(ctx.Resp); err != nil {
			log.Error("Failed writing sitemap: %v", err)
		}
		return
	}

	ctx.Data["Keyword"] = keyword
	ctx.Data["Total"] = count
	ctx.Data["Repos"] = repos
	ctx.Data["IsRepoIndexerEnabled"] = setting.Indexer.RepoIndexerEnabled

	pager := context.NewPagination(int(count), opts.PageSize, page, 5)
	pager.SetDefaultParams(ctx)
	pager.AddParam(ctx, "topic", "TopicOnly")
	pager.AddParam(ctx, "language", "Language")
	pager.AddParamString(relevantReposOnlyParam, fmt.Sprint(opts.OnlyShowRelevant))
	ctx.Data["Page"] = pager
	ctx.Data["CodeHub"] = opts.OnlyShowCodeHub

	ctx.HTML(http.StatusOK, opts.TplName)
}

// Repos render explore repositories page
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

	ctx.Data["UsersIsDisabled"] = setting.Service.Explore.DisableUsersPage
	ctx.Data["Title"] = ctx.Tr("explore")
	ctx.Data["PageIsExplore"] = true
	ctx.Data["PageIsExploreRepositories"] = true
	ctx.Data["IsRepoIndexerEnabled"] = setting.Indexer.RepoIndexerEnabled

	owner := ctx.Doer
	var ownerID int64
	if owner != nil && owner.IsOrganization() {
		ownerID = owner.ID
	}
	if setting.SourceControl.TenantWithRoleModeEnabled {
		var tenantID string
		var err error
		if owner != nil && owner.IsOrganization() {
			tenantID, err = tenant.GetTenantByOrgIdOrDefault(ctx, ownerID)
		} else {
			tenantID, err = role_model.GetUserTenantId(ctx, owner.ID)
			ownerID = owner.ID
		}
		if err != nil {
			ctx.Error(http.StatusInternalServerError, err.Error())
			return
		}
		ctx.Data["TenantID"] = tenantID
	}

	onlyShowRelevant := setting.UI.OnlyShowRelevantRepos
	codeHub := false

	_ = ctx.Req.ParseForm() // parse the form first, to prepare the ctx.Req.Form field
	if len(ctx.Req.Form[relevantReposOnlyParam]) != 0 {
		onlyShowRelevant = ctx.FormBool(relevantReposOnlyParam)
	}
	if len(ctx.Req.Form[codeHubParam]) != 0 {
		codeHub = ctx.FormBool(codeHubParam)
	}

	s.RenderRepoSearch(ctx, &RepoSearchOptions{
		PageSize:         setting.UI.ExplorePagingNum,
		OwnerID:          ownerID,
		Private:          ctx.Doer != nil,
		TplName:          tplExploreRepos,
		OnlyShowRelevant: onlyShowRelevant,
		OnlyShowCodeHub:  codeHub,
	})
}
