package user_or_organization

import (
	"fmt"
	"net/http"
	"strings"

	activities_model "code.gitea.io/gitea/models/activities"
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/organization"
	project_model "code.gitea.io/gitea/models/project"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/role_model"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/base"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/markup"
	"code.gitea.io/gitea/modules/markup/markdown"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/trace"
	"code.gitea.io/gitea/modules/util"
	"code.gitea.io/gitea/routers/utils"
	"code.gitea.io/gitea/routers/web/feed"
	"code.gitea.io/gitea/routers/web/user/accesser"
)

const (
	tplProfile base.TplName = "user/profile"
)

type TabHandler interface {
	Handle(ctx *context.Context)
}

func (s Server) Profile(ctx *context.Context) {
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

	if s.handleEarlyExitConditions(ctx) {
		return
	}

	profileData, err := s.prepareProfileData(ctx)
	if err != nil {
		return
	}

	tabHandler := s.getTabHandler(profileData)
	tabHandler.Handle(ctx)

	s.renderProfile(ctx)
}

type ProfileData struct {
	ContextUser    *user_model.User
	Doer           *user_model.User
	Organizations  []*organization.Organization
	AllowedRepoIDs []int64
	ShowPrivate    bool
	Tab            string
	Page           int
	PagingNum      int
}

// Обработчики условий раннего выхода
func (s Server) handleEarlyExitConditions(ctx *context.Context) bool {
	if handleFeedRequests(ctx) {
		return true
	}

	if ctx.ContextUser.IsOrganization() {
		s.HomeOrg(ctx)
		return true
	}

	if !user_model.IsUserVisibleToViewer(ctx, ctx.ContextUser, ctx.Doer) {
		ctx.NotFound("user", fmt.Errorf(ctx.ContextUser.Name))
		return true
	}

	return false
}

// Подготовка данных профиля
func (s Server) prepareProfileData(ctx *context.Context) (*ProfileData, error) {
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

	data := &ProfileData{
		ContextUser: ctx.ContextUser,
		Doer:        ctx.Doer,
		ShowPrivate: ctx.IsSigned && (ctx.Doer.IsAdmin || ctx.Doer.ID == ctx.ContextUser.ID),
		Tab:         ctx.FormString("tab"),
		Page:        ctx.FormInt("page"),
	}

	if data.Page <= 0 {
		data.Page = 1
	}

	// Инициализация базовых данных профиля
	if err := s.initBasicProfileData(ctx, data); err != nil {
		return nil, err
	}

	// Получение организаций
	orgs, err := s.getUserOrganizations(ctx, data.ShowPrivate)
	if err != nil {
		return nil, err
	}
	data.Organizations = orgs

	// Получение разрешенных репозиториев
	if setting.SourceControl.TenantWithRoleModeEnabled {
		data.AllowedRepoIDs = s.getAllowedRepoIDs(ctx, getOrganizationIDs(orgs), data.ShowPrivate)
	}

	// Установка дополнительных данных
	if err := s.setAdditionalProfileData(ctx, data); err != nil {
		return nil, err
	}

	return data, nil
}

// Получение обработчика вкладки
func (s Server) getTabHandler(data *ProfileData) TabHandler {
	switch data.Tab {
	case "followers":
		return &FollowersTabHandler{data: data}
	case "following":
		return &FollowingTabHandler{data: data}
	case "activity":
		return &ActivityTabHandler{data: data}
	case "stars":
		return &StarsTabHandler{data: data}
	case "projects":
		return &ProjectsTabHandler{data: data}
	case "watching":
		return &WatchingTabHandler{data: data}
	default:
		return &RepositoriesTabHandler{data: data}
	}
}

type FollowersTabHandler struct {
	data *ProfileData
}

func (h *FollowersTabHandler) Handle(ctx *context.Context) {
	followers, count, err := user_model.GetUserFollowers(ctx, h.data.ContextUser, h.data.Doer,
		db.ListOptions{PageSize: h.data.PagingNum, Page: h.data.Page})
	if err != nil {
		ctx.ServerError("GetUserFollowers", err)
		return
	}

	ctx.Data["Cards"] = followers
	ctx.Data["Total"] = int(count)
	ctx.Data["NumFollowers"] = count
}

