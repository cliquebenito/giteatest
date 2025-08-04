package convert

import (
	"code.gitea.io/gitea/models/organization"
	"code.gitea.io/gitea/routers/sbt/response"
	"context"
)

// ToOrganization конвертирует organization.Organization в response.Organization
func ToOrganization(ctx context.Context, org *organization.Organization) *response.Organization {
	return &response.Organization{
		ID:                        org.ID,
		AvatarURL:                 org.AsUser().AvatarLink(ctx),
		Name:                      org.Name,
		FullName:                  org.FullName,
		Description:               org.Description,
		Website:                   org.Website,
		Location:                  org.Location,
		Visibility:                org.Visibility.String(),
		RepoAdminChangeTeamAccess: org.RepoAdminChangeTeamAccess,
	}
}

// ToOrganizationSettings конвертирует organization.Organization в response.OrganizationSettings
func ToOrganizationSettings(org *organization.Organization) *response.OrganizationSettings {
	return &response.OrganizationSettings{
		Name:                      org.Name,
		Description:               org.Description,
		FullName:                  org.FullName,
		RepoAdminChangeTeamAccess: org.RepoAdminChangeTeamAccess,
		Location:                  org.Location,
		Visibility:                org.Visibility.String(),
		Website:                   org.Website,
		MaxRepoCreation:           org.MaxRepoCreation,
	}
}
