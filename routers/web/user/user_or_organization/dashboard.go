package user_or_organization

import (
	"net/http"

	activities_model "code.gitea.io/gitea/models/activities"
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/organization"
	"code.gitea.io/gitea/models/role_model"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/base"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/trace"
	"code.gitea.io/gitea/routers/utils"
	"code.gitea.io/gitea/routers/web/user/accesser"
)

const (
	tplDashboard base.TplName = "user/dashboard/dashboard"
)

// Dashboard render the dashboard page
func (s Server) Dashboard(ctx *context.Context) {
	logTracer := trace.NewLogTracer()
	message := logTracer.CreateTraceMessage(ctx)
	err := logTracer.Trace(message)
	if err != nil {
		log.Error("Error has occurred while creating trace message: %v", err)
	}
	defer func() {
		err = logTracer.TraceTime(message)
		if err != nil {
			log.Error("Error has occurred while creating trace time message: %v", err)
		}
	}()

	ctxUser := getDashboardContextUser(ctx)
	if ctx.Written() {
		return
	}

	var (
		date = ctx.FormString("date")
		page = ctx.FormInt("page")
	)

	// Make sure page number is at least 1. Will be posted to ctx.Data.
	if page <= 1 {
		page = 1
	}

	ctx.Data["Title"] = ctxUser.DisplayName() + " - " + ctx.Tr("dashboard")
	ctx.Data["PageIsDashboard"] = true
	ctx.Data["PageIsNews"] = true
	cnt := ctx.Data["OrgsCount"]
	ctx.Data["UserOrgsCount"] = cnt
	ctx.Data["MirrorsEnabled"] = setting.Mirror.Enabled
	ctx.Data["Date"] = date

	var uid int64
	if ctxUser != nil {
		uid = ctxUser.ID
	}

	ctx.PageData["dashboardRepoList"] = map[string]interface{}{
		"searchLimit": setting.UI.User.RepoPagingNum,
		"uid":         uid,
	}
	orgs := ctx.Data["Orgs"].([]*organization.MinimalOrg)
	organizationIDs := make([]int64, len(orgs))
	for idx, org := range orgs {
		organizationIDs[idx] = org.ID
	}

	// уникальные ids для проектов
	uniqueOrgIDs := make(map[int64]int)
	// доступные репозитории для пользователя
	allowedRepoIds := make([]int64, 0)
	if setting.SourceControl.TenantWithRoleModeEnabled {
		tenantId, err := role_model.GetUserTenantId(ctx, ctx.Doer.ID)
		if err != nil {
			log.Error("Error has occurred while getting tenant by user id: %v", err)
			ctx.ServerError("Error has occurred while getting tenant by user id: %v", err)
			return
		}

		for _, orgId := range organizationIDs {
			repos, err := organization.GetOrgRepositories(ctx, orgId)
			if err != nil {
				log.Error("Error has occurred while getting repositories by org id: %v", err)
				ctx.ServerError("Error has occurred while getting repositories by org id: %v", err)
				return
			}

			countRepos := 0
			for _, repo := range repos {
				action := role_model.READ
				if repo.IsPrivate {
					action = role_model.READ_PRIVATE
				}
				allowed, err := s.orgRequestAccesser.IsAccessGranted(*ctx, accesser.OrgAccessRequest{
					DoerID:         ctx.Doer.ID,
					TargetOrgID:    orgId,
					TargetTenantID: tenantId,
					Action:         action,
				})
				if err != nil {
					log.Error("Error has occurred while checking user's permissions: %v", err)
					ctx.ServerError("Error has occurred while checking user's permissions: %v", err)
					return
				}
				if !allowed {
					allow, err := s.repoRequestAccessor.AccessesByCustomPrivileges(*ctx, accesser.RepoAccessRequest{
						DoerID:          ctx.Doer.ID,
						OrgID:           orgId,
						TargetTenantID:  tenantId,
						RepoID:          repo.ID,
						CustomPrivilege: role_model.ViewBranch.String(),
					})
					if err != nil {
						log.Error("Error has occurred while checking user's permissions: %v", err)
						ctx.ServerError("Error has occurred while checking user's permissions: %v", err)
						return
					}
					if !allow {
						continue
					}
				}

				countRepos++
				allowedRepoIds = append(allowedRepoIds, repo.ID)

			}
			uniqueOrgIDs[orgId] = countRepos
		}
	}

	for idx := range orgs {
		orgs[idx].NumRepos = uniqueOrgIDs[orgs[idx].ID]
	}
	ctx.Data["Orgs"] = orgs

	if setting.Service.EnableUserHeatmap {
		data, err := activities_model.GetUserHeatmapDataByUserTeam(ctxUser, ctx.Org.Team, ctx.Doer, organizationIDs, allowedRepoIds)
		if err != nil {
			ctx.ServerError("GetUserHeatmapDataByUserTeam", err)
			return
		}
		ctx.Data["HeatmapData"] = data
		ctx.Data["HeatmapTotalContributions"] = activities_model.GetTotalContributionsInHeatmap(data)
	}

	feeds, count, err := activities_model.GetFeeds(ctx, activities_model.GetFeedsOptions{
		RequestedUser:   ctxUser,
		RequestedTeam:   ctx.Org.Team,
		Actor:           ctx.Doer,
		IncludePrivate:  true,
		OnlyPerformedBy: false,
		IncludeDeleted:  false,
		Date:            ctx.FormString("date"),
		ListOptions: db.ListOptions{
			Page:     page,
			PageSize: setting.UI.FeedPagingNum,
		},
		OrganizationIDs: organizationIDs,
		AllowedRepoIDs:  allowedRepoIds,
	})
	if err != nil {
		ctx.ServerError("GetFeeds", err)
		return
	}

	ctx.Data["Feeds"] = feeds

	pager := context.NewPagination(int(count), setting.UI.FeedPagingNum, page, 5)
	pager.AddParam(ctx, "date", "Date")
	ctx.Data["Page"] = pager

	ctx.HTML(http.StatusOK, tplDashboard)
}

