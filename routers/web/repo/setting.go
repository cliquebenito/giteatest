// Copyright 2014 The Gogs Authors. All rights reserved.
// Copyright 2018 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"code.gitea.io/gitea/modules/trace"
	"gitlab.com/gitlab-org/gitaly/v16/proto/go/gitalypb"

	"code.gitea.io/gitea/models"
	asymkey_model "code.gitea.io/gitea/models/asymkey"
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/organization"
	"code.gitea.io/gitea/models/perm"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/role_model"
	"code.gitea.io/gitea/models/tenant"
	unit_model "code.gitea.io/gitea/models/unit"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/base"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/indexer/code"
	"code.gitea.io/gitea/modules/indexer/stats"
	"code.gitea.io/gitea/modules/json"
	"code.gitea.io/gitea/modules/lfs"
	"code.gitea.io/gitea/modules/log"
	mirror_module "code.gitea.io/gitea/modules/mirror"
	repo_module "code.gitea.io/gitea/modules/repository"
	"code.gitea.io/gitea/modules/sbt/audit"
	audit2 "code.gitea.io/gitea/modules/sbt/audit/utils"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/modules/typesniffer"
	"code.gitea.io/gitea/modules/util"
	"code.gitea.io/gitea/modules/validation"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/routers/utils"
	asymkey_service "code.gitea.io/gitea/services/asymkey"
	"code.gitea.io/gitea/services/forms"
	"code.gitea.io/gitea/services/mailer"
	"code.gitea.io/gitea/services/migrations"
	mirror_service "code.gitea.io/gitea/services/mirror"
	org_service "code.gitea.io/gitea/services/org"
	repo_service "code.gitea.io/gitea/services/repository"
	wiki_service "code.gitea.io/gitea/services/wiki"
)

const (
	tplSettingsOptions base.TplName = "repo/settings/options"
	tplCollaboration   base.TplName = "repo/settings/collaboration"
	tplBranches        base.TplName = "repo/settings/branches"
	tplTags            base.TplName = "repo/settings/tags"
	tplGithooks        base.TplName = "repo/settings/githooks"
	tplGithookEdit     base.TplName = "repo/settings/githook_edit"
	tplDeployKeys      base.TplName = "repo/settings/deploy_keys"
	tplDeleteRepo      base.TplName = "repo/settings/delete"
)

// SettingsCtxData is a middleware that sets all the general context data for the
// settings template.
func SettingsCtxData(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("repo.settings.options")
	ctx.Data["PageIsSettingsOptions"] = true
	ctx.Data["ForcePrivate"] = setting.Repository.ForcePrivate
	ctx.Data["MirrorsEnabled"] = setting.Mirror.Enabled
	ctx.Data["DisableNewPushMirrors"] = setting.Mirror.DisableNewPush
	ctx.Data["DefaultMirrorInterval"] = setting.Mirror.DefaultInterval
	ctx.Data["MinimumMirrorInterval"] = setting.Mirror.MinInterval
	ctx.Data["username"] = ctx.ContextUser.Name
	ctx.Data["reponame"] = ctx.Repo.Repository.Name

	signing, _ := asymkey_service.SigningKey(ctx, ctx.Repo.Repository.RepoPath())
	ctx.Data["SigningKeyAvailable"] = len(signing) > 0
	ctx.Data["SigningSettings"] = setting.Repository.Signing
	ctx.Data["CodeIndexerEnabled"] = setting.Indexer.RepoIndexerEnabled

	if ctx.Doer.IsAdmin {
		if setting.Indexer.RepoIndexerEnabled {
			status, err := repo_model.GetIndexerStatus(ctx, ctx.Repo.Repository, repo_model.RepoIndexerTypeCode)
			if err != nil {
				ctx.ServerError("repo.indexer_status", err)
				return
			}
			ctx.Data["CodeIndexerStatus"] = status
		}
		status, err := repo_model.GetIndexerStatus(ctx, ctx.Repo.Repository, repo_model.RepoIndexerTypeStats)
		if err != nil {
			ctx.ServerError("repo.indexer_status", err)
			return
		}
		ctx.Data["StatsIndexerStatus"] = status
	}
	pushMirrors, _, err := repo_model.GetPushMirrorsByRepoID(ctx, ctx.Repo.Repository.ID, db.ListOptions{})
	if err != nil {
		ctx.ServerError("GetPushMirrorsByRepoID", err)
		return
	}
	codeOwners, err := repo_model.GetCodeOwnersSettings(ctx, ctx.Repo.Repository.ID)
	if err != nil {
		ctx.ServerError("Get Code Owners", err)
		return
	}
	ctx.Data["CodeOwnersSettings"] = codeOwners
	ctx.Data["PushMirrors"] = pushMirrors
}

// Settings show a repository's settings page
func Settings(ctx *context.Context) {
	ctx.HTML(http.StatusOK, tplSettingsOptions)
}

