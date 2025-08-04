package task_tracker_db

import (
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/issues"
	"code.gitea.io/gitea/models/unit_links"
	"code.gitea.io/gitea/modules/log"
	"context"
	"fmt"
	"xorm.io/builder"
)

func (t taskTrackerDB) IsActiveOfPullRequestStatus(ctx context.Context, pullRequestID int64) (bool, error) {
	_, committer, err := db.TxContext(ctx)
	if err != nil {
		return false, err
	}
	defer committer.Close()
	var pullRequest issues.PullRequest
	has, err := t.engine.Where(builder.Eq{"id": pullRequestID}).Get(&pullRequest)
	if err != nil {
		log.Error("Error has occurred while fetching pull request: %v", err)
		return false, fmt.Errorf("getting pull request by id: %w", err)
	}
	if !has {
		log.Error("Pull Request is not found")
		return false, nil
	}

	var unitLinks []unit_links.UnitLinks

	if err = t.engine.
		Where(builder.Eq{"is_active": 1}, builder.Eq{"from_unit_id": pullRequestID}).
		Table("unit_links").
		Find(&unitLinks); err != nil {
		return false, fmt.Errorf("find unit links: %w", err)
	}
	return len(unitLinks) > 0, committer.Commit()
}
