package repo_server

import (
	"net/http"
	"time"

	activities_model "code.gitea.io/gitea/models/activities"
	"code.gitea.io/gitea/models/role_model"
	"code.gitea.io/gitea/models/unit"
	"code.gitea.io/gitea/modules/base"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/trace"
	"code.gitea.io/gitea/routers/web/user/accesser"
)

const (
	tplActivity base.TplName = "repo/activity"
)

// Activity render the page to show repository latest changes
func (s *Server) Activity(ctx *context.Context) {
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

	ctx.Data["Title"] = ctx.Tr("repo.activity")
	ctx.Data["PageIsActivity"] = true

	ctx.Data["Period"] = ctx.Params("period")

	timeUntil := time.Now()
	var timeFrom time.Time

	switch ctx.Data["Period"] {
	case "daily":
		timeFrom = timeUntil.Add(-time.Hour * 24)
	case "halfweekly":
		timeFrom = timeUntil.Add(-time.Hour * 72)
	case "weekly":
		timeFrom = timeUntil.Add(-time.Hour * 168)
	case "monthly":
		timeFrom = timeUntil.AddDate(0, -1, 0)
	case "quarterly":
		timeFrom = timeUntil.AddDate(0, -3, 0)
	case "semiyearly":
		timeFrom = timeUntil.AddDate(0, -6, 0)
	case "yearly":
		timeFrom = timeUntil.AddDate(-1, 0, 0)
	default:
		ctx.Data["Period"] = "weekly"
		timeFrom = timeUntil.Add(-time.Hour * 168)
	}
	ctx.Data["DateFrom"] = timeFrom.UTC().Format(time.RFC3339)
	ctx.Data["DateUntil"] = timeUntil.UTC().Format(time.RFC3339)
	ctx.Data["PeriodText"] = ctx.Tr("repo.activity.period." + ctx.Data["Period"].(string))

	var err error
	if ctx.Data["Activity"], err = activities_model.GetActivityStats(ctx, ctx.Repo.Repository, timeFrom,
		ctx.Repo.CanRead(unit.TypeReleases),
		ctx.Repo.CanRead(unit.TypeIssues),
		ctx.Repo.CanRead(unit.TypePullRequests),
		ctx.Repo.CanRead(unit.TypeCode)); err != nil {
		ctx.ServerError("GetActivityStats", err)
		return
	}

	if ctx.PageData["repoActivityTopAuthors"], ctx.Data["AuthorCountInAllBranches"], err = activities_model.GetActivityStatsTopAuthors(ctx, ctx.Repo.Repository, timeFrom, 10); err != nil {
		ctx.ServerError("GetActivityStatsTopAuthors", err)
		return
	}

	action := role_model.READ
	if ctx.Repo.Repository.IsPrivate {
		action = role_model.READ_PRIVATE
	}

	allowed, err := s.orgRequestAccessor.IsAccessGranted(ctx, accesser.OrgAccessRequest{
		DoerID:         ctx.Doer.ID,
		TargetTenantID: ctx.Data["TenantID"].(string),
		TargetOrgID:    ctx.Repo.Repository.OwnerID,
		Action:         action,
	})
	if err != nil {
		log.Error("Error has occurred while checking user's permissions: %v", err)
		ctx.ServerError("Error has occurred while checking user's permissions: %v", err)
		return
	}
	if !allowed {
		allow, err := s.repoRequestAccessor.AccessesByCustomPrivileges(ctx, accesser.RepoAccessRequest{
			DoerID:          ctx.Doer.ID,
			RepoID:          ctx.Repo.Repository.ID,
			CustomPrivilege: role_model.ViewBranch.String(),
			OrgID:           ctx.Repo.Repository.OwnerID,
			TargetTenantID:  ctx.Data["TenantID"].(string),
		})
		if err != nil {
			log.Error("Error has occurred while checking user's permissions: %v", err)
			ctx.ServerError("Error has occurred while checking user's permissions: %v", err)
			return
		}
		allowed = allow
	}

	if !allowed {
		ctx.Error(http.StatusForbidden, "Forbidden")
		return
	}
	ctx.HTML(http.StatusOK, tplActivity)
}
