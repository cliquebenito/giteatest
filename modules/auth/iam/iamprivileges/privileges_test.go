//go:build !correct

package iampriveleges

import (
	"testing"

	"github.com/stretchr/testify/require"

	"code.gitea.io/gitea/models/role_model"
)

func Test_parsePrivilege(t *testing.T) {
	tests := []struct {
		name         string
		rawPrivilege string
		want         Privilege
	}{
		{rawPrivilege: "tenant_sc_project_r", want: Privilege{TenantName: "tenant", ToolName: "sc", ProjectName: "project", Role: role_model.READER}},
		{rawPrivilege: "tenant_sc_project_w", want: Privilege{TenantName: "tenant", ToolName: "sc", ProjectName: "project", Role: role_model.WRITER}},
		{rawPrivilege: "tenant_sc_project_x", want: Privilege{TenantName: "tenant", ToolName: "sc", ProjectName: "project", Role: role_model.MANAGER}},
		{rawPrivilege: "tenant_sc_project_a", want: Privilege{TenantName: "tenant", ToolName: "sc", ProjectName: "project", Role: role_model.OWNER}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parsePrivilege(tt.rawPrivilege)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func Test_parsePrivilege_negative(t *testing.T) {
	tests := []struct {
		name         string
		rawPrivilege string
	}{
		{rawPrivilege: "tenant_sc_project_rw"},
		{rawPrivilege: ""},
		{rawPrivilege: "tenant_snakecase_name_sc_project_x"},
		{rawPrivilege: "tenant_sc_snakecase_name_project_a"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parsePrivilege(tt.rawPrivilege)
			require.Error(t, err)
		})
	}
}

func Test_parsePrivileges(t *testing.T) {
	tests := []struct {
		name         string
		rawPrivilege []string
		want         []Privilege
	}{
		{
			rawPrivilege: []string{
				"tenant_sc_asdf_r",
				"tenant_sc_asdf_w",
				"tenant_sc_asdf_x",
				"tenant_sc_asdf_x",
				"tenant_sc_asdf_x",
				"tenant_sc_asdf_x",
			},
			want: Privileges{
				Privilege{TenantName: "tenant", ToolName: "sc", ProjectName: "asdf", Role: role_model.READER},
				Privilege{TenantName: "tenant", ToolName: "sc", ProjectName: "asdf", Role: role_model.WRITER},
				Privilege{TenantName: "tenant", ToolName: "sc", ProjectName: "asdf", Role: role_model.MANAGER},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parsePrivileges(tt.rawPrivilege)
			require.NoError(t, err)
			require.ElementsMatch(t, tt.want, got)
		})
	}
}

func Test_parsePrivileges_negative(t *testing.T) {
	tests := []struct {
		name         string
		rawPrivilege []string
	}{
		{rawPrivilege: []string{"tenant_sc_asdf_r", "tenant_sc_asdf_w", "tenant_sc_asdf_qqqq"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parsePrivileges(tt.rawPrivilege)
			require.Error(t, err)
		})
	}
}
