package protected_brancher

import (
	"testing"

	"code.gitea.io/gitea/models/git/protected_branch"
	"code.gitea.io/gitea/models/perm"
	access_model "code.gitea.io/gitea/models/perm/access"
	"code.gitea.io/gitea/models/unit"
	user_model "code.gitea.io/gitea/models/user"

	"github.com/gobwas/glob"
	"github.com/stretchr/testify/require"
)

var checker *ProtectedBranchChecker

func init() {
	checker = NewProtectedBranchChecker()
}

func TestCheckUserCanPush(t *testing.T) {
	cases := []struct {
		name           string
		protectBranch  protected_branch.ProtectedBranch
		user           *user_model.User
		expectedResult bool
	}{
		{
			name: "EnableWhitelist is false",
			protectBranch: protected_branch.ProtectedBranch{
				EnableWhitelist: false,
			},
			user: &user_model.User{
				ID: 1,
			},
			expectedResult: true,
		},
		{
			name: "EnableWhitelist is true, user.ID is in WhitelistUserIDs",
			protectBranch: protected_branch.ProtectedBranch{
				EnableWhitelist:  true,
				WhitelistUserIDs: []int64{1, 2, 3},
			},
			user: &user_model.User{
				ID: 2,
			},
			expectedResult: true,
		},
		{
			name: "EnableWhitelist is true, user.ID is not in WhitelistUserIDs",
			protectBranch: protected_branch.ProtectedBranch{
				EnableWhitelist:  true,
				WhitelistUserIDs: []int64{1, 2, 3},
			},
			user: &user_model.User{
				ID: 4,
			},
			expectedResult: false,
		},
		{
			name: "EnableWhitelist is true, WhitelistUserIDs is empty",
			protectBranch: protected_branch.ProtectedBranch{
				EnableWhitelist:  true,
				WhitelistUserIDs: []int64{},
			},
			user: &user_model.User{
				ID: 1,
			},
			expectedResult: false,
		},
		{
			name: "EnableWhitelist is true, user.ID is in WhitelistUserIDs, but WhitelistUserIDs contains only one element",
			protectBranch: protected_branch.ProtectedBranch{
				EnableWhitelist:  true,
				WhitelistUserIDs: []int64{1},
			},
			user: &user_model.User{
				ID: 1,
			},
			expectedResult: true,
		},
		{
			name: "EnableWhitelist is true, user.ID is not in WhitelistUserIDs, but WhitelistUserIDs contains only one element",
			protectBranch: protected_branch.ProtectedBranch{
				EnableWhitelist:  true,
				WhitelistUserIDs: []int64{1},
			},
			user: &user_model.User{
				ID: 2,
			},
			expectedResult: false,
		},
		{
			name: "EnableWhitelist is true, user.ID is in WhitelistUserIDs, but WhitelistUserIDs contains only two elements",
			protectBranch: protected_branch.ProtectedBranch{
				EnableWhitelist:  true,
				WhitelistUserIDs: []int64{1, 2},
			},
			user: &user_model.User{
				ID: 1,
			},
			expectedResult: true,
		},
		{
			name: "EnableWhitelist is true, user.ID is not in WhitelistUserIDs, but WhitelistUserIDs contains only two elements",
			protectBranch: protected_branch.ProtectedBranch{
				EnableWhitelist:  true,
				WhitelistUserIDs: []int64{1, 2},
			},
			user: &user_model.User{
				ID: 3,
			},
			expectedResult: false,
		},
		{
			name: "EnableWhitelist is true, user.ID is in WhitelistUserIDs, but WhitelistUserIDs contains only three elements",
			protectBranch: protected_branch.ProtectedBranch{
				EnableWhitelist:  true,
				WhitelistUserIDs: []int64{1, 2, 3},
			},
			user: &user_model.User{
				ID: 1,
			},
			expectedResult: true,
		},
		{
			name: "EnableWhitelist is true, user.ID is not in WhitelistUserIDs, but WhitelistUserIDs contains only three elements",
			protectBranch: protected_branch.ProtectedBranch{
				EnableWhitelist:  true,
				WhitelistUserIDs: []int64{1, 2, 3},
			},
			user: &user_model.User{
				ID: 4,
			},
			expectedResult: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := checker.CheckUserCanPush(nil, tc.protectBranch, tc.user)
			require.Equal(t, tc.expectedResult, result)
		})
	}
}

