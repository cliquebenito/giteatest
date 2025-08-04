package unit_links_db

import (
	goCtx "context"
	"fmt"

	"xorm.io/builder"

	"code.gitea.io/gitea/models/issues"
	"code.gitea.io/gitea/models/unit_links"
)

// GetUnitLinks получить список линк
func (u unitLinkDB) GetUnitLinks(_ goCtx.Context, pullRequestID int64) (unit_links.AllUnitLinks, error) {
	pullRequests := issues.PullRequest{}

	count, err := u.engine.Where(builder.Eq{"id": pullRequestID}).Count(&pullRequests)
	if err != nil {
		return nil, fmt.Errorf("find pull request links: %w", err)
	}

	if count == 0 {
		return unit_links.AllUnitLinks{}, NewPullRequestNotFoundError(pullRequestID)
	}

	unitLinks := make([]unit_links.UnitLinks, 0)

	if err = u.engine.
		Where(builder.Eq{"is_active": 1}, builder.Eq{"from_unit_id": pullRequestID}).
		Table("unit_links").
		Find(&unitLinks); err != nil {
		return nil, fmt.Errorf("find unit links: %w", err)
	}

	return unitLinks, nil
}
