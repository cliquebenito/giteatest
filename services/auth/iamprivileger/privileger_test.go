package iamprivileger

import (
	"testing"

	"code.gitea.io/gitea/models/role_model"
	"github.com/stretchr/testify/require"

	"code.gitea.io/gitea/models/organization"
	iampriveleges "code.gitea.io/gitea/modules/auth/iam/iamprivileges"
)

func TestPrivileger_getOrganizationToPrivileges(t *testing.T) {
	tests := []struct {
		name          string
		organizations []*organization.Organization
		privileges    iampriveleges.Privileges
		want          map[organization.Organization]iampriveleges.Privilege
	}{
		{
			organizations: []*organization.Organization{},
			privileges:    iampriveleges.Privileges{},
			want:          map[organization.Organization]iampriveleges.Privilege{},
		},
		{
			organizations: []*organization.Organization{
				{ID: 1, Name: "org1", LowerName: "org1"},
			},
			privileges: iampriveleges.Privileges{
				{TenantName: "tenant", ToolName: "sc", ProjectName: "org1", Role: role_model.OWNER},
			},
			want: map[organization.Organization]iampriveleges.Privilege{
				{ID: 1, LowerName: "org1", Name: "org1"}: {TenantName: "tenant", ToolName: "sc", ProjectName: "org1", Role: role_model.OWNER},
			},
		},
		{
			organizations: []*organization.Organization{
				{ID: 1, Name: "org1", LowerName: "org1"},
			},
			privileges: iampriveleges.Privileges{
				{TenantName: "tenant", ToolName: "sc", ProjectName: "org1", Role: role_model.READER},
				{TenantName: "tenant", ToolName: "sc", ProjectName: "org1", Role: role_model.OWNER},
			},
			want: map[organization.Organization]iampriveleges.Privilege{
				{ID: 1, LowerName: "org1", Name: "org1"}: {TenantName: "tenant", ToolName: "sc", ProjectName: "org1", Role: role_model.OWNER},
			},
		},
		{
			organizations: []*organization.Organization{
				{ID: 1, Name: "org1", LowerName: "org1"},
				{ID: 2, Name: "org2", LowerName: "org2"},
			},
			privileges: iampriveleges.Privileges{
				{TenantName: "tenant", ToolName: "sc", ProjectName: "org1", Role: role_model.READER},
				{TenantName: "tenant", ToolName: "sc", ProjectName: "org2", Role: role_model.OWNER},
			},
			want: map[organization.Organization]iampriveleges.Privilege{
				{ID: 1, LowerName: "org1", Name: "org1"}: {TenantName: "tenant", ToolName: "sc", ProjectName: "org1", Role: role_model.READER},
				{ID: 2, LowerName: "org2", Name: "org2"}: {TenantName: "tenant", ToolName: "sc", ProjectName: "org2", Role: role_model.OWNER},
			},
		},
		{
			organizations: []*organization.Organization{
				{ID: 1, Name: "org1", LowerName: "org1"},
				{ID: 2, Name: "org2", LowerName: "org2"},
			},
			privileges: iampriveleges.Privileges{
				{TenantName: "tenant", ToolName: "sc", ProjectName: "org1", Role: role_model.READER},
			},
			want: map[organization.Organization]iampriveleges.Privilege{
				{ID: 1, LowerName: "org1", Name: "org1"}: {TenantName: "tenant", ToolName: "sc", ProjectName: "org1", Role: role_model.READER},
			},
		},
	}

	p := Privileger{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := p.getOrganizationToPrivileges(tt.organizations, tt.privileges)

			require.Equal(t, tt.want, got)
		})
	}
}