func TestCheckUserCanDeleteBranch(t *testing.T) {
	cases := []struct {
		name           string
		protectBranch  protected_branch.ProtectedBranch
		user           *user_model.User
		expectedResult bool
	}{
		{
			name: "EnableDeleterWhitelist is false",
			protectBranch: protected_branch.ProtectedBranch{
				EnableDeleterWhitelist: false,
			},
			user: &user_model.User{
				ID: 1,
			},
			expectedResult: true,
		},
		{
			name: "EnableDeleterWhitelist is true, user.ID is in DeleterWhitelistUserIDs",
			protectBranch: protected_branch.ProtectedBranch{
				EnableDeleterWhitelist:  true,
				DeleterWhitelistUserIDs: []int64{1, 2, 3},
			},
			user: &user_model.User{
				ID: 2,
			},
			expectedResult: true,
		},
		{
			name: "EnableDeleterWhitelist is true, user.ID is not in DeleterWhitelistUserIDs",
			protectBranch: protected_branch.ProtectedBranch{
				EnableDeleterWhitelist:  true,
				DeleterWhitelistUserIDs: []int64{1, 2, 3},
			},
			user: &user_model.User{
				ID: 4,
			},
			expectedResult: false,
		},
		{
			name: "EnableDeleterWhitelist is true, DeleterWhitelistUserIDs is empty",
			protectBranch: protected_branch.ProtectedBranch{
				EnableDeleterWhitelist:  true,
				DeleterWhitelistUserIDs: []int64{},
			},
			user: &user_model.User{
				ID: 1,
			},
			expectedResult: false,
		},
		{
			name: "EnableDeleterWhitelist is true, user.ID is in DeleterWhitelistUserIDs, but DeleterWhitelistUserIDs contains only one element",
			protectBranch: protected_branch.ProtectedBranch{
				EnableDeleterWhitelist:  true,
				DeleterWhitelistUserIDs: []int64{1},
			},
			user: &user_model.User{
				ID: 1,
			},
			expectedResult: true,
		},
		{
			name: "EnableDeleterWhitelist is true, user.ID is not in DeleterWhitelistUserIDs, but DeleterWhitelistUserIDs contains only one element",
			protectBranch: protected_branch.ProtectedBranch{
				EnableDeleterWhitelist:  true,
				DeleterWhitelistUserIDs: []int64{1},
			},
			user: &user_model.User{
				ID: 2,
			},
			expectedResult: false,
		},
		{
			name: "EnableDeleterWhitelist is true, user.ID is in DeleterWhitelistUserIDs, but DeleterWhitelistUserIDs contains only two elements",
			protectBranch: protected_branch.ProtectedBranch{
				EnableDeleterWhitelist:  true,
				DeleterWhitelistUserIDs: []int64{1, 2},
			},
			user: &user_model.User{
				ID: 1,
			},
			expectedResult: true,
		},
		{
			name: "EnableDeleterWhitelist is true, user.ID is not in DeleterWhitelistUserIDs, but DeleterWhitelistUserIDs contains only two elements",
			protectBranch: protected_branch.ProtectedBranch{
				EnableDeleterWhitelist:  true,
				DeleterWhitelistUserIDs: []int64{1, 2},
			},
			user: &user_model.User{
				ID: 3,
			},
			expectedResult: false,
		},
		{
			name: "EnableDeleterWhitelist is true, user.ID is in DeleterWhitelistUserIDs, but DeleterWhitelistUserIDs contains only three elements",
			protectBranch: protected_branch.ProtectedBranch{
				EnableDeleterWhitelist:  true,
				DeleterWhitelistUserIDs: []int64{1, 2, 3},
			},
			user: &user_model.User{
				ID: 1,
			},
			expectedResult: true,
		},
		{
			name: "EnableDeleterWhitelist is true, user.ID is not in DeleterWhitelistUserIDs, but DeleterWhitelistUserIDs contains only three elements",
			protectBranch: protected_branch.ProtectedBranch{
				EnableDeleterWhitelist:  true,
				DeleterWhitelistUserIDs: []int64{1, 2, 3},
			},
			user: &user_model.User{
				ID: 4,
			},
			expectedResult: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := checker.CheckUserCanDeleteBranch(nil, tc.protectBranch, tc.user)
			require.Equal(t, tc.expectedResult, result)
		})
	}
}