// ApplyCodeOwnersSettings applies the code owners settings
func ApplyCodeOwnersSettings(ctx *context.Context, form forms.RepoSettingForm) {
	auditValues := audit2.NewRequiredAuditParams(ctx)

	auditParams := map[string]string{
		"owner":           ctx.Repo.Repository.OwnerName,
		"repository":      ctx.Repo.Repository.Name,
		"repository_id":   strconv.FormatInt(ctx.Repo.Repository.ID, 10),
		"approval_status": strconv.FormatBool(form.CodeOwners.ApprovalStatus),
		"amount_users":    strconv.FormatInt(form.CodeOwners.AmountUsers, 10),
	}

	codeOwnersInfo, err := repo_model.GetCodeOwnersSettings(ctx, ctx.Repo.Repository.ID)
	if err != nil {
		auditParams["error"] = "Error has occurred while getting code owners settings"
		audit.CreateAndSendEvent(audit.CodeOwnersSettingsChangeEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
		ctx.ServerError("Get Code Owners", err)
		return
	}

	codeOwners := &repo_model.CodeOwnersSettings{
		RepositoryID:   ctx.Repo.Repository.ID,
		ApprovalStatus: form.CodeOwners.ApprovalStatus,
		AmountUsers:    form.CodeOwners.AmountUsers,
	}

	switch {
	case codeOwnersInfo.RepositoryID != codeOwners.RepositoryID:
		err = repo_model.InsertCodeOwners(ctx, codeOwners)
		if err != nil {
			auditParams["error"] = "Error has occurred while inserting code owners settings"
			audit.CreateAndSendEvent(audit.CodeOwnersSettingsGrantEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
			ctx.ServerError("Insert CodeOwners", err)
			return
		}
		audit.CreateAndSendEvent(audit.CodeOwnersSettingsGrantEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusSuccess, auditValues.RemoteAddress, auditParams)

	case !codeOwners.ApprovalStatus:
		err = repo_model.DeleteCodeOwners(ctx, codeOwners)
		if err != nil {
			auditParams["error"] = "Error has occurred while deleting code owners settings"
			audit.CreateAndSendEvent(audit.CodeOwnersSettingsRevokeEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
			ctx.ServerError("Delete CodeOwners", err)
			return
		}
		audit.CreateAndSendEvent(audit.CodeOwnersSettingsRevokeEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusSuccess, auditValues.RemoteAddress, auditParams)

	case codeOwnersInfo.RepositoryID == codeOwners.RepositoryID:
		err = repo_model.UpdateCodeOwners(ctx, codeOwners)
		if err != nil {
			auditParams["error"] = "Error has occurred while updating code owners settings"
			audit.CreateAndSendEvent(audit.CodeOwnersSettingsUpdateEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
			ctx.ServerError("Update CodeOwners", err)
			return
		}
		audit.CreateAndSendEvent(audit.CodeOwnersSettingsUpdateEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusSuccess, auditValues.RemoteAddress, auditParams)

	}
	audit.CreateAndSendEvent(audit.CodeOwnersSettingsChangeEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusSuccess, auditValues.RemoteAddress, auditParams)
	ctx.Flash.Success(ctx.Tr("repo.settings.update_settings_success"))
	ctx.Redirect(ctx.Repo.RepoLink + "/settings")
}

// SettingsPost response for changes of a repository
func SettingsPost(ctx *context.Context) {
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

	form := web.GetForm(ctx).(*forms.RepoSettingForm)

	ctx.Data["ForcePrivate"] = setting.Repository.ForcePrivate
	ctx.Data["MirrorsEnabled"] = setting.Mirror.Enabled
	ctx.Data["DisableNewPushMirrors"] = setting.Mirror.DisableNewPush
	ctx.Data["DefaultMirrorInterval"] = setting.Mirror.DefaultInterval
	ctx.Data["MinimumMirrorInterval"] = setting.Mirror.MinInterval

	signing, _ := asymkey_service.SigningKey(ctx, ctx.Repo.Repository.RepoPath())
	ctx.Data["SigningKeyAvailable"] = len(signing) > 0
	ctx.Data["SigningSettings"] = setting.Repository.Signing
	ctx.Data["CodeIndexerEnabled"] = setting.Indexer.RepoIndexerEnabled

	repo := ctx.Repo.Repository

	auditParams := map[string]string{
		"repository":    repo.Name,
		"repository_id": strconv.FormatInt(repo.ID, 10),
		"owner":         ctx.Repo.Owner.Name,
	}

	switch ctx.FormString("action") {
	case "owners":
		if !ctx.Repo.IsOwner() {
			ctx.Error(http.StatusForbidden)
			return
		}
		ApplyCodeOwnersSettings(ctx, *form)

	case "update":
		type auditValue struct {
			Name        string
			LowerName   string
			Description string
			Website     string
			IsTemplate  bool
			IsPrivate   bool
		}

		oldValue := auditValue{
			Name:        repo.Name,
			LowerName:   repo.LowerName,
			Description: repo.Description,
			Website:     repo.Website,
			IsTemplate:  repo.IsTemplate,
			IsPrivate:   repo.IsPrivate,
		}

		newValue := auditValue{
			Name:        repo.Name,
			LowerName:   repo.LowerName,
			Description: form.Description,
			Website:     form.Website,
			IsTemplate:  form.Template,
			IsPrivate:   form.Private,
		}

		oldValueBytes, _ := json.Marshal(oldValue)
		auditParams["old_value"] = string(oldValueBytes)

		newValueBytes, _ := json.Marshal(newValue)
		auditParams["new_value"] = string(newValueBytes)

		if ctx.HasError() {
			ctx.HTML(http.StatusOK, tplSettingsOptions)
			auditParams["error"] = "Error occurs in form validation"
			audit.CreateAndSendEvent(audit.RepositorySettingsChangeEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}

		// Check if repository name has been changed.
		if repo.LowerName != strings.ToLower(form.RepoName) {
			log.Warn("Repository name cannot change")
		}

		oldValueBytes, _ = json.Marshal(oldValue)

		auditParams["old_value"] = string(oldValueBytes)
		auditParams["new_value"] = string(newValueBytes)

		repo.Description = form.Description
		repo.Website = form.Website
		repo.IsTemplate = form.Template

		// Visibility of forked repository is forced sync with base repository.
		if repo.IsFork {
			form.Private = repo.BaseRepo.IsPrivate || repo.BaseRepo.Owner.Visibility == structs.VisibleTypePrivate
		}

		visibilityChanged := repo.IsPrivate != form.Private
		// when ForcePrivate enabled, you could change public repo to private, but only admin users can change private to public
		if visibilityChanged && setting.Repository.ForcePrivate && !form.Private && !ctx.Doer.IsAdmin {
			ctx.RenderWithErr(ctx.Tr("form.repository_force_private"), tplSettingsOptions, form)
			auditParams["error"] = "Only admin users can change private to public repository"
			audit.CreateAndSendEvent(audit.RepositorySettingsChangeEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}

		repo.IsPrivate = form.Private
		if err := repo_service.UpdateRepository(ctx, repo, visibilityChanged); err != nil {
			ctx.ServerError("UpdateRepository", err)
			auditParams["error"] = "Error has occurred while updating repository"
			audit.CreateAndSendEvent(audit.RepositorySettingsChangeEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}
		log.Trace("Repository basic settings updated: %s/%s", ctx.Repo.Owner.Name, repo.Name)

		audit.CreateAndSendEvent(audit.RepositorySettingsChangeEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)
		ctx.Flash.Success(ctx.Tr("repo.settings.update_settings_success"))
		ctx.Redirect(repo.Link() + "/settings")

	case "mirror":
		if !setting.Mirror.Enabled {
			break
		}

		if !setting.Mirror.Enabled || !repo.IsMirror {
			ctx.NotFound("", nil)
			return
		}

		pullMirror, err := repo_model.GetMirrorByRepoID(ctx, ctx.Repo.Repository.ID)
		if err == repo_model.ErrMirrorNotExist {
			ctx.NotFound("", nil)
			return
		}
		if err != nil {
			ctx.ServerError("GetMirrorByRepoID", err)
			return
		}
		// This section doesn't require repo_name/RepoName to be set in the form, don't show it
		// as an error on the UI for this action
		ctx.Data["Err_RepoName"] = nil

		interval, err := time.ParseDuration(form.Interval)
		if err != nil || (interval != 0 && interval < setting.Mirror.MinInterval) {
			ctx.Data["Err_Interval"] = true
			ctx.RenderWithErr(ctx.Tr("repo.mirror_interval_invalid"), tplSettingsOptions, &form)
			return
		}

		pullMirror.EnablePrune = form.EnablePrune
		pullMirror.Interval = interval
		pullMirror.ScheduleNextUpdate()
		if err := repo_model.UpdateMirror(ctx, pullMirror); err != nil {
			ctx.ServerError("UpdateMirror", err)
			return
		}

		u, err := git.GetRemoteURL(ctx, ctx.Repo.Repository.RepoPath(), pullMirror.GetRemoteName())
		if err != nil {
			ctx.Data["Err_MirrorAddress"] = true
			handleSettingRemoteAddrError(ctx, err, form)
			return
		}
		if u.User != nil && form.MirrorPassword == "" && form.MirrorUsername == u.User.Username() {
			form.MirrorPassword, _ = u.User.Password()
		}

		address, err := forms.ParseRemoteAddr(form.MirrorAddress, form.MirrorUsername, form.MirrorPassword)
		if err == nil {
			err = migrations.IsMigrateURLAllowed(address, ctx.Doer)
		}
		if err != nil {
			ctx.Data["Err_MirrorAddress"] = true
			handleSettingRemoteAddrError(ctx, err, form)
			return
		}

		if err := mirror_service.UpdateAddress(ctx, pullMirror, address); err != nil {
			ctx.ServerError("UpdateAddress", err)
			return
		}

		form.LFS = form.LFS && setting.LFS.StartServer

		if len(form.LFSEndpoint) > 0 {
			ep := lfs.DetermineEndpoint("", form.LFSEndpoint)
			if ep == nil {
				ctx.Data["Err_LFSEndpoint"] = true
				ctx.RenderWithErr(ctx.Tr("repo.migrate.invalid_lfs_endpoint"), tplSettingsOptions, &form)
				return
			}
			err = migrations.IsMigrateURLAllowed(ep.String(), ctx.Doer)
			if err != nil {
				ctx.Data["Err_LFSEndpoint"] = true
				handleSettingRemoteAddrError(ctx, err, form)
				return
			}
		}

		pullMirror.LFS = form.LFS
		pullMirror.LFSEndpoint = form.LFSEndpoint
		if err := repo_model.UpdateMirror(ctx, pullMirror); err != nil {
			ctx.ServerError("UpdateMirror", err)
			return
		}

		ctx.Flash.Success(ctx.Tr("repo.settings.update_settings_success"))
		ctx.Redirect(repo.Link() + "/settings")

	case "mirror-sync":
		if !setting.Mirror.Enabled {
			break
		}

		if !setting.Mirror.Enabled || !repo.IsMirror {
			ctx.NotFound("", nil)
			return
		}

		mirror_module.AddPullMirrorToQueue(repo.ID)

		ctx.Flash.Info(ctx.Tr("repo.settings.mirror_sync_in_progress"))
		ctx.Redirect(repo.Link() + "/settings")

	case "push-mirror-sync":
		if !setting.Mirror.Enabled {
			break
		}

		if !setting.Mirror.Enabled {
			ctx.NotFound("", nil)
			return
		}

		m, err := selectPushMirrorByForm(ctx, form, repo)
		if err != nil {
			ctx.NotFound("", nil)
			return
		}

		mirror_module.AddPushMirrorToQueue(m.ID)

		ctx.Flash.Info(ctx.Tr("repo.settings.mirror_sync_in_progress"))
		ctx.Redirect(repo.Link() + "/settings")

	case "push-mirror-remove":
		if !setting.Mirror.Enabled {
			break
		}

		if !setting.Mirror.Enabled {
			ctx.NotFound("", nil)
			return
		}

		// This section doesn't require repo_name/RepoName to be set in the form, don't show it
		// as an error on the UI for this action
		ctx.Data["Err_RepoName"] = nil

		m, err := selectPushMirrorByForm(ctx, form, repo)
		if err != nil {
			ctx.NotFound("", nil)
			return
		}

		if err = mirror_service.RemovePushMirrorRemote(ctx, m); err != nil {
			ctx.ServerError("RemovePushMirrorRemote", err)
			return
		}

		if err = repo_model.DeletePushMirrors(ctx, repo_model.PushMirrorOptions{ID: m.ID, RepoID: m.RepoID}); err != nil {
			ctx.ServerError("DeletePushMirrorByID", err)
			return
		}

		ctx.Flash.Success(ctx.Tr("repo.settings.update_settings_success"))
		ctx.Redirect(repo.Link() + "/settings")

	case "push-mirror-add":
		if !setting.Mirror.Enabled {
			break
		}

		if setting.Mirror.DisableNewPush {
			ctx.NotFound("", nil)
			return
		}

		// This section doesn't require repo_name/RepoName to be set in the form, don't show it
		// as an error on the UI for this action
		ctx.Data["Err_RepoName"] = nil

		interval, err := time.ParseDuration(form.PushMirrorInterval)
		if err != nil || (interval != 0 && interval < setting.Mirror.MinInterval) {
			ctx.Data["Err_PushMirrorInterval"] = true
			ctx.RenderWithErr(ctx.Tr("repo.mirror_interval_invalid"), tplSettingsOptions, &form)
			return
		}

		address, err := forms.ParseRemoteAddr(form.PushMirrorAddress, form.PushMirrorUsername, form.PushMirrorPassword)
		if err == nil {
			err = migrations.IsMigrateURLAllowed(address, ctx.Doer)
		}
		if err != nil {
			ctx.Data["Err_PushMirrorAddress"] = true
			handleSettingRemoteAddrError(ctx, err, form)
			return
		}

		remoteSuffix, err := util.CryptoRandomString(10)
		if err != nil {
			ctx.ServerError("RandomString", err)
			return
		}

		m := &repo_model.PushMirror{
			RepoID:       repo.ID,
			Repo:         repo,
			RemoteName:   fmt.Sprintf("remote_mirror_%s", remoteSuffix),
			SyncOnCommit: form.PushMirrorSyncOnCommit,
			Interval:     interval,
		}
		if err := repo_model.InsertPushMirror(ctx, m); err != nil {
			ctx.ServerError("InsertPushMirror", err)
			return
		}

		if err := mirror_service.AddPushMirrorRemote(ctx, m, address); err != nil {
			if err := repo_model.DeletePushMirrors(ctx, repo_model.PushMirrorOptions{ID: m.ID, RepoID: m.RepoID}); err != nil {
				log.Error("DeletePushMirrors %v", err)
			}
			ctx.ServerError("AddPushMirrorRemote", err)
			return
		}

		ctx.Flash.Success(ctx.Tr("repo.settings.update_settings_success"))
		ctx.Redirect(repo.Link() + "/settings")

	case "advanced":
		var repoChanged bool
		var units []repo_model.RepoUnit
		var deleteUnitTypes []unit_model.Type
		var auditEvents []audit.Event

		type auditValue struct {
			EnableCode                            bool
			EnableWiki                            bool
			EnableExternalWiki                    bool
			ExternalWikiURL                       string
			EnableIssues                          bool
			EnableExternalTracker                 bool
			ExternalTrackerURL                    string
			TrackerURLFormat                      string
			TrackerIssueStyle                     string
			ExternalTrackerRegexpPattern          string
			EnableCloseIssuesViaCommitInAnyBranch bool
			EnableProjects                        bool
			EnableReleases                        bool
			EnablePackages                        bool
			EnablePulls                           bool
			EnableActions                         bool
			PullsIgnoreWhitespace                 bool
			PullsAllowMerge                       bool
			PullsAdminCanMergeWithoutChecks       bool
			PullsAllowRebase                      bool
			PullsAllowRebaseMerge                 bool
			PullsAllowSquash                      bool
			PullsAllowManualMerge                 bool
			PullsDefaultMergeStyle                string
			EnableAutodetectManualMerge           bool
			PullsAllowRebaseUpdate                bool
			DefaultDeleteBranchAfterMerge         bool
			DefaultAllowMaintainerEdit            bool
			EnableTimetracker                     bool
			AllowOnlyContributorsToTrackTime      bool
			EnableIssueDependencies               bool
			IsEmptyProtectedBranchRule            bool
		}

		newValue := auditValue{
			EnableCode:                            form.EnableCode,
			EnableWiki:                            form.EnableWiki,
			EnableExternalWiki:                    form.EnableExternalWiki,
			ExternalWikiURL:                       form.ExternalWikiURL,
			EnableIssues:                          form.EnableIssues,
			EnableExternalTracker:                 form.EnableExternalTracker,
			ExternalTrackerURL:                    form.ExternalTrackerURL,
			TrackerURLFormat:                      form.TrackerURLFormat,
			TrackerIssueStyle:                     form.TrackerIssueStyle,
			ExternalTrackerRegexpPattern:          form.ExternalTrackerRegexpPattern,
			EnableCloseIssuesViaCommitInAnyBranch: form.EnableCloseIssuesViaCommitInAnyBranch,
			EnableProjects:                        form.EnableProjects,
			EnableReleases:                        form.EnableReleases,
			EnablePackages:                        form.EnablePackages,
			EnablePulls:                           form.EnablePulls,
			EnableActions:                         form.EnableActions,
			PullsIgnoreWhitespace:                 form.PullsIgnoreWhitespace,
			PullsAllowMerge:                       form.PullsAllowMerge,
			PullsAdminCanMergeWithoutChecks:       form.PullsAdminCanMergeWithoutChecks,
			PullsAllowRebase:                      form.PullsAllowRebase,
			PullsAllowRebaseMerge:                 form.PullsAllowRebaseMerge,
			PullsAllowSquash:                      form.PullsAllowSquash,
			PullsAllowManualMerge:                 form.PullsAllowManualMerge,
			PullsDefaultMergeStyle:                form.PullsDefaultMergeStyle,
			EnableAutodetectManualMerge:           form.EnableAutodetectManualMerge,
			PullsAllowRebaseUpdate:                form.PullsAllowRebaseUpdate,
			DefaultDeleteBranchAfterMerge:         form.DefaultDeleteBranchAfterMerge,
			DefaultAllowMaintainerEdit:            form.DefaultAllowMaintainerEdit,
			EnableTimetracker:                     form.EnableTimetracker,
			AllowOnlyContributorsToTrackTime:      form.AllowOnlyContributorsToTrackTime,
			EnableIssueDependencies:               form.EnableIssueDependencies,
			IsEmptyProtectedBranchRule:            form.IsEmptyProtectedBranchRule,
		}

		oldValue := auditValue{
			EnableCode:            unitTypeExistInSliceOfUnits(repo.Units, unit_model.TypeCode),
			EnableWiki:            unitTypeExistInSliceOfUnits(repo.Units, unit_model.TypeWiki),
			EnableExternalWiki:    unitTypeExistInSliceOfUnits(repo.Units, unit_model.TypeExternalWiki),
			EnableIssues:          unitTypeExistInSliceOfUnits(repo.Units, unit_model.TypeIssues),
			EnableExternalTracker: unitTypeExistInSliceOfUnits(repo.Units, unit_model.TypeExternalTracker),
			EnableProjects:        unitTypeExistInSliceOfUnits(repo.Units, unit_model.TypeProjects),
			EnableReleases:        unitTypeExistInSliceOfUnits(repo.Units, unit_model.TypeReleases),
			EnablePackages:        unitTypeExistInSliceOfUnits(repo.Units, unit_model.TypePackages),
			EnableActions:         unitTypeExistInSliceOfUnits(repo.Units, unit_model.TypeActions),
			EnablePulls:           unitTypeExistInSliceOfUnits(repo.Units, unit_model.TypePullRequests),
		}
		if oldValue.EnableExternalWiki {
			oldValue.ExternalWikiURL = findUnitByType(repo.Units, unit_model.TypeExternalWiki).Config.(*repo_model.ExternalWikiConfig).ExternalWikiURL
		}
		if oldValue.EnableExternalTracker {
			config := findUnitByType(repo.Units, unit_model.TypeExternalTracker).Config.(*repo_model.ExternalTrackerConfig)
			oldValue.ExternalTrackerURL = config.ExternalTrackerURL
			oldValue.TrackerURLFormat = config.ExternalTrackerFormat
			oldValue.TrackerIssueStyle = config.ExternalTrackerStyle
			oldValue.ExternalTrackerRegexpPattern = config.ExternalTrackerRegexpPattern
		}
		if oldValue.EnableIssues {
			config := findUnitByType(repo.Units, unit_model.TypeIssues).Config.(*repo_model.IssuesConfig)
			oldValue.EnableTimetracker = config.EnableTimetracker
			oldValue.AllowOnlyContributorsToTrackTime = config.AllowOnlyContributorsToTrackTime
			oldValue.EnableIssueDependencies = config.EnableDependencies
		}
		if oldValue.EnablePulls {
			config := findUnitByType(repo.Units, unit_model.TypePullRequests).Config.(*repo_model.PullRequestsConfig)
			oldValue.PullsIgnoreWhitespace = config.IgnoreWhitespaceConflicts
			oldValue.PullsAllowMerge = config.AllowMerge
			oldValue.PullsAdminCanMergeWithoutChecks = config.AdminCanMergeWithoutChecks
			oldValue.PullsAllowRebase = config.AllowRebase
			oldValue.PullsAllowRebaseMerge = config.AllowRebaseMerge
			oldValue.PullsAllowSquash = config.AllowSquash
			oldValue.PullsAllowManualMerge = config.AllowManualMerge
			oldValue.EnableAutodetectManualMerge = config.AutodetectManualMerge
			oldValue.PullsAllowRebaseUpdate = config.AllowRebaseUpdate
			oldValue.DefaultDeleteBranchAfterMerge = config.DefaultDeleteBranchAfterMerge
			oldValue.PullsDefaultMergeStyle = string(config.DefaultMergeStyle)
			oldValue.DefaultAllowMaintainerEdit = config.DefaultAllowMaintainerEdit
		}

		oldValueBytes, _ := json.Marshal(oldValue)
		auditParams["old_value"] = string(oldValueBytes)
		newValueBytes, _ := json.Marshal(newValue)
		auditParams["new_value"] = string(newValueBytes)

		// если у нас включена ролевая модель SourceControl, то запускается проверка привилегий на возможность включения слияния без проверок
		if setting.SourceControl.TenantWithRoleModeEnabled {
			if oldValue.PullsAdminCanMergeWithoutChecks != newValue.PullsAdminCanMergeWithoutChecks {
				tenantId, err := tenant.GetTenantByOrgIdOrDefault(ctx, ctx.Repo.Repository.OwnerID)
				if err != nil {
					ctx.NotFound(ctx.Req.URL.RequestURI(), nil)
					return
				}

				allowed, err := role_model.CheckUserPermissionToOrganization(ctx, ctx.Doer, tenantId, &organization.Organization{ID: ctx.Repo.Repository.OwnerID}, role_model.MERGE_WITHOUT_CHECK)
				if err != nil || !allowed {
					ctx.NotFound(ctx.Req.URL.RequestURI(), nil)
					return
				}
			}
		}

		// This section doesn't require repo_name/RepoName to be set in the form, don't show it
		// as an error on the UI for this action
		ctx.Data["Err_RepoName"] = nil

		if repo.CloseIssuesViaCommitInAnyBranch != form.EnableCloseIssuesViaCommitInAnyBranch {
			repo.CloseIssuesViaCommitInAnyBranch = form.EnableCloseIssuesViaCommitInAnyBranch
			repoChanged = true
		}

		if form.EnableCode && !unit_model.TypeCode.UnitGlobalDisabled() {
			units = append(units, repo_model.RepoUnit{
				RepoID: repo.ID,
				Type:   unit_model.TypeCode,
			})
		} else if !unit_model.TypeCode.UnitGlobalDisabled() {
			deleteUnitTypes = append(deleteUnitTypes, unit_model.TypeCode)
		}

		if form.EnableWiki && form.EnableExternalWiki && !unit_model.TypeExternalWiki.UnitGlobalDisabled() {
			if !validation.IsValidExternalURL(form.ExternalWikiURL) {
				ctx.Flash.Error(ctx.Tr("repo.settings.external_wiki_url_error"))
				ctx.Redirect(repo.Link() + "/settings")
				auditParams["error"] = "External wiki url is invalid"
				audit.CreateAndSendEvent(audit.RepositorySettingsChangeEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
				return
			}

			units = append(units, repo_model.RepoUnit{
				RepoID: repo.ID,
				Type:   unit_model.TypeExternalWiki,
				Config: &repo_model.ExternalWikiConfig{
					ExternalWikiURL: form.ExternalWikiURL,
				},
			})
			deleteUnitTypes = append(deleteUnitTypes, unit_model.TypeWiki)
		} else if form.EnableWiki && !form.EnableExternalWiki && !unit_model.TypeWiki.UnitGlobalDisabled() {
			units = append(units, repo_model.RepoUnit{
				RepoID: repo.ID,
				Type:   unit_model.TypeWiki,
				Config: new(repo_model.UnitConfig),
			})
			deleteUnitTypes = append(deleteUnitTypes, unit_model.TypeExternalWiki)
		} else {
			if !unit_model.TypeExternalWiki.UnitGlobalDisabled() {
				deleteUnitTypes = append(deleteUnitTypes, unit_model.TypeExternalWiki)
			}
			if !unit_model.TypeWiki.UnitGlobalDisabled() {
				deleteUnitTypes = append(deleteUnitTypes, unit_model.TypeWiki)
			}
		}

		if form.EnableIssues && form.EnableExternalTracker && !unit_model.TypeExternalTracker.UnitGlobalDisabled() {
			if !validation.IsValidExternalURL(form.ExternalTrackerURL) {
				ctx.Flash.Error(ctx.Tr("repo.settings.external_tracker_url_error"))
				ctx.Redirect(repo.Link() + "/settings")
				auditParams["error"] = "External tracker url is invalid"
				audit.CreateAndSendEvent(audit.RepositorySettingsChangeEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
				return
			}
			if len(form.TrackerURLFormat) != 0 && !validation.IsValidExternalTrackerURLFormat(form.TrackerURLFormat) {
				ctx.Flash.Error(ctx.Tr("repo.settings.tracker_url_format_error"))
				ctx.Redirect(repo.Link() + "/settings")
				auditParams["error"] = "External tracker url format is invalid"
				audit.CreateAndSendEvent(audit.RepositorySettingsChangeEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
				return
			}
			units = append(units, repo_model.RepoUnit{
				RepoID: repo.ID,
				Type:   unit_model.TypeExternalTracker,
				Config: &repo_model.ExternalTrackerConfig{
					ExternalTrackerURL:           form.ExternalTrackerURL,
					ExternalTrackerFormat:        form.TrackerURLFormat,
					ExternalTrackerStyle:         form.TrackerIssueStyle,
					ExternalTrackerRegexpPattern: form.ExternalTrackerRegexpPattern,
				},
			})
			deleteUnitTypes = append(deleteUnitTypes, unit_model.TypeIssues)
		} else if form.EnableIssues && !form.EnableExternalTracker && !unit_model.TypeIssues.UnitGlobalDisabled() {
			units = append(units, repo_model.RepoUnit{
				RepoID: repo.ID,
				Type:   unit_model.TypeIssues,
				Config: &repo_model.IssuesConfig{
					EnableTimetracker:                form.EnableTimetracker,
					AllowOnlyContributorsToTrackTime: form.AllowOnlyContributorsToTrackTime,
					EnableDependencies:               form.EnableIssueDependencies,
				},
			})
			deleteUnitTypes = append(deleteUnitTypes, unit_model.TypeExternalTracker)
		} else {
			if !unit_model.TypeExternalTracker.UnitGlobalDisabled() {
				deleteUnitTypes = append(deleteUnitTypes, unit_model.TypeExternalTracker)
			}
			if !unit_model.TypeIssues.UnitGlobalDisabled() {
				deleteUnitTypes = append(deleteUnitTypes, unit_model.TypeIssues)
			}
		}

		if form.EnableProjects && !unit_model.TypeProjects.UnitGlobalDisabled() {
			units = append(units, repo_model.RepoUnit{
				RepoID: repo.ID,
				Type:   unit_model.TypeProjects,
			})
		} else if !unit_model.TypeProjects.UnitGlobalDisabled() {
			deleteUnitTypes = append(deleteUnitTypes, unit_model.TypeProjects)
		}

		if form.EnableReleases && !unit_model.TypeReleases.UnitGlobalDisabled() {
			units = append(units, repo_model.RepoUnit{
				RepoID: repo.ID,
				Type:   unit_model.TypeReleases,
			})
		} else if !unit_model.TypeReleases.UnitGlobalDisabled() {
			deleteUnitTypes = append(deleteUnitTypes, unit_model.TypeReleases)
		}

		if form.EnablePackages && !unit_model.TypePackages.UnitGlobalDisabled() {
			units = append(units, repo_model.RepoUnit{
				RepoID: repo.ID,
				Type:   unit_model.TypePackages,
			})
		} else if !unit_model.TypePackages.UnitGlobalDisabled() {
			deleteUnitTypes = append(deleteUnitTypes, unit_model.TypePackages)
		}

		if form.EnableActions && !unit_model.TypeActions.UnitGlobalDisabled() {
			units = append(units, repo_model.RepoUnit{
				RepoID: repo.ID,
				Type:   unit_model.TypeActions,
			})
		} else if !unit_model.TypeActions.UnitGlobalDisabled() {
			deleteUnitTypes = append(deleteUnitTypes, unit_model.TypeActions)
		}

		if form.EnablePulls && !unit_model.TypePullRequests.UnitGlobalDisabled() {
			units = append(units, repo_model.RepoUnit{
				RepoID: repo.ID,
				Type:   unit_model.TypePullRequests,
				Config: &repo_model.PullRequestsConfig{
					IgnoreWhitespaceConflicts:     form.PullsIgnoreWhitespace,
					AllowMerge:                    form.PullsAllowMerge,
					AdminCanMergeWithoutChecks:    form.PullsAdminCanMergeWithoutChecks,
					AllowRebase:                   form.PullsAllowRebase,
					AllowRebaseMerge:              form.PullsAllowRebaseMerge,
					AllowSquash:                   form.PullsAllowSquash,
					AllowManualMerge:              form.PullsAllowManualMerge,
					AutodetectManualMerge:         form.EnableAutodetectManualMerge,
					AllowRebaseUpdate:             form.PullsAllowRebaseUpdate,
					DefaultDeleteBranchAfterMerge: form.DefaultDeleteBranchAfterMerge,
					DefaultMergeStyle:             repo_model.MergeStyle(form.PullsDefaultMergeStyle),
					DefaultAllowMaintainerEdit:    form.DefaultAllowMaintainerEdit,
				},
			})
			if form.DefaultDeleteBranchAfterMerge {
				auditEvents = append(auditEvents, audit.BranchDeleteAfterMergeSettingEnableEvent)
			} else {
				auditEvents = append(auditEvents, audit.BranchDeleteAfterMergeSettingDisableEvent)
			}
			auditEvents = append(auditEvents, audit.PRMergeSettingUpdateEvent)
		} else if !unit_model.TypePullRequests.UnitGlobalDisabled() {
			deleteUnitTypes = append(deleteUnitTypes, unit_model.TypePullRequests)
			auditEvents = append(auditEvents, audit.PRMergeSettingDeleteEvent)
		}

		if len(units) == 0 {
			ctx.Flash.Error(ctx.Tr("repo.settings.update_settings_no_unit"))
			ctx.Redirect(ctx.Repo.RepoLink + "/settings")
			auditParams["error"] = "Update settings hasn't enable units"
			for _, event := range auditEvents {
				audit.CreateAndSendEvent(event, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			}
			audit.CreateAndSendEvent(audit.RepositorySettingsChangeEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}

		if err := repo_model.UpdateRepositoryUnits(repo, units, deleteUnitTypes); err != nil {
			ctx.ServerError("UpdateRepositoryUnits", err)
			auditParams["error"] = "Error has occurred while updating repository units"
			for _, event := range auditEvents {
				audit.CreateAndSendEvent(event, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			}
			audit.CreateAndSendEvent(audit.RepositorySettingsChangeEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}

		for _, event := range auditEvents {
			audit.CreateAndSendEvent(event, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)
		}

		if repoChanged {
			if err := repo_service.UpdateRepository(ctx, repo, false); err != nil {
				ctx.ServerError("UpdateRepository", err)
				auditParams["error"] = "Error has occurred while updating repository"
				audit.CreateAndSendEvent(audit.RepositorySettingsChangeEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
				return
			}
		}
		log.Trace("Repository advanced settings updated: %s/%s", ctx.Repo.Owner.Name, repo.Name)

		audit.CreateAndSendEvent(audit.RepositorySettingsChangeEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)
		ctx.Flash.Success(ctx.Tr("repo.settings.update_settings_success"))
		ctx.Redirect(ctx.Repo.RepoLink + "/settings")

	case "signing":
		changed := false
		trustModel := repo_model.ToTrustModel(form.TrustModel)

		type auditValue struct {
			TrustModel string
		}

		oldValue := auditValue{
			TrustModel: repo.TrustModel.String(),
		}
		newValue := auditValue{
			TrustModel: trustModel.String(),
		}

		oldValueBytes, _ := json.Marshal(oldValue)
		auditParams["old_value"] = string(oldValueBytes)
		newValueBytes, _ := json.Marshal(newValue)
		auditParams["new_value"] = string(newValueBytes)

		if trustModel != repo.TrustModel {
			repo.TrustModel = trustModel
			changed = true
		}

		if changed {
			if err := repo_service.UpdateRepository(ctx, repo, false); err != nil {
				ctx.ServerError("UpdateRepository", err)
				auditParams["error"] = "Error has occurred while updating repository"
				audit.CreateAndSendEvent(audit.RepositorySettingsChangeEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
				return
			}
			audit.CreateAndSendEvent(audit.RepositorySettingsChangeEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)
		}
		log.Trace("Repository signing settings updated: %s/%s", ctx.Repo.Owner.Name, repo.Name)

		ctx.Flash.Success(ctx.Tr("repo.settings.update_settings_success"))
		ctx.Redirect(ctx.Repo.RepoLink + "/settings")

	case "admin":
		type auditValue struct {
			EnableHealthCheck string
		}

		oldValue := auditValue{
			EnableHealthCheck: strconv.FormatBool(repo.IsFsckEnabled),
		}
		newValue := auditValue{
			EnableHealthCheck: strconv.FormatBool(form.EnableHealthCheck),
		}

		oldValueBytes, _ := json.Marshal(oldValue)
		auditParams["old_value"] = string(oldValueBytes)
		newValueBytes, _ := json.Marshal(newValue)
		auditParams["new_value"] = string(newValueBytes)

		if !ctx.Doer.IsAdmin {
			ctx.Error(http.StatusForbidden)
			auditParams["error"] = "Only admin users can change this settings"
			audit.CreateAndSendEvent(audit.RepositorySettingsChangeEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}

		if repo.IsFsckEnabled != form.EnableHealthCheck {
			repo.IsFsckEnabled = form.EnableHealthCheck
		}

		if err := repo_service.UpdateRepository(ctx, repo, false); err != nil {
			ctx.ServerError("UpdateRepository", err)
			auditParams["error"] = "Error has occurred while updating repository"
			audit.CreateAndSendEvent(audit.RepositorySettingsChangeEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}
		audit.CreateAndSendEvent(audit.RepositorySettingsChangeEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)

		log.Trace("Repository admin settings updated: %s/%s", ctx.Repo.Owner.Name, repo.Name)

		ctx.Flash.Success(ctx.Tr("repo.settings.update_settings_success"))
		ctx.Redirect(ctx.Repo.RepoLink + "/settings")

	case "admin_index":
		if !ctx.Doer.IsAdmin {
			ctx.Error(http.StatusForbidden)
			return
		}

		switch form.RequestReindexType {
		case "stats":
			if err := stats.UpdateRepoIndexer(ctx.Repo.Repository); err != nil {
				ctx.ServerError("UpdateStatsRepondexer", err)
				return
			}
		case "code":
			if !setting.Indexer.RepoIndexerEnabled {
				ctx.Error(http.StatusForbidden)
				return
			}
			code.UpdateRepoIndexer(ctx.Repo.Repository)
		default:
			ctx.NotFound("", nil)
			return
		}

		log.Trace("Repository reindex for %s requested: %s/%s", form.RequestReindexType, ctx.Repo.Owner.Name, repo.Name)

		ctx.Flash.Success(ctx.Tr("repo.settings.reindex_requested"))
		ctx.Redirect(ctx.Repo.RepoLink + "/settings")

	case "convert":
		if !ctx.Repo.IsOwner() {
			ctx.Error(http.StatusNotFound)
			return
		}
		if repo.Name != form.RepoName {
			ctx.RenderWithErr(ctx.Tr("form.enterred_invalid_repo_name"), tplSettingsOptions, nil)
			return
		}

		if !repo.IsMirror {
			ctx.Error(http.StatusNotFound)
			return
		}
		repo.IsMirror = false

		if _, err := repo_module.CleanUpMigrateInfo(ctx, repo); err != nil {
			ctx.ServerError("CleanUpMigrateInfo", err)
			return
		} else if err = repo_model.DeleteMirrorByRepoID(ctx.Repo.Repository.ID); err != nil {
			ctx.ServerError("DeleteMirrorByRepoID", err)
			return
		}
		log.Trace("Repository converted from mirror to regular: %s", repo.FullName())
		ctx.Flash.Success(ctx.Tr("repo.settings.convert_succeed"))
		ctx.Redirect(repo.Link())

	case "convert_fork":
		if !ctx.Repo.IsOwner() {
			ctx.Error(http.StatusNotFound)
			return
		}
		if err := repo.LoadOwner(ctx); err != nil {
			ctx.ServerError("Convert Fork", err)
			return
		}
		if repo.Name != form.RepoName {
			ctx.RenderWithErr(ctx.Tr("form.enterred_invalid_repo_name"), tplSettingsOptions, nil)
			return
		}

		if !repo.IsFork {
			ctx.Error(http.StatusNotFound)
			return
		}

		if !ctx.Repo.Owner.CanCreateRepo() {
			maxCreationLimit := ctx.Repo.Owner.MaxCreationLimit()
			msg := ctx.TrN(maxCreationLimit, "repo.form.reach_limit_of_creation_1", "repo.form.reach_limit_of_creation_n", maxCreationLimit)
			ctx.Flash.Error(msg)
			ctx.Redirect(repo.Link() + "/settings")
			return
		}

		if err := repo_service.ConvertForkToNormalRepository(ctx, repo); err != nil {
			log.Error("Unable to convert repository %-v from fork. Error: %v", repo, err)
			ctx.ServerError("Convert Fork", err)
			return
		}

		log.Trace("Repository converted from fork to regular: %s", repo.FullName())
		ctx.Flash.Success(ctx.Tr("repo.settings.convert_fork_succeed"))
		ctx.Redirect(repo.Link())

	case "transfer":
		// если у нас включена ролевая модель SourceControl, то функция передачи прав на репозиторий не доступна
		if setting.SourceControl.TenantWithRoleModeEnabled {
			ctx.NotFound(ctx.Req.URL.RequestURI(), nil)
			return
		}
		if !ctx.Repo.IsOwner() {
			ctx.Error(http.StatusNotFound)
			return
		}
		if repo.Name != form.RepoName {
			ctx.RenderWithErr(ctx.Tr("form.enterred_invalid_repo_name"), tplSettingsOptions, nil)
			return
		}

		newOwner, err := user_model.GetUserByName(ctx, ctx.FormString("new_owner_name"))
		if err != nil {
			if user_model.IsErrUserNotExist(err) {
				ctx.RenderWithErr(ctx.Tr("form.enterred_invalid_owner_name"), tplSettingsOptions, nil)
				return
			}
			ctx.ServerError("IsUserExist", err)
			return
		}

		if newOwner.Type == user_model.UserTypeOrganization {
			if !ctx.Doer.IsAdmin && newOwner.Visibility == structs.VisibleTypePrivate && !organization.OrgFromUser(newOwner).HasMemberWithUserID(ctx.Doer.ID) {
				// The user shouldn't know about this organization
				ctx.RenderWithErr(ctx.Tr("form.enterred_invalid_owner_name"), tplSettingsOptions, nil)
				return
			}
		}

		// Close the GitRepo if open
		if ctx.Repo.GitRepo != nil {
			ctx.Repo.GitRepo.Close()
			ctx.Repo.GitRepo = nil
		}

		if err := repo_service.StartRepositoryTransfer(ctx, ctx.Doer, newOwner, repo, nil); err != nil {
			if repo_model.IsErrRepoAlreadyExist(err) {
				ctx.RenderWithErr(ctx.Tr("repo.settings.new_owner_has_same_repo"), tplSettingsOptions, nil)
			} else if models.IsErrRepoTransferInProgress(err) {
				ctx.RenderWithErr(ctx.Tr("repo.settings.transfer_in_progress"), tplSettingsOptions, nil)
			} else {
				ctx.ServerError("TransferOwnership", err)
			}

			return
		}

		log.Trace("Repository transfer process was started: %s/%s -> %s", ctx.Repo.Owner.Name, repo.Name, newOwner)
		ctx.Flash.Success(ctx.Tr("repo.settings.transfer_started", newOwner.DisplayName()))
		ctx.Redirect(repo.Link() + "/settings")

	case "cancel_transfer":
		// если у нас включена ролевая модель SourceControl, то функция передачи прав на репозиторий не доступна
		if setting.SourceControl.TenantWithRoleModeEnabled {
			ctx.NotFound(ctx.Req.URL.RequestURI(), nil)
			return
		}
		if !ctx.Repo.IsOwner() {
			ctx.Error(http.StatusNotFound)
			return
		}

		repoTransfer, err := models.GetPendingRepositoryTransfer(ctx, ctx.Repo.Repository)
		if err != nil {
			if models.IsErrNoPendingTransfer(err) {
				ctx.Flash.Error("repo.settings.transfer_abort_invalid")
				ctx.Redirect(repo.Link() + "/settings")
			} else {
				ctx.ServerError("GetPendingRepositoryTransfer", err)
			}

			return
		}

		if err := repoTransfer.LoadAttributes(ctx); err != nil {
			ctx.ServerError("LoadRecipient", err)
			return
		}

		if err := models.CancelRepositoryTransfer(ctx.Repo.Repository); err != nil {
			ctx.ServerError("CancelRepositoryTransfer", err)
			return
		}

		log.Trace("Repository transfer process was cancelled: %s/%s ", ctx.Repo.Owner.Name, repo.Name)
		ctx.Flash.Success(ctx.Tr("repo.settings.transfer_abort_success", repoTransfer.Recipient.Name))
		ctx.Redirect(repo.Link() + "/settings")

	case "delete":
		// если у нас включена ролевая модель SourceControl, то запускается проверка привилегий на удаление репозитория
		if setting.SourceControl.TenantWithRoleModeEnabled {
			tenantId, err := tenant.GetTenantByOrgIdOrDefault(ctx, ctx.Repo.Repository.OwnerID)
			if err != nil {
				ctx.NotFound(ctx.Req.URL.RequestURI(), nil)
				return
			}

			allowed, err := role_model.CheckUserPermissionToOrganization(ctx, ctx.Doer, tenantId, &organization.Organization{ID: ctx.Repo.Repository.OwnerID}, role_model.DELETE)
			if err != nil || !allowed {
				ctx.NotFound(ctx.Req.URL.RequestURI(), nil)
				return
			}
		} else if !ctx.Repo.IsOwner() {
			ctx.Error(http.StatusNotFound)
			auditParams["error"] = "Current user isn't the owner of repository"
			audit.CreateAndSendEvent(audit.RepositoryDeleteEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}
		if repo.Name != form.RepoName {
			ctx.RenderWithErr(ctx.Tr("form.enterred_invalid_repo_name"), tplSettingsOptions, nil)
			auditParams["error"] = "Enterred invalid repository name"
			audit.CreateAndSendEvent(audit.RepositoryDeleteEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}

		// Close the gitrepository before doing this.
		if ctx.Repo.GitRepo != nil {
			ctx.Repo.GitRepo.Close()
		}

		if err := repo_service.DeleteRepository(ctx, ctx.Doer, ctx.Repo.Repository, true); err != nil {
			ctx.ServerError("DeleteRepository", err)
			auditParams["error"] = "Error has occurred while deleting repository"
			audit.CreateAndSendEvent(audit.RepositoryDeleteEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}
		log.Trace("Repository deleted: %s/%s", ctx.Repo.Owner.Name, repo.Name)

		ctx.Flash.Success(ctx.Tr("repo.settings.deletion_success"))
		ctx.Redirect(ctx.Repo.Owner.DashboardLink())

		audit.CreateAndSendEvent(audit.RepositoryDeleteEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)

		repository, err := git.OpenRepository(ctx, ctx.Repo.Repository.OwnerName, ctx.Repo.Repository.Name, ctx.Repo.Repository.RepoPath())
		defer repository.Close()
		if err != nil {
			ctx.RenderWithErr(ctx.Tr("git.OpenRepository"), tplDeleteRepo, nil)
			return
		}

		resp, err := repository.RepoClient.RemoveRepository(ctx, &gitalypb.RemoveRepositoryRequest{
			Repository: repository.GitalyRepo,
		})
		if err != nil {
			ctx.RenderWithErr(ctx.Tr("repository.RepoClient.RemoveRepository"), tplDeleteRepo, nil)
			return
		}

		if resp.String() != "" {
			ctx.RenderWithErr(ctx.Tr(resp.String()), tplDeleteRepo, nil)
			return
		}
	case "delete-wiki":
		if !ctx.Repo.IsOwner() {
			ctx.Error(http.StatusNotFound)
			return
		}
		if repo.Name != form.RepoName {
			ctx.RenderWithErr(ctx.Tr("form.enterred_invalid_repo_name"), tplSettingsOptions, nil)
			return
		}

		err := wiki_service.DeleteWiki(ctx, repo)
		if err != nil {
			log.Error("Delete Wiki: %v", err.Error())
		}
		log.Trace("Repository wiki deleted: %s/%s", ctx.Repo.Owner.Name, repo.Name)

		ctx.Flash.Success(ctx.Tr("repo.settings.wiki_deletion_success"))
		ctx.Redirect(ctx.Repo.RepoLink + "/settings")
	case "archive":
		if !ctx.Repo.IsOwner() {
			ctx.Error(http.StatusForbidden)
			return
		}

		if repo.IsMirror {
			ctx.Flash.Error(ctx.Tr("repo.settings.archive.error_ismirror"))
			ctx.Redirect(ctx.Repo.RepoLink + "/settings")
			return
		}

		if err := repo_model.SetArchiveRepoState(repo, true); err != nil {
			log.Error("Tried to archive a repo: %s", err)
			ctx.Flash.Error(ctx.Tr("repo.settings.archive.error"))
			ctx.Redirect(ctx.Repo.RepoLink + "/settings")
			return
		}

		ctx.Flash.Success(ctx.Tr("repo.settings.archive.success"))

		log.Trace("Repository was archived: %s/%s", ctx.Repo.Owner.Name, repo.Name)
		ctx.Redirect(ctx.Repo.RepoLink + "/settings")

	case "unarchive":
		if !ctx.Repo.IsOwner() {
			ctx.Error(http.StatusForbidden)
			return
		}

		if err := repo_model.SetArchiveRepoState(repo, false); err != nil {
			log.Error("Tried to unarchive a repo: %s", err)
			ctx.Flash.Error(ctx.Tr("repo.settings.unarchive.error"))
			ctx.Redirect(ctx.Repo.RepoLink + "/settings")
			return
		}

		ctx.Flash.Success(ctx.Tr("repo.settings.unarchive.success"))

		log.Trace("Repository was un-archived: %s/%s", ctx.Repo.Owner.Name, repo.Name)
		ctx.Redirect(ctx.Repo.RepoLink + "/settings")

	default:
		ctx.NotFound("", nil)
	}
}

func handleSettingRemoteAddrError(ctx *context.Context, err error, form *forms.RepoSettingForm) {
	if models.IsErrInvalidCloneAddr(err) {
		addrErr := err.(*models.ErrInvalidCloneAddr)
		switch {
		case addrErr.IsProtocolInvalid:
			ctx.RenderWithErr(ctx.Tr("repo.mirror_address_protocol_invalid"), tplSettingsOptions, form)
		case addrErr.IsURLError:
			ctx.RenderWithErr(ctx.Tr("form.url_error", addrErr.Host), tplSettingsOptions, form)
		case addrErr.IsPermissionDenied:
			if addrErr.LocalPath {
				ctx.RenderWithErr(ctx.Tr("repo.migrate.permission_denied"), tplSettingsOptions, form)
			} else {
				ctx.RenderWithErr(ctx.Tr("repo.migrate.permission_denied_blocked"), tplSettingsOptions, form)
			}
		case addrErr.IsInvalidPath:
			ctx.RenderWithErr(ctx.Tr("repo.migrate.invalid_local_path"), tplSettingsOptions, form)
		default:
			ctx.ServerError("Unknown error", err)
		}
		return
	}
	ctx.RenderWithErr(ctx.Tr("repo.mirror_address_url_invalid"), tplSettingsOptions, form)
}

// Collaboration render a repository's collaboration page
func Collaboration(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("repo.settings.collaboration")
	ctx.Data["PageIsSettingsCollaboration"] = true

	users, err := repo_model.GetCollaborators(ctx, ctx.Repo.Repository.ID, db.ListOptions{})
	if err != nil {
		ctx.ServerError("GetCollaborators", err)
		return
	}
	ctx.Data["Collaborators"] = users

	teams, err := organization.GetRepoTeams(ctx, ctx.Repo.Repository)
	if err != nil {
		ctx.ServerError("GetRepoTeams", err)
		return
	}
	ctx.Data["Teams"] = teams
	ctx.Data["Repo"] = ctx.Repo.Repository
	ctx.Data["OrgID"] = ctx.Repo.Repository.OwnerID
	ctx.Data["OrgName"] = ctx.Repo.Repository.OwnerName
	ctx.Data["Org"] = ctx.Repo.Repository.Owner
	ctx.Data["Units"] = unit_model.Units

	ctx.HTML(http.StatusOK, tplCollaboration)
}

// CollaborationPost response for actions for a collaboration of a repository
func CollaborationPost(ctx *context.Context) {
	name := utils.RemoveUsernameParameterSuffix(strings.ToLower(ctx.FormString("collaborator")))
	auditParams := map[string]string{
		"repository":    ctx.Repo.Repository.Name,
		"repository_id": strconv.FormatInt(ctx.Repo.Repository.ID, 10),
		"affected_user": name,
		"rights_mode":   perm.AccessModeWrite.String(),
	}
	if len(name) == 0 || ctx.Repo.Owner.LowerName == name {
		ctx.Redirect(setting.AppSubURL + ctx.Req.URL.EscapedPath())
		auditParams["error"] = "Name is not specified or it is the same user"
		audit.CreateAndSendEvent(audit.RepositoryRightsGrantedEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}

	u, err := user_model.GetUserByName(ctx, name)
	if err != nil {
		if user_model.IsErrUserNotExist(err) { // если не найден в БД
			u, err = user_model.GetAndCreateUserByNameOrEmailFromKeycloak(name, ctx.Locale, ctx) // попробуем найти в keycloak если это возможно
			if err != nil {
				switch true {
				case user_model.IsErrUserNotExist(err):
					ctx.Flash.Error(ctx.Tr("form.user_not_exist"))
					ctx.Redirect(setting.AppSubURL + ctx.Req.URL.EscapedPath())
					auditParams["error"] = "User not exist"
					audit.CreateAndSendEvent(audit.RepositoryRightsGrantedEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
					return
				case user_model.IsErrEmailAddressNotExist(err):
					ctx.Flash.Error(ctx.Tr("form.email_is_empty"))
					ctx.Redirect(setting.AppSubURL + ctx.Req.URL.EscapedPath())
					auditParams["error"] = "Email is empty"
					audit.CreateAndSendEvent(audit.RepositoryRightsGrantedEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
					return
				default:
					log.Error("Error has occurred while try get and add user from keycloak to db with name: %s, err: %v", name, err)
					ctx.ServerError("GetUserByName", err)
					auditParams["error"] = "Error has occurred while try get and add user from keycloak to db"
					audit.CreateAndSendEvent(audit.RepositoryRightsGrantedEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
					return
				}
			}
		} else {
			ctx.ServerError("GetUserByName", err)
			auditParams["error"] = "Error has occurred while getting user by name"
			audit.CreateAndSendEvent(audit.RepositoryRightsGrantedEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}
	}

	auditParams["affected_user_id"] = strconv.FormatInt(u.ID, 10)

	if !u.IsActive {
		ctx.Flash.Error(ctx.Tr("repo.settings.add_collaborator_inactive_user"))
		ctx.Redirect(setting.AppSubURL + ctx.Req.URL.EscapedPath())
		auditParams["error"] = "Cannot add an inactive user as a collaborator"
		audit.CreateAndSendEvent(audit.RepositoryRightsGrantedEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}

	// Organization is not allowed to be added as a collaborator.
	if u.IsOrganization() {
		ctx.Flash.Error(ctx.Tr("repo.settings.org_not_allowed_to_be_collaborator"))
		ctx.Redirect(setting.AppSubURL + ctx.Req.URL.EscapedPath())
		auditParams["error"] = "Organizations cannot be added as a collaborator"
		audit.CreateAndSendEvent(audit.RepositoryRightsGrantedEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}

	if got, err := repo_model.IsCollaborator(ctx, ctx.Repo.Repository.ID, u.ID); err == nil && got {
		ctx.Flash.Error(ctx.Tr("repo.settings.add_collaborator_duplicate"))
		ctx.Redirect(ctx.Repo.RepoLink + "/settings/collaboration")
		auditParams["error"] = "The collaborator is already added to this repository"
		audit.CreateAndSendEvent(audit.RepositoryRightsGrantedEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}

	// find the owner team of the organization the repo belongs too and
	// check if the user we're trying to add is an owner.
	if ctx.Repo.Repository.Owner.IsOrganization() {
		if isOwner, err := organization.IsOrganizationOwner(ctx, ctx.Repo.Repository.Owner.ID, u.ID); err != nil {
			ctx.ServerError("IsOrganizationOwner", err)
			auditParams["error"] = "Error has occurred while checking owner of organization"
			audit.CreateAndSendEvent(audit.RepositoryRightsGrantedEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		} else if isOwner {
			ctx.Flash.Error(ctx.Tr("repo.settings.add_collaborator_owner"))
			ctx.Redirect(setting.AppSubURL + ctx.Req.URL.EscapedPath())
			auditParams["error"] = "Cannot add an owner as a collaborator"
			audit.CreateAndSendEvent(audit.RepositoryRightsGrantedEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}
	}

	if err = repo_module.AddCollaborator(ctx, ctx.Repo.Repository, u); err != nil {
		ctx.ServerError("AddCollaborator", err)
		auditParams["error"] = "Error has occurred while adding a collaborator"
		audit.CreateAndSendEvent(audit.RepositoryRightsGrantedEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}

	if setting.Service.EnableNotifyMail {
		mailer.SendCollaboratorMail(u, ctx.Doer, ctx.Repo.Repository)
	}

	audit.CreateAndSendEvent(audit.RepositoryRightsGrantedEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)
	ctx.Flash.Success(ctx.Tr("repo.settings.add_collaborator_success"))
	ctx.Redirect(setting.AppSubURL + ctx.Req.URL.EscapedPath())
}

// ChangeCollaborationAccessMode response for changing access of a collaboration
func ChangeCollaborationAccessMode(ctx *context.Context) {
	auditParams := map[string]string{
		"repository":       ctx.Repo.Repository.Name,
		"repository_id":    strconv.FormatInt(ctx.Repo.Repository.ID, 10),
		"affected_user_id": strconv.FormatInt(ctx.FormInt64("uid"), 10),
		"rights_mode":      perm.AccessMode(ctx.FormInt("mode")).String(),
	}
	if err := repo_model.ChangeCollaborationAccessMode(
		ctx,
		ctx.Repo.Repository,
		ctx.FormInt64("uid"),
		perm.AccessMode(ctx.FormInt("mode"))); err != nil {
		log.Error("ChangeCollaborationAccessMode: %v", err)
		auditParams["error"] = "Error has occurred while changing collaboration access mode"
		audit.CreateAndSendEvent(audit.RepositoryRightsChangeEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}
	audit.CreateAndSendEvent(audit.RepositoryRightsChangeEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)
}

// DeleteCollaboration delete a collaboration for a repository
func DeleteCollaboration(ctx *context.Context) {
	auditParams := map[string]string{
		"repository":       ctx.Repo.Repository.Name,
		"repository_id":    strconv.FormatInt(ctx.Repo.Repository.ID, 10),
		"affected_user_id": strconv.FormatInt(ctx.FormInt64("id"), 10),
	}
	if err := models.DeleteCollaboration(ctx.Repo.Repository, ctx.FormInt64("id")); err != nil {
		ctx.Flash.Error("DeleteCollaboration: " + err.Error())

		auditParams["error"] = "Error has occurred while deleting collaboration"
		audit.CreateAndSendEvent(audit.RepositoryRightsRemoveEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		ctx.JSON(http.StatusOK, map[string]interface{}{
			"redirect": ctx.Repo.RepoLink + "/settings/collaboration",
		})
		return
	}

	audit.CreateAndSendEvent(audit.RepositoryRightsRemoveEvent, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)
	ctx.Flash.Success(ctx.Tr("repo.settings.remove_collaborator_success"))
	ctx.JSON(http.StatusOK, map[string]interface{}{
		"redirect": ctx.Repo.RepoLink + "/settings/collaboration",
	})
}

// AddTeamPost response for adding a team to a repository
func AddTeamPost(ctx *context.Context) {
	if !ctx.Repo.Owner.RepoAdminChangeTeamAccess && !ctx.Repo.IsOwner() {
		ctx.Flash.Error(ctx.Tr("repo.settings.change_team_access_not_allowed"))
		ctx.Redirect(ctx.Repo.RepoLink + "/settings/collaboration")
		return
	}

	name := utils.RemoveUsernameParameterSuffix(strings.ToLower(ctx.FormString("team")))
	if len(name) == 0 {
		ctx.Redirect(ctx.Repo.RepoLink + "/settings/collaboration")
		return
	}

	team, err := organization.OrgFromUser(ctx.Repo.Owner).GetTeam(ctx, name)
	if err != nil {
		if organization.IsErrTeamNotExist(err) {
			ctx.Flash.Error(ctx.Tr("form.team_not_exist"))
			ctx.Redirect(ctx.Repo.RepoLink + "/settings/collaboration")
		} else {
			ctx.ServerError("GetTeam", err)
		}
		return
	}

	if team.OrgID != ctx.Repo.Repository.OwnerID {
		ctx.Flash.Error(ctx.Tr("repo.settings.team_not_in_organization"))
		ctx.Redirect(ctx.Repo.RepoLink + "/settings/collaboration")
		return
	}

	if organization.HasTeamRepo(ctx, ctx.Repo.Repository.OwnerID, team.ID, ctx.Repo.Repository.ID) {
		ctx.Flash.Error(ctx.Tr("repo.settings.add_team_duplicate"))
		ctx.Redirect(ctx.Repo.RepoLink + "/settings/collaboration")
		return
	}

	if err = org_service.TeamAddRepository(team, ctx.Repo.Repository); err != nil {
		ctx.ServerError("TeamAddRepository", err)
		return
	}

	ctx.Flash.Success(ctx.Tr("repo.settings.add_team_success"))
	ctx.Redirect(ctx.Repo.RepoLink + "/settings/collaboration")
}

// DeleteTeam response for deleting a team from a repository
func DeleteTeam(ctx *context.Context) {
	if !ctx.Repo.Owner.RepoAdminChangeTeamAccess && !ctx.Repo.IsOwner() {
		ctx.Flash.Error(ctx.Tr("repo.settings.change_team_access_not_allowed"))
		ctx.Redirect(ctx.Repo.RepoLink + "/settings/collaboration")
		return
	}

	team, err := organization.GetTeamByID(ctx, ctx.FormInt64("id"))
	if err != nil {
		ctx.ServerError("GetTeamByID", err)
		return
	}

	if err = models.RemoveRepository(team, ctx.Repo.Repository.ID); err != nil {
		ctx.ServerError("team.RemoveRepositorys", err)
		return
	}

	ctx.Flash.Success(ctx.Tr("repo.settings.remove_team_success"))
	ctx.JSON(http.StatusOK, map[string]interface{}{
		"redirect": ctx.Repo.RepoLink + "/settings/collaboration",
	})
}

// GitHooks hooks of a repository
func GitHooks(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("repo.settings.githooks")
	ctx.Data["PageIsSettingsGitHooks"] = true

	hooks, err := ctx.Repo.GitRepo.Hooks()
	if err != nil {
		ctx.ServerError("Hooks", err)
		return
	}
	ctx.Data["Hooks"] = hooks

	ctx.HTML(http.StatusOK, tplGithooks)
}

// GitHooksEdit render for editing a hook of repository page
func GitHooksEdit(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("repo.settings.githooks")
	ctx.Data["PageIsSettingsGitHooks"] = true

	name := ctx.Params(":name")
	hook, err := ctx.Repo.GitRepo.GetHook(name)
	if err != nil {
		if err == git.ErrNotValidHook {
			ctx.NotFound("GetHook", err)
		} else {
			ctx.ServerError("GetHook", err)
		}
		return
	}
	ctx.Data["Hook"] = hook
	ctx.HTML(http.StatusOK, tplGithookEdit)
}

// GitHooksEditPost response for editing a git hook of a repository
func GitHooksEditPost(ctx *context.Context) {
	name := ctx.Params(":name")
	hook, err := ctx.Repo.GitRepo.GetHook(name)
	if err != nil {
		if err == git.ErrNotValidHook {
			ctx.NotFound("GetHook", err)
		} else {
			ctx.ServerError("GetHook", err)
		}
		return
	}
	hook.Content = ctx.FormString("content")
	if err = hook.Update(); err != nil {
		ctx.ServerError("hook.Update", err)
		return
	}
	ctx.Redirect(ctx.Repo.RepoLink + "/settings/hooks/git")
}

// DeployKeys render the deploy keys list of a repository page
func DeployKeys(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("repo.settings.deploy_keys") + " / " + ctx.Tr("secrets.secrets")
	ctx.Data["PageIsSettingsKeys"] = true
	ctx.Data["DisableSSH"] = setting.SSH.Disabled

	keys, err := asymkey_model.ListDeployKeys(ctx, &asymkey_model.ListDeployKeysOptions{RepoID: ctx.Repo.Repository.ID})
	if err != nil {
		ctx.ServerError("ListDeployKeys", err)
		return
	}
	ctx.Data["Deploykeys"] = keys

	ctx.HTML(http.StatusOK, tplDeployKeys)
}

// DeployKeysPost response for adding a deploy key of a repository
func DeployKeysPost(ctx *context.Context) {
	form := web.GetForm(ctx).(*forms.AddKeyForm)
	ctx.Data["Title"] = ctx.Tr("repo.settings.deploy_keys")
	ctx.Data["PageIsSettingsKeys"] = true
	ctx.Data["DisableSSH"] = setting.SSH.Disabled

	keys, err := asymkey_model.ListDeployKeys(ctx, &asymkey_model.ListDeployKeysOptions{RepoID: ctx.Repo.Repository.ID})
	if err != nil {
		ctx.ServerError("ListDeployKeys", err)
		return
	}
	ctx.Data["Deploykeys"] = keys

	if ctx.HasError() {
		ctx.HTML(http.StatusOK, tplDeployKeys)
		return
	}

	content, err := asymkey_model.CheckPublicKeyString(form.Content)
	if err != nil {
		if db.IsErrSSHDisabled(err) {
			ctx.Flash.Info(ctx.Tr("settings.ssh_disabled"))
		} else if asymkey_model.IsErrKeyUnableVerify(err) {
			ctx.Flash.Info(ctx.Tr("form.unable_verify_ssh_key"))
		} else if err == asymkey_model.ErrKeyIsPrivate {
			ctx.Data["HasError"] = true
			ctx.Data["Err_Content"] = true
			ctx.Flash.Error(ctx.Tr("form.must_use_public_key"))
		} else {
			ctx.Data["HasError"] = true
			ctx.Data["Err_Content"] = true
			ctx.Flash.Error(ctx.Tr("form.invalid_ssh_key", err.Error()))
		}
		ctx.Redirect(ctx.Repo.RepoLink + "/settings/keys")
		return
	}

	key, err := asymkey_model.AddDeployKey(ctx.Repo.Repository.ID, form.Title, content, !form.IsWritable)
	if err != nil {
		ctx.Data["HasError"] = true
		switch {
		case asymkey_model.IsErrDeployKeyAlreadyExist(err):
			ctx.Data["Err_Content"] = true
			ctx.RenderWithErr(ctx.Tr("repo.settings.key_been_used"), tplDeployKeys, &form)
		case asymkey_model.IsErrKeyAlreadyExist(err):
			ctx.Data["Err_Content"] = true
			ctx.RenderWithErr(ctx.Tr("settings.ssh_key_been_used"), tplDeployKeys, &form)
		case asymkey_model.IsErrKeyNameAlreadyUsed(err):
			ctx.Data["Err_Title"] = true
			ctx.RenderWithErr(ctx.Tr("repo.settings.key_name_used"), tplDeployKeys, &form)
		case asymkey_model.IsErrDeployKeyNameAlreadyUsed(err):
			ctx.Data["Err_Title"] = true
			ctx.RenderWithErr(ctx.Tr("repo.settings.key_name_used"), tplDeployKeys, &form)
		default:
			ctx.ServerError("AddDeployKey", err)
		}
		return
	}

	log.Trace("Deploy key added: %d", ctx.Repo.Repository.ID)
	ctx.Flash.Success(ctx.Tr("repo.settings.add_key_success", key.Name))
	ctx.Redirect(ctx.Repo.RepoLink + "/settings/keys")
}

// DeleteDeployKey response for deleting a deploy key
func DeleteDeployKey(ctx *context.Context) {
	if err := asymkey_service.DeleteDeployKey(ctx.Doer, ctx.FormInt64("id")); err != nil {
		ctx.Flash.Error("DeleteDeployKey: " + err.Error())
	} else {
		ctx.Flash.Success(ctx.Tr("repo.settings.deploy_key_deletion_success"))
	}

	ctx.JSON(http.StatusOK, map[string]interface{}{
		"redirect": ctx.Repo.RepoLink + "/settings/keys",
	})
}

// UpdateAvatarSetting update repo's avatar
func UpdateAvatarSetting(ctx *context.Context, form forms.AvatarForm) error {
	ctxRepo := ctx.Repo.Repository

	if form.Avatar == nil {
		// No avatar is uploaded and we not removing it here.
		// No random avatar generated here.
		// Just exit, no action.
		if ctxRepo.CustomAvatarRelativePath() == "" {
			log.Trace("No avatar was uploaded for repo: %d. Default icon will appear instead.", ctxRepo.ID)
		}
		return nil
	}

	r, err := form.Avatar.Open()
	if err != nil {
		return fmt.Errorf("Avatar.Open: %w", err)
	}
	defer r.Close()

	if form.Avatar.Size > setting.Avatar.MaxFileSize {
		return errors.New(ctx.Tr("settings.uploaded_avatar_is_too_big"))
	}

	data, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("io.ReadAll: %w", err)
	}
	st := typesniffer.DetectContentType(data)
	if !(st.IsImage() && !st.IsSvgImage()) {
		return errors.New(ctx.Tr("settings.uploaded_avatar_not_a_image"))
	}
	if err = repo_service.UploadAvatar(ctx, ctxRepo, data); err != nil {
		return fmt.Errorf("UploadAvatar: %w", err)
	}
	return nil
}

// SettingsAvatar save new POSTed repository avatar
func SettingsAvatar(ctx *context.Context) {
	form := web.GetForm(ctx).(*forms.AvatarForm)
	form.Source = forms.AvatarLocal
	if err := UpdateAvatarSetting(ctx, *form); err != nil {
		ctx.Flash.Error(err.Error())
	} else {
		ctx.Flash.Success(ctx.Tr("repo.settings.update_avatar_success"))
	}
	ctx.Redirect(ctx.Repo.RepoLink + "/settings")
}

// SettingsDeleteAvatar delete repository avatar
func SettingsDeleteAvatar(ctx *context.Context) {
	if err := repo_service.DeleteAvatar(ctx, ctx.Repo.Repository); err != nil {
		ctx.Flash.Error(fmt.Sprintf("DeleteAvatar: %v", err))
	}
	ctx.Redirect(ctx.Repo.RepoLink + "/settings")
}

func selectPushMirrorByForm(ctx *context.Context, form *forms.RepoSettingForm, repo *repo_model.Repository) (*repo_model.PushMirror, error) {
	id, err := strconv.ParseInt(form.PushMirrorID, 10, 64)
	if err != nil {
		return nil, err
	}

	pushMirrors, _, err := repo_model.GetPushMirrorsByRepoID(ctx, repo.ID, db.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, m := range pushMirrors {
		if m.ID == id {
			m.Repo = repo
			return m, nil
		}
	}

	return nil, fmt.Errorf("PushMirror[%v] not associated to repository %v", id, repo)
}

func unitTypeExistInSliceOfUnits(units []*repo_model.RepoUnit, findedUnit unit_model.Type) bool {
	return findUnitByType(units, findedUnit) != nil
}

func findUnitByType(units []*repo_model.RepoUnit, findedUnit unit_model.Type) *repo_model.RepoUnit {
	for _, unit := range units {
		if unit.Type == findedUnit {
			return unit
		}
	}
	return nil
}
