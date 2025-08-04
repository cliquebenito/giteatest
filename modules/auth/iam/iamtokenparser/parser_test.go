package iamtokenparser

import (
	"testing"

	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/require"

	"code.gitea.io/gitea/modules/auth/iam/iamtoken"
)

func Test_getString(t *testing.T) {
	value := "value"
	name := "key"
	claims := jwt.MapClaims{
		name: value,
	}

	got, err := getString(name, claims)

	require.NoError(t, err)
	require.Equal(t, value, got)
}

func Test_getString_Negative(t *testing.T) {
	tests := []struct {
		name   string
		claims jwt.MapClaims
	}{
		{
			claims: jwt.MapClaims(nil),
		},
		{
			name:   "key",
			claims: jwt.MapClaims{"key1": ""},
		},
		{
			name:   "key",
			claims: jwt.MapClaims{"key": 4},
		},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got, err := getString(tt.name, tt.claims)

			require.Error(t, err)
			require.Empty(t, got)
		})
	}
}

func TestIAMJWTTokenParser_getRole(t *testing.T) {
	tests := []struct {
		name                string
		claims              jwt.MapClaims
		want                iamtoken.SourceControlGlobalRole
		wantIsGroupsInToken bool

		whiteListRolesAdmin map[string]struct{}
		whiteListRolesUser  map[string]struct{}
	}{
		{
			claims:              jwt.MapClaims{"groups": []interface{}{"ROLE_USER", "ROLE_ADMIN"}},
			whiteListRolesAdmin: map[string]struct{}{"ROLE_ADMIN": {}},
			whiteListRolesUser:  map[string]struct{}{"ROLE_USER": {}},
			want:                iamtoken.AdminRole,
			wantIsGroupsInToken: true,
		},
		{
			claims:              jwt.MapClaims{"groups": []interface{}{"ROLE_ADMIN"}},
			whiteListRolesAdmin: map[string]struct{}{"ROLE_ADMIN": {}},
			want:                iamtoken.AdminRole,
			wantIsGroupsInToken: true,
		},
		{
			claims:              jwt.MapClaims{"groups": []interface{}{"ROLE_USER"}},
			whiteListRolesUser:  map[string]struct{}{"ROLE_USER": {}},
			want:                iamtoken.UserRole,
			wantIsGroupsInToken: true,
		},
		{
			claims:              jwt.MapClaims{"groups": []interface{}{}},
			want:                iamtoken.UserRole,
			wantIsGroupsInToken: false,
		},
		{
			claims:              jwt.MapClaims{},
			want:                iamtoken.UserRole,
			wantIsGroupsInToken: false,
		},
		{
			claims:              jwt.MapClaims{"groups": []interface{}{"ROLE_UNKNOWN"}},
			want:                iamtoken.UserRole,
			wantIsGroupsInToken: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := IAMJWTTokenParser{
				WhiteListRoles: WhiteListRoles{
					User:  tt.whiteListRolesUser,
					Admin: tt.whiteListRolesAdmin,
				},
			}

			got, isGroupsInToken, err := parser.getRole(tt.claims)

			require.NoError(t, err)
			require.Equal(t, tt.wantIsGroupsInToken, isGroupsInToken)
			require.Equal(t, got, tt.want)
		})
	}
}

func TestIAMJWTTokenParser_getRole_Negative(t *testing.T) {
	parser := IAMJWTTokenParser{
		WhiteListRoles: WhiteListRoles{
			Admin: map[string]struct{}{"ROLE_ADMIN": {}},
		},
	}

	got, isGroupsInToken, err := parser.getRole(jwt.MapClaims{"groups": []interface{}{"ROLE_UNKNOWN"}})

	require.Error(t, err)
	require.False(t, isGroupsInToken)
	require.Empty(t, got)
}