func TestCheckUserCanForcePush(t *testing.T) {
	cases := []struct {
		name           string
		protectBranch  protected_branch.ProtectedBranch
		user           *user_model.User
		expectedResult bool
	}{
		{
			name: "EnableForcePushWhitelist is false",
			protectBranch: protected_branch.ProtectedBranch{
				EnableForcePushWhitelist: false,
			},
			user: &user_model.User{
				ID: 1,
			},
			expectedResult: true,
		},
		{
			name: "EnableForcePushWhitelist is true, user.ID is in ForcePushWhitelistUserIDs",
			protectBranch: protected_branch.ProtectedBranch{
				EnableForcePushWhitelist:  true,
				ForcePushWhitelistUserIDs: []int64{1, 2, 3},
			},
			user: &user_model.User{
				ID: 2,
			},
			expectedResult: true,
		},
		{
			name: "EnableForcePushWhitelist is true, user.ID is not in ForcePushWhitelistUserIDs",
			protectBranch: protected_branch.ProtectedBranch{
				EnableForcePushWhitelist:  true,
				ForcePushWhitelistUserIDs: []int64{1, 2, 3},
			},
			user: &user_model.User{
				ID: 4,
			},
			expectedResult: false,
		},
		{
			name: "EnableForcePushWhitelist is true, ForcePushWhitelistUserIDs is empty",
			protectBranch: protected_branch.ProtectedBranch{
				EnableForcePushWhitelist:  true,
				ForcePushWhitelistUserIDs: []int64{},
			},
			user: &user_model.User{
				ID: 1,
			},
			expectedResult: false,
		},
		{
			name: "EnableForcePushWhitelist is true, user.ID is in ForcePushWhitelistUserIDs, but ForcePushWhitelistUserIDs contains only one element",
			protectBranch: protected_branch.ProtectedBranch{
				EnableForcePushWhitelist:  true,
				ForcePushWhitelistUserIDs: []int64{1},
			},
			user: &user_model.User{
				ID: 1,
			},
			expectedResult: true,
		},
		{
			name: "EnableForcePushWhitelist is true, user.ID is not in ForcePushWhitelistUserIDs, but ForcePushWhitelistUserIDs contains only one element",
			protectBranch: protected_branch.ProtectedBranch{
				EnableForcePushWhitelist:  true,
				ForcePushWhitelistUserIDs: []int64{1},
			},
			user: &user_model.User{
				ID: 2,
			},
			expectedResult: false,
		},
		{
			name: "EnableForcePushWhitelist is true, user.ID is in ForcePushWhitelistUserIDs, but ForcePushWhitelistUserIDs contains only two elements",
			protectBranch: protected_branch.ProtectedBranch{
				EnableForcePushWhitelist:  true,
				ForcePushWhitelistUserIDs: []int64{1, 2},
			},
			user: &user_model.User{
				ID: 1,
			},
			expectedResult: true,
		},
		{
			name: "EnableForcePushWhitelist is true, user.ID is not in ForcePushWhitelistUserIDs, but ForcePushWhitelistUserIDs contains only two elements",
			protectBranch: protected_branch.ProtectedBranch{
				EnableForcePushWhitelist:  true,
				ForcePushWhitelistUserIDs: []int64{1, 2},
			},
			user: &user_model.User{
				ID: 3,
			},
			expectedResult: false,
		},
		{
			name: "EnableForcePushWhitelist is true, user.ID is in ForcePushWhitelistUserIDs, but ForcePushWhitelistUserIDs contains only three elements",
			protectBranch: protected_branch.ProtectedBranch{
				EnableForcePushWhitelist:  true,
				ForcePushWhitelistUserIDs: []int64{1, 2, 3},
			},
			user: &user_model.User{
				ID: 1,
			},
			expectedResult: true,
		},
		{
			name: "EnableForcePushWhitelist is true, user.ID is not in ForcePushWhitelistUserIDs, but ForcePushWhitelistUserIDs contains only three elements",
			protectBranch: protected_branch.ProtectedBranch{
				EnableForcePushWhitelist:  true,
				ForcePushWhitelistUserIDs: []int64{1, 2, 3},
			},
			user: &user_model.User{
				ID: 4,
			},
			expectedResult: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := checker.CheckUserCanForcePush(nil, tc.protectBranch, tc.user)
			require.Equal(t, tc.expectedResult, result)
		})
	}
}

