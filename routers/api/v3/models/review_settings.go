package models

import (
	"context"
	"fmt"
	"strconv"

	"code.gitea.io/gitea/models/default_reviewers"
	"code.gitea.io/gitea/models/review_settings"
	"code.gitea.io/gitea/models/user"
)

// swagger:response ReviewSettings
type ReviewSettings struct {
	// Список настроек ревью для веток
	// in:body
	// required: true
	ReviewSettings []BranchReviewSetting `json:"review_settings"`
}

// ReviewSettingsRequest Параметры для создания правила ревью
// swagger:model
type ReviewSettingsRequest struct {
	// Название ветки, к которой применяются настройки (например, "*")
	// required: true
	BranchName string `json:"branch_name" binding:"Required"`

	// Настройки ревью
	// required: true
	ApprovalSettings ApprovalSettings `json:"approval_settings"`

	// Ограничения на слияние
	// required: true
	MergeRestrictions MergeRestrictions `json:"merge_restrictions"`

	// Настройки слияния
	// required: true
	MergeSettings MergeSettings `json:"merge_settings"`

	// Проверки CI статусов
	// required: true
	StatusChecks StatusChecks `json:"status_checks"`
}

// BranchReviewSetting описывает настройки ревью для конкретной ветки
// swagger:model
type BranchReviewSetting struct {
	// Название ветки, к которой применяются настройки (например, "*")
	// required: true
	BranchName string `json:"branch_name"`

	// Настройки ревью
	// required: true
	ApprovalSettings ApprovalSettings `json:"approval_settings"`

	// Ограничения на слияние
	// required: true
	MergeRestrictions MergeRestrictions `json:"merge_restrictions"`

	// Настройки слияния
	// required: true
	MergeSettings MergeSettings `json:"merge_settings"`

	// Проверки CI статусов
	// required: true
	StatusChecks StatusChecks `json:"status_checks"`
}

// ApprovalSettings содержит информацию о необходимых ревью перед слиянием
// swagger:model
type ApprovalSettings struct {
	// Назначать ли ревьюеров по умолчанию
	// required: true
	RequireDefaultReviewers bool `json:"require_default_reviewers"`

	// Список наборов ревьюеров по умолчанию
	// required: true
	DefaultReviewers []DefaultReviewerSet `json:"default_reviewers"`
}

// DefaultReviewerSet описывает один набор ревьюеров и требуемое количество аппрувов
// swagger:model
type DefaultReviewerSet struct {
	// Список ревьюеров по умолчанию
	// required: true
	DefaultReviewersList []string `json:"default_reviewers_list"`

	// Минимальное количество необходимых аппрувов
	// required: true
	RequiredApprovalsCount int `json:"required_approvals_count"`
}

// MergeRestrictions определяет, когда слияние должно быть заблокировано
// swagger:model
type MergeRestrictions struct {
	// Блокировать слияние, если есть запросы на официальное ревью
	BlockOnOfficialReviewRequests bool `json:"block_on_official_review_requests"`

	// Блокировать слияние, если ветка отстаёт от основной
	BlockOnOutdatedBranch bool `json:"block_on_outdated_branch"`

	// Блокировать слияние при наличии отклонённых ревью
	BlockOnRejectedReviews bool `json:"block_on_rejected_reviews"`

	// Сбрасывать аппрувы при новых изменениях
	DismissStaleApprovals bool `json:"dismiss_stale_approvals"`

	// Требовать прохождения SonarQube Quality Gate
	RequireSonarqubeQualityGate bool `json:"require_sonarqube_quality_gate"`
}

// MergeSettings определяет, кто может выполнять слияние
// swagger:model
type MergeSettings struct {
	// Требовать белый список на слияние
	RequireMergeWhitelist bool `json:"require_merge_whitelist"`

	// Список пользователей, которым разрешено слияние
	MergeWhitelistUsernames []string `json:"merge_whitelist_usernames"`
}

// StatusChecks описывает необходимые CI проверки перед слиянием
// swagger:model
type StatusChecks struct {
	// Включить проверку CI статусов
	EnableStatusCheck bool `json:"enable_status_check"`

	// Список обязательных CI проверок
	StatusCheckContexts []string `json:"status_check_contexts"`
}

func (r ReviewSettingsRequest) Validate() error {
	for _, dr := range r.ApprovalSettings.DefaultReviewers {
		if dr.RequiredApprovalsCount < 0 {
			return fmt.Errorf("negative required approvals count")
		}
	}
	return nil
}

func ConvertDefaultReviewersToAPIModel(dbDefaultReviewer *default_reviewers.DefaultReviewers) DefaultReviewerSet {
	reviewersList := make([]string, len(dbDefaultReviewer.DefaultReviewersList))
	for i, v := range dbDefaultReviewer.DefaultReviewersList {
		reviewersList[i] = strconv.FormatInt(v, 10)
	}
	return DefaultReviewerSet{
		RequiredApprovalsCount: int(dbDefaultReviewer.RequiredApprovals),
		DefaultReviewersList:   reviewersList,
	}
}

