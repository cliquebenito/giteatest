package convert_v3

import (
	"context"

	"code.gitea.io/gitea/models/git/protected_branch"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/routers/api/v3/models"
)

type BranchProtectionConverter struct{}

func NewBranchProtectionConverter() *BranchProtectionConverter {
	return &BranchProtectionConverter{}
}

func (b BranchProtectionConverter) ToBranchProtectionBody(rule protected_branch.ProtectedBranch) models.BranchProtectionBody {
	pushWhitelistUsernames, err := user_model.GetUserNamesByIDs(rule.WhitelistUserIDs)
	if err != nil {
		log.Error("Eror has occrured while GetUserNamesByIDs (WhitelistUserIDs): %v", err)
	}

	forcePushWhitelistUsernames, err := user_model.GetUserNamesByIDs(rule.ForcePushWhitelistUserIDs)
	if err != nil {
		log.Error("Eror has occrured while GetUserNamesByIDs (ForcePushWhitelistUserIDs): %v", err)
	}

	deletionWhitelistUsernames, err := user_model.GetUserNamesByIDs(rule.DeleterWhitelistUserIDs)
	if err != nil {
		log.Error("Eror has occrured while GetUserNamesByIDs (DeleterWhitelistUserIDs): %v", err)
	}

	return models.BranchProtectionBody{
		BranchName: rule.RuleName,
		PushSettings: models.PushSettings{
			RequirePushWhitelist:   rule.EnableWhitelist,
			PushWhitelistUsernames: pushWhitelistUsernames,
			AllowPushDeployKeys:    rule.WhitelistDeployKeys,
		},
		ForcePushSettings: models.ForcePushSettings{
			RequireForcePushWhitelist:   rule.EnableForcePushWhitelist,
			ForcePushWhitelistUsernames: forcePushWhitelistUsernames,
			AllowForcePushDeployKeys:    rule.ForcePushWhitelistDeployKeys,
		},
		DeletionSettings: models.DeletionSettings{
			RequireDeletionWhitelist:   rule.EnableDeleterWhitelist,
			DeletionWhitelistUsernames: deletionWhitelistUsernames,
			AllowDeletionDeployKeys:    rule.DeleterWhitelistDeployKeys,
		},
		AdditionalRestrictions: models.AdditionalRestrictions{
			// RequireSignedCommits:    rule.RequireSignedCommits,
			ProtectedFilePatterns:   rule.ProtectedFilePatterns,
			UnprotectedFilePatterns: rule.UnprotectedFilePatterns,
		},
	}
}

func (b BranchProtectionConverter) ToBranchProtectionRulesBody(rules protected_branch.ProtectedBranchRules) []models.BranchProtectionBody {
	result := make([]models.BranchProtectionBody, 0)
	for _, rule := range rules {
		result = append(result, b.ToBranchProtectionBody(*rule))
	}

	return result
}

func (b BranchProtectionConverter) ToProtectedBranch(ctx context.Context, protectedBranchRequest models.BranchProtectionBody) *protected_branch.ProtectedBranch {
	pushWhitelistIDs, err := user_model.GetUserIDsByNames(ctx, protectedBranchRequest.PushSettings.PushWhitelistUsernames, true)
	if err != nil {
		log.Error("GetUserIDsByNames (PushWhitelistUsernames): %v", err)
	}

	forcePushWhitelistIDs, err := user_model.GetUserIDsByNames(ctx, protectedBranchRequest.ForcePushSettings.ForcePushWhitelistUsernames, true)
	if err != nil {
		log.Error("GetUserIDsByNames (ForcePushWhitelistUsernames): %v", err)
	}

	deleterWhitelistIDs, err := user_model.GetUserIDsByNames(ctx, protectedBranchRequest.DeletionSettings.DeletionWhitelistUsernames, true)
	if err != nil {
		log.Error("GetUserIDsByNames (DeletionWhitelistUsernames): %v", err)
	}

	return &protected_branch.ProtectedBranch{
		RuleName: protectedBranchRequest.BranchName,

		EnableWhitelist:     protectedBranchRequest.PushSettings.RequirePushWhitelist,
		WhitelistUserIDs:    pushWhitelistIDs,
		WhitelistDeployKeys: protectedBranchRequest.PushSettings.AllowPushDeployKeys,

		EnableForcePushWhitelist:     protectedBranchRequest.ForcePushSettings.RequireForcePushWhitelist,
		ForcePushWhitelistDeployKeys: protectedBranchRequest.ForcePushSettings.AllowForcePushDeployKeys,
		ForcePushWhitelistUserIDs:    forcePushWhitelistIDs,

		EnableDeleterWhitelist:     protectedBranchRequest.DeletionSettings.RequireDeletionWhitelist,
		DeleterWhitelistUserIDs:    deleterWhitelistIDs,
		DeleterWhitelistDeployKeys: protectedBranchRequest.DeletionSettings.AllowDeletionDeployKeys,

		// RequireSignedCommits:    protectedBranchRequest.AdditionalRestrictions.RequireSignedCommits,
		ProtectedFilePatterns:   protectedBranchRequest.AdditionalRestrictions.ProtectedFilePatterns,
		UnprotectedFilePatterns: protectedBranchRequest.AdditionalRestrictions.UnprotectedFilePatterns,
	}
}