func TestIsUserMergeWhitelisted(t *testing.T) {
	cases := []struct {
		name             string
		protectBranch    protected_branch.ProtectedBranch
		userID           int64
		permissionInRepo access_model.Permission
		expectedResult   bool
	}{
		{
			name: "EnableMergeWhitelist is false, user has write permission",
			protectBranch: protected_branch.ProtectedBranch{
				EnableMergeWhitelist: false,
			},
			userID: 1,
			permissionInRepo: access_model.Permission{
				UnitsMode: map[unit.Type]perm.AccessMode{unit.TypeCode: perm.AccessModeWrite},
			},
			expectedResult: true,
		},
		{
			name: "EnableMergeWhitelist is false, user does not have write permission",
			protectBranch: protected_branch.ProtectedBranch{
				EnableMergeWhitelist: false,
			},
			userID: 1,
			permissionInRepo: access_model.Permission{
				UnitsMode: map[unit.Type]perm.AccessMode{unit.TypeCode: perm.AccessModeRead},
			},
			expectedResult: false,
		},
		{
			name: "EnableMergeWhitelist is true, userID is in MergeWhitelistUserIDs",
			protectBranch: protected_branch.ProtectedBranch{
				EnableMergeWhitelist:  true,
				MergeWhitelistUserIDs: []int64{1, 2, 3},
			},
			userID: 2,
			permissionInRepo: access_model.Permission{
				UnitsMode: map[unit.Type]perm.AccessMode{unit.TypeCode: perm.AccessModeRead},
			},
			expectedResult: true,
		},
		{
			name: "EnableMergeWhitelist is true, userID is not in MergeWhitelistUserIDs",
			protectBranch: protected_branch.ProtectedBranch{
				EnableMergeWhitelist:  true,
				MergeWhitelistUserIDs: []int64{1, 2, 3},
			},
			userID: 4,
			permissionInRepo: access_model.Permission{
				UnitsMode: map[unit.Type]perm.AccessMode{unit.TypeCode: perm.AccessModeRead},
			},
			expectedResult: false,
		},
		{
			name: "EnableMergeWhitelist is true, MergeWhitelistUserIDs is empty",
			protectBranch: protected_branch.ProtectedBranch{
				EnableMergeWhitelist:  true,
				MergeWhitelistUserIDs: []int64{},
			},
			userID: 1,
			permissionInRepo: access_model.Permission{
				UnitsMode: map[unit.Type]perm.AccessMode{unit.TypeCode: perm.AccessModeRead},
			},
			expectedResult: false,
		},
		{
			name: "EnableMergeWhitelist is true, userID is in MergeWhitelistUserIDs, but MergeWhitelistUserIDs contains only one element",
			protectBranch: protected_branch.ProtectedBranch{
				EnableMergeWhitelist:  true,
				MergeWhitelistUserIDs: []int64{1},
			},
			userID: 1,
			permissionInRepo: access_model.Permission{
				UnitsMode: map[unit.Type]perm.AccessMode{unit.TypeCode: perm.AccessModeRead},
			},
			expectedResult: true,
		},
		{
			name: "EnableMergeWhitelist is true, userID is not in MergeWhitelistUserIDs, but MergeWhitelistUserIDs contains only one element",
			protectBranch: protected_branch.ProtectedBranch{
				EnableMergeWhitelist:  true,
				MergeWhitelistUserIDs: []int64{1},
			},
			userID: 2,
			permissionInRepo: access_model.Permission{
				UnitsMode: map[unit.Type]perm.AccessMode{unit.TypeCode: perm.AccessModeRead},
			},
			expectedResult: false,
		},
		{
			name: "EnableMergeWhitelist is true, userID is in MergeWhitelistUserIDs, but MergeWhitelistUserIDs contains only two elements",
			protectBranch: protected_branch.ProtectedBranch{
				EnableMergeWhitelist:  true,
				MergeWhitelistUserIDs: []int64{1, 2},
			},
			userID: 1,
			permissionInRepo: access_model.Permission{
				UnitsMode: map[unit.Type]perm.AccessMode{unit.TypeCode: perm.AccessModeRead},
			},
			expectedResult: true,
		},
		{
			name: "EnableMergeWhitelist is true, userID is not in MergeWhitelistUserIDs, but MergeWhitelistUserIDs contains only two elements",
			protectBranch: protected_branch.ProtectedBranch{
				EnableMergeWhitelist:  true,
				MergeWhitelistUserIDs: []int64{1, 2},
			},
			userID: 3,
			permissionInRepo: access_model.Permission{
				UnitsMode: map[unit.Type]perm.AccessMode{unit.TypeCode: perm.AccessModeRead},
			},
			expectedResult: false,
		},
		{
			name: "EnableMergeWhitelist is true, userID is in MergeWhitelistUserIDs, but MergeWhitelistUserIDs contains only three elements",
			protectBranch: protected_branch.ProtectedBranch{
				EnableMergeWhitelist:  true,
				MergeWhitelistUserIDs: []int64{1, 2, 3},
			},
			userID: 1,
			permissionInRepo: access_model.Permission{
				UnitsMode: map[unit.Type]perm.AccessMode{unit.TypeCode: perm.AccessModeRead},
			},
			expectedResult: true,
		},
		{
			name: "EnableMergeWhitelist is true, userID is not in MergeWhitelistUserIDs, but MergeWhitelistUserIDs contains only three elements",
			protectBranch: protected_branch.ProtectedBranch{
				EnableMergeWhitelist:  true,
				MergeWhitelistUserIDs: []int64{1, 2, 3},
			},
			userID: 4,
			permissionInRepo: access_model.Permission{
				UnitsMode: map[unit.Type]perm.AccessMode{unit.TypeCode: perm.AccessModeRead},
			},
			expectedResult: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := checker.IsUserMergeWhitelisted(nil, tc.protectBranch, tc.userID, tc.permissionInRepo)
			require.Equal(t, tc.expectedResult, result)
		})
	}
}

