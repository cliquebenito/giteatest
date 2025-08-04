package protected_branch

type AuditProtectedBranch struct {
	BranchName             string                                     `json:"branch_name"`
	PushSettings           AuditProtectedBranchPushSettings           `json:"push_settings"`
	ForcePushSettings      AuditProtectedBranchForcePushSettings      `json:"force_push_settings"`
	DeletionSettings       AuditProtectedBranchDeletionSettings       `json:"deletion_settings"`
	AdditionalRestrictions AuditProtectedBranchAdditionalRestrictions `json:"additional_restrictions"`
}

type AuditProtectedBranchPushSettings struct {
	RequirePushWhitelist   bool     `json:"require_push_whitelist"`
	PushWhitelistUsernames []string `json:"push_whitelist_usernames"`
	AllowPushDeployKeys    bool     `json:"allow_push_deploy_keys"`
}

type AuditProtectedBranchForcePushSettings struct {
	RequireForcePushWhitelist   bool     `json:"require_force_push_whitelist"`
	ForcePushWhitelistUsernames []string `json:"force_push_whitelist_usernames"`
	AllowForcePushDeployKeys    bool     `json:"allow_force_push_deploy_keys"`
}

type AuditProtectedBranchDeletionSettings struct {
	RequireDeletionWhitelist   bool     `json:"require_deletion_whitelist"`
	DeletionWhitelistUsernames []string `json:"branch_deletion_whitelist_usernames"`
	AllowDeletionDeployKeys    bool     `json:"allow_deletion_deploy_keys"`
}

type AuditProtectedBranchAdditionalRestrictions struct {
	RequireSignedCommits    bool   `json:"require_signed_commits"`
	ProtectedFilePatterns   string `json:"protected_file_patterns"`
	UnprotectedFilePatterns string `json:"unprotected_file_patterns"`
}
