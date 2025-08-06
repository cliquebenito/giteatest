package pulls

import (
	gocontext "context"
	"fmt"
	"net/http"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/default_reviewers"
	access_model "code.gitea.io/gitea/models/perm/access"
	"code.gitea.io/gitea/models/review_settings"
	"code.gitea.io/gitea/models/review_settings/review_settings_db"
	"code.gitea.io/gitea/models/role_model"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/json"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/sbt/audit"
	auditutils "code.gitea.io/gitea/modules/sbt/audit/utils"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/routers/api/v3/models"
	"code.gitea.io/gitea/routers/web/user/accesser"
)

const (
	tmplReviewSettings      = "repo/settings/review"
	tmplReviewSettingCreate = "repo/settings/new_review_settings"
)

type server struct {
	orgRequestAccessor
	defaultReviewersDB
	reviewSettingsDB
}

type orgRequestAccessor interface {
	IsReadAccessGranted(ctx gocontext.Context, request accesser.OrgAccessRequest) (bool, error)
	IsAccessGranted(ctx gocontext.Context, request accesser.OrgAccessRequest) (bool, error)
}

func NewReviewSettingsServer(orgAccessor orgRequestAccessor, defaultReviewersDB defaultReviewersDB, reviewSettingsDB reviewSettingsDB) *server {
	return &server{orgRequestAccessor: orgAccessor, defaultReviewersDB: defaultReviewersDB, reviewSettingsDB: reviewSettingsDB}
}

type defaultReviewersDB interface {
	GetDefaultReviewers(ctx gocontext.Context, settingID int64) ([]*default_reviewers.DefaultReviewers, error)
	InsertDefaultReviewers(ctx gocontext.Context, defaultReviewers []*default_reviewers.DefaultReviewers) error
	DeleteDefaultReviewers(ctx gocontext.Context, defaultReviewers []*default_reviewers.DefaultReviewers) error
	DeleteDefaultReviewersBySettingID(ctx gocontext.Context, settingID int64) error
}

type reviewSettingsDB interface {
	GetReviewSettings(_ gocontext.Context, repoID int64) ([]*review_settings.ReviewSettings, error)
	GetReviewSettingsByBranchPattern(_ gocontext.Context, repoID int64, branchName string) (*review_settings.ReviewSettings, error)
	UpsertReviewSettings(_ gocontext.Context, rs *review_settings.ReviewSettings) error
	DeleteReviewSettingsByRepoID(_ gocontext.Context, repoID int64, branchName string) error
}

func (s server) CreateReviewSettings(ctx *context.Context) {

	s.createReviewSettings(ctx)
}

