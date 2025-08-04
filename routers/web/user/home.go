// Copyright 2014 The Gogs Authors. All rights reserved.
// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package user

import (
	"bytes"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"code.gitea.io/gitea/modules/trace"
	"github.com/keybase/go-crypto/openpgp"
	"github.com/keybase/go-crypto/openpgp/armor"
	"xorm.io/builder"

	asymkey_model "code.gitea.io/gitea/models/asymkey"
	"code.gitea.io/gitea/models/db"
	issues_model "code.gitea.io/gitea/models/issues"
	"code.gitea.io/gitea/models/organization"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/role_model"
	"code.gitea.io/gitea/models/unit"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/base"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/markup"
	"code.gitea.io/gitea/modules/markup/markdown"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/util"
	"code.gitea.io/gitea/routers/utils"
)

const (
	tplDashboard  base.TplName = "user/dashboard/dashboard"
	tplIssues     base.TplName = "user/dashboard/issues"
	tplMilestones base.TplName = "user/dashboard/milestones"
)

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

// Milestones render the user milestones page
func Milestones(ctx *context.Context) {
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

	if unit.TypeIssues.UnitGlobalDisabled() && unit.TypePullRequests.UnitGlobalDisabled() {
		log.Debug("Milestones overview page not available as both issues and pull requests are globally disabled")
		ctx.Status(http.StatusNotFound)
		return
	}

	ctx.Data["Title"] = ctx.Tr("milestones")
	ctx.Data["PageIsMilestonesDashboard"] = true

	ctxUser := getDashboardContextUser(ctx)
	if ctx.Written() {
		return
	}

	repoOpts := repo_model.SearchRepoOptions{
		Actor:         ctxUser,
		OwnerID:       ctxUser.ID,
		Private:       true,
		AllPublic:     false, // Include also all public repositories of users and public organisations
		AllLimited:    false, // Include also all public repositories of limited organisations
		Archived:      util.OptionalBoolFalse,
		HasMilestones: util.OptionalBoolTrue, // Just needs display repos has milestones
	}

	if ctxUser.IsOrganization() && ctx.Org.Team != nil {
		repoOpts.TeamID = ctx.Org.Team.ID
	}

	var (
		userRepoCond = repo_model.SearchRepositoryCondition(&repoOpts) // all repo condition user could visit
		repoCond     = userRepoCond
		repoIDs      []int64

		reposQuery   = ctx.FormString("repos")
		isShowClosed = ctx.FormString("state") == "closed"
		sortType     = ctx.FormString("sort")
		page         = ctx.FormInt("page")
		keyword      = ctx.FormTrim("q")
	)

	if page <= 1 {
		page = 1
	}

	if len(reposQuery) != 0 {
		if issueReposQueryPattern.MatchString(reposQuery) {
			// remove "[" and "]" from string
			reposQuery = reposQuery[1 : len(reposQuery)-1]
			// for each ID (delimiter ",") add to int to repoIDs

			for _, rID := range strings.Split(reposQuery, ",") {
				// Ensure nonempty string entries
				if rID != "" && rID != "0" {
					rIDint64, err := strconv.ParseInt(rID, 10, 64)
					// If the repo id specified by query is not parseable or not accessible by user, just ignore it.
					if err == nil {
						repoIDs = append(repoIDs, rIDint64)
					}
				}
			}
			if len(repoIDs) > 0 {
				// Don't just let repoCond = builder.In("id", repoIDs) because user may has no permission on repoIDs
				// But the original repoCond has a limitation
				repoCond = repoCond.And(builder.In("id", repoIDs))
			}
		} else {
			log.Warn("issueReposQueryPattern not match with query")
		}
	}

	counts, err := issues_model.CountMilestonesByRepoCondAndKw(userRepoCond, keyword, isShowClosed)
	if err != nil {
		ctx.ServerError("CountMilestonesByRepoIDs", err)
		return
	}

	milestones, err := issues_model.SearchMilestones(repoCond, page, isShowClosed, sortType, keyword)
	if err != nil {
		ctx.ServerError("SearchMilestones", err)
		return
	}

	showRepos, _, err := repo_model.SearchRepositoryByCondition(ctx, &repoOpts, userRepoCond, false)
	if err != nil {
		ctx.ServerError("SearchRepositoryByCondition", err)
		return
	}
	sort.Sort(showRepos)

	for i := 0; i < len(milestones); {
		for _, repo := range showRepos {
			if milestones[i].RepoID == repo.ID {
				milestones[i].Repo = repo
				break
			}
		}
		if milestones[i].Repo == nil {
			log.Warn("Cannot find milestone %d 's repository %d", milestones[i].ID, milestones[i].RepoID)
			milestones = append(milestones[:i], milestones[i+1:]...)
			continue
		}

		milestones[i].RenderedContent, err = markdown.RenderString(&markup.RenderContext{
			URLPrefix: milestones[i].Repo.Link(),
			Metas:     milestones[i].Repo.ComposeMetas(),
			Ctx:       ctx,
		}, milestones[i].Content)
		if err != nil {
			ctx.ServerError("RenderString", err)
			return
		}

		if milestones[i].Repo.IsTimetrackerEnabled(ctx) {
			err := milestones[i].LoadTotalTrackedTime()
			if err != nil {
				ctx.ServerError("LoadTotalTrackedTime", err)
				return
			}
		}
		i++
	}

	milestoneStats, err := issues_model.GetMilestonesStatsByRepoCondAndKw(repoCond, keyword)
	if err != nil {
		ctx.ServerError("GetMilestoneStats", err)
		return
	}

	var totalMilestoneStats *issues_model.MilestonesStats
	if len(repoIDs) == 0 {
		totalMilestoneStats = milestoneStats
	} else {
		totalMilestoneStats, err = issues_model.GetMilestonesStatsByRepoCondAndKw(userRepoCond, keyword)
		if err != nil {
			ctx.ServerError("GetMilestoneStats", err)
			return
		}
	}

	var pagerCount int
	if isShowClosed {
		ctx.Data["State"] = "closed"
		ctx.Data["Total"] = totalMilestoneStats.ClosedCount
		pagerCount = int(milestoneStats.ClosedCount)
	} else {
		ctx.Data["State"] = "open"
		ctx.Data["Total"] = totalMilestoneStats.OpenCount
		pagerCount = int(milestoneStats.OpenCount)
	}

	ctx.Data["Milestones"] = milestones
	ctx.Data["Repos"] = showRepos
	ctx.Data["Counts"] = counts
	ctx.Data["MilestoneStats"] = milestoneStats
	ctx.Data["SortType"] = sortType
	ctx.Data["Keyword"] = keyword
	if milestoneStats.Total() != totalMilestoneStats.Total() {
		ctx.Data["RepoIDs"] = repoIDs
	}
	ctx.Data["IsShowClosed"] = isShowClosed

	pager := context.NewPagination(pagerCount, setting.UI.IssuePagingNum, page, 5)
	pager.AddParam(ctx, "q", "Keyword")
	pager.AddParam(ctx, "repos", "RepoIDs")
	pager.AddParam(ctx, "sort", "SortType")
	pager.AddParam(ctx, "state", "State")
	ctx.Data["Page"] = pager

	ctx.HTML(http.StatusOK, tplMilestones)
}