// renderProfile выполняет финальный рендеринг страницы профиля
func (s Server) renderProfile(ctx *context.Context) {
	// Безопасное получение параметров пагинации
	tab, _ := ctx.Data["TabName"].(string)
	total, _ := ctx.Data["Total"].(int)

	// Установка значения по умолчанию для pagingNum
	pagingNum := setting.UI.User.RepoPagingNum
	if ctxPagingNum, ok := ctx.Data["PagingNum"].(int); ok {
		pagingNum = ctxPagingNum
	}

	// Установка значения по умолчанию для page
	page := 1
	if ctxPage, ok := ctx.Data["Page"].(int); ok {
		page = ctxPage
	}

	pager := context.NewPagination(total, pagingNum, page, 5)
	pager.SetDefaultParams(ctx)
	pager.AddParam(ctx, "tab", "TabName")

	if tab != "followers" && tab != "following" && tab != "activity" && tab != "projects" {
		pager.AddParam(ctx, "language", "Language")
	}
	if tab == "activity" {
		pager.AddParam(ctx, "date", "Date")
	}

	ctx.Data["Page"] = pager
	ctx.Data["IsProjectEnabled"] = true
	ctx.Data["IsPackageEnabled"] = setting.Packages.Enabled
	ctx.Data["IsRepoIndexerEnabled"] = setting.Indexer.RepoIndexerEnabled

	showEmail := setting.UI.ShowUserEmail &&
		ctx.ContextUser.Email != "" &&
		ctx.IsSigned &&
		!ctx.ContextUser.KeepEmailPrivate
	ctx.Data["ShowUserEmail"] = showEmail

	ctx.HTML(http.StatusOK, tplProfile)
}

// handleFeedRequests проверяет запросы RSS/Atom и обрабатывает их
func handleFeedRequests(ctx *context.Context) bool {
	if strings.Contains(ctx.Req.Header.Get("Accept"), "application/rss+xml") {
		feed.ShowUserFeedRSS(ctx)
		return true
	}
	if strings.Contains(ctx.Req.Header.Get("Accept"), "application/atom+xml") {
		feed.ShowUserFeedAtom(ctx)
		return true
	}
	return false
}

func (s Server) initBasicProfileData(ctx *context.Context, data *ProfileData) error {
	ctx.Data["FeedURL"] = data.ContextUser.HomeLink()
	ctx.Data["Title"] = data.ContextUser.DisplayName()
	ctx.Data["PageIsUserProfile"] = true
	ctx.Data["ContextUser"] = data.ContextUser
	ctx.Data["TabName"] = data.Tab

	// OpenID URIs
	openIDs, err := user_model.GetUserOpenIDs(data.ContextUser.ID)
	if err != nil {
		ctx.ServerError("GetUserOpenIDs", err)
		return err
	}
	ctx.Data["OpenIDs"] = openIDs

	// Following status
	var isFollowing bool
	if data.Doer != nil {
		isFollowing = user_model.IsFollowing(data.Doer.ID, data.ContextUser.ID)
	}
	ctx.Data["IsFollowing"] = isFollowing

	return nil
}

// getUserOrganizations получает организации пользователя
func (s Server) getUserOrganizations(ctx *context.Context, showPrivate bool) ([]*organization.Organization, error) {
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

	if setting.SourceControl.TenantWithRoleModeEnabled {
		return s.getTenantAwareOrganizations(ctx)
	}
	return organization.FindOrgs(organization.FindOrgOptions{
		UserID:         ctx.ContextUser.ID,
		IncludePrivate: showPrivate,
	})
}

// getOrganizationIDs возвращает ID организаций
func getOrganizationIDs(orgs []*organization.Organization) []int64 {
	ids := make([]int64, len(orgs))
	for i, org := range orgs {
		ids[i] = org.ID
	}
	return ids
}

