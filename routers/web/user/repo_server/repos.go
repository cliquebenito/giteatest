package repo_server

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"code.gitea.io/gitea/models/db"
	repo_model "code.gitea.io/gitea/models/repo"
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
	tplSettingsRepositories base.TplName = "user/settings/repos"
)

// Repos display a list of all repositories of the user
func (s *Server) Repos(ctx *context.Context) {
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

	ctx.Data["Title"] = ctx.Tr("settings.repos")
	ctx.Data["PageIsSettingsRepos"] = true
	ctx.Data["allowAdopt"] = ctx.IsUserSiteAdmin() || setting.Repository.AllowAdoptionOfUnadoptedRepositories
	ctx.Data["allowDelete"] = ctx.IsUserSiteAdmin() || setting.Repository.AllowDeleteOfUnadoptedRepositories

	opts := db.ListOptions{
		PageSize: setting.UI.Admin.UserPagingNum,
		Page:     ctx.FormInt("page"),
	}

	if opts.Page <= 0 {
		opts.Page = 1
	}
	start := (opts.Page - 1) * opts.PageSize
	end := start + opts.PageSize

	adoptOrDelete := ctx.IsUserSiteAdmin() || (setting.Repository.AllowAdoptionOfUnadoptedRepositories && setting.Repository.AllowDeleteOfUnadoptedRepositories)

	ctxUser := ctx.Doer
	count := 0

	if adoptOrDelete && !setting.SourceControl.TenantWithRoleModeEnabled {
		repoNames := make([]string, 0, setting.UI.Admin.UserPagingNum)
		repos := map[string]*repo_model.Repository{}
		// We're going to iterate by pagesize.
		root := user_model.UserPath(ctxUser.Name)
		if err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				if os.IsNotExist(err) {
					return nil
				}
				return err
			}
			if !d.IsDir() || path == root {
				return nil
			}
			name := d.Name()
			if !strings.HasSuffix(name, ".git") {
				return filepath.SkipDir
			}
			name = name[:len(name)-4]
			if repo_model.IsUsableRepoName(name) != nil || strings.ToLower(name) != name {
				return filepath.SkipDir
			}
			if count >= start && count < end {
				repoNames = append(repoNames, name)
			}
			count++
			return filepath.SkipDir
		}); err != nil {
			ctx.ServerError("filepath.WalkDir", err)
			return
		}

		userRepos, _, err := repo_model.GetUserRepositories(&repo_model.SearchRepoOptions{
			Actor:   ctxUser,
			Private: true,
			ListOptions: db.ListOptions{
				Page:     1,
				PageSize: setting.UI.Admin.UserPagingNum,
			},
			LowerNames: repoNames,
		})
		if err != nil {
			ctx.ServerError("GetUserRepositories", err)
			return
		}
		for _, repo := range userRepos {
			if repo.IsFork {
				if err := repo.GetBaseRepo(ctx); err != nil {
					ctx.ServerError("GetBaseRepo", err)
					return
				}
			}
			repos[repo.LowerName] = repo
		}
		ctx.Data["Dirs"] = repoNames
		ctx.Data["ReposMap"] = repos
	} else if !setting.SourceControl.TenantWithRoleModeEnabled {
		repos, count64, err := repo_model.GetUserRepositories(&repo_model.SearchRepoOptions{Actor: ctxUser, Private: true, ListOptions: opts})
		if err != nil {
			ctx.ServerError("GetUserRepositories", err)
			return
		}
		count = int(count64)

		for i := range repos {
			if repos[i].IsFork {
				if err := repos[i].GetBaseRepo(ctx); err != nil {
					ctx.ServerError("GetBaseRepo", err)
					return
				}
			}
		}

		ctx.Data["Repos"] = repos
	} else {
		// TenantWithRoleModeEnabled = true получаем репозитории по проектам из тенатнов
		repoNames := make([]string, 0, setting.UI.Admin.UserPagingNum)
		repos := map[string]*repo_model.Repository{}
		organizationIDs := make([]int64, 0)
		tenantID, errGetTenantIdByUserId := role_model.GetUserTenantId(ctx, ctx.Doer.ID)
		if errGetTenantIdByUserId != nil {
			log.Error("Repos role_model.GetUserTenantId failed: %v", errGetTenantIdByUserId)
			return
		}
		if setting.SourceControl.TenantWithRoleModeEnabled {
			usersPrivileges, err := utils.GetTenantsPrivilegesByUserID(ctx, ctx.Doer.ID)
			if err != nil {
				ctx.Error(http.StatusNotFound, fmt.Sprintf("Error has occurred while getting user's privileges: %v", err))
				return
			}
			organizationsRepository := utils.ConvertPrivilegesTenantFromOrganizationsOrUsers(usersPrivileges, user_model.UserTypeOrganization)
			for organizationID := range organizationsRepository {
				allowed, errCheckPermission := s.orgRequestAccessor.IsReadAccessGranted(ctx, accesser.OrgAccessRequest{
					DoerID:         ctx.Doer.ID,
					TargetOrgID:    organizationID,
					TargetTenantID: tenantID,
					Action:         role_model.READ,
				})
				if errCheckPermission != nil {
					log.Error("Repos role_model.CheckUserPermissionToOrganization failed: %v", errCheckPermission)
					ctx.Error(http.StatusNotFound, fmt.Sprintf("Repos role_model.CheckUserPermissionToOrganization failed: %v", errCheckPermission))
					return
				}
				if allowed {
					organizationIDs = append(organizationIDs, organizationID)
				}
			}
		}
		userRepos, _, err := repo_model.GetUserRepositories(&repo_model.SearchRepoOptions{
			Actor:   ctxUser,
			Private: true,
			ListOptions: db.ListOptions{
				Page:     1,
				PageSize: setting.UI.Admin.UserPagingNum,
			},
			LowerNames: repoNames,
			OwnerIDs:   organizationIDs,
		})
		if err != nil {
			ctx.ServerError("GetUserRepositories", err)
			return
		}
		for _, orgId := range organizationIDs {
			for _, repo := range userRepos {
				allowed, err := s.orgRequestAccessor.IsReadAccessGranted(ctx, accesser.OrgAccessRequest{
					DoerID:         ctxUser.ID,
					TargetOrgID:    orgId,
					TargetTenantID: tenantID,
					Action:         role_model.READ,
				})
				if err != nil {
					log.Error("Error has occurred while checking permission's: %v", err)
					ctx.Error(http.StatusNotFound, fmt.Sprintf("Error has occurred while checking permission's: %v", err))
				}
				if !allowed {
					allow, err := s.repoRequestAccessor.AccessesByCustomPrivileges(ctx, accesser.RepoAccessRequest{
						DoerID:          ctxUser.ID,
						RepoID:          repo.ID,
						TargetTenantID:  tenantID,
						OrgID:           orgId,
						CustomPrivilege: role_model.ViewBranch.String(),
					})
					if err != nil {
						log.Error("Error has occurred while checking custom permission's: %v", err)
						ctx.Error(http.StatusNotFound, fmt.Sprintf("Error has occurred while checking custom permission's: %v", err))
						return
					}
					if !allow {
						continue
					}
				}
				repos[repo.LowerName] = repo
			}
		}
		for _, repo := range userRepos {
			if repo.IsFork {
				if err := repo.GetBaseRepo(ctx); err != nil {
					ctx.ServerError("GetBaseRepo", err)
					return
				}
			}
			repos[repo.LowerName] = repo
		}
		repNames := make([]string, 0, len(repos))
		for _, repo := range repos {
			repNames = append(repNames, repo.Name)
		}
		ctx.Data["Dirs"] = repNames
		ctx.Data["ReposMap"] = repos
		ctx.Data["Repos"] = repos
	}

	ctx.Data["ContextUser"] = ctxUser
	pager := context.NewPagination(count, opts.PageSize, opts.Page, 5)
	pager.SetDefaultParams(ctx)
	ctx.Data["Page"] = pager
	ctx.HTML(http.StatusOK, tplSettingsRepositories)
}