var issueReposQueryPattern = regexp.MustCompile(`^\[\d+(,\d+)*,?\]$`)

// ShowSSHKeys output all the ssh keys of user by uid
func ShowSSHKeys(ctx *context.Context) {
	keys, err := asymkey_model.ListPublicKeys(ctx.ContextUser.ID, db.ListOptions{})
	if err != nil {
		ctx.ServerError("ListPublicKeys", err)
		return
	}

	var buf bytes.Buffer
	for i := range keys {
		buf.WriteString(keys[i].OmitEmail())
		buf.WriteString("\n")
	}
	ctx.PlainTextBytes(http.StatusOK, buf.Bytes())
}

// ShowGPGKeys output all the public GPG keys of user by uid
func ShowGPGKeys(ctx *context.Context) {
	keys, err := asymkey_model.ListGPGKeys(ctx, ctx.ContextUser.ID, db.ListOptions{})
	if err != nil {
		ctx.ServerError("ListGPGKeys", err)
		return
	}

	entities := make([]*openpgp.Entity, 0)
	failedEntitiesID := make([]string, 0)
	for _, k := range keys {
		e, err := asymkey_model.GPGKeyToEntity(k)
		if err != nil {
			if asymkey_model.IsErrGPGKeyImportNotExist(err) {
				failedEntitiesID = append(failedEntitiesID, k.KeyID)
				continue // Skip previous import without backup of imported armored key
			}
			ctx.ServerError("ShowGPGKeys", err)
			return
		}
		entities = append(entities, e)
	}
	var buf bytes.Buffer

	headers := make(map[string]string)
	if len(failedEntitiesID) > 0 { // If some key need re-import to be exported
		headers["Note"] = fmt.Sprintf("The keys with the following IDs couldn't be exported and need to be reuploaded %s", strings.Join(failedEntitiesID, ", "))
	} else if len(entities) == 0 {
		headers["Note"] = "This user hasn't uploaded any GPG keys."
	}
	writer, _ := armor.Encode(&buf, "PGP PUBLIC KEY BLOCK", headers)
	for _, e := range entities {
		err = e.Serialize(writer) // TODO find why key are exported with a different cipherTypeByte as original (should not be blocking but strange)
		if err != nil {
			ctx.ServerError("ShowGPGKeys", err)
			return
		}
	}
	writer.Close()
	ctx.PlainTextBytes(http.StatusOK, buf.Bytes())
}