// setAdditionalProfileData устанавливает дополнительные данные профиля
func (s Server) setAdditionalProfileData(ctx *context.Context, data *ProfileData) error {
	// Установка данных организаций
	ctx.Data["Orgs"] = data.Organizations
	ctx.Data["HasOrgsVisible"] = organization.HasOrgsVisible(data.Organizations, data.Doer)

	// Heatmap data
	if setting.Service.EnableUserHeatmap {
		heatmapData, err := activities_model.GetUserHeatmapDataByUser(
			data.ContextUser, data.Doer, getOrganizationIDs(data.Organizations), data.AllowedRepoIDs)
		if err != nil {
			ctx.ServerError("GetUserHeatmapDataByUser", err)
			return err
		}
		ctx.Data["HeatmapData"] = heatmapData
		ctx.Data["HeatmapTotalContributions"] = activities_model.GetTotalContributionsInHeatmap(heatmapData)
	}

	// Profile description
	if len(data.ContextUser.Description) > 0 {
		content, err := markdown.RenderString(&markup.RenderContext{
			URLPrefix: ctx.Repo.RepoLink,
			Metas:     map[string]string{"mode": "document"},
			GitRepo:   ctx.Repo.GitRepo,
			Ctx:       ctx,
		}, data.ContextUser.Description)
		if err != nil {
			ctx.ServerError("RenderString", err)
			return err
		}
		ctx.Data["RenderedDescription"] = content
	}

	// Profile README
	repo, err := repo_model.GetRepositoryByName(data.ContextUser.ID, ".profile")
	if err == nil && !repo.IsEmpty {
		gitRepo, err := git.OpenRepository(ctx, repo.OwnerName, repo.Name, repo.RepoPath())
		if err != nil {
			ctx.ServerError("OpenRepository", err)
			return err
		}
		defer gitRepo.Close()

		commit, err := gitRepo.GetBranchCommit(repo.DefaultBranch)
		if err != nil {
			ctx.ServerError("GetBranchCommit", err)
			return err
		}

		blob, err := commit.GetBlobByPath("README.md")
		if err == nil {
			bytes, err := blob.GetBlobContent()
			if err != nil {
				ctx.ServerError("GetBlobContent", err)
				return err
			}

			profileContent, err := markdown.RenderString(&markup.RenderContext{
				Ctx:     ctx,
				GitRepo: gitRepo,
			}, bytes)
			if err != nil {
				ctx.ServerError("RenderString", err)
				return err
			}
			ctx.Data["ProfileReadme"] = profileContent
		}
	}

	// Badges
	badges, _, err := user_model.GetUserBadges(ctx, data.ContextUser)
	if err != nil {
		ctx.ServerError("GetUserBadges", err)
		return err
	}
	ctx.Data["Badges"] = badges

	return nil
}

// Реализации обработчиков вкладок

type FollowingTabHandler struct {
	data *ProfileData
}

func (h *FollowingTabHandler) Handle(ctx *context.Context) {
	following, count, err := user_model.GetUserFollowing(ctx, h.data.ContextUser, h.data.Doer,
		db.ListOptions{PageSize: h.data.PagingNum, Page: h.data.Page})
	if err != nil {
		ctx.ServerError("GetUserFollowing", err)
		return
	}

	ctx.Data["Cards"] = following
	ctx.Data["Total"] = int(count)
	ctx.Data["NumFollowing"] = count
}

type ActivityTabHandler struct {
	data *ProfileData
}

func (h *ActivityTabHandler) Handle(ctx *context.Context) {
	date := ctx.FormString("date")
	items, count, err := activities_model.GetFeeds(ctx, activities_model.GetFeedsOptions{
		RequestedUser:   h.data.ContextUser,
		Actor:           h.data.Doer,
		IncludePrivate:  h.data.ShowPrivate,
		OnlyPerformedBy: true,
		IncludeDeleted:  false,
		Date:            date,
		OrganizationIDs: getOrganizationIDs(h.data.Organizations),
		ListOptions: db.ListOptions{
			PageSize: h.data.PagingNum,
			Page:     h.data.Page,
		},
		AllowedRepoIDs: h.data.AllowedRepoIDs,
	})
	if err != nil {
		ctx.ServerError("GetFeeds", err)
		return
	}

	if setting.SourceControl.TenantWithRoleModeEnabled {
		ctx.ContextUser.NumRepos = len(h.data.AllowedRepoIDs)
	}

	ctx.Data["Feeds"] = items
	ctx.Data["Date"] = date
	ctx.Data["Total"] = int(count)
}

type StarsTabHandler struct {
	data *ProfileData
}

