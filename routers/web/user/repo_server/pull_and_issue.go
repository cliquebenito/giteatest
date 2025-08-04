package repo_server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"

	issues_model "code.gitea.io/gitea/models/issues"
	"code.gitea.io/gitea/models/organization"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/role_model"
	"code.gitea.io/gitea/models/role_model/casbin_role_manager"
	"code.gitea.io/gitea/models/unit"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/base"
	"code.gitea.io/gitea/modules/context"
	issue_indexer "code.gitea.io/gitea/modules/indexer/issues"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/trace"
	"code.gitea.io/gitea/modules/util"
	"code.gitea.io/gitea/routers/utils"
	"code.gitea.io/gitea/routers/web/user/accesser"
	issue_service "code.gitea.io/gitea/services/issue"
	pull_service "code.gitea.io/gitea/services/pull"
)

const (
	//tplIssuesEs base.TplName = "repo/issue/list"
	tplIssues base.TplName = "repo/issue/list"

	tplPulls base.TplName = "user/dashboard/issues"
)

// Pulls renders the user's pull request overview page
func (s *Server) Pulls(ctx *context.Context) {
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

	if unit.TypePullRequests.UnitGlobalDisabled() {
		log.Debug("Pull request overview page not available as it is globally disabled.")
		ctx.Status(http.StatusNotFound)
		return
	}
	ctx.Data["Title"] = ctx.Tr("pull_requests")
	ctx.Data["PageIsPulls"] = true
	ctx.Data["SingleRepoAction"] = "pull"
	s.buildIssueOverview(ctx, unit.TypePullRequests)
}

// Issues renders the user's issues overview page
func (s *Server) Issues(ctx *context.Context) {
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

	if unit.TypeIssues.UnitGlobalDisabled() {
		log.Debug("Issues overview page not available as it is globally disabled.")
		ctx.Status(http.StatusNotFound)
		return
	}

	ctx.Data["Title"] = ctx.Tr("issues")
	ctx.Data["PageIsIssues"] = true
	ctx.Data["SingleRepoAction"] = "issue"

	s.buildIssueOverview(ctx, unit.TypeIssues)
}

// Regexp for repos query
var issueReposQueryPattern = regexp.MustCompile(`^\[\d+(,\d+)*,?\]$`)

