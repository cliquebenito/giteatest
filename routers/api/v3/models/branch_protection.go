package models

// This struct is common body for request and response
// response wrapping in BranchProtectionRulesResponse or BranchProtectionResponse
type BranchProtectionBody struct {
	BranchName             string                 `json:"branch_name"`
	PushSettings           PushSettings           `json:"push_settings"`
	ForcePushSettings      ForcePushSettings      `json:"force_push_settings"`
	DeletionSettings       DeletionSettings       `json:"deletion_settings"`
	AdditionalRestrictions AdditionalRestrictions `json:"additional_restrictions"`
}

// Accumulates push settings
type PushSettings struct {
	RequirePushWhitelist   bool     `json:"require_push_whitelist"`
	PushWhitelistUsernames []string `json:"push_whitelist_usernames"`
	AllowPushDeployKeys    bool     `json:"allow_push_deploy_keys"`
}

// Accumulates forec push settings
type ForcePushSettings struct {
	RequireForcePushWhitelist   bool     `json:"require_force_push_whitelist"`
	ForcePushWhitelistUsernames []string `json:"force_push_whitelist_usernames"`
	AllowForcePushDeployKeys    bool     `json:"allow_force_push_deploy_keys"`
}

// Accumulates deletion settings
type DeletionSettings struct {
	RequireDeletionWhitelist   bool     `json:"require_deletion_whitelist"`
	DeletionWhitelistUsernames []string `json:"branch_deletion_whitelist_usernames"`
	AllowDeletionDeployKeys    bool     `json:"allow_deletion_deploy_keys"`
}

// Accumulates additional restrictions
type AdditionalRestrictions struct {
	// RequireSignedCommits    bool   `json:"require_signed_commits"`
	ProtectedFilePatterns   string `json:"protected_file_patterns"`
	UnprotectedFilePatterns string `json:"unprotected_file_patterns"`
}

func (b BranchProtectionBody) Validate() error {
	if b.BranchName == "" {
		return NewErrProtectedBranchBranchNameEmpty()
	}
	if len(b.BranchName) > 255 {
		return NewErrProtectedBranchBranchNameTooLong(b.BranchName)
	}

	if b.PushSettings.RequirePushWhitelist && b.PushSettings.PushWhitelistUsernames == nil {
		return NewErrPushWhitelistRequired()
	}

	if !b.PushSettings.RequirePushWhitelist && b.PushSettings.PushWhitelistUsernames != nil {
		return NewErrPushWhitelistUnexpected()
	}

	if b.ForcePushSettings.RequireForcePushWhitelist && b.ForcePushSettings.ForcePushWhitelistUsernames == nil {
		return NewErrForcePushWhitelistRequired()
	}

	if !b.ForcePushSettings.RequireForcePushWhitelist && b.ForcePushSettings.ForcePushWhitelistUsernames != nil {
		return NewErrForcePushWhitelistUnexpected()
	}

	if b.DeletionSettings.RequireDeletionWhitelist && b.DeletionSettings.DeletionWhitelistUsernames == nil {
		return NewErrDeleteWhitelistRequired()
	}

	if !b.DeletionSettings.RequireDeletionWhitelist && b.DeletionSettings.DeletionWhitelistUsernames != nil {
		return NewErrDeleteWhitelistUnexpected()
	}

	return nil
}