func ConvertReviewSettingsToAPIModel(
	ctx context.Context,
	dbModel *review_settings.ReviewSettings,
	dbDefaultReviewers []*default_reviewers.DefaultReviewers,
) (*BranchReviewSetting, error) {
	userNames, err := convertUserIDsToUsernames(ctx, dbModel.MergeWhitelistUserIDs, resolveUsername)
	if err != nil {
		return nil, fmt.Errorf("convert user ids: %w", err)
	}
	reviewersList := make([]DefaultReviewerSet, len(dbDefaultReviewers))
	for i, v := range dbDefaultReviewers {
		reviewersList[i] = ConvertDefaultReviewersToAPIModel(v)
	}
	return &BranchReviewSetting{
		BranchName: dbModel.RuleName,
		ApprovalSettings: ApprovalSettings{
			RequireDefaultReviewers: dbModel.EnableDefaultReviewers,
			DefaultReviewers:        reviewersList,
		},
		MergeRestrictions: MergeRestrictions{
			BlockOnOfficialReviewRequests: dbModel.BlockOnOfficialReviewRequests,
			BlockOnOutdatedBranch:         dbModel.BlockOnOutdatedBranch,
			BlockOnRejectedReviews:        dbModel.BlockOnRejectedReviews,
			DismissStaleApprovals:         dbModel.DismissStaleApprovals,
			RequireSonarqubeQualityGate:   dbModel.EnableSonarQube,
		},
		MergeSettings: MergeSettings{
			RequireMergeWhitelist:   dbModel.EnableMergeWhitelist,
			MergeWhitelistUsernames: userNames,
		},
		StatusChecks: StatusChecks{
			EnableStatusCheck:   dbModel.EnableStatusCheck,
			StatusCheckContexts: dbModel.StatusCheckContexts,
		},
	}, nil
}

func ConvertDefaultReviewerSetsToDBModel(sets []DefaultReviewerSet) ([]*default_reviewers.DefaultReviewers, error) {
	result := make([]*default_reviewers.DefaultReviewers, 0, len(sets))

	for _, set := range sets {
		var reviewerIDs []int64
		for _, userID := range set.DefaultReviewersList {
			id, err := strconv.ParseInt(userID, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("convert user id to int: %w", err)
			}
			if id != 0 {
				reviewerIDs = append(reviewerIDs, id)
			}
		}

		result = append(result, &default_reviewers.DefaultReviewers{
			RequiredApprovals:    int64(set.RequiredApprovalsCount),
			DefaultReviewersList: reviewerIDs,
		})
	}

	return result, nil
}

func ConvertAPIToReviewSettingsModel(
	ctx context.Context,
	apiModel ReviewSettingsRequest,
	repoID int64,
) (*review_settings.ReviewSettings, error) {
	whiteListUserIDs, err := convertUsernamesToUserIDs(ctx, apiModel.MergeSettings.MergeWhitelistUsernames, resolveUserID)
	if err != nil {
		return nil, fmt.Errorf("convert usernames: %w", err)
	}
	return &review_settings.ReviewSettings{
			RepoID:                        repoID,
			RuleName:                      apiModel.BranchName,
			EnableDefaultReviewers:        apiModel.ApprovalSettings.RequireDefaultReviewers,
			BlockOnOfficialReviewRequests: apiModel.MergeRestrictions.BlockOnOfficialReviewRequests,
			BlockOnOutdatedBranch:         apiModel.MergeRestrictions.BlockOnOutdatedBranch,
			BlockOnRejectedReviews:        apiModel.MergeRestrictions.BlockOnRejectedReviews,
			DismissStaleApprovals:         apiModel.MergeRestrictions.DismissStaleApprovals,
			EnableSonarQube:               apiModel.MergeRestrictions.RequireSonarqubeQualityGate,
			EnableMergeWhitelist:          apiModel.MergeSettings.RequireMergeWhitelist,
			MergeWhitelistUserIDs:         whiteListUserIDs,
			EnableStatusCheck:             apiModel.StatusChecks.EnableStatusCheck,
			StatusCheckContexts:           apiModel.StatusChecks.StatusCheckContexts,
		},
		nil
}

func convertUsernamesToUserIDs(ctx context.Context, usernames []string, resolver func(ctx context.Context, username string) (int64, error)) ([]int64, error) {
	userIDs := make([]int64, 0, len(usernames))
	for _, name := range usernames {
		id, err := resolver(ctx, name)
		if err != nil {
			return nil, fmt.Errorf("resolve username: %w", err)
		}
		if id != 0 {
			userIDs = append(userIDs, id)
		}
	}
	return userIDs, nil
}

func convertUserIDsToUsernames(ctx context.Context, userIDs []int64, resolver func(ctx context.Context, userID int64) (string, error)) ([]string, error) {
	usernames := make([]string, 0, len(userIDs))
	for _, id := range userIDs {
		username, err := resolver(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("resolve username: %w", err)
		}
		if username != "" {
			usernames = append(usernames, username)
		}
	}
	return usernames, nil
}

func resolveUsername(ctx context.Context, userID int64) (string, error) {
	user, err := user.GetUserByID(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("get user by id: %w", err)
	}
	return user.Name, nil
}

func resolveUserID(ctx context.Context, username string) (int64, error) {
	user, err := user.GetUserByName(ctx, username)
	if err != nil {
		return 0, fmt.Errorf("get user by name: %w", err)
	}
	return user.ID, nil
}