func (s *Server) buildIssueOverview(ctx *context.Context, unitType unit.Type) {
	// ----------------------------------------------------
	// Determine user; can be either user or organization.
	// Return with NotFound or ServerError if unsuccessful.
	// ----------------------------------------------------
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
		viewType   string
		sortType   = ctx.FormString("sort")
		filterMode int
	)

	// Default to recently updated, unlike repository issues list
	if sortType == "" {
		sortType = "recentupdate"
	}

	// --------------------------------------------------------------------------------
	// Distinguish User from Organization.
	// Org:
	// - Remember pre-determined viewType string for later. Will be posted to ctx.Data.
	//   Organization does not have view type and filter mode.
	// User:
	// - Use ctx.FormString("type") to determine filterMode.
	//  The type is set when clicking for example "assigned to me" on the overview page.
	// - Remember either this or a fallback. Will be posted to ctx.Data.
	// --------------------------------------------------------------------------------

	// TODO: distinguish during routing

	viewType = ctx.FormString("type")
	switch viewType {
	case "assigned":
		filterMode = issues_model.FilterModeAssign
	case "created_by":
		filterMode = issues_model.FilterModeCreate
	case "mentioned":
		filterMode = issues_model.FilterModeMention
	case "review_requested":
		filterMode = issues_model.FilterModeReviewRequested
	case "reviewed_by":
		filterMode = issues_model.FilterModeReviewed
	case "your_repositories":
		fallthrough
	default:
		filterMode = issues_model.FilterModeYourRepositories
		viewType = "your_repositories"
	}

	// --------------------------------------------------------------------------
	// Build opts (IssuesOptions), which contains filter information.
	// Will eventually be used to retrieve issues relevant for the overview page.
	// Note: Non-final states of opts are used in-between, namely for:
	//       - Keyword search
	//       - Count Issues by repo
	// --------------------------------------------------------------------------

	// Get repository IDs where User/Org/Team has access.
	var team *organization.Team
	var org *organization.Organization
	if ctx.Org != nil {
		org = ctx.Org.Organization
		team = ctx.Org.Team
	}

	isPullList := unitType == unit.TypePullRequests
	opts := &issues_model.IssuesOptions{
		IsPull:     util.OptionalBoolOf(isPullList),
		SortType:   sortType,
		IsArchived: util.OptionalBoolFalse,
		Org:        org,
		Team:       team,
		User:       ctx.Doer,
	}

	// Search all repositories which
	//
	// As user:
	// - Owns the repository.
	// - Have collaborator permissions in repository.
	//
	// As org:
	// - Owns the repository.
	//
	// As team:
	// - Team org's owns the repository.
	// - Team has read permission to repository.
	repoOpts := &repo_model.SearchRepoOptions{
		Actor:      ctx.Doer,
		OwnerID:    ctx.Doer.ID,
		Private:    true,
		AllPublic:  false,
		AllLimited: false,
	}

	if team != nil {
		repoOpts.TeamID = team.ID
	}

	switch filterMode {
	case issues_model.FilterModeAll:
	case issues_model.FilterModeYourRepositories:
	case issues_model.FilterModeAssign:
		opts.AssigneeID = ctx.Doer.ID
	case issues_model.FilterModeCreate:
		opts.PosterID = ctx.Doer.ID
	case issues_model.FilterModeMention:
		opts.MentionedID = ctx.Doer.ID
	case issues_model.FilterModeReviewRequested:
		opts.ReviewRequestedID = ctx.Doer.ID
	case issues_model.FilterModeReviewed:
		opts.ReviewedID = ctx.Doer.ID
	}

	// keyword holds the search term entered into the search field.
	keyword := strings.Trim(ctx.FormString("q"), " ")
	ctx.Data["Keyword"] = keyword

	// Execute keyword search for issues.
	// USING NON-FINAL STATE OF opts FOR A QUERY.
	issueIDsFromSearch, err := issueIDsFromSearch(ctx, ctxUser, keyword, opts)
	if err != nil {
		ctx.ServerError("issueIDsFromSearch", err)
		return
	}

	// Ensure no issues are returned if a keyword was provided that didn't match any issues.
	var forceEmpty bool

	if len(issueIDsFromSearch) > 0 {
		opts.IssueIDs = issueIDsFromSearch
	} else if len(keyword) > 0 {
		forceEmpty = true
	}

	// Educated guess: Do or don't show closed issues.
	isShowClosed := ctx.FormString("state") == "closed"
	opts.IsClosed = util.OptionalBoolOf(isShowClosed)

	// Parse ctx.FormString("repos") and remember matched repo IDs for later.
	// Gets set when clicking filters on the issues overview page.
	repoIDs := make([]int64, 0)

	tenantID, err := role_model.GetUserTenantId(ctx, ctxUser.ID)
	if err != nil {
		log.Error("Error has occurred while getting user tenant id: %v", err)
		return
	}

	if setting.SourceControl.TenantWithRoleModeEnabled && ctxUser.Type == user_model.UserTypeIndividual {
		repoIDs = repoIDsGet(ctx, ctxUser, tenantID)
	} else if setting.SourceControl.TenantWithRoleModeEnabled && ctxUser.Type == user_model.UserTypeOrganization {
		repos, err := organization.GetOrgRepositories(ctx, ctxUser.ID)
		if err != nil {
			log.Error("Error has occurred while getting repositories by org: %v", err)
			ctx.Error(http.StatusNotFound, fmt.Sprintf("Error has occurred while getting repositories by org: %v", err))
			return
		}

		for idx := range repos {
			repoIDs = append(repoIDs, repos[idx].ID)
		}
	} else {
		repoIDs = getRepoIDs(ctx.FormString("repos"))
	}
	opts.RepoIDs = repoIDs

	// Filter repos and count issues in them. Count will be used later.
	// USING NON-FINAL STATE OF opts FOR A QUERY.
	var issueCountByRepo map[int64]int64
	if !forceEmpty {
		issueCountByRepo, err = issues_model.CountIssuesByRepo(ctx, opts)
		if err != nil {
			ctx.ServerError("CountIssuesByRepo", err)
			return
		}
	}

	// Make sure page number is at least 1. Will be posted to ctx.Data.
	page := ctx.FormInt("page")
	if page <= 1 {
		page = 1
	}
	opts.Page = page
	opts.PageSize = setting.UI.IssuePagingNum

	// Get IDs for labels (a filter option for issues/pulls).
	// Required for IssuesOptions.
	var labelIDs []int64
	selectedLabels := ctx.FormString("labels")
	if len(selectedLabels) > 0 && selectedLabels != "0" {
		labelIDs, err = base.StringsToInt64s(strings.Split(selectedLabels, ","))
		if err != nil {
			ctx.ServerError("StringsToInt64s", err)
			return
		}
	}
	opts.LabelIDs = labelIDs

	// ------------------------------
	// Get issues as defined by opts.
	// ------------------------------

	// Slice of Issues that will be displayed on the overview page
	// USING FINAL STATE OF opts FOR A QUERY.
	repoIdsAllow := repoIDs
	var issues []*issues_model.Issue
	if !forceEmpty {
		issues, err = issues_model.Issues(ctx, opts)
		if err != nil {
			ctx.ServerError("Issues", err)
			return
		}
		organizations := make([]*organization.Organization, 0)
		if ctx.Data["Orgs"] != nil {
			organizations = ctx.Data["Orgs"].([]*organization.Organization)
		}
		for _, orgEntity := range organizations {
			for _, repoID := range repoIDs {
				repo, err := repo_model.GetRepositoryByID(ctx, repoID)
				if err != nil {
					log.Error("Error has occurred while getting repository: %v", err)
					ctx.ServerError("Error has occurred while getting repository: %v", err)
					return
				}

				action := role_model.READ
				if repo.IsPrivate {
					action = role_model.READ_PRIVATE
				}
				allowed, err := s.orgRequestAccessor.IsAccessGranted(*ctx, accesser.OrgAccessRequest{
					DoerID:         ctx.Doer.ID,
					TargetOrgID:    orgEntity.ID,
					TargetTenantID: tenantID,
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
						OrgID:           orgEntity.ID,
						TargetTenantID:  tenantID,
						RepoID:          repoID,
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
			}
		}

		ctx.Data["Orgs"] = organizations
		repoIDs = repoIdsAllow
	}

	// ----------------------------------
	// Add repository pointers to Issues.
	// ----------------------------------

	// showReposMap maps repository IDs to their Repository pointers.
	showReposMap, err := loadRepoByIDs(ctxUser, issueCountByRepo, unitType)
	if err != nil {
		if repo_model.IsErrRepoNotExist(err) {
			ctx.NotFound("GetRepositoryByID", err)
			return
		}
		ctx.ServerError("loadRepoByIDs", err)
		return
	}

	// a RepositoryList
	showRepos := repo_model.RepositoryListOfMap(showReposMap)
	sort.Sort(showRepos)

	// maps pull request IDs to their CommitStatus. Will be posted to ctx.Data.
	for _, issue := range issues {
		if issue.Repo == nil {
			issue.Repo = showReposMap[issue.RepoID]
		}
	}

	commitStatuses, lastStatus, err := pull_service.GetIssuesAllCommitStatus(ctx, issues)
	if err != nil {
		ctx.ServerError("GetIssuesLastCommitStatus", err)
		return
	}

	// -------------------------------
	// Fill stats to post to ctx.Data.
	// -------------------------------
	var issueStats *issues_model.IssueStats
	if !forceEmpty {
		statsOpts := issues_model.IssuesOptions{
			User:       ctx.Doer,
			IsPull:     util.OptionalBoolOf(isPullList),
			IsClosed:   util.OptionalBoolOf(isShowClosed),
			IssueIDs:   issueIDsFromSearch,
			IsArchived: util.OptionalBoolFalse,
			LabelIDs:   opts.LabelIDs,
			Org:        org,
			Team:       team,
			RepoCond:   opts.RepoCond,
			RepoIDs:    repoIdsAllow,
		}

		issueStats, err = issues_model.GetUserIssueStats(filterMode, statsOpts)
		if err != nil {
			ctx.ServerError("GetUserIssueStats Shown", err)
			return
		}
	} else {
		issueStats = &issues_model.IssueStats{}
	}

	// Will be posted to ctx.Data.
	var shownIssues int
	if !isShowClosed {
		shownIssues = int(issueStats.OpenCount)
	} else {
		shownIssues = int(issueStats.ClosedCount)
	}
	if len(opts.RepoIDs) != 0 {
		shownIssues = 0
		for _, repoID := range opts.RepoIDs {
			shownIssues += int(issueCountByRepo[repoID])
		}
	}

	var allIssueCount int64
	for _, issueCount := range issueCountByRepo {
		allIssueCount += issueCount
	}
	ctx.Data["TotalIssueCount"] = allIssueCount

	if len(opts.RepoIDs) == 1 {
		repo := showReposMap[opts.RepoIDs[0]]
		if repo != nil {
			ctx.Data["SingleRepoLink"] = repo.Link()
		}
	}

	ctx.Data["IsShowClosed"] = isShowClosed

	ctx.Data["IssueRefEndNames"], ctx.Data["IssueRefURLs"] = issue_service.GetRefEndNamesAndURLs(issues, ctx.FormString("RepoLink"))

	ctx.Data["Issues"] = issues

	approvalCounts, err := issues_model.IssueList(issues).GetApprovalCounts(ctx)
	if err != nil {
		ctx.ServerError("ApprovalCounts", err)
		return
	}
	ctx.Data["ApprovalCounts"] = func(issueID int64, typ string) int64 {
		counts, ok := approvalCounts[issueID]
		if !ok || len(counts) == 0 {
			return 0
		}
		reviewTyp := issues_model.ReviewTypeApprove
		if typ == "reject" {
			reviewTyp = issues_model.ReviewTypeReject
		} else if typ == "waiting" {
			reviewTyp = issues_model.ReviewTypeRequest
		}
		for _, count := range counts {
			if count.Type == reviewTyp {
				return count.Count
			}
		}
		return 0
	}
	ctx.Data["CommitLastStatus"] = lastStatus
	ctx.Data["CommitStatuses"] = commitStatuses
	ctx.Data["Repos"] = showRepos
	ctx.Data["Counts"] = issueCountByRepo
	ctx.Data["IssueStats"] = issueStats
	ctx.Data["ViewType"] = viewType
	ctx.Data["SortType"] = sortType
	ctx.Data["RepoIDs"] = opts.RepoIDs
	ctx.Data["IsShowClosed"] = isShowClosed
	ctx.Data["SelectLabels"] = selectedLabels

	if isShowClosed {
		ctx.Data["State"] = "closed"
	} else {
		ctx.Data["State"] = "open"
	}

	// Convert []int64 to string
	reposParam, _ := json.Marshal(opts.RepoIDs)

	ctx.Data["ReposParam"] = string(reposParam)

	pager := context.NewPagination(shownIssues, setting.UI.IssuePagingNum, page, 5)
	pager.AddParam(ctx, "q", "Keyword")
	pager.AddParam(ctx, "type", "ViewType")
	pager.AddParam(ctx, "repos", "ReposParam")
	pager.AddParam(ctx, "sort", "SortType")
	pager.AddParam(ctx, "state", "State")
	pager.AddParam(ctx, "labels", "SelectLabels")
	pager.AddParam(ctx, "milestone", "MilestoneID")
	pager.AddParam(ctx, "assignee", "AssigneeID")
	ctx.Data["Page"] = pager

	ctx.HTML(http.StatusOK, tplPulls)
}
func repoIDsGet(ctx *context.Context, ctxUser *user_model.User, tenantID string) []int64 {
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

	repoIDs := make([]int64, 0)

	roleManager := casbin_role_manager.New()

	privileges, err := utils.GetTenantsPrivilegesByUserID(ctx, ctxUser.ID)
	if err != nil {
		log.Error("Error has occurred while getting privileges by tenant: %v", err)
		ctx.Error(http.StatusNotFound, fmt.Sprintf("Error has occurred while getting privileges by tenant: %v", err))
		return nil
	}

	userPrivileges := []role_model.EnrichedPrivilege{}
	for _, v := range privileges {
		if v.User.ID == ctxUser.ID {
			userPrivileges = append(userPrivileges, v)
		}
	}

	privilegeOrganizations := utils.ConvertTenantPrivilegesInOrganizations(userPrivileges)
	for _, privilegeOrg := range privilegeOrganizations {
		repos, err := organization.GetOrgRepositories(ctx, privilegeOrg.ID)
		if err != nil {
			log.Error("Error has occurred while getting repositories by org: %v", err)
			ctx.Error(http.StatusNotFound, fmt.Sprintf("Error has occurred while getting repositories by org: %v", err))
			return nil
		}
		for _, v := range repos {
			action := role_model.READ
			if v.IsPrivate {
				action = role_model.READ_PRIVATE
			}
			allowed, err := roleManager.CheckUserPermissionToOrganization(ctx, ctxUser, tenantID, privilegeOrg, action)
			if err != nil {
				log.Error("Error has occurred while checking user permission to organization: %v", err)
				return nil
			}
			if !allowed {
				allow, err := role_model.CheckUserPermissionToTeam(ctx, ctxUser, tenantID, &organization.Organization{ID: privilegeOrg.ID},
					&repo_model.Repository{ID: v.ID}, role_model.ViewBranch.String())
				if err != nil {
					log.Error("Error has occurred while checking user permission to team: %v", err)
					return nil
				}
				if !allow {
					continue
				}
			}
			repoIDs = append(repoIDs, v.ID)
		}
	}
	return repoIDs
}

func getRepoIDs(reposQuery string) []int64 {
	if len(reposQuery) == 0 || reposQuery == "[]" {
		return []int64{}
	}
	if !issueReposQueryPattern.MatchString(reposQuery) {
		log.Warn("issueReposQueryPattern does not match query")
		return []int64{}
	}

	var repoIDs []int64
	// remove "[" and "]" from string
	reposQuery = reposQuery[1 : len(reposQuery)-1]
	// for each ID (delimiter ",") add to int to repoIDs
	for _, rID := range strings.Split(reposQuery, ",") {
		// Ensure nonempty string entries
		if rID != "" && rID != "0" {
			rIDint64, err := strconv.ParseInt(rID, 10, 64)
			if err == nil {
				repoIDs = append(repoIDs, rIDint64)
			}
		}
	}

	return repoIDs
}

func issueIDsFromSearch(ctx *context.Context, ctxUser *user_model.User, keyword string, opts *issues_model.IssuesOptions) ([]int64, error) {
	if len(keyword) == 0 {
		return []int64{}, nil
	}

	searchRepoIDs, err := issues_model.GetRepoIDsForIssuesOptions(opts, ctxUser)
	if err != nil {
		return nil, fmt.Errorf("GetRepoIDsForIssuesOptions: %w", err)
	}
	issueIDsFromSearch, err := issue_indexer.SearchIssuesByKeyword(ctx, searchRepoIDs, keyword)
	if err != nil {
		return nil, fmt.Errorf("SearchIssuesByKeyword: %w", err)
	}

	return issueIDsFromSearch, nil
}

func loadRepoByIDs(ctxUser *user_model.User, issueCountByRepo map[int64]int64, unitType unit.Type) (map[int64]*repo_model.Repository, error) {
	totalRes := make(map[int64]*repo_model.Repository, len(issueCountByRepo))
	repoIDs := make([]int64, 0, 500)
	for id := range issueCountByRepo {
		if id <= 0 {
			continue
		}
		repoIDs = append(repoIDs, id)
		if len(repoIDs) == 500 {
			if err := repo_model.FindReposMapByIDs(repoIDs, totalRes); err != nil {
				return nil, err
			}
			repoIDs = repoIDs[:0]
		}
	}
	if len(repoIDs) > 0 {
		if err := repo_model.FindReposMapByIDs(repoIDs, totalRes); err != nil {
			return nil, err
		}
	}
	return totalRes, nil
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
