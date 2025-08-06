package auth

import (
	"testing"

	"github.com/stretchr/testify/require"

	"code.gitea.io/gitea/modules/setting"
)

func TestShouldUseWsPrivileges(t *testing.T) {
	tests := []struct {
		name                string
		gitProtocolHeader   string
		wsPrivilegesEnabled bool
		want                bool
	}{
		{
			name:                "Empty Git Protocol Header, WsPrivileges Enabled",
			gitProtocolHeader:   "",
			wsPrivilegesEnabled: true,
			want:                true,
		},
		{
			name:                "Non-empty Git Protocol Header, WsPrivileges Enabled",
			gitProtocolHeader:   "some-header",
			wsPrivilegesEnabled: true,
			want:                false,
		},
		{
			name:                "Empty Git Protocol Header, WsPrivileges Disabled",
			gitProtocolHeader:   "",
			wsPrivilegesEnabled: false,
			want:                false,
		},
		{
			name:                "Non-empty Git Protocol Header, WsPrivileges Disabled",
			gitProtocolHeader:   "some-header",
			wsPrivilegesEnabled: false,
			want:                false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setting.IAM.WsPrivilegesEnabled = tt.wsPrivilegesEnabled

			got := shouldUseWsPrivileges(tt.gitProtocolHeader)

			require.Equal(t, tt.want, got)
		})
	}
}