func TestIsUserOfficialReviewer(t *testing.T) {
	cases := []struct {
		name           string
		protectBranch  protected_branch.ProtectedBranch
		user           *user_model.User
		expectedResult bool
	}{
		{
			name: "EnableApprovalsWhitelist is false",
			protectBranch: protected_branch.ProtectedBranch{
				EnableApprovalsWhitelist: false,
			},
			user: &user_model.User{
				ID: 1,
			},
			expectedResult: true,
		},
		{
			name: "EnableApprovalsWhitelist is true, user.ID is in ApprovalsWhitelistUserIDs",
			protectBranch: protected_branch.ProtectedBranch{
				EnableApprovalsWhitelist:  true,
				ApprovalsWhitelistUserIDs: []int64{1, 2, 3},
			},
			user: &user_model.User{
				ID: 2,
			},
			expectedResult: true,
		},
		{
			name: "EnableApprovalsWhitelist is true, user.ID is not in ApprovalsWhitelistUserIDs",
			protectBranch: protected_branch.ProtectedBranch{
				EnableApprovalsWhitelist:  true,
				ApprovalsWhitelistUserIDs: []int64{1, 2, 3},
			},
			user: &user_model.User{
				ID: 4,
			},
			expectedResult: false,
		},
		{
			name: "EnableApprovalsWhitelist is true, ApprovalsWhitelistUserIDs is empty",
			protectBranch: protected_branch.ProtectedBranch{
				EnableApprovalsWhitelist:  true,
				ApprovalsWhitelistUserIDs: []int64{},
			},
			user: &user_model.User{
				ID: 1,
			},
			expectedResult: false,
		},
		{
			name: "EnableApprovalsWhitelist is true, user.ID is in ApprovalsWhitelistUserIDs, but ApprovalsWhitelistUserIDs contains only one element",
			protectBranch: protected_branch.ProtectedBranch{
				EnableApprovalsWhitelist:  true,
				ApprovalsWhitelistUserIDs: []int64{1},
			},
			user: &user_model.User{
				ID: 1,
			},
			expectedResult: true,
		},
		{
			name: "EnableApprovalsWhitelist is true, user.ID is not in ApprovalsWhitelistUserIDs, but ApprovalsWhitelistUserIDs contains only one element",
			protectBranch: protected_branch.ProtectedBranch{
				EnableApprovalsWhitelist:  true,
				ApprovalsWhitelistUserIDs: []int64{1},
			},
			user: &user_model.User{
				ID: 2,
			},
			expectedResult: false,
		},
		{
			name: "EnableApprovalsWhitelist is true, user.ID is in ApprovalsWhitelistUserIDs, but ApprovalsWhitelistUserIDs contains only two elements",
			protectBranch: protected_branch.ProtectedBranch{
				EnableApprovalsWhitelist:  true,
				ApprovalsWhitelistUserIDs: []int64{1, 2},
			},
			user: &user_model.User{
				ID: 1,
			},
			expectedResult: true,
		},
		{
			name: "EnableApprovalsWhitelist is true, user.ID is not in ApprovalsWhitelistUserIDs, but ApprovalsWhitelistUserIDs contains only two elements",
			protectBranch: protected_branch.ProtectedBranch{
				EnableApprovalsWhitelist:  true,
				ApprovalsWhitelistUserIDs: []int64{1, 2},
			},
			user: &user_model.User{
				ID: 3,
			},
			expectedResult: false,
		},
		{
			name: "EnableApprovalsWhitelist is true, user.ID is in ApprovalsWhitelistUserIDs, but ApprovalsWhitelistUserIDs contains only three elements",
			protectBranch: protected_branch.ProtectedBranch{
				EnableApprovalsWhitelist:  true,
				ApprovalsWhitelistUserIDs: []int64{1, 2, 3},
			},
			user: &user_model.User{
				ID: 1,
			},
			expectedResult: true,
		},
		{
			name: "EnableApprovalsWhitelist is true, user.ID is not in ApprovalsWhitelistUserIDs, but ApprovalsWhitelistUserIDs contains only three elements",
			protectBranch: protected_branch.ProtectedBranch{
				EnableApprovalsWhitelist:  true,
				ApprovalsWhitelistUserIDs: []int64{1, 2, 3},
			},
			user: &user_model.User{
				ID: 4,
			},
			expectedResult: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := checker.IsUserOfficialReviewer(nil, tc.protectBranch, tc.user)
			require.Equal(t, tc.expectedResult, result)
		})
	}
}

