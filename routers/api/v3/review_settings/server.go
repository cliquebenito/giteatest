package review_settings

import (
	gocontext "context"
	"fmt"
	"net/http"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/default_reviewers"
	"code.gitea.io/gitea/models/review_settings"
	"code.gitea.io/gitea/models/review_settings/review_settings_db"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/json"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/sbt/audit"
	auditutils "code.gitea.io/gitea/modules/sbt/audit/utils"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/routers/api/v3/models"
)

type server struct {
	defaultReviewersDB
	reviewSettingsDB
}

func NewServer(defaultReviewersDB defaultReviewersDB, reviewSettingsDB reviewSettingsDB) *server {
	return &server{defaultReviewersDB: defaultReviewersDB, reviewSettingsDB: reviewSettingsDB}
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

func (s server) GetReviewSettingsHandler(ctx *context.APIContext) {
	// swagger:operation GET /repos/{tenant}/{project}/{repo}/review_settings GetReviewSettings
	// ---
	// summary: Returns all review settings for repository
	// produces:
	// - application/json
	// parameters:
	// - name: tenant
	//   in: path
	//   required: true
	//   type: string
	//   description: Tenant identifier
	// - name: project
	//   in: path
	//   required: true
	//   type: string
	//   description: Project identifier
	// - name: repo
	//   in: path
	//   required: true
	//   type: string
	//   description: Repository identifier
	// responses:
	//   200:
	//     "$ref": "#/responses/ReviewSettings"
	//   400:
	//     description: Bad request
	//     schema:
	//       type: object
	//       properties:
	//         errors:
	//           type: array
	//           items:
	//             type: string
	//           description: List of error messages
	//         message:
	//           type: string
	//           description: Error message
	//         url:
	//           type: string
	//           description: Link to Swagger documentation
	//   404:
	//     description: Not found
	//   500:
	//     description: Internal server error
	//     schema:
	//       type: object
	//       properties:
	//         errors:
	//           type: array
	//           items:
	//             type: string
	//           description: List of error messages
	//         message:
	//           type: string
	//           description: Error message
	//         url:
	//           type: string
	//           description: Link to Swagger documentation

	s.getReviewSettings(ctx)
}

func (s server) getReviewSettings(ctx *context.APIContext) {
	reviewSettings, err := s.GetReviewSettings(ctx, ctx.Repo.Repository.ID)
	if err != nil {
		if review_settings_db.IsErrReviewSettingsDoesntExistsError(err) {
			log.Error("Review settings for repo with id %d do no exist: %v", ctx.Repo.Repository.ID, err)
			ctx.Error(http.StatusNotFound, "Review settings do not exist", err)
		} else {
			log.Error("Error has occurred while getting review settings by repository id %d: %v", ctx.Repo.Repository.ID, err)
			ctx.Error(http.StatusInternalServerError, "Fail to get review settings", err)
		}
		return
	}
	result := make([]models.BranchReviewSetting, len(reviewSettings))
	for i, rs := range reviewSettings {
		defaultReviewers, err := s.GetDefaultReviewers(ctx, rs.ID)
		if err != nil {
			log.Error("Error has occurred while getting default reviewers by setting id %d: %v", rs.ID, err)
			ctx.Error(http.StatusInternalServerError, "Fail to get default reviewers", err)
			return
		}
		apiReview, err := models.ConvertReviewSettingsToAPIModel(ctx, reviewSettings[0], defaultReviewers)
		if err != nil {
			log.Error("Error has occurred while converting review setting: %v", err)
			ctx.Error(http.StatusInternalServerError, "Fail to convert review setting", err)
			return
		}
		result[i] = *apiReview
	}
	ctx.JSON(http.StatusOK, result)
}

func (s server) GetBranchReviewSettings(ctx *context.APIContext) {
	// swagger:operation GET /repos/{tenant}/{project}/{repo}/review_settings/{branch_name} GetBranchReviewSettings
	// ---
	// summary: Returns branch review settings for repository
	// produces:
	// - application/json
	// parameters:
	// - name: tenant
	//   in: path
	//   required: true
	//   type: string
	//   description: Tenant identifier
	// - name: project
	//   in: path
	//   required: true
	//   type: string
	//   description: Project identifier
	// - name: repo
	//   in: path
	//   required: true
	//   type: string
	//   description: Repository identifier
	// - name: branch_name
	//   in: path
	//   required: true
	//   type: string
	//   description: Branch identifier
	// responses:
	//   200:
	//     "$ref": "#/responses/ReviewSettings"
	//   400:
	//     description: Bad request
	//     schema:
	//       type: object
	//       properties:
	//         errors:
	//           type: array
	//           items:
	//             type: string
	//           description: List of error messages
	//         message:
	//           type: string
	//           description: Error message
	//         url:
	//           type: string
	//           description: Link to Swagger documentation
	//   404:
	//     description: Not found
	//   500:
	//     description: Internal server error
	//     schema:
	//       type: object
	//       properties:
	//         errors:
	//           type: array
	//           items:
	//             type: string
	//           description: List of error messages
	//         message:
	//           type: string
	//           description: Error message
	//         url:
	//           type: string
	//           description: Link to Swagger documentation

	s.getBranchReviewSettings(ctx)
}

func (s server) getBranchReviewSettings(ctx *context.APIContext) {
	branchName := ctx.Params(":branch_name")
	reviewSetting, err := s.GetReviewSettingsByBranchPattern(ctx, ctx.Repo.Repository.ID, branchName)
	if err != nil {
		if review_settings_db.IsErrReviewSettingsDoesntExistsError(err) {
			log.Error("Review settings for repo with id %d and branch name %s do no exist: %v", ctx.Repo.Repository.ID, branchName, err)
			ctx.Error(http.StatusNotFound, "Review settings do not exist", err)
		} else {
			log.Error("Error has occurred while getting review setting by repository id %d and branch name %s: %v", ctx.Repo.Repository.ID, branchName, err)
			ctx.Error(http.StatusInternalServerError, "Fail to get review settings", err)
		}
		return
	}
	defaultReviewers, err := s.GetDefaultReviewers(ctx, reviewSetting.ID)
	if err != nil {
		log.Error("Error has occurred while getting default reviewers by setting id %d: %v", reviewSetting.ID, err)
		ctx.Error(http.StatusInternalServerError, "Fail to get default reviewers", err)
		return
	}
	apiReview, err := models.ConvertReviewSettingsToAPIModel(ctx, reviewSetting, defaultReviewers)
	if err != nil {
		log.Error("Error has occurred while converting review setting: %v", err)
		ctx.Error(http.StatusInternalServerError, "Fail to convert review setting", err)
		return
	}
	ctx.JSON(http.StatusOK, apiReview)
}

func (s server) CreateReviewSettings(ctx *context.APIContext) {
	// swagger:operation POST /repos/{tenant}/{project}/{repo}/review_settings CreateReviewSettings
	// ---
	// summary: Creates review settings for repository
	// produces:
	// - application/json
	// parameters:
	// - name: tenant
	//   in: path
	//   required: true
	//   type: string
	//   description: Tenant identifier
	// - name: project
	//   in: path
	//   required: true
	//   type: string
	//   description: Project identifier
	// - name: repo
	//   in: path
	//   required: true
	//   type: string
	//   description: Repository identifier
	// - name: body
	//   in: body
	//   description: Параметры для создания правила ревью
	//   required: true
	//   schema:
	//     type: object
	//     required:
	//       - branch_name
	//       - approval_settings
	//       - merge_restrictions
	//       - merge_settings
	//       - status_checks
	//     properties:
	//       branch_name:
	//         type: string
	//         description: Название ветки, к которой применяются настройки (например, "*")
	//       approval_settings:
	//         type: object
	//         description: Настройки ревью
	//         required:
	//           - require_default_reviewers
	//           - default_reviewers
	//         properties:
	//           require_default_reviewers:
	//             type: boolean
	//             description: Назначать ли ревьюеров по умолчанию
	//           default_reviewers:
	//             type: array
	//             description: Список наборов ревьюеров по умолчанию
	//             items:
	//               type: object
	//               required:
	//                 - default_reviewers_list
	//                 - required_approvals_count
	//               properties:
	//                 default_reviewers_list:
	//                   type: array
	//                   description: Список ревьюеров по умолчанию
	//                   items:
	//                     type: string
	//                 required_approvals_count:
	//                   type: integer
	//                   description: Минимальное количество необходимых аппрувов
	//       merge_restrictions:
	//         type: object
	//         description: Ограничения на слияние
	//         properties:
	//           block_on_official_review_requests:
	//             type: boolean
	//             description: Блокировать слияние, если есть запросы на официальное ревью
	//           block_on_outdated_branch:
	//             type: boolean
	//             description: Блокировать слияние, если ветка отстаёт от основной
	//           block_on_rejected_reviews:
	//             type: boolean
	//             description: Блокировать слияние при наличии отклонённых ревью
	//           dismiss_stale_approvals:
	//             type: boolean
	//             description: Сбрасывать аппрувы при новых изменениях
	//           require_sonarqube_quality_gate:
	//             type: boolean
	//             description: Требовать прохождения SonarQube Quality Gate
	//       merge_settings:
	//         type: object
	//         description: Настройки слияния
	//         properties:
	//           require_merge_whitelist:
	//             type: boolean
	//             description: Требовать белый список на слияние
	//           merge_whitelist_usernames:
	//             type: array
	//             description: Список пользователей, которым разрешено слияние
	//             items:
	//               type: string
	//       status_checks:
	//         type: object
	//         description: Проверки CI статусов
	//         properties:
	//           enable_status_check:
	//             type: boolean
	//             description: Включить проверку CI статусов
	//           status_check_contexts:
	//             type: array
	//             description: Список обязательных CI проверок
	//             items:
	//               type: string
	// responses:
	//   201:
	//     description: Ok
	//   400:
	//     description: Bad request
	//     schema:
	//       type: object
	//       properties:
	//         errors:
	//           type: array
	//           items:
	//             type: string
	//           description: List of error messages
	//         message:
	//           type: string
	//           description: Error message
	//         url:
	//           type: string
	//           description: Link to Swagger documentation
	//   500:
	//     description: Internal server error
	//     schema:
	//       type: object
	//       properties:
	//         errors:
	//           type: array
	//           items:
	//             type: string
	//           description: List of error messages
	//         message:
	//           type: string
	//           description: Error message
	//         url:
	//           type: string
	//           description: Link to Swagger documentation

	s.createReviewSettings(ctx)
}

func (s server) createReviewSettings(ctx *context.APIContext) {
	opt := web.GetForm(ctx).(*models.ReviewSettingsRequest)
	newValue, err := json.Marshal(opt)
	if err != nil {
		log.Error("Error has occurred while serializing new value: %v", err)
	}
	auditParams := map[string]string{
		"new_value": string(newValue),
	}
	auditValues := auditutils.NewRequiredAuditParamsFromApiContext(ctx)

	if err := opt.Validate(); err != nil {
		log.Debug("Error has occurred while validating review setting: %v", err)
		ctx.Error(http.StatusBadRequest, "Fail to validate review setting", err)
		return
	}
	reviewSetting, err := models.ConvertAPIToReviewSettingsModel(ctx, *opt, ctx.Repo.Repository.ID)
	if err != nil {
		log.Error("Error has occurred while converting review setting: %v", err)
		auditParams["error"] = "Error has occurred while converting review setting"
		audit.CreateAndSendEvent(audit.ReviewSettingCreateEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
		ctx.Error(http.StatusBadRequest, "Fail to convert review setting", err)
		return
	}
	defaultReviewers, err := models.ConvertDefaultReviewerSetsToDBModel(opt.ApprovalSettings.DefaultReviewers)
	if err != nil {
		log.Error("Error has occurred while converting default reviewers: %v", err)
		auditParams["error"] = "Error has occurred while converting default reviewers"
		audit.CreateAndSendEvent(audit.ReviewSettingCreateEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
		ctx.Error(http.StatusBadRequest, "Fail to convert default reviewers", err)
		return
	}

	// Check if already exists
	_, err = s.GetReviewSettingsByBranchPattern(ctx, ctx.Repo.Repository.ID, opt.BranchName)
	if err != nil {
		if !review_settings_db.IsErrReviewSettingsDoesntExistsError(err) {
			log.Error("Error has occurred while getting review setting by repository id %d and branch name %s: %v", ctx.Repo.Repository.ID, opt.BranchName, err)
			auditParams["error"] = "Error has occurred while getting review setting by repository id and branch name"
			audit.CreateAndSendEvent(audit.ReviewSettingCreateEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
			ctx.Error(http.StatusInternalServerError, "Fail to get review settings", err)
			return
		}
	} else {
		log.Debug("Review setting with repository id %d and branch name %s already exists", ctx.Repo.Repository.ID, opt.BranchName)
		auditParams["error"] = "Review setting with same repository id and branch name already exists"
		audit.CreateAndSendEvent(audit.ReviewSettingCreateEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
		ctx.Error(http.StatusConflict, "Review setting already exists", err)
		return
	}

	if err := db.WithTx(db.DefaultContext, func(context gocontext.Context) error {
		if err := s.UpsertReviewSettings(context, reviewSetting); err != nil {
			return fmt.Errorf("Error has occurred while updating default review settings: %w", err)
		}

		rs, err := s.GetReviewSettingsByBranchPattern(context, ctx.Repo.Repository.ID, opt.BranchName)
		if err != nil {
			return fmt.Errorf("Error has occurred while getting default review settings: %w", err)
		}

		for _, df := range defaultReviewers {
			df.ReviewSettingID = rs.ID
		}

		if err := s.InsertDefaultReviewers(context, defaultReviewers); err != nil {
			return fmt.Errorf("Error has occurred while inserting default reviewers: %w", err)
		}
		return nil
	}); err != nil {
		log.Error("Error has occurred while creating review settings: %w", err)
		auditParams["error"] = "Error has occurred while creating review settings"
		audit.CreateAndSendEvent(audit.ReviewSettingCreateEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
		ctx.Error(http.StatusInternalServerError, "Fail to create review settings", err)
		return
	}
	audit.CreateAndSendEvent(audit.ReviewSettingCreateEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusSuccess, auditValues.RemoteAddress, auditParams)
	ctx.Status(http.StatusCreated)
}

func (s server) UpdateReviewSettings(ctx *context.APIContext) {
	// swagger:operation PUT /repos/{tenant}/{project}/{repo}/review_settings/{branch_name} UpdateReviewSettings
	// ---
	// summary: Updates review settings for repository
	// produces:
	// - application/json
	// parameters:
	// - name: tenant
	//   in: path
	//   required: true
	//   type: string
	//   description: Tenant identifier
	// - name: project
	//   in: path
	//   required: true
	//   type: string
	//   description: Project identifier
	// - name: repo
	//   in: path
	//   required: true
	//   type: string
	//   description: Repository identifier
	// - name: branch_name
	//   in: path
	//   required: true
	//   type: string
	//   description: Branch identifier
	// - name: body
	//   in: body
	//   description: Параметры для создания правила ревью
	//   required: true
	//   schema:
	//     type: object
	//     required:
	//       - branch_name
	//       - approval_settings
	//       - merge_restrictions
	//       - merge_settings
	//       - status_checks
	//     properties:
	//       branch_name:
	//         type: string
	//         description: Название ветки, к которой применяются настройки (например, "*")
	//       approval_settings:
	//         type: object
	//         description: Настройки ревью
	//         required:
	//           - require_default_reviewers
	//           - default_reviewers
	//         properties:
	//           require_default_reviewers:
	//             type: boolean
	//             description: Назначать ли ревьюеров по умолчанию
	//           default_reviewers:
	//             type: array
	//             description: Список наборов ревьюеров по умолчанию
	//             items:
	//               type: object
	//               required:
	//                 - default_reviewers_list
	//                 - required_approvals_count
	//               properties:
	//                 default_reviewers_list:
	//                   type: array
	//                   description: Список ревьюеров по умолчанию
	//                   items:
	//                     type: string
	//                 required_approvals_count:
	//                   type: integer
	//                   description: Минимальное количество необходимых аппрувов
	//       merge_restrictions:
	//         type: object
	//         description: Ограничения на слияние
	//         properties:
	//           block_on_official_review_requests:
	//             type: boolean
	//             description: Блокировать слияние, если есть запросы на официальное ревью
	//           block_on_outdated_branch:
	//             type: boolean
	//             description: Блокировать слияние, если ветка отстаёт от основной
	//           block_on_rejected_reviews:
	//             type: boolean
	//             description: Блокировать слияние при наличии отклонённых ревью
	//           dismiss_stale_approvals:
	//             type: boolean
	//             description: Сбрасывать аппрувы при новых изменениях
	//           require_sonarqube_quality_gate:
	//             type: boolean
	//             description: Требовать прохождения SonarQube Quality Gate
	//       merge_settings:
	//         type: object
	//         description: Настройки слияния
	//         properties:
	//           require_merge_whitelist:
	//             type: boolean
	//             description: Требовать белый список на слияние
	//           merge_whitelist_usernames:
	//             type: array
	//             description: Список пользователей, которым разрешено слияние
	//             items:
	//               type: string
	//       status_checks:
	//         type: object
	//         description: Проверки CI статусов
	//         properties:
	//           enable_status_check:
	//             type: boolean
	//             description: Включить проверку CI статусов
	//           status_check_contexts:
	//             type: array
	//             description: Список обязательных CI проверок
	//             items:
	//               type: string
	// responses:
	//   200:
	//     description: Ok
	//   400:
	//     description: Bad request
	//     schema:
	//       type: object
	//       properties:
	//         errors:
	//           type: array
	//           items:
	//             type: string
	//           description: List of error messages
	//         message:
	//           type: string
	//           description: Error message
	//         url:
	//           type: string
	//           description: Link to Swagger documentation
	//   404:
	//     description: Not found
	//   500:
	//     description: Internal server error
	//     schema:
	//       type: object
	//       properties:
	//         errors:
	//           type: array
	//           items:
	//             type: string
	//           description: List of error messages
	//         message:
	//           type: string
	//           description: Error message
	//         url:
	//           type: string
	//           description: Link to Swagger documentation

	s.updateReviewSettings(ctx)
}

func (s server) updateReviewSettings(ctx *context.APIContext) {
	branchName := ctx.Params(":branch_name")
	opt := web.GetForm(ctx).(*models.ReviewSettingsRequest)
	newValue, err := json.Marshal(opt)
	if err != nil {
		log.Error("Error has occurred while serializing new value: %v", err)
	}
	auditParams := map[string]string{
		"new_value": string(newValue),
	}
	auditValues := auditutils.NewRequiredAuditParamsFromApiContext(ctx)

	reviewSetting, err := models.ConvertAPIToReviewSettingsModel(ctx, *opt, ctx.Repo.Repository.ID)
	if err != nil {
		log.Error("Error has occurred while converting review setting: %v", err)
		auditParams["error"] = "Error has occurred while converting review setting"
		audit.CreateAndSendEvent(audit.ReviewSettingUpdateEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
		ctx.Error(http.StatusBadRequest, "Fail to convert review setting", err)
		return
	}
	defaultReviewers, err := models.ConvertDefaultReviewerSetsToDBModel(opt.ApprovalSettings.DefaultReviewers)
	if err != nil {
		log.Error("Error has occurred while converting default reviewers: %v", err)
		auditParams["error"] = "Error has occurred while converting default reviewers"
		audit.CreateAndSendEvent(audit.ReviewSettingUpdateEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
		ctx.Error(http.StatusBadRequest, "Fail to convert default reviewers", err)
		return
	}

	// Check that exists
	_, err = s.GetReviewSettingsByBranchPattern(ctx, ctx.Repo.Repository.ID, branchName)
	if err != nil {
		if review_settings_db.IsErrReviewSettingsDoesntExistsError(err) {
			log.Error("Review settings for repo with id %d and branch name %s do not exist: %v", ctx.Repo.Repository.ID, branchName, err)
			auditParams["error"] = "Review setting for repo with requested id and branch name does not exist"
			audit.CreateAndSendEvent(audit.ReviewSettingUpdateEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
			ctx.Error(http.StatusNotFound, "Review settings do not exist", err)
		} else {
			log.Error("Error has occurred while getting review setting by repository id %d and branch name %s: %v", ctx.Repo.Repository.ID, branchName, err)
			auditParams["error"] = "Error has occurred while getting review setting by repository id and branch name"
			audit.CreateAndSendEvent(audit.ReviewSettingUpdateEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
			ctx.Error(http.StatusInternalServerError, "Fail to get review settings", err)
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
				ctx.Error(http.StatusInternalServerError, "Fail to get review settings", err)
				return
			}
		} else {
			log.Debug("Review setting with repository id %d and branch name %s already exists", ctx.Repo.Repository.ID, opt.BranchName)
			auditParams["error"] = "Review setting with requested repository id and branch name already exists"
			audit.CreateAndSendEvent(audit.ReviewSettingUpdateEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
			ctx.Error(http.StatusConflict, "Review setting already exists", err)
			return
		}
	}

	if err := db.WithTx(db.DefaultContext, func(context gocontext.Context) error {
		rs, err := s.GetReviewSettingsByBranchPattern(context, ctx.Repo.Repository.ID, branchName)
		if err != nil {
			return fmt.Errorf("Error has occurred while getting review settings: %w", err)
		}
		err = s.DeleteReviewSettingsByRepoID(context, ctx.Repo.Repository.ID, branchName)
		if err != nil {
			return fmt.Errorf("Error has occurred while deleting review settings: %w", err)
		}
		err = s.DeleteDefaultReviewersBySettingID(context, rs.ID)
		if err != nil {
			return fmt.Errorf("Error has occurred while deleting default reviewers: %w", err)
		}

		if err := s.UpsertReviewSettings(context, reviewSetting); err != nil {
			return fmt.Errorf("Error has occurred while updating default review settings: %w", err)
		}
		rs, err = s.GetReviewSettingsByBranchPattern(context, ctx.Repo.Repository.ID, opt.BranchName)
		if err != nil {
			return fmt.Errorf("Error has occurred while getting review settings: %w", err)
		}
		for _, df := range defaultReviewers {
			df.ReviewSettingID = rs.ID
		}

		if err := s.InsertDefaultReviewers(context, defaultReviewers); err != nil {
			return fmt.Errorf("Error has occurred while inserting default reviewers: %w", err)
		}

		return nil
	}); err != nil {
		log.Error("Error has occurred while updating review settings: %w", err)
		auditParams["error"] = "Error has occurred while updating review settings"
		audit.CreateAndSendEvent(audit.ReviewSettingUpdateEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
		ctx.Error(http.StatusInternalServerError, "Fail to update review settings", err)
		return
	}
	audit.CreateAndSendEvent(audit.ReviewSettingUpdateEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusSuccess, auditValues.RemoteAddress, auditParams)
	ctx.Status(http.StatusCreated)
}

func (s server) DeleteReviewSettings(ctx *context.APIContext) {
	// swagger:operation DELETE /repos/{tenant}/{project}/{repo}/review_settings/{branch_name} DeleteReviewSettings
	// ---
	// summary: Deletes branch review settings
	// produces:
	// - application/json
	// parameters:
	// - name: tenant
	//   in: path
	//   required: true
	//   type: string
	//   description: Tenant identifier
	// - name: project
	//   in: path
	//   required: true
	//   type: string
	//   description: Project identifier
	// - name: repo
	//   in: path
	//   required: true
	//   type: string
	//   description: Repository identifier
	// - name: branch_name
	//   in: path
	//   required: true
	//   type: string
	//   description: Branch identifier
	// responses:
	//   204:
	//     description: Ok
	//   400:
	//     description: Bad request
	//     schema:
	//       type: object
	//       properties:
	//         errors:
	//           type: array
	//           items:
	//             type: string
	//           description: List of error messages
	//         message:
	//           type: string
	//           description: Error message
	//         url:
	//           type: string
	//           description: Link to Swagger documentation
	//   404:
	//     description: Not found
	//   500:
	//     description: Internal server error
	//     schema:
	//       type: object
	//       properties:
	//         errors:
	//           type: array
	//           items:
	//             type: string
	//           description: List of error messages
	//         message:
	//           type: string
	//           description: Error message
	//         url:
	//           type: string
	//           description: Link to Swagger documentation

	s.deleteReviewSettings(ctx)
}

func (s server) deleteReviewSettings(ctx *context.APIContext) {
	branchName := ctx.Params(":branch_name")
	auditParams := map[string]string{}
	auditValues := auditutils.NewRequiredAuditParamsFromApiContext(ctx)

	// Check that exists
	value, err := s.GetReviewSettingsByBranchPattern(ctx, ctx.Repo.Repository.ID, branchName)
	if err != nil {
		if review_settings_db.IsErrReviewSettingsDoesntExistsError(err) {
			log.Error("Review settings for repo with id %d and branch name %s do no exist: %v", ctx.Repo.Repository.ID, branchName, err)
			auditParams["error"] = "Review settings for repo with requested id and branch name do no exist"
			audit.CreateAndSendEvent(audit.ReviewSettingDeleteEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
			ctx.Error(http.StatusNotFound, "Review settings do not exist", err)
		} else {
			log.Error("Error has occurred while getting review setting by repository id %d and branch name %s: %v", ctx.Repo.Repository.ID, branchName, err)
			auditParams["error"] = "Error has occurred while getting review setting by repository id and branch name"
			audit.CreateAndSendEvent(audit.ReviewSettingDeleteEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
			ctx.Error(http.StatusInternalServerError, "Fail to get review settings", err)
		}
		return
	}
	newValue, err := json.Marshal(value)
	if err != nil {
		log.Error("Error has occurred while serializing new value: %v", err)
	}
	auditParams["old_value"] = string(newValue)

	if err := db.WithTx(db.DefaultContext, func(context gocontext.Context) error {
		rs, err := s.GetReviewSettingsByBranchPattern(context, ctx.Repo.Repository.ID, branchName)
		if err != nil {
			return fmt.Errorf("Error has occurred while getting review settings: %w", err)
		}
		err = s.DeleteReviewSettingsByRepoID(ctx, ctx.Repo.Repository.ID, branchName)
		if err != nil {
			return fmt.Errorf("Error has occurred while deleting review settings: %w", err)
		}
		err = s.DeleteDefaultReviewersBySettingID(context, rs.ID)
		if err != nil {
			return fmt.Errorf("Error has occurred while deleting default reviewers: %w", err)
		}
		return nil
	}); err != nil {
		log.Error("Error has occurred while deleting review settings: %w", err)
		auditParams["error"] = "Error has occurred while deleting review settings"
		audit.CreateAndSendEvent(audit.ReviewSettingDeleteEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusFailure, auditValues.RemoteAddress, auditParams)
		ctx.Error(http.StatusInternalServerError, "Fail to delete review settings", err)
		return
	}
	audit.CreateAndSendEvent(audit.ReviewSettingDeleteEvent, auditValues.DoerName, auditValues.DoerID, audit.StatusSuccess, auditValues.RemoteAddress, auditParams)
	ctx.Status(http.StatusNoContent)
}
