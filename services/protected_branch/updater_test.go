package protected_brancher

import (
	"testing"

	"code.gitea.io/gitea/models/git/protected_branch"
	protectd_branch_mocks "code.gitea.io/gitea/services/protected_branch/mocks"

	"github.com/stretchr/testify/require"
)

var updater *ProtectedBranchUpdater
var mockUpdater *protectd_branch_mocks.ProtectedBranchUpdater

func init() {
	updater = NewProtectedBranchUpdater()
}

func TestUpdateModelProtectedBranch(t *testing.T) {
	tests := []struct {
		name     string
		existing *protected_branch.ProtectedBranch
		new      *protected_branch.ProtectedBranch
		expected *protected_branch.ProtectedBranch
	}{
		{
			name: "Update all whitelists true -> false",
			existing: &protected_branch.ProtectedBranch{
				EnableWhitelist:              true,
				WhitelistDeployKeys:          true,
				EnableForcePushWhitelist:     true,
				ForcePushWhitelistDeployKeys: true,
				EnableDeleterWhitelist:       true,
				DeleterWhitelistDeployKeys:   true,
			},
			new: &protected_branch.ProtectedBranch{
				EnableWhitelist:              false,
				WhitelistDeployKeys:          false,
				EnableForcePushWhitelist:     false,
				ForcePushWhitelistDeployKeys: false,
				EnableDeleterWhitelist:       false,
				DeleterWhitelistDeployKeys:   false,
			},
			expected: &protected_branch.ProtectedBranch{
				EnableWhitelist:              false,
				WhitelistDeployKeys:          false,
				EnableForcePushWhitelist:     false,
				ForcePushWhitelistDeployKeys: false,
				EnableDeleterWhitelist:       false,
				DeleterWhitelistDeployKeys:   false,
			},
		},
		{
			name: "Update all whitelists false -> true",
			existing: &protected_branch.ProtectedBranch{
				EnableWhitelist:              false,
				WhitelistDeployKeys:          false,
				EnableForcePushWhitelist:     false,
				ForcePushWhitelistDeployKeys: false,
				EnableDeleterWhitelist:       false,
				DeleterWhitelistDeployKeys:   false,
			},
			new: &protected_branch.ProtectedBranch{
				EnableWhitelist:              true,
				WhitelistDeployKeys:          true,
				EnableForcePushWhitelist:     true,
				ForcePushWhitelistDeployKeys: true,
				EnableDeleterWhitelist:       true,
				DeleterWhitelistDeployKeys:   true,
			},
			expected: &protected_branch.ProtectedBranch{
				EnableWhitelist:              true,
				WhitelistDeployKeys:          true,
				EnableForcePushWhitelist:     true,
				ForcePushWhitelistDeployKeys: true,
				EnableDeleterWhitelist:       true,
				DeleterWhitelistDeployKeys:   true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := updater.UpdateModelProtectedBranch(tt.existing, tt.new)
			require.Equal(t, tt.expected, result)
		})
	}
}
