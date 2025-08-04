//go:build !correct

package iampriveleges

import (
	"testing"

	"github.com/stretchr/testify/require"

	"code.gitea.io/gitea/models/role_model"
)

func TestPrivileges_GetMaxPrivilege(t *testing.T) {
	tests := []struct {
		name string
		p    Privileges
		want Privilege
	}{
		{
			p:    Privileges{{ToolName: "1", Role: role_model.MANAGER}, {ToolName: "2", Role: role_model.READER}, {ToolName: "3", Role: role_model.WRITER}, {ToolName: "4", Role: role_model.OWNER}},
			want: Privilege{ToolName: "4", Role: role_model.OWNER},
		},
		{
			p:    Privileges{{ToolName: "1", Role: role_model.MANAGER}},
			want: Privilege{ToolName: "1", Role: role_model.MANAGER},
		},
		{
			p:    Privileges{{ToolName: "sc", Role: role_model.READER}, {ToolName: "tracker", Role: role_model.MANAGER}},
			want: Privilege{ToolName: "tracker", Role: role_model.MANAGER},
		},
		{
			p:    Privileges{{ToolName: "sc", Role: role_model.MANAGER}},
			want: Privilege{ToolName: "sc", Role: role_model.MANAGER},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.p.GetMaxPrivilege()
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
