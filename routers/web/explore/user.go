// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package explore

import (
	"bytes"
	"fmt"
	//"code.gitea.io/gitea/models/role_model"
	"net/http"
	"strings"

	"code.gitea.io/gitea/models/organization"
	"code.gitea.io/gitea/models/role_model"
	"code.gitea.io/gitea/models/tenant"
	"code.gitea.io/gitea/modules/trace"
	"code.gitea.io/gitea/routers/utils"

	"code.gitea.io/gitea/models/db"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/base"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/sitemap"
	"code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/modules/util"
)

const (
	// tplExploreUsers explore users page template
	tplExploreUsers base.TplName = "explore/users"
)

// UserSearchDefaultSortType is the default sort type for user search
const (
	UserSearchDefaultSortType  = "recentupdate"
	UserSearchDefaultAdminSort = "alphabetically"
)

var nullByte = []byte{0x00}

func isKeywordValid(keyword string) bool {
	return !bytes.Contains([]byte(keyword), nullByte)
}

// RenderUserSearch render user search page
func RenderUserSearch(ctx *context.Context, opts *user_model.SearchUserOptions, tplName base.TplName) {
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

	opts.Page = int(ctx.ParamsInt64("idx"))
	isSitemap := ctx.Params("idx") != ""
	if opts.Page <= 1 {
		opts.Page = ctx.FormInt("page")
	}
	if opts.Page <= 1 {
		opts.Page = 1
	}

	if isSitemap {
		opts.PageSize = setting.UI.SitemapPagingNum
	}

	var (
		users   []*user_model.User
		count   int64
		err     error
		orderBy db.SearchOrderBy
	)

	// we can not set orderBy to `models.SearchOrderByXxx`, because there may be a JOIN in the statement, different tables may have the same name columns

	ctx.Data["SortType"] = ctx.FormString("sort")
	switch ctx.FormString("sort") {
	case "newest":
		orderBy = "`user`.id DESC"
	case "oldest":
		orderBy = "`user`.id ASC"
	case "leastupdate":
		orderBy = "`user`.updated_unix ASC"
	case "reversealphabetically":
		orderBy = "`user`.name DESC"
	case "lastlogin":
		orderBy = "`user`.last_login_unix ASC"
	case "reverselastlogin":
		orderBy = "`user`.last_login_unix DESC"
	case "alphabetically":
		orderBy = "`user`.name ASC"
	case "recentupdate":
		fallthrough
	default:
		// in case the sortType is not valid, we set it to recentupdate
		ctx.Data["SortType"] = "recentupdate"
		orderBy = "`user`.updated_unix DESC"
	}

	opts.Keyword = ctx.FormTrim("q")
	opts.OrderBy = orderBy
	// TenantWithRoleModeEnabled = true получаем проекты по тенатнам
	if setting.SourceControl.TenantWithRoleModeEnabled {
		// для страницы админской панели
		if strings.Contains(ctx.Link, "/admin/users") {
			opts.IsAminPanel = true
		}
		tenantID, errGetTenantID := role_model.GetUserTenantId(ctx, ctx.Doer.ID)
		if errGetTenantID != nil {
			log.Error("RenderUserSearch role_model.GetUserTenantId failed: %v", errGetTenantID)
			ctx.Error(http.StatusNotFound, fmt.Sprintf("RenderUserSearch role_model.GetUserTenantId failed: %v", errGetTenantID))
			return
		}
		tenantEntity, errGetTenant := tenant.GetTenantByID(ctx, tenantID)
		if errGetTenant != nil {
			log.Error("RenderUserSearch tenant.GetTenantByID failed: %v", errGetTenant)
			ctx.Error(http.StatusInternalServerError, "RenderUserSearch tenant.GetTenantByID failed")
			return
		}
		privilegesByTenantID, errGetPrivileges := role_model.GetPrivilegesByTenant(tenantID)
		if errGetPrivileges != nil {
			log.Error("RenderUserSearch role_model.GetPrivilegesByTenant failed: %v", errGetPrivileges)
			ctx.Error(http.StatusNotFound, fmt.Sprintf("RenderUserSearch role_model.GetPrivilegesByTenant failed: %v", errGetPrivileges))
			return
		}
		usersPrivileges, errGetTenantPrivilege := utils.GetTenantsPrivilegesByUserID(ctx, ctx.Doer.ID)
		if errGetTenantPrivilege != nil {
			log.Error("RenderUserSearch utils.GetTenantsPrivilegesByUserID failed: %v", errGetTenantPrivilege)
			ctx.Error(http.StatusNotFound, fmt.Sprintf("RenderUserSearch utils.GetUsersPrivilegesByUserID failed: %v", errGetTenantPrivilege))
			return
		}
		organizationsPrivileges := make(map[int64]struct{})
		organizationIDs := make([]int64, 0)
		switch tplName {
		case tplExploreOrganizations:
			organizationsPrivileges = utils.ConvertPrivilegesTenantFromOrganizationsOrUsers(usersPrivileges, user_model.UserTypeOrganization)
			organizations := utils.ConvertMapUserOrOrganizationsInSlice(organizationsPrivileges)
			for _, organizationID := range organizations {

				allowed, errCheckPermission := role_model.CheckUserPermissionToOrganization(ctx, ctx.Doer, tenantID, &organization.Organization{ID: organizationID}, role_model.READ)
				if errCheckPermission != nil {
					log.Error("RenderUserSearch role_model.CheckUserPermissionToOrganization failed: %v", errCheckPermission)
					ctx.Error(http.StatusNotFound, fmt.Sprintf("RenderUserSearch role_model.CheckUserPermissionToOrganization failed: %v", errCheckPermission))
					return
				}
				if allowed {
					organizationIDs = append(organizationIDs, organizationID)
				}
			}
		case tplExploreUsers:
			organizationsPrivileges = utils.ConvertPrivilegesTenantFromOrganizationsOrUsers(privilegesByTenantID, user_model.UserTypeIndividual)
			organizationsIDs := utils.ConvertMapUserOrOrganizationsInSlice(organizationsPrivileges)
			if tenantEntity.Default {
				organizationUniqueIDs := make(map[int64]struct{})
				for _, organizationID := range organizationIDs {
					organizationUniqueIDs[organizationID] = struct{}{}
				}
				allPrivileges, _ := role_model.GetAllPrivileges()
				orgIDS := make([]int64, 0)
				for _, p := range allPrivileges {
					if _, ok := organizationUniqueIDs[p.User.ID]; !ok {
						if p.TenantID != tenantID {
							orgIDS = append(orgIDS, p.User.ID)
						}
					}
				}
				orgIDs := append(organizationsIDs, orgIDS...)
				usersList, errGetUsersNotInUserIDs := user_model.GetUsersNotInUserIDs(orgIDs)
				if errGetUsersNotInUserIDs != nil {
					log.Error("RenderUserSearch user_model.GetUsersNotInUserIDs failed: %v", errGetUsersNotInUserIDs)
					ctx.Error(http.StatusNotFound, fmt.Sprintf("RenderUserSearch user_model.GetUsersNotInUserIDs failed: %v", errGetUsersNotInUserIDs))
					return
				}
				organizationIDs = append(organizationsIDs, usersList.GetUserIDs()...)
			} else {
				organizationIDs = organizationsIDs
			}
		}
		opts.UserIDs = organizationIDs
	}

	if len(opts.Keyword) == 0 || isKeywordValid(opts.Keyword) {
		users, count, err = user_model.SearchUsers(opts)
		if err != nil {
			ctx.ServerError("SearchUsers", err)
			return
		}
	}
	if isSitemap {
		m := sitemap.NewSitemap()
		for _, item := range users {
			m.Add(sitemap.URL{URL: item.HTMLURL(), LastMod: item.UpdatedUnix.AsTimePtr()})
		}
		ctx.Resp.Header().Set("Content-Type", "text/xml")
		if _, err := m.WriteTo(ctx.Resp); err != nil {
			log.Error("Failed writing sitemap: %v", err)
		}
		return
	}

	// Исключаем ТУЗа из выдачи
	if !opts.SearchWithTuz {
		var filteredUsers []*user_model.User
		for _, usr := range users {
			tuz, err := role_model.CheckIsUserTuz(usr.ID)
			if err != nil {
				log.Error("Error has occurred while checking is user tuz: %v", err)
				ctx.Error(http.StatusNotFound, fmt.Sprintf("Error has occurred while checking is user tuz: %v", err))
				return
			}
			if !tuz {
				filteredUsers = append(filteredUsers, usr)
			}
		}
		users = filteredUsers
	}

	ctx.Data["Keyword"] = opts.Keyword
	ctx.Data["Total"] = count
	ctx.Data["Users"] = users
	ctx.Data["UsersTwoFaStatus"] = user_model.UserList(users).GetTwoFaStatus()
	ctx.Data["ShowUserEmail"] = setting.UI.ShowUserEmail
	ctx.Data["IsRepoIndexerEnabled"] = setting.Indexer.RepoIndexerEnabled

	pager := context.NewPagination(int(count), opts.PageSize, opts.Page, 5)
	pager.SetDefaultParams(ctx)
	for paramKey, paramVal := range opts.ExtraParamStrings {
		pager.AddParamString(paramKey, paramVal)
	}
	ctx.Data["Page"] = pager

	ctx.HTML(http.StatusOK, tplName)
}

// Users render explore users page
func Users(ctx *context.Context) {
	if setting.Service.Explore.DisableUsersPage {
		ctx.Redirect(setting.AppSubURL + "/explore/repos")
		return
	}
	ctx.Data["Title"] = ctx.Tr("explore")
	ctx.Data["PageIsExplore"] = true
	ctx.Data["PageIsExploreUsers"] = true
	ctx.Data["IsRepoIndexerEnabled"] = setting.Indexer.RepoIndexerEnabled

	if ctx.FormString("sort") == "" {
		ctx.SetFormString("sort", UserSearchDefaultSortType)
	}

	RenderUserSearch(ctx, &user_model.SearchUserOptions{
		Actor:       ctx.Doer,
		Type:        user_model.UserTypeIndividual,
		ListOptions: db.ListOptions{PageSize: setting.UI.ExplorePagingNum},
		IsActive:    util.OptionalBoolTrue,
		Visible:     []structs.VisibleType{structs.VisibleTypePublic, structs.VisibleTypeLimited, structs.VisibleTypePrivate},
	}, tplExploreUsers)
}