func TestIsProtectedFile(t *testing.T) {
	cases := []struct {
		name           string
		protectBranch  protected_branch.ProtectedBranch
		patterns       []glob.Glob
		path           string
		expectedResult bool
	}{
		{
			name: "patterns is empty, GetProtectedFilePatterns returns empty slice",
			protectBranch: protected_branch.ProtectedBranch{
				ProtectedFilePatterns: "",
			},
			patterns:       []glob.Glob{},
			path:           "test",
			expectedResult: false,
		},
		{
			name: "patterns is empty, GetProtectedFilePatterns returns non-empty slice, path does not match any pattern",
			protectBranch: protected_branch.ProtectedBranch{
				ProtectedFilePatterns: "pattern1;pattern2",
			},
			patterns:       []glob.Glob{},
			path:           "test",
			expectedResult: false,
		},
		{
			name: "patterns is empty, GetProtectedFilePatterns returns non-empty slice, path matches one of the patterns",
			protectBranch: protected_branch.ProtectedBranch{
				ProtectedFilePatterns: "pattern1;pattern2",
			},
			patterns:       []glob.Glob{},
			path:           "pattern1",
			expectedResult: true,
		},
		{
			name:           "patterns is non-empty, path does not match any pattern",
			protectBranch:  protected_branch.ProtectedBranch{},
			patterns:       []glob.Glob{glob.MustCompile("pattern1"), glob.MustCompile("pattern2")},
			path:           "test",
			expectedResult: false,
		},
		{
			name:           "patterns is non-empty, path matches one of the patterns",
			protectBranch:  protected_branch.ProtectedBranch{},
			patterns:       []glob.Glob{glob.MustCompile("pattern1"), glob.MustCompile("pattern2")},
			path:           "pattern1",
			expectedResult: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := checker.IsProtectedFile(nil, tc.protectBranch, tc.patterns, tc.path)
			require.Equal(t, tc.expectedResult, result)
		})
	}
}

