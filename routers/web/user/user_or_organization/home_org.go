package user_or_organization

import (
	"fmt"
	"net/http"
	"strings"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/organization"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/role_model"
	"code.gitea.io/gitea/models/tenant"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/base"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/markup"
	"code.gitea.io/gitea/modules/markup/markdown"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/trace"
	"code.gitea.io/gitea/routers/utils"
	"code.gitea.io/gitea/routers/web/user/accesser"
)

const (
	tplOrgHome base.TplName = "org/home"
)

// HomeOrg show organization home page
func (s Server) HomeOrg(ctx *context.Context) {
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

	uname := ctx.Params(":username")

	if strings.HasSuffix(uname, ".keys") || strings.HasSuffix(uname, ".gpg") {
		ctx.NotFound("", nil)
		return
	}

	ctx.SetParams(":org", uname)
	context.HandleOrgAssignment(ctx)
	if ctx.Written() {
		return
	}

	org := ctx.Org.Organization

	ctx.Data["PageIsUserProfile"] = true
	ctx.Data["Title"] = org.DisplayName()
	if len(org.Description) != 0 {
		desc, err := markdown.RenderString(&markup.RenderContext{
			Ctx:       ctx,
			URLPrefix: ctx.Repo.RepoLink,
			Metas:     map[string]string{"mode": "document"},
			GitRepo:   ctx.Repo.GitRepo,
		}, org.Description)
		if err != nil {
			ctx.ServerError("RenderString", err)
			return
		}
		ctx.Data["RenderedDescription"] = desc
	}

	var orderBy db.SearchOrderBy
	ctx.Data["SortType"] = ctx.FormString("sort")
	switch ctx.FormString("sort") {
	case "newest":
		orderBy = db.SearchOrderByNewest
	case "oldest":
		orderBy = db.SearchOrderByOldest
	case "recentupdate":
		orderBy = db.SearchOrderByRecentUpdated
	case "leastupdate":
		orderBy = db.SearchOrderByLeastUpdated
	case "reversealphabetically":
		orderBy = db.SearchOrderByAlphabeticallyReverse
	case "alphabetically":
		orderBy = db.SearchOrderByAlphabetically
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

	keyword := ctx.FormTrim("q")
	ctx.Data["Keyword"] = keyword

	language := ctx.FormTrim("language")
	ctx.Data["Language"] = language

	page := ctx.FormInt("page")
	if page <= 0 {
		page = 1
	}

	var (
		repos           []*repo_model.Repository
		count           int64
		organizationIDs []int64
	)

	allowedRepoIDs := make([]int64, 0)
	if setting.SourceControl.TenantWithRoleModeEnabled && ctx.ContextUser != nil && ctx.ContextUser.Type != user_model.UserTypeOrganization {

		privileges, errGetTenantsPrivileges := utils.GetTenantsPrivilegesByUserID(ctx, ctx.ContextUser.ID)
		if errGetTenantsPrivileges != nil {
			log.Error("Error has occurred while getting privileges: %v", errGetTenantsPrivileges)
			ctx.Error(http.StatusNotFound, fmt.Sprintf("Error has occurred while getting privileges: %v", errGetTenantsPrivileges))
			return
		}
		orgs := utils.ConvertTenantPrivilegesInOrganizations(privileges)
		organizationIDs = make([]int64, 0)
		for _, organizationEntity := range orgs {
			organizationIDs = append(organizationIDs, organizationEntity.ID)
		}
	} else {
		if setting.SourceControl.TenantWithRoleModeEnabled {
			tenantID, errGetTenantID := role_model.GetUserTenantId(ctx, ctx.Doer.ID)
			if errGetTenantID != nil {
				log.Error("Error has occurred while getting tenantID by user id %d: %v", ctx.Doer.ID, errGetTenantID)
				ctx.Error(http.StatusNotFound, fmt.Sprintf("Error has occurred while getting tenantID by user id %d: %v", ctx.Doer.ID, errGetTenantID))
				return
			}

			privileges, errGetTenantPrivilege := role_model.GetPrivilegesByOrgId(org.ID)
			if errGetTenantPrivilege != nil {
				log.Error("Error has occurred while getting privileges for org: %v", errGetTenantPrivilege)
				ctx.Error(http.StatusNotFound, fmt.Sprintf("Error has occurred while getting privileges for org: %v", errGetTenantPrivilege))
				return
			}

			privilegeOrganizations := utils.ConvertPrivilegesTenantFromOrganizationsOrUsers(privileges, user_model.UserTypeOrganization)
			uniqueOrgIDs := make(map[int64]struct{})
			for orgPrivilegeID := range privilegeOrganizations {
				repositories, err := organization.GetOrgRepositories(ctx, orgPrivilegeID)
				if err != nil {
					log.Error("Error has occurred while getting all repositories for org: %v", err)
					ctx.Error(http.StatusNotFound, fmt.Sprintf("Error has occurred while getting all repositories for org: %v", err))
					return
				}

				for _, repo := range repositories {
					action := role_model.READ
					if repo.IsPrivate {
						action = role_model.READ_PRIVATE
					}
					allowed, err := s.orgRequestAccesser.IsAccessGranted(*ctx, accesser.OrgAccessRequest{
						DoerID:         ctx.Doer.ID,
						TargetOrgID:    orgPrivilegeID,
						Action:         action,
						TargetTenantID: tenantID,
					})
					if err != nil {
						log.Error("Error has occurred while checking user's permissions: %v", err)
						ctx.Error(http.StatusNotFound, fmt.Sprintf("Error has occurred while checking user's permissions: %v", err))
						return
					}
					if !allowed {
						allow, err := s.repoRequestAccessor.AccessesByCustomPrivileges(*ctx, accesser.RepoAccessRequest{
							DoerID:          ctx.Doer.ID,
							OrgID:           orgPrivilegeID,
							RepoID:          repo.ID,
							CustomPrivilege: role_model.ViewBranch.String(),
							TargetTenantID:  tenantID,
						})
						if err != nil {
							log.Error("Error has occurred while checking user's permissions: %v", err)
							ctx.Error(http.StatusNotFound, fmt.Sprintf("Error has occurred while checking user's permissions: %v", err))
							return
						}
						if !allow {
							continue
						}
					}
					allowedRepoIDs = append(allowedRepoIDs, repo.ID)
				}
			}
			ownerIDs := make([]int64, 0, len(uniqueOrgIDs))
			for orgID := range uniqueOrgIDs {
				ownerIDs = append(ownerIDs, orgID)
			}
			organizationIDs = ownerIDs
		}
	}

	repos, count, err = repo_model.SearchRepository(ctx, &repo_model.SearchRepoOptions{
		ListOptions: db.ListOptions{
			PageSize: setting.UI.User.RepoPagingNum,
			Page:     page,
		},
		Keyword:            keyword,
		OwnerID:            org.ID,
		OrderBy:            orderBy,
		Private:            ctx.IsSigned,
		Actor:              ctx.Doer,
		Language:           language,
		IncludeDescription: setting.UI.SearchRepoDescription,
		OwnerIDs:           organizationIDs,
		AllowedRepoIDs:     allowedRepoIDs,
	})
	if err != nil {
		ctx.ServerError("SearchRepository", err)
		return
	}

	opts := &organization.FindOrgMembersOpts{
		OrgID:       org.ID,
		PublicOnly:  true,
		ListOptions: db.ListOptions{Page: 1, PageSize: 25},
	}

	if ctx.Doer != nil {
		isMember, err := org.IsOrgMember(ctx.Doer.ID)
		if err != nil {
			ctx.Error(http.StatusInternalServerError, "IsOrgMember")
			return
		}
		opts.PublicOnly = !isMember && !ctx.Doer.IsAdmin
	}

	members, _, err := organization.FindOrgMembers(opts)
	if err != nil {
		ctx.ServerError("FindOrgMembers", err)
		return
	}

	membersCount, err := organization.CountOrgMembers(opts)
	if err != nil {
		ctx.ServerError("CountOrgMembers", err)
		return
	}

	var isFollowing bool
	if ctx.Doer != nil {
		isFollowing = user_model.IsFollowing(ctx.Doer.ID, ctx.ContextUser.ID)
	}

	var visibleReposCount int
	var tenantId string

	if setting.SourceControl.TenantWithRoleModeEnabled {
		tenantId, err = tenant.GetTenantByOrgIdOrDefault(ctx, ctx.Org.Organization.ID)
		if err != nil {
			ctx.Error(http.StatusInternalServerError, err.Error())
			return
		}
		visibleReposCount = 0
		for _, repo := range repos {
			action := role_model.READ
			if repo.IsPrivate {
				action = role_model.READ_PRIVATE
			}
			allowed, err := s.orgRequestAccesser.IsAccessGranted(*ctx, accesser.OrgAccessRequest{
				DoerID:         ctx.Doer.ID,
				TargetOrgID:    ctx.Org.Organization.ID,
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
					OrgID:           ctx.Org.Organization.ID,
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
			visibleReposCount += 1
		}
		visibleReposCount = len(repos)
	}

	ctx.Data["VisibleReposCount"] = visibleReposCount
	ctx.Data["TenantID"] = tenantId
	ctx.Data["Repos"] = repos
	ctx.Data["Total"] = count
	ctx.Data["MembersTotal"] = membersCount
	ctx.Data["Members"] = members
	ctx.Data["Teams"] = ctx.Org.Teams
	ctx.Data["DisableNewPullMirrors"] = setting.Mirror.DisableNewPull
	ctx.Data["PageIsViewRepositories"] = true
	ctx.Data["IsFollowing"] = isFollowing

	pager := context.NewPagination(int(count), setting.UI.User.RepoPagingNum, page, 5)
	pager.SetDefaultParams(ctx)
	pager.AddParam(ctx, "language", "Language")
	ctx.Data["Page"] = pager
	ctx.Data["ContextUser"] = ctx.ContextUser

	ctx.HTML(http.StatusOK, tplOrgHome)
}
