package convert

import (
	"testing"

	"code.gitea.io/gitea/models/git/protected_branch"
	"code.gitea.io/gitea/models/git/protected_branch/convert/mocks"

	"github.com/stretchr/testify/require"
)

func TestConvert(t *testing.T) {
	mockUserConverterDB := mocks.NewUserConverterDb(t)
	auditConverter := NewAuditConverter(mockUserConverterDB)

	// Тестовый случай 1: все поля имеют значения по умолчанию
	protectBranch := protected_branch.ProtectedBranch{}
	var expectedNilUserIds []int64
	var returnNilUserNames []string
	mockUserConverterDB.On("GetUserNames", expectedNilUserIds).Return(returnNilUserNames)
	require.Equal(t, protected_branch.AuditProtectedBranch{}, auditConverter.Convert(protectBranch))

	// Тестовый случай 2: все поля имеют значения true
	protectBranch = protected_branch.ProtectedBranch{
		RuleName:                     "test",
		EnableWhitelist:              true,
		WhitelistUserIDs:             []int64{1, 2, 3},
		WhitelistDeployKeys:          true,
		EnableForcePushWhitelist:     true,
		ForcePushWhitelistUserIDs:    []int64{4, 5, 6},
		ForcePushWhitelistDeployKeys: true,
		EnableDeleterWhitelist:       true,
		DeleterWhitelistUserIDs:      []int64{7, 8, 9},
		DeleterWhitelistDeployKeys:   true,
		RequireSignedCommits:         true,
		ProtectedFilePatterns:        `"pattern1", "pattern2"`,
		UnprotectedFilePatterns:      `"pattern3", "pattern4"`,
	}
	mockUserConverterDB.On("GetUserNames", []int64{1, 2, 3}).Return([]string{"user1", "user2", "user3"})
	mockUserConverterDB.On("GetUserNames", []int64{4, 5, 6}).Return([]string{"user4", "user5", "user6"})
	mockUserConverterDB.On("GetUserNames", []int64{7, 8, 9}).Return([]string{"user7", "user8", "user9"})
	expected := protected_branch.AuditProtectedBranch{
		BranchName: "test",
		PushSettings: protected_branch.AuditProtectedBranchPushSettings{
			RequirePushWhitelist:   true,
			PushWhitelistUsernames: []string{"user1", "user2", "user3"},
			AllowPushDeployKeys:    true,
		},
		ForcePushSettings: protected_branch.AuditProtectedBranchForcePushSettings{
			RequireForcePushWhitelist:   true,
			ForcePushWhitelistUsernames: []string{"user4", "user5", "user6"},
			AllowForcePushDeployKeys:    true,
		},
		DeletionSettings: protected_branch.AuditProtectedBranchDeletionSettings{
			RequireDeletionWhitelist:   true,
			DeletionWhitelistUsernames: []string{"user7", "user8", "user9"},
			AllowDeletionDeployKeys:    true,
		},
		AdditionalRestrictions: protected_branch.AuditProtectedBranchAdditionalRestrictions{
			RequireSignedCommits:    true,
			ProtectedFilePatterns:   `"pattern1", "pattern2"`,
			UnprotectedFilePatterns: `"pattern3", "pattern4"`,
		},
	}
	require.Equal(t, expected, auditConverter.Convert(protectBranch))

	// Тестовый случай 3: все поля имеют значения false
	protectBranch = protected_branch.ProtectedBranch{
		RuleName:                     "test",
		EnableWhitelist:              false,
		WhitelistUserIDs:             []int64{},
		WhitelistDeployKeys:          false,
		EnableForcePushWhitelist:     false,
		ForcePushWhitelistUserIDs:    []int64{},
		ForcePushWhitelistDeployKeys: false,
		EnableDeleterWhitelist:       false,
		DeleterWhitelistUserIDs:      []int64{},
		DeleterWhitelistDeployKeys:   false,
		RequireSignedCommits:         false,
		ProtectedFilePatterns:        "",
		UnprotectedFilePatterns:      "",
	}
	mockUserConverterDB.On("GetUserNames", []int64{}).Return([]string{})
	expected = protected_branch.AuditProtectedBranch{
		BranchName: "test",
		PushSettings: protected_branch.AuditProtectedBranchPushSettings{
			RequirePushWhitelist:   false,
			PushWhitelistUsernames: []string{},
			AllowPushDeployKeys:    false,
		},
		ForcePushSettings: protected_branch.AuditProtectedBranchForcePushSettings{
			RequireForcePushWhitelist:   false,
			ForcePushWhitelistUsernames: []string{},
			AllowForcePushDeployKeys:    false,
		},
		DeletionSettings: protected_branch.AuditProtectedBranchDeletionSettings{
			RequireDeletionWhitelist:   false,
			DeletionWhitelistUsernames: []string{},
			AllowDeletionDeployKeys:    false,
		},
		AdditionalRestrictions: protected_branch.AuditProtectedBranchAdditionalRestrictions{
			RequireSignedCommits:    false,
			ProtectedFilePatterns:   "",
			UnprotectedFilePatterns: "",
		},
	}
	require.Equal(t, expected, auditConverter.Convert(protectBranch))

	// Тестовый случай 4: некоторые поля имеют значения true, некоторые - false
	protectBranch = protected_branch.ProtectedBranch{
		RuleName:                     "test",
		EnableWhitelist:              true,
		WhitelistUserIDs:             []int64{1, 2, 3},
		WhitelistDeployKeys:          false,
		EnableForcePushWhitelist:     false,
		ForcePushWhitelistUserIDs:    []int64{},
		ForcePushWhitelistDeployKeys: false,
		EnableDeleterWhitelist:       true,
		DeleterWhitelistUserIDs:      []int64{7, 8, 9},
		DeleterWhitelistDeployKeys:   false,
		RequireSignedCommits:         false,
		ProtectedFilePatterns:        `"pattern1", "pattern2"`,
		UnprotectedFilePatterns:      "",
	}
	mockUserConverterDB.On("GetUserNames", []int64{1, 2, 3}).Return([]string{"user1", "user2", "user3"})
	mockUserConverterDB.On("GetUserNames", []int64{}).Return([]string{})
	mockUserConverterDB.On("GetUserNames", []int64{7, 8, 9}).Return([]string{"user7", "user8", "user9"})
	expected = protected_branch.AuditProtectedBranch{
		BranchName: "test",
		PushSettings: protected_branch.AuditProtectedBranchPushSettings{
			RequirePushWhitelist:   true,
			PushWhitelistUsernames: []string{"user1", "user2", "user3"},
			AllowPushDeployKeys:    false,
		},
		ForcePushSettings: protected_branch.AuditProtectedBranchForcePushSettings{
			RequireForcePushWhitelist:   false,
			ForcePushWhitelistUsernames: []string{},
			AllowForcePushDeployKeys:    false,
		},
		DeletionSettings: protected_branch.AuditProtectedBranchDeletionSettings{
			RequireDeletionWhitelist:   true,
			DeletionWhitelistUsernames: []string{"user7", "user8", "user9"},
			AllowDeletionDeployKeys:    false,
		},
		AdditionalRestrictions: protected_branch.AuditProtectedBranchAdditionalRestrictions{
			RequireSignedCommits:    false,
			ProtectedFilePatterns:   `"pattern1", "pattern2"`,
			UnprotectedFilePatterns: "",
		},
	}
	require.Equal(t, expected, auditConverter.Convert(protectBranch))

	// Тестовый случай 5: некоторые поля имеют значения false, некоторые - true
	protectBranch = protected_branch.ProtectedBranch{
		RuleName:                     "test",
		EnableWhitelist:              false,
		WhitelistUserIDs:             []int64{},
		WhitelistDeployKeys:          true,
		EnableForcePushWhitelist:     true,
		ForcePushWhitelistUserIDs:    []int64{4, 5, 6},
		ForcePushWhitelistDeployKeys: true,
		EnableDeleterWhitelist:       false,
		DeleterWhitelistUserIDs:      []int64{},
		DeleterWhitelistDeployKeys:   true,
		RequireSignedCommits:         true,
		ProtectedFilePatterns:        "",
		UnprotectedFilePatterns:      `"pattern3", "pattern4"`,
	}
	mockUserConverterDB.On("GetUserNames", []int64{}).Return([][]string{})
	mockUserConverterDB.On("GetUserNames", []int64{4, 5, 6}).Return([]string{"user4", "user5", "user6"})
	expected = protected_branch.AuditProtectedBranch{
		BranchName: "test",
		PushSettings: protected_branch.AuditProtectedBranchPushSettings{
			RequirePushWhitelist:   false,
			PushWhitelistUsernames: []string{},
			AllowPushDeployKeys:    true,
		},
		ForcePushSettings: protected_branch.AuditProtectedBranchForcePushSettings{
			RequireForcePushWhitelist:   true,
			ForcePushWhitelistUsernames: []string{"user4", "user5", "user6"},
			AllowForcePushDeployKeys:    true,
		},
		DeletionSettings: protected_branch.AuditProtectedBranchDeletionSettings{
			RequireDeletionWhitelist:   false,
			DeletionWhitelistUsernames: []string{},
			AllowDeletionDeployKeys:    true,
		},
		AdditionalRestrictions: protected_branch.AuditProtectedBranchAdditionalRestrictions{
			RequireSignedCommits:    true,
			ProtectedFilePatterns:   "",
			UnprotectedFilePatterns: `"pattern3", "pattern4"`,
		},
	}
	require.Equal(t, expected, auditConverter.Convert(protectBranch))

	// Тестовый случай 6: некоторые поля имеют значения true, некоторые - false, некоторые - пустые строки
	protectBranch = protected_branch.ProtectedBranch{
		RuleName:                     "",
		EnableWhitelist:              true,
		WhitelistUserIDs:             []int64{1, 2, 3},
		WhitelistDeployKeys:          false,
		EnableForcePushWhitelist:     false,
		ForcePushWhitelistUserIDs:    []int64{},
		ForcePushWhitelistDeployKeys: false,
		EnableDeleterWhitelist:       true,
		DeleterWhitelistUserIDs:      []int64{7, 8, 9},
		DeleterWhitelistDeployKeys:   false,
		RequireSignedCommits:         false,
		ProtectedFilePatterns:        "",
		UnprotectedFilePatterns:      "",
	}
	mockUserConverterDB.On("GetUserNames", []int64{1, 2, 3}).Return([]string{"user1", "user2", "user3"})
	mockUserConverterDB.On("GetUserNames", []int64{}).Return([]string{})
	mockUserConverterDB.On("GetUserNames", []int64{7, 8, 9}).Return([]string{"user7", "user8", "user9"})
	expected = protected_branch.AuditProtectedBranch{
		BranchName: "",
		PushSettings: protected_branch.AuditProtectedBranchPushSettings{
			RequirePushWhitelist:   true,
			PushWhitelistUsernames: []string{"user1", "user2", "user3"},
			AllowPushDeployKeys:    false,
		},
		ForcePushSettings: protected_branch.AuditProtectedBranchForcePushSettings{
			RequireForcePushWhitelist:   false,
			ForcePushWhitelistUsernames: []string{},
			AllowForcePushDeployKeys:    false,
		},
		DeletionSettings: protected_branch.AuditProtectedBranchDeletionSettings{
			RequireDeletionWhitelist:   true,
			DeletionWhitelistUsernames: []string{"user7", "user8", "user9"},
			AllowDeletionDeployKeys:    false,
		},
		AdditionalRestrictions: protected_branch.AuditProtectedBranchAdditionalRestrictions{
			RequireSignedCommits:    false,
			ProtectedFilePatterns:   "",
			UnprotectedFilePatterns: "",
		},
	}
	require.Equal(t, expected, auditConverter.Convert(protectBranch))

	// Тестовый случай 7: некоторые поля имеют значения false, некоторые - true, некоторые - пустые строки
	protectBranch = protected_branch.ProtectedBranch{
		RuleName:                     "",
		EnableWhitelist:              false,
		WhitelistUserIDs:             []int64{},
		WhitelistDeployKeys:          true,
		EnableForcePushWhitelist:     true,
		ForcePushWhitelistUserIDs:    []int64{4, 5, 6},
		ForcePushWhitelistDeployKeys: true,
		EnableDeleterWhitelist:       false,
		DeleterWhitelistUserIDs:      []int64{},
		DeleterWhitelistDeployKeys:   true,
		RequireSignedCommits:         true,
		ProtectedFilePatterns:        "",
		UnprotectedFilePatterns:      "",
	}
	mockUserConverterDB.On("GetUserNames", []int64{}).Return([]string{})
	mockUserConverterDB.On("GetUserNames", []int64{4, 5, 6}).Return([]string{"user4", "user5", "user6"})
	expected = protected_branch.AuditProtectedBranch{
		BranchName: "",
		PushSettings: protected_branch.AuditProtectedBranchPushSettings{
			RequirePushWhitelist:   false,
			PushWhitelistUsernames: []string{},
			AllowPushDeployKeys:    true,
		},
		ForcePushSettings: protected_branch.AuditProtectedBranchForcePushSettings{
			RequireForcePushWhitelist:   true,
			ForcePushWhitelistUsernames: []string{"user4", "user5", "user6"},
			AllowForcePushDeployKeys:    true,
		},
		DeletionSettings: protected_branch.AuditProtectedBranchDeletionSettings{
			RequireDeletionWhitelist:   false,
			DeletionWhitelistUsernames: []string{},
			AllowDeletionDeployKeys:    true,
		},
		AdditionalRestrictions: protected_branch.AuditProtectedBranchAdditionalRestrictions{
			RequireSignedCommits:    true,
			ProtectedFilePatterns:   "",
			UnprotectedFilePatterns: "",
		},
	}
	require.Equal(t, expected, auditConverter.Convert(protectBranch))

	// Тестовый случай 8: некоторые поля имеют значения true, некоторые - false, некоторые - пустые строки, некоторые - nil
	protectBranch = protected_branch.ProtectedBranch{
		RuleName:                     "",
		EnableWhitelist:              true,
		WhitelistUserIDs:             []int64{1, 2, 3},
		WhitelistDeployKeys:          false,
		EnableForcePushWhitelist:     false,
		ForcePushWhitelistUserIDs:    nil,
		ForcePushWhitelistDeployKeys: false,
		EnableDeleterWhitelist:       true,
		DeleterWhitelistUserIDs:      []int64{7, 8, 9},
		DeleterWhitelistDeployKeys:   false,
		RequireSignedCommits:         false,
		ProtectedFilePatterns:        "",
		UnprotectedFilePatterns:      "",
	}
	mockUserConverterDB.On("GetUserNames", []int64{1, 2, 3}).Return([]string{"user1", "user2", "user3"})
	mockUserConverterDB.On("GetUserNames", expectedNilUserIds).Return(returnNilUserNames)
	mockUserConverterDB.On("GetUserNames", []int64{7, 8, 9}).Return([]string{"user7", "user8", "user9"})
	expected = protected_branch.AuditProtectedBranch{
		BranchName: "",
		PushSettings: protected_branch.AuditProtectedBranchPushSettings{
			RequirePushWhitelist:   true,
			PushWhitelistUsernames: []string{"user1", "user2", "user3"},
			AllowPushDeployKeys:    false,
		},
		ForcePushSettings: protected_branch.AuditProtectedBranchForcePushSettings{
			RequireForcePushWhitelist:   false,
			ForcePushWhitelistUsernames: returnNilUserNames,
			AllowForcePushDeployKeys:    false,
		},
		DeletionSettings: protected_branch.AuditProtectedBranchDeletionSettings{
			RequireDeletionWhitelist:   true,
			DeletionWhitelistUsernames: []string{"user7", "user8", "user9"},
			AllowDeletionDeployKeys:    false,
		},
		AdditionalRestrictions: protected_branch.AuditProtectedBranchAdditionalRestrictions{
			RequireSignedCommits:    false,
			ProtectedFilePatterns:   "",
			UnprotectedFilePatterns: "",
		},
	}
	require.Equal(t, expected, auditConverter.Convert(protectBranch))

	// Тестовый случай 9: некоторые поля имеют значения false, некоторые - true, некоторые - пустые строки, некоторые - nil
	protectBranch = protected_branch.ProtectedBranch{
		RuleName:                     "",
		EnableWhitelist:              false,
		WhitelistUserIDs:             []int64{},
		WhitelistDeployKeys:          true,
		EnableForcePushWhitelist:     true,
		ForcePushWhitelistUserIDs:    []int64{4, 5, 6},
		ForcePushWhitelistDeployKeys: true,
		EnableDeleterWhitelist:       false,
		DeleterWhitelistUserIDs:      nil,
		DeleterWhitelistDeployKeys:   true,
		RequireSignedCommits:         true,
		ProtectedFilePatterns:        "",
		UnprotectedFilePatterns:      "",
	}
	mockUserConverterDB.On("GetUserNames", expectedNilUserIds).Return(returnNilUserNames)
	mockUserConverterDB.On("GetUserNames", []int64{4, 5, 6}).Return([]string{"user4", "user5", "user6"})
	expected = protected_branch.AuditProtectedBranch{
		BranchName: "",
		PushSettings: protected_branch.AuditProtectedBranchPushSettings{
			RequirePushWhitelist:   false,
			PushWhitelistUsernames: []string{},
			AllowPushDeployKeys:    true,
		},
		ForcePushSettings: protected_branch.AuditProtectedBranchForcePushSettings{
			RequireForcePushWhitelist:   true,
			ForcePushWhitelistUsernames: []string{"user4", "user5", "user6"},
			AllowForcePushDeployKeys:    true,
		},
		DeletionSettings: protected_branch.AuditProtectedBranchDeletionSettings{
			RequireDeletionWhitelist:   false,
			DeletionWhitelistUsernames: returnNilUserNames,
			AllowDeletionDeployKeys:    true,
		},
		AdditionalRestrictions: protected_branch.AuditProtectedBranchAdditionalRestrictions{
			RequireSignedCommits:    true,
			ProtectedFilePatterns:   "",
			UnprotectedFilePatterns: "",
		},
	}
	require.Equal(t, expected, auditConverter.Convert(protectBranch))

	// Тестовый случай 10: все поля имеют значения nil
	protectBranch = protected_branch.ProtectedBranch{
		RuleName:                     "",
		EnableWhitelist:              false,
		WhitelistUserIDs:             nil,
		WhitelistDeployKeys:          false,
		EnableForcePushWhitelist:     false,
		ForcePushWhitelistUserIDs:    nil,
		ForcePushWhitelistDeployKeys: false,
		EnableDeleterWhitelist:       false,
		DeleterWhitelistUserIDs:      nil,
		DeleterWhitelistDeployKeys:   false,
		RequireSignedCommits:         false,
		ProtectedFilePatterns:        "",
		UnprotectedFilePatterns:      "",
	}
	mockUserConverterDB.On("GetUserNames", expectedNilUserIds).Return(returnNilUserNames)
	expected = protected_branch.AuditProtectedBranch{
		BranchName: "",
		PushSettings: protected_branch.AuditProtectedBranchPushSettings{
			RequirePushWhitelist:   false,
			PushWhitelistUsernames: returnNilUserNames,
			AllowPushDeployKeys:    false,
		},
		ForcePushSettings: protected_branch.AuditProtectedBranchForcePushSettings{
			RequireForcePushWhitelist:   false,
			ForcePushWhitelistUsernames: returnNilUserNames,
			AllowForcePushDeployKeys:    false,
		},
		DeletionSettings: protected_branch.AuditProtectedBranchDeletionSettings{
			RequireDeletionWhitelist:   false,
			DeletionWhitelistUsernames: returnNilUserNames,
			AllowDeletionDeployKeys:    false,
		},
		AdditionalRestrictions: protected_branch.AuditProtectedBranchAdditionalRestrictions{
			RequireSignedCommits:    false,
			ProtectedFilePatterns:   "",
			UnprotectedFilePatterns: "",
		},
	}
	require.Equal(t, expected, auditConverter.Convert(protectBranch))
}