func TestIsUnprotectedFile(t *testing.T) {
	cases := []struct {
		name           string
		protectBranch  protected_branch.ProtectedBranch
		patterns       []glob.Glob
		path           string
		expectedResult bool
	}{
		{
			name: "patterns is empty, GetUnprotectedFilePatterns returns empty slice",
			protectBranch: protected_branch.ProtectedBranch{
				UnprotectedFilePatterns: "",
			},
			patterns:       []glob.Glob{},
			path:           "test",
			expectedResult: false,
		},
		{
			name: "patterns is empty, GetUnprotectedFilePatterns returns non-empty slice, path does not match any pattern",
			protectBranch: protected_branch.ProtectedBranch{
				UnprotectedFilePatterns: "pattern1;pattern2",
			},
			patterns:       []glob.Glob{},
			path:           "test",
			expectedResult: false,
		},
		{
			name: "patterns is empty, GetUnprotectedFilePatterns returns non-empty slice, path matches one of the patterns",
			protectBranch: protected_branch.ProtectedBranch{
				UnprotectedFilePatterns: "pattern1;pattern2",
			},
			patterns:       []glob.Glob{},
			path:           "pattern1",
			expectedResult: true,
		},
		{
			name:           "patterns is non-empty, path does not match any pattern",
			protectBranch:  protected_branch.ProtectedBranch{},
			patterns:       []glob.Glob{glob.MustCompile("pattern1"), glob.MustCompile("pattern2")},
			path:           "test",
			expectedResult: false,
		},
		{
			name:           "patterns is non-empty, path matches one of the patterns",
			protectBranch:  protected_branch.ProtectedBranch{},
			patterns:       []glob.Glob{glob.MustCompile("pattern1"), glob.MustCompile("pattern2")},
			path:           "pattern1",
			expectedResult: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := checker.IsUnprotectedFile(nil, tc.protectBranch, tc.patterns, tc.path)
			require.Equal(t, tc.expectedResult, result)
		})
	}
}

func TestMergeBlockedByProtectedFiles(t *testing.T) {
	cases := []struct {
		name                  string
		protectBranch         protected_branch.ProtectedBranch
		changedProtectedFiles []string
		expectedResult        bool
	}{
		{
			name: "GetProtectedFilePatterns returns empty slice",
			protectBranch: protected_branch.ProtectedBranch{
				ProtectedFilePatterns: "",
			},
			changedProtectedFiles: []string{},
			expectedResult:        false,
		},
		{
			name: "GetProtectedFilePatterns returns non-empty slice, changedProtectedFiles is empty",
			protectBranch: protected_branch.ProtectedBranch{
				ProtectedFilePatterns: "pattern1;pattern2",
			},
			changedProtectedFiles: []string{},
			expectedResult:        false,
		},
		{
			name: "GetProtectedFilePatterns returns non-empty slice, changedProtectedFiles is non-empty",
			protectBranch: protected_branch.ProtectedBranch{
				ProtectedFilePatterns: "pattern1;pattern2",
			},
			changedProtectedFiles: []string{"file1", "file2"},
			expectedResult:        true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := checker.MergeBlockedByProtectedFiles(nil, tc.protectBranch, tc.changedProtectedFiles)
			require.Equal(t, tc.expectedResult, result)
		})
	}
}
