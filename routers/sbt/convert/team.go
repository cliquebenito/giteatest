package convert

import (
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/organization"
	"code.gitea.io/gitea/routers/sbt/response"
	"context"
)

// ToTeams конвертирует список models.Team в список response.Team
func ToTeams(ctx context.Context, teams []*organization.Team, loadOrgs bool) ([]*response.Team, error) {
	if len(teams) == 0 || teams[0] == nil {
		return nil, nil
	}

	cache := make(map[int64]*response.Organization)
	apiTeams := make([]*response.Team, len(teams))
	for i := range teams {
		if err := teams[i].LoadUnits(ctx); err != nil {
			return nil, err
		}

		apiTeams[i] = &response.Team{
			ID:                      teams[i].ID,
			Name:                    teams[i].Name,
			Description:             teams[i].Description,
			IncludesAllRepositories: teams[i].IncludesAllRepositories,
			CanCreateOrgRepo:        teams[i].CanCreateOrgRepo,
			Permission:              teams[i].AccessMode.String(),
			UnitsMap:                teams[i].GetUnitsMap(),
		}

		if loadOrgs {
			apiOrg, ok := cache[teams[i].OrgID]
			if !ok {
				org, err := organization.GetOrgByID(db.DefaultContext, teams[i].OrgID)
				if err != nil {
					return nil, err
				}
				apiOrg = ToOrganization(ctx, org)
				cache[teams[i].OrgID] = apiOrg
			}
			apiTeams[i].Organization = apiOrg
		}
	}
	return apiTeams, nil
}