func (h *StarsTabHandler) Handle(ctx *context.Context) {
	ctx.Data["PageIsProfileStarList"] = true

	keyword := ctx.FormTrim("q")
	language := ctx.FormTrim("language")
	topicOnly := ctx.FormBool("topic")

	repos, count, err := repo_model.SearchRepository(ctx, &repo_model.SearchRepoOptions{
		ListOptions: db.ListOptions{
			PageSize: h.data.PagingNum,
			Page:     h.data.Page,
		},
		Actor:              h.data.Doer,
		Keyword:            keyword,
		OrderBy:            getSearchOrderBy(ctx),
		Private:            ctx.IsSigned,
		StarredByID:        h.data.ContextUser.ID,
		Collaborate:        util.OptionalBoolFalse,
		TopicOnly:          topicOnly,
		Language:           language,
		IncludeDescription: setting.UI.SearchRepoDescription,
		OwnerIDs:           getOrganizationIDs(h.data.Organizations),
		AllowedRepoIDs:     h.data.AllowedRepoIDs,
	})
	if err != nil {
		ctx.ServerError("SearchRepository", err)
		return
	}

	if setting.SourceControl.TenantWithRoleModeEnabled {
		ctx.ContextUser.NumRepos = len(h.data.AllowedRepoIDs)
		ctx.ContextUser.NumStars = int(count)
	}

	ctx.Data["Repos"] = repos
	ctx.Data["Total"] = int(count)
}

type ProjectsTabHandler struct {
	data *ProfileData
}

func (h *ProjectsTabHandler) Handle(ctx *context.Context) {
	projects, _, err := project_model.FindProjects(ctx, project_model.SearchOptions{
		Page:     -1,
		IsClosed: util.OptionalBoolFalse,
		Type:     project_model.TypeIndividual,
	})
	if err != nil {
		ctx.ServerError("GetProjects", err)
		return
	}
	ctx.Data["OpenProjects"] = projects
}

type WatchingTabHandler struct {
	data *ProfileData
}

func (h *WatchingTabHandler) Handle(ctx *context.Context) {
	keyword := ctx.FormTrim("q")
	language := ctx.FormTrim("language")
	topicOnly := ctx.FormBool("topic")

	repos, count, err := repo_model.SearchRepository(ctx, &repo_model.SearchRepoOptions{
		ListOptions: db.ListOptions{
			PageSize: h.data.PagingNum,
			Page:     h.data.Page,
		},
		Actor:              h.data.Doer,
		Keyword:            keyword,
		OrderBy:            getSearchOrderBy(ctx),
		Private:            ctx.IsSigned,
		WatchedByID:        h.data.ContextUser.ID,
		Collaborate:        util.OptionalBoolFalse,
		TopicOnly:          topicOnly,
		Language:           language,
		IncludeDescription: setting.UI.SearchRepoDescription,
		OwnerIDs:           getOrganizationIDs(h.data.Organizations),
		AllowedRepoIDs:     h.data.AllowedRepoIDs,
	})
	if err != nil {
		ctx.ServerError("SearchRepository", err)
		return
	}

	if setting.SourceControl.TenantWithRoleModeEnabled {
		ctx.Data["Total"] = int(count)
	}

	ctx.Data["Repos"] = repos
	ctx.Data["Total"] = int(count)
}

type RepositoriesTabHandler struct {
	data *ProfileData
}

func (h *RepositoriesTabHandler) Handle(ctx *context.Context) {
	keyword := ctx.FormTrim("q")
	language := ctx.FormTrim("language")
	topicOnly := ctx.FormBool("topic")

	repos, count, err := repo_model.SearchRepository(ctx, &repo_model.SearchRepoOptions{
		ListOptions: db.ListOptions{
			PageSize: h.data.PagingNum,
			Page:     h.data.Page,
		},
		Actor:              h.data.Doer,
		Keyword:            keyword,
		OwnerID:            h.data.ContextUser.ID,
		OrderBy:            getSearchOrderBy(ctx),
		Private:            ctx.IsSigned,
		Collaborate:        util.OptionalBoolFalse,
		TopicOnly:          topicOnly,
		Language:           language,
		IncludeDescription: setting.UI.SearchRepoDescription,
		OwnerIDs:           getOrganizationIDs(h.data.Organizations),
		AllowedRepoIDs:     h.data.AllowedRepoIDs,
	})
	if err != nil {
		ctx.ServerError("SearchRepository", err)
		return
	}

	if setting.SourceControl.TenantWithRoleModeEnabled {
		countStars := 0
		for _, repo := range repos {
			stargazers, err := repo_model.GetStargazers(repo, db.ListOptions{})
			if err != nil {
				log.Error("Error getting stars: %v", err)
				continue
			}
			for _, user := range stargazers {
				if user.ID == h.data.ContextUser.ID {
					countStars++
				}
			}
		}
		ctx.ContextUser.NumStars = countStars
		ctx.ContextUser.NumRepos = int(count)
	}

	ctx.Data["Repos"] = repos
	ctx.Data["Total"] = int(count)
}

