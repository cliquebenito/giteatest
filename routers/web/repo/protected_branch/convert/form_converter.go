package form_converter

import (
	"fmt"
	"strings"

	"code.gitea.io/gitea/models/git/protected_branch"
	"code.gitea.io/gitea/modules/base"
	"code.gitea.io/gitea/services/forms"
)

type FormConverter struct{}

func NewFormConverter() FormConverter {
	return FormConverter{}
}

func (s FormConverter) ConvertProtectBranchFormToProtectedBranchRule(form *forms.ProtectBranchForm, protectBranch *protected_branch.ProtectedBranch) (*protected_branch.ProtectedBranch, error) {
	var err error
	protectBranch.RuleName = form.RuleName

	switch form.EnablePush {
	case "all":
		protectBranch.EnableWhitelist = false
		protectBranch.WhitelistDeployKeys = false
	case "whitelist":
		protectBranch.EnableWhitelist = true
		protectBranch.WhitelistDeployKeys = form.WhitelistDeployKeys
		var whitelistUsers []int64
		if strings.TrimSpace(form.WhitelistUsers) != "" {
			whitelistUsers, err = base.StringsToInt64s(strings.Split(form.WhitelistUsers, ","))
			if err != nil {
				return nil, fmt.Errorf("Err: convert string to int64: %w", err)
			}
		}
		protectBranch.WhitelistUserIDs = whitelistUsers
	default:
		protectBranch.EnableWhitelist = false
		protectBranch.WhitelistDeployKeys = false
	}

	switch form.EnableDelete {
	case "all":
		protectBranch.EnableDeleterWhitelist = false
		protectBranch.DeleterWhitelistDeployKeys = false
	case "whitelist":
		protectBranch.EnableDeleterWhitelist = true
		protectBranch.DeleterWhitelistDeployKeys = form.DeleterWhitelistDeployKeys
		var deleterWhitelistUsers []int64
		if strings.TrimSpace(form.DeleterWhitelistUsers) != "" {
			deleterWhitelistUsers, err = base.StringsToInt64s(strings.Split(form.DeleterWhitelistUsers, ","))
			if err != nil {
				return nil, fmt.Errorf("Err: convert string to int64: %w", err)
			}
		}
		protectBranch.DeleterWhitelistUserIDs = deleterWhitelistUsers
	default:
		protectBranch.EnableDeleterWhitelist = false
		protectBranch.DeleterWhitelistDeployKeys = false
	}

	switch form.EnableForcePush {
	case "all":
		protectBranch.EnableForcePushWhitelist = false
		protectBranch.ForcePushWhitelistDeployKeys = false
	case "whitelist":
		protectBranch.EnableForcePushWhitelist = true
		protectBranch.ForcePushWhitelistDeployKeys = form.ForcePusherWhitelistDeployKeys
		var forcePusherWhitelistUsers []int64
		if strings.TrimSpace(form.ForcePusherWhitelistUsers) != "" {
			forcePusherWhitelistUsers, err = base.StringsToInt64s(strings.Split(form.ForcePusherWhitelistUsers, ","))
			if err != nil {
				return nil, fmt.Errorf("Err: convert string to int64: %w", err)
			}
		}
		protectBranch.ForcePushWhitelistUserIDs = forcePusherWhitelistUsers
	default:
		protectBranch.EnableForcePushWhitelist = false
		protectBranch.ForcePushWhitelistDeployKeys = false
	}

	protectBranch.RequireSignedCommits = form.RequireSignedCommits
	protectBranch.ProtectedFilePatterns = form.ProtectedFilePatterns
	protectBranch.UnprotectedFilePatterns = form.UnprotectedFilePatterns

	return protectBranch, nil
}
