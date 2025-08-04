//go:build !correct

package iampriveleges

import (
	"testing"

	"github.com/stretchr/testify/require"

	"code.gitea.io/gitea/models/role_model"
)

func TestOpenFromString(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    SourceControlPrivilegesByTenant
	}{
		{
			content: `[{"organization":"rum","rolesMapping":{"project_coordinator":["rum_sc_skey_x"]}}]`,
			want:    SourceControlPrivilegesByTenant{"rum": Privileges{Privilege{TenantName: "rum", ToolName: "sc", ProjectName: "skey", Role: 2}}},
		},
		{
			content: `[{"organization":"rum","rolesMapping":{"devops":["rum_sc_skey_a"]}}]`,
			want:    SourceControlPrivilegesByTenant{"rum": Privileges{Privilege{TenantName: "rum", ToolName: "sc", ProjectName: "skey", Role: 1}}},
		},
		{
			content: `[{"organization":"rum","rolesMapping":{"some_other_role":["rum_sc_skey_a"]}}]`,
			want:    SourceControlPrivilegesByTenant{"rum": Privileges{Privilege{TenantName: "rum", ToolName: "sc", ProjectName: "skey", Role: 1}}},
		},
		{
			content: `[{"organization":"rum","rolesMapping":{"some_other_role":["rum_wrongtool_skey_w"]}}]`,
			want:    SourceControlPrivilegesByTenant{"rum": Privileges(nil)},
		},
		{
			content: `[{"organization":"rum","rolesMapping":{"some_other_role":[]}}]`,
			want:    SourceControlPrivilegesByTenant{"rum": Privileges(nil)},
		},

		{
			content: `[{"organization":"rum","rolesMapping":{"some_other_role":["rum_wrongtool_skey_w"],"devops":["sbt_tool_skey_w"]}}]`,
			want:    SourceControlPrivilegesByTenant{"rum": Privileges(nil)},
		},
		{
			content: `[{"organization":"rum","rolesMapping":{"some_other_role":["rum_sc_skey_w"],"devops":["sbt_tool_skey_w"]}}]`,
			want:    SourceControlPrivilegesByTenant{"rum": Privileges{Privilege{TenantName: "rum", ToolName: "sc", ProjectName: "skey", Role: role_model.WRITER}}},
		},
		{
			content: `[{"organization":"rum","rolesMapping":{"some_other_role":["rum_sc_skey_w"],"devops":["sbt_tool_skey_r"],"project_coordinator":["sbt_sc_project_a"]}}]`,
			want:    SourceControlPrivilegesByTenant{"rum": Privileges{Privilege{TenantName: "rum", ToolName: "sc", ProjectName: "skey", Role: role_model.WRITER}, Privilege{TenantName: "sbt", ToolName: "sc", ProjectName: "project", Role: role_model.OWNER}}},
		},

		{
			content: `[{"organization":"rum","rolesMapping":{}}]`,
			want:    SourceControlPrivilegesByTenant{},
		},
		{
			content: `[{"organization":"rum"}]`,
			want:    SourceControlPrivilegesByTenant{},
		},
		{
			content: `[]`,
			want:    SourceControlPrivilegesByTenant{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := OpenFromString(tt.content)
			require.NoError(t, err)

			for k, gotSlice := range got {
				wantSlice, exists := tt.want[k]
				if !exists {
					t.Errorf("want to exist in got: %v", k)
				}

				require.ElementsMatch(t, wantSlice, gotSlice)
			}
		})
	}
}
