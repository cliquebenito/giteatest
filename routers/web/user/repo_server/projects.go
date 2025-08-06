package repo_server

import (
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"

	"code.gitea.io/gitea/modules/trace"

	project_model "code.gitea.io/gitea/models/project"
	"code.gitea.io/gitea/models/role_model"
	"code.gitea.io/gitea/models/unit"
	"code.gitea.io/gitea/modules/base"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/markup"
	"code.gitea.io/gitea/modules/markup/markdown"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/util"
	"code.gitea.io/gitea/routers/web/user/accesser"
)

const (
	tplProjects base.TplName = "repo/projects/list"
)

// Projects renders the home page of projects
func (s *Server) Projects(ctx *context.Context) {
	logTracer := trace.NewLogTracer()
	message := logTracer.CreateTraceMessage(ctx)
	err := logTracer.Trace(message)
	if err != nil {
		log.Errorf("Error has occurred while creating trace message: %v", err)
	}
	defer func() {
		err = logTracer.TraceTime(message)
		if err != nil {
			log.Errorf("Error has occurred while creating trace time message: %v", err)
		}
	}()

	ctx.Data["Title"] = ctx.Tr("repo.project_board")

	sortType := ctx.FormTrim("sort")

	isShowClosed := strings.ToLower(ctx.FormTrim("state")) == "closed"
	repo := ctx.Repo.Repository
	page := ctx.FormInt("page")
	if page <= 1 {
		page = 1
	}

	action := role_model.READ
	if ctx.Repo.Repository.IsPrivate {
		action = role_model.READ_PRIVATE
	}

	allowed, err := s.orgRequestAccessor.IsAccessGranted(*ctx, accesser.OrgAccessRequest{
		DoerID:         ctx.Doer.ID,
		TargetOrgID:    repo.OwnerID,
		TargetTenantID: ctx.Data["TenantID"].(string),
		Action:         action,
	})
	if err != nil {
		log.Errorf("Error has occurred while checking user's permissions: %v", err)
		ctx.ServerError("Error has occurred while checking user's permissions: %v", err)
		return
	}
	if !allowed {
		allow, err := s.repoRequestAccessor.AccessesByCustomPrivileges(*ctx, accesser.RepoAccessRequest{
			DoerID:          ctx.Doer.ID,
			OrgID:           repo.OwnerID,
			TargetTenantID:  ctx.Data["TenantID"].(string),
			RepoID:          repo.ID,
			CustomPrivilege: role_model.ViewBranch.String(),
		})
		if err != nil {
			log.Errorf("Error has occurred while checking user's permissions: %v", err)
			ctx.ServerError("Error has occurred while checking user's permissions: %v", err)
			return
		}
		if !allow {
			ctx.NotFound("Error has occurred while checking user's custom permissions", nil)
			return
		}
	}

	ctx.Data["OpenCount"] = repo.NumOpenProjects
	ctx.Data["ClosedCount"] = repo.NumClosedProjects

	var total int
	if !isShowClosed {
		total = repo.NumOpenProjects
	} else {
		total = repo.NumClosedProjects
	}

	projects, count, err := project_model.FindProjects(ctx, project_model.SearchOptions{
		RepoID:   repo.ID,
		Page:     page,
		IsClosed: util.OptionalBoolOf(isShowClosed),
		SortType: sortType,
		Type:     project_model.TypeRepository,
	})
	if err != nil {
		ctx.ServerError("GetProjects", err)
		return
	}

	for i := range projects {
		projects[i].RenderedContent, err = markdown.RenderString(&markup.RenderContext{
			URLPrefix: ctx.Repo.RepoLink,
			Metas:     ctx.Repo.Repository.ComposeMetas(),
			GitRepo:   ctx.Repo.GitRepo,
			Ctx:       ctx,
		}, projects[i].Description)
		if err != nil {
			ctx.ServerError("RenderString", err)
			return
		}
	}

	ctx.Data["Projects"] = projects

	if isShowClosed {
		ctx.Data["State"] = "closed"
	} else {
		ctx.Data["State"] = "open"
	}

	numPages := 0
	if count > 0 {
		numPages = (int(count) - 1/setting.UI.IssuePagingNum)
	}

	pager := context.NewPagination(total, setting.UI.IssuePagingNum, page, numPages)
	pager.AddParam(ctx, "state", "State")
	ctx.Data["Page"] = pager

	ctx.Data["CanWriteProjects"] = ctx.Repo.Permission.CanWrite(unit.TypeProjects)
	ctx.Data["IsShowClosed"] = isShowClosed
	ctx.Data["IsProjectsPage"] = true
	ctx.Data["SortType"] = sortType

	ctx.HTML(http.StatusOK, tplProjects)
}