// getSearchOrderBy возвращает порядок сортировки для поиска
func getSearchOrderBy(ctx *context.Context) db.SearchOrderBy {
	switch ctx.FormString("sort") {
	case "newest":
		return db.SearchOrderByNewest
	case "oldest":
		return db.SearchOrderByOldest
	case "reversealphabetically":
		return db.SearchOrderByAlphabeticallyReverse
	case "alphabetically":
		return db.SearchOrderByAlphabetically
	case "moststars":
		return db.SearchOrderByStarsReverse
	case "feweststars":
		return db.SearchOrderByStars
	case "mostforks":
		return db.SearchOrderByForksReverse
	case "fewestforks":
		return db.SearchOrderByForks
	default:
		ctx.Data["SortType"] = "recentupdate"
		return db.SearchOrderByRecentUpdated
	}
}

func (s Server) getAllowedRepoIDs(ctx *context.Context, organizationIDs []int64, showPrivate bool) []int64 {
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

	tenantId, err := role_model.GetUserTenantId(ctx, ctx.Doer.ID)
	if err != nil {
		log.Error("Error getting tenant by user id: %v", err)
		return nil
	}

	var allowedRepoIds []int64
	for _, orgId := range organizationIDs {
		repos, err := organization.GetOrgRepositories(ctx, orgId)
		if err != nil {
			log.Error("Error getting repositories by orgId %v: %v", orgId, err)
			continue
		}

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
				log.Error("Error checking permissions: %v", err)
				continue
			}

			if !allowed {
				allow, err := s.repoRequestAccessor.AccessesByCustomPrivileges(*ctx, accesser.RepoAccessRequest{
					DoerID:          ctx.Doer.ID,
					OrgID:           orgId,
					TargetTenantID:  tenantId,
					RepoID:          repo.ID,
					CustomPrivilege: role_model.ViewBranch.String(),
				})
				if err != nil || !allow {
					continue
				}
			}
			allowedRepoIds = append(allowedRepoIds, repo.ID)
		}
	}
	return allowedRepoIds
}

func (s Server) getTenantAwareOrganizations(ctx *context.Context) ([]*organization.Organization, error) {
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

	doerTenantIDs, err := role_model.GetUserTenantIDsOrDefaultTenantID(ctx.Doer)
	if err != nil {
		log.Error("Profile role_model.GetUserTenantIds failed: %v", err)
		return nil, err
	}

	contextUserTenantIDs, err := role_model.GetUserTenantIDsOrDefaultTenantID(ctx.ContextUser)
	if err != nil {
		log.Error("Profile role_model.GetUserTenantIds failed: %v", err)
		return nil, err
	}

	var commonTenantIDs []string
	if !ctx.Doer.IsAdmin {
		for _, doerTenantID := range doerTenantIDs {
			for _, contextUserTenantID := range contextUserTenantIDs {
				if doerTenantID == contextUserTenantID {
					commonTenantIDs = append(commonTenantIDs, doerTenantID)
				}
			}
		}
	} else {
		commonTenantIDs = doerTenantIDs
	}

	if len(commonTenantIDs) == 0 {
		return nil, fmt.Errorf("no common tenants found")
	}

	var organizations []*organization.Organization
	for _, commonTenantID := range commonTenantIDs {
		privileges, err := utils.GetTenantsPrivilegesByUserID(ctx, ctx.Doer.ID)
		if err != nil {
			log.Error("Error getting user's privileges: %v", err)
			return nil, err
		}

		orgPrivileges := utils.ConvertTenantPrivilegesInOrganizations(privileges)
		for _, org := range orgPrivileges {
			allowed, err := role_model.CheckUserPermissionToOrganization(ctx, ctx.Doer, commonTenantID, org, role_model.READ)
			if err != nil {
				log.Error("Error checking permissions: %v", err)
				return nil, err
			}
			if allowed {
				organizations = append(organizations, org)
			}
		}
	}

	ctx.Data["TenantID"] = commonTenantIDs[0]
	return organizations, nil
}