// getDashboardContextUser finds out which context user dashboard is being viewed as .
func getDashboardContextUser(ctx *context.Context) *user_model.User {
	logTracer := trace.NewLogTracer()
	message := logTracer.CreateTraceMessage(ctx)
	err := logTracer.Trace(message)
	if err != nil {
		log.Error("Error has occurred while creating trace message: %v", err)
	}
	defer func() {
		err = logTracer.TraceTime(message)
		if err != nil {
			log.Error("Error has occurred while creating trace time message: %v", err)
		}
	}()

	ctxUser := ctx.Doer
	orgName := ctx.Params(":org")
	if len(orgName) > 0 {
		ctxUser = ctx.Org.Organization.AsUser()
		ctx.Data["Teams"] = ctx.Org.Teams
	}
	ctx.Data["ContextUser"] = ctxUser
	//signedUser := ctx.Data["SignedUser"].(*user_model.User)
	orgs, err := organization.GetUserOrgsList(ctx.Doer)
	if err != nil {
		ctx.ServerError("GetUserOrgsList", err)
		return nil
	}

	// TenantWithRoleModeEnabled = true выводим dashboard по проектам из тенантов
	if setting.SourceControl.TenantWithRoleModeEnabled {
		tenantID, errGetTenantIdToDoer := role_model.GetUserTenantId(ctx, ctx.Doer.ID)
		if errGetTenantIdToDoer != nil {
			log.Error("getDashboardContextUser role_model.GetUserTenantId failed: %v", errGetTenantIdToDoer)
			return nil
		}
		privileges, errGetUserByID := utils.GetTenantsPrivilegesByUserID(ctx, ctx.Doer.ID)
		if errGetUserByID != nil {
			log.Error("getDashboardContextUser utils.GetTenantsPrivilegesByUserID failed: %v", errGetUserByID)
			return nil
		}
		orgPrivilege := utils.ConvertTenantPrivilegesInOrganizations(privileges)
		organizations := make([]*organization.Organization, 0)
		for _, org := range orgPrivilege {
			allowed, errCheckPermission := role_model.CheckUserPermissionToOrganization(ctx, ctx.Doer, tenantID, org, role_model.READ)
			if errCheckPermission != nil {
				log.Error("getDashboardContextUser role_model.CheckUserPermissionToOrganization failed: %v", errCheckPermission)
				return nil
			}
			if allowed {
				organizations = append(organizations, org)
			}
		}
		orgs = organizations
	}
	ctx.Data["Orgs"] = orgs
	ctx.Data["OrgsCount"] = len(orgs)

	return ctxUser
}