func (s server) createReviewSettings(ctx *context.Context) {
	tenantID, err := role_model.GetUserTenantId(ctx, ctx.Doer.ID)
	if err != nil {
		log.Error("Error has occurred while getting user tenant id: %v", err)
		ctx.Error(http.StatusInternalServerError, "Fail to get user tenant id")
		return
	}
	allowed, err := s.orgRequestAccessor.IsAccessGranted(ctx, accesser.OrgAccessRequest{
		DoerID:         ctx.Doer.ID,
		TargetOrgID:    ctx.Repo.Repository.OwnerID,
		TargetTenantID: tenantID,
		Action:         role_model.EDIT,
	})
	if err != nil {
		log.Error("Error has occurred while checking user permission: %v", err)
		ctx.Error(http.StatusInternalServerError, "Fail to check user permission")
		return
	}

	if !allowed {
		log.Debug("User does not have permission to create review settings")
		ctx.Status(http.StatusNotFound)
		return
	}

	opt := web.GetForm(ctx).(*models.ReviewSettingsRequest)
	newValue, err := json.Marshal(opt)
	if err != nil {
		log.Error("Error has occurred while serializing new value: %v", err)
		ctx.Error(http.StatusInternalServerError, "Fail to serialize")
		return
	}
	auditParams := map[string]string{
		"new_value": string(newValue),
	}
	auditValues := auditutils.NewRequiredAuditParams(ctx)
	if err := opt.Validate(); err != nil {
		log.Debug("Error has occurred while validating review setting: %v", err)
		auditParams["error"] = "Error has occurred while validating review setting"
		audit.CreateAndSendEvent(audit.ReviewSettingCreateEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
		ctx.Error(http.StatusBadRequest, "Fail to validate review setting")
		return
	}
	reviewSetting, err := models.ConvertAPIToReviewSettingsModel(ctx, *opt, ctx.Repo.Repository.ID)
	if err != nil {
		log.Error("Error has occurred while converting review setting: %v", err)
		auditParams["error"] = "Error has occurred while converting review setting"
		audit.CreateAndSendEvent(audit.ReviewSettingCreateEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
		ctx.Error(http.StatusBadRequest, "Fail to convert review setting")
		return
	}
	defaultReviewers, err := models.ConvertDefaultReviewerSetsToDBModel(opt.ApprovalSettings.DefaultReviewers)
	if err != nil {
		log.Error("Error has occurred while converting default reviewers: %v", err)
		auditParams["error"] = "Error has occurred while converting default reviewers"
		audit.CreateAndSendEvent(audit.ReviewSettingCreateEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
		ctx.Error(http.StatusBadRequest, "Fail to convert default reviewers")
		return
	}

	// Check if already exists
	_, err = s.GetReviewSettingsByBranchPattern(ctx, ctx.Repo.Repository.ID, opt.BranchName)
	if err != nil {
		if !review_settings_db.IsErrReviewSettingsDoesntExistsError(err) {
			log.Error("Error has occurred while getting review setting by repository id %d and branch name %s: %v", ctx.Repo.Repository.ID, opt.BranchName, err)
			auditParams["error"] = "Error has occurred while getting review setting by repository id and branch name"
			audit.CreateAndSendEvent(audit.ReviewSettingCreateEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
			ctx.Error(http.StatusInternalServerError, "Fail to get review settings")
			return
		}
	} else {
		log.Debug("Review setting with repository id %d and branch name %s already exists", ctx.Repo.Repository.ID, opt.BranchName)
		auditParams["error"] = "Review setting with same repository id and branch name already exists"
		audit.CreateAndSendEvent(audit.ReviewSettingCreateEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
		ctx.Error(http.StatusConflict, "Review setting already exists")
		return
	}

	if err := db.WithTx(db.DefaultContext, func(context gocontext.Context) error {
		if err := s.UpsertReviewSettings(context, reviewSetting); err != nil {
			return fmt.Errorf("fail to update default review settings: %v", err)
		}

		rs, err := s.GetReviewSettingsByBranchPattern(context, ctx.Repo.Repository.ID, opt.BranchName)
		if err != nil {
			return fmt.Errorf("fail to get default review settings: %v", err)
		}

		for _, df := range defaultReviewers {
			df.ReviewSettingID = rs.ID
		}

		if err := s.InsertDefaultReviewers(context, defaultReviewers); err != nil {
			return fmt.Errorf("fail to insert default reviewers: %v", err)
		}
		return nil
	}); err != nil {
		log.Error("Error has occurred while creating review settings: %w", err)
		auditParams["error"] = "Error has occurred while creating review settings"
		audit.CreateAndSendEvent(audit.ReviewSettingCreateEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
		ctx.Error(http.StatusInternalServerError, "Fail to create review settings")
		return
	}
	audit.CreateAndSendEvent(audit.ReviewSettingCreateEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusSuccess, auditValues.RemoteAddress, auditParams)
	ctx.Status(http.StatusCreated)
}

func (s server) UpdateReviewSettings(ctx *context.Context) {

	s.updateReviewSettings(ctx)
}

func (s server) updateReviewSettings(ctx *context.Context) {
	tenantID, err := role_model.GetUserTenantId(ctx, ctx.Doer.ID)
	if err != nil {
		log.Error("Error has occurred while getting user tenant id: %v", err)
		ctx.Error(http.StatusInternalServerError, "Fail to get user tenant id")
		return
	}
	allowed, err := s.orgRequestAccessor.IsAccessGranted(ctx, accesser.OrgAccessRequest{
		DoerID:         ctx.Doer.ID,
		TargetOrgID:    ctx.Repo.Repository.OwnerID,
		TargetTenantID: tenantID,
		Action:         role_model.EDIT,
	})
	if err != nil {
		log.Error("Error has occurred while checking user permission: %v", err)
		ctx.Error(http.StatusInternalServerError, "Fail to check user permission")
		return
	}

	if !allowed {
		log.Debug("User does not have permission to edit review settings")
		ctx.Status(http.StatusNotFound)
		return
	}

	branchName := ctx.Params(":branch_name")
	opt := web.GetForm(ctx).(*models.ReviewSettingsRequest)
	newValue, err := json.Marshal(opt)
	if err != nil {
		log.Error("Error has occurred while serializing new value: %v", err)
	}
	auditParams := map[string]string{
		"new_value": string(newValue),
	}
	auditValues := auditutils.NewRequiredAuditParams(ctx)

	reviewSetting, err := models.ConvertAPIToReviewSettingsModel(ctx, *opt, ctx.Repo.Repository.ID)
	if err != nil {
		log.Error("Error has occurred while converting review setting: %v", err)
		auditParams["error"] = "Error has occurred while converting review setting"
		audit.CreateAndSendEvent(audit.ReviewSettingUpdateEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
		ctx.Error(http.StatusBadRequest, "Fail to convert review setting")
		return
	}
	defaultReviewers, err := models.ConvertDefaultReviewerSetsToDBModel(opt.ApprovalSettings.DefaultReviewers)
	if err != nil {
		log.Error("Error has occurred while converting default reviewers: %v", err)
		auditParams["error"] = "Error has occurred while converting default reviewers"
		audit.CreateAndSendEvent(audit.ReviewSettingUpdateEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
		ctx.Error(http.StatusBadRequest, "Fail to convert default reviewers")
		return
	}

	// Check that exists
	_, err = s.GetReviewSettingsByBranchPattern(ctx, ctx.Repo.Repository.ID, branchName)
	if err != nil {
		if review_settings_db.IsErrReviewSettingsDoesntExistsError(err) {
			log.Error("Review settings for repo with id %d and branch name %s do no exist: %v", ctx.Repo.Repository.ID, branchName, err)
			auditParams["error"] = "Review setting for repo with requested id and branch name does not exist"
			audit.CreateAndSendEvent(audit.ReviewSettingUpdateEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
			ctx.Error(http.StatusNotFound, "Review settings do not exist")
		} else {
			log.Error("Error has occurred while getting review setting by repository id %d and branch name %s: %v", ctx.Repo.Repository.ID, branchName, err)
			auditParams["error"] = "Error has occurred while getting review setting by repository id and branch name"
			audit.CreateAndSendEvent(audit.ReviewSettingUpdateEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
			ctx.Error(http.StatusInternalServerError, "Fail to get review settings")
		}
		return
	}

	// Check if already exists
	if branchName != opt.BranchName {
		_, err = s.GetReviewSettingsByBranchPattern(ctx, ctx.Repo.Repository.ID, opt.BranchName)
		if err != nil {
			if !review_settings_db.IsErrReviewSettingsDoesntExistsError(err) {
				log.Error("Error has occurred while getting review setting by repository id %d and branch name %s: %v", ctx.Repo.Repository.ID, opt.BranchName, err)
				auditParams["error"] = "Error has occurred while getting review setting by repository id and branch name"
				audit.CreateAndSendEvent(audit.ReviewSettingUpdateEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
				ctx.Error(http.StatusInternalServerError, "Fail to get review settings")
				return
			}
		} else {
			log.Debug("Review setting with repository id %d and branch name %s already exists", ctx.Repo.Repository.ID, opt.BranchName)
			auditParams["error"] = "Review setting with requested repository id and branch name already exists"
			audit.CreateAndSendEvent(audit.ReviewSettingUpdateEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
			ctx.Error(http.StatusConflict, "Review setting already exists")
			return
		}
	}

	if err := db.WithTx(db.DefaultContext, func(context gocontext.Context) error {
		rs, err := s.GetReviewSettingsByBranchPattern(context, ctx.Repo.Repository.ID, branchName)
		if err != nil {
			return fmt.Errorf("Fail to get review settings: %w", err)
		}
		err = s.DeleteReviewSettingsByRepoID(context, ctx.Repo.Repository.ID, branchName)
		if err != nil {
			return fmt.Errorf("Fail to delete review settings: %w", err)
		}
		err = s.DeleteDefaultReviewersBySettingID(context, rs.ID)
		if err != nil {
			return fmt.Errorf("fail to delete default reviewers: %w", err)
		}

		if err := s.UpsertReviewSettings(context, reviewSetting); err != nil {
			return fmt.Errorf("fail to update default review settings: %w", err)
		}
		rs, err = s.GetReviewSettingsByBranchPattern(context, ctx.Repo.Repository.ID, opt.BranchName)
		if err != nil {
			return fmt.Errorf("fail to get review settings: %w", err)
		}
		for _, df := range defaultReviewers {
			df.ReviewSettingID = rs.ID
		}

		if err := s.InsertDefaultReviewers(context, defaultReviewers); err != nil {
			return fmt.Errorf("fail to insert default reviewers: %w", err)
		}

		return nil
	}); err != nil {
		log.Error("Error has occurred while updating review settings: %w", err)
		auditParams["error"] = "Error has occurred while updating review settings"
		audit.CreateAndSendEvent(audit.ReviewSettingUpdateEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
		ctx.Error(http.StatusInternalServerError, "Fail to update review settings")
		return
	}
	audit.CreateAndSendEvent(audit.ReviewSettingUpdateEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusSuccess, auditValues.RemoteAddress, auditParams)
	ctx.Status(http.StatusCreated)
}

func (s server) DeleteReviewSettings(ctx *context.Context) {

	s.deleteReviewSettings(ctx)
}

func (s server) deleteReviewSettings(ctx *context.Context) {
	tenantID, err := role_model.GetUserTenantId(ctx, ctx.Doer.ID)
	if err != nil {
		log.Error("Error has occurred while getting user tenant id: %v", err)
		ctx.Error(http.StatusInternalServerError, "Fail to get user tenant id")
		return
	}
	allowed, err := s.orgRequestAccessor.IsAccessGranted(ctx, accesser.OrgAccessRequest{
		DoerID:         ctx.Doer.ID,
		TargetOrgID:    ctx.Repo.Repository.OwnerID,
		TargetTenantID: tenantID,
		Action:         role_model.EDIT,
	})
	if err != nil {
		log.Error("Error has occurred while checking user permission: %v", err)
		ctx.Error(http.StatusInternalServerError, "Fail to check user permission")
		return
	}

	if !allowed {
		log.Debug("User does not have permission to delete review settings")
		ctx.Status(http.StatusNotFound)
		return
	}

	branchName := ctx.Params(":branch_name")
	auditParams := map[string]string{}
	auditValues := auditutils.NewRequiredAuditParams(ctx)

	// Check that exists
	value, err := s.GetReviewSettingsByBranchPattern(ctx, ctx.Repo.Repository.ID, branchName)
	if err != nil {
		if review_settings_db.IsErrReviewSettingsDoesntExistsError(err) {
			log.Error("Review settings for repo with id %d and branch name %s do no exist: %v", ctx.Repo.Repository.ID, branchName, err)
			auditParams["error"] = "Review settings for repo with requested id and branch name do no exist"
			audit.CreateAndSendEvent(audit.ReviewSettingDeleteEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
			ctx.Error(http.StatusNotFound, "Review settings do not exist")
		} else {
			log.Error("Error has occurred while getting review setting by repository id %d and branch name %s: %v", ctx.Repo.Repository.ID, branchName, err)
			auditParams["error"] = "Error has occurred while getting review setting by repository id and branch name"
			audit.CreateAndSendEvent(audit.ReviewSettingDeleteEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
			ctx.Error(http.StatusInternalServerError, "Fail to get review settings")
		}
		return
	}

	newValue, err := json.Marshal(value)
	if err != nil {
		log.Error("Error has occurred while serializing new value: %v", err)
		ctx.Error(http.StatusInternalServerError, "Fail to serialize new value")
		return
	}
	auditParams["old_value"] = string(newValue)

	if err := db.WithTx(db.DefaultContext, func(context gocontext.Context) error {
		rs, err := s.GetReviewSettingsByBranchPattern(context, ctx.Repo.Repository.ID, branchName)
		if err != nil {
			return fmt.Errorf("fail to get review settings: %w", err)
		}
		err = s.DeleteReviewSettingsByRepoID(ctx, ctx.Repo.Repository.ID, branchName)
		if err != nil {
			return fmt.Errorf("fail to delete review settings: %w", err)
		}
		err = s.DeleteDefaultReviewersBySettingID(context, rs.ID)
		if err != nil {
			return fmt.Errorf("fail to delete default reviewers: %w", err)
		}
		return nil
	}); err != nil {
		log.Error("Error has occurred while deleting review settings: %w", err)
		auditParams["error"] = "Error has occurred while deleting review settings"
		audit.CreateAndSendEvent(audit.ReviewSettingDeleteEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
		ctx.Error(http.StatusInternalServerError, "Fail to delete review settings")
		return
	}
	audit.CreateAndSendEvent(audit.ReviewSettingDeleteEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusSuccess, auditValues.RemoteAddress, auditParams)
	ctx.Status(http.StatusNoContent)
}

func (s server) ReviewSettings(ctx *context.Context) {
	reviewSettings, err := s.GetReviewSettings(ctx, ctx.Repo.Repository.ID)
	if err != nil && !review_settings_db.IsErrReviewSettingsDoesntExistsError(err) {
		log.Error("Error has occurred while getting review setting by repository id %d: %v", ctx.Repo.Repository.ID, err)
		ctx.Error(http.StatusInternalServerError, "Fail to get review settings")
		return
	}
	result := make([]models.BranchReviewSetting, len(reviewSettings))
	for i, rs := range reviewSettings {
		defaultReviewers, err := s.GetDefaultReviewers(ctx, rs.ID)
		if err != nil {
			log.Error("Error has occurred while getting default reviewers by setting id %d: %v", rs.ID, err)
			ctx.Error(http.StatusInternalServerError, "Fail to get default reviewers")
			return
		}
		apiReview, err := models.ConvertReviewSettingsToAPIModel(ctx, rs, defaultReviewers)
		if err != nil {
			log.Error("Error has occurred while converting review setting: %v", err)
			ctx.Error(http.StatusInternalServerError, "Fail to convert review setting")
			return
		}
		result[i] = *apiReview
	}

	ctx.Data["ReviewSettings"] = result
	ctx.Data["OrgLink"] = ctx.Org.Organization.AsUser().OrganisationLink()
	ctx.HTML(http.StatusOK, tmplReviewSettings)
}

func (s server) EditReviewSetting(ctx *context.Context) {
	branchName := ctx.FormString("setting_name")

	reviewSetting, err := s.GetReviewSettingsByBranchPattern(ctx, ctx.Repo.Repository.ID, branchName)
	if err != nil {
		if review_settings_db.IsErrReviewSettingsDoesntExistsError(err) {
			log.Error("Review settings for repo with id %d and branch name %s do no exist: %v", ctx.Repo.Repository.ID, branchName, err)
			ctx.Error(http.StatusNotFound, "Review settings do not exist")
		} else {
			log.Error("Error has occurred while getting review setting by repository id %d and branch name %s: %v", ctx.Repo.Repository.ID, branchName, err)
			ctx.Error(http.StatusInternalServerError, "Fail to get review settings")
		}
		return
	}
	defaultReviewers, err := s.GetDefaultReviewers(ctx, reviewSetting.ID)
	if err != nil {
		log.Error("Error has occurred while getting default reviewers by setting id %d: %v", reviewSetting.ID, err)
		ctx.Error(http.StatusInternalServerError, "Fail to get default reviewers")
		return
	}
	apiReview, err := models.ConvertReviewSettingsToAPIModel(ctx, reviewSetting, defaultReviewers)
	if err != nil {
		log.Error("Error has occurred while converting review setting: %v", err)
		ctx.Error(http.StatusInternalServerError, "Fail to convert review setting")
		return
	}
	users, err := access_model.GetRepoReaders(ctx.Repo.Repository)
	if err != nil {
		ctx.ServerError("Fail to get users to display", err)
		return
	}
	ctx.Data["Users"] = users
	ctx.Data["ReviewSetting"] = apiReview
	ctx.Data["OrgLink"] = ctx.Org.Organization.AsUser().OrganisationLink()
	ctx.HTML(http.StatusOK, tmplReviewSettingCreate)
}

func (s server) CreateReviewSetting(ctx *context.Context) {
	users, err := access_model.GetRepoReaders(ctx.Repo.Repository)
	if err != nil {
		ctx.ServerError("Fail to get users to display", err)
		return
	}
	ctx.Data["Users"] = users
	ctx.Data["OrgLink"] = ctx.Org.Organization.AsUser().OrganisationLink()
	ctx.HTML(http.StatusOK, tmplReviewSettingCreate)
}
