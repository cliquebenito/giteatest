package convert

import (
	"code.gitea.io/gitea/models/git/protected_branch"
)

// //go:generate mockery --name=userConverterDb --exported
type userConverterManager interface {
	GetUserNames(ids []int64) []string
}

type AuditConverter struct {
	userManager userConverterManager
}

func NewAuditConverter(userManager userConverterManager) AuditConverter {
	return AuditConverter{userManager: userManager}
}

func (c AuditConverter) Convert(protectBranch protected_branch.ProtectedBranch) protected_branch.AuditProtectedBranch {
	auditProtectedBranch := protected_branch.AuditProtectedBranch{
		BranchName: protectBranch.RuleName,
		PushSettings: protected_branch.AuditProtectedBranchPushSettings{
			RequirePushWhitelist:   protectBranch.EnableWhitelist,
			PushWhitelistUsernames: c.userManager.GetUserNames(protectBranch.WhitelistUserIDs),
			AllowPushDeployKeys:    protectBranch.WhitelistDeployKeys,
		},
		ForcePushSettings: protected_branch.AuditProtectedBranchForcePushSettings{
			RequireForcePushWhitelist:   protectBranch.EnableForcePushWhitelist,
			ForcePushWhitelistUsernames: c.userManager.GetUserNames(protectBranch.ForcePushWhitelistUserIDs),
			AllowForcePushDeployKeys:    protectBranch.ForcePushWhitelistDeployKeys,
		},
		DeletionSettings: protected_branch.AuditProtectedBranchDeletionSettings{
			RequireDeletionWhitelist:   protectBranch.EnableDeleterWhitelist,
			DeletionWhitelistUsernames: c.userManager.GetUserNames(protectBranch.DeleterWhitelistUserIDs),
			AllowDeletionDeployKeys:    protectBranch.DeleterWhitelistDeployKeys,
		},
		AdditionalRestrictions: protected_branch.AuditProtectedBranchAdditionalRestrictions{
			RequireSignedCommits:    protectBranch.RequireSignedCommits,
			ProtectedFilePatterns:   protectBranch.ProtectedFilePatterns,
			UnprotectedFilePatterns: protectBranch.UnprotectedFilePatterns,
		},
	}
	return auditProtectedBranch
}
