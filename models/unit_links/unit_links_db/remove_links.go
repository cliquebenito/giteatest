package unit_links_db

import (
	"context"
	"fmt"
	"time"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/unit_links"
	"code.gitea.io/gitea/models/unit_links_sender"
)

// RemoveLinks метод удаляет список линков
func (u unitLinkDB) RemoveLinks(
	ctx context.Context,
	fromUnitID int64,
	links unit_links.AllUnitLinks,
	userName, pullRequestURL string,
) error {
	removeLinks := func(ctx context.Context) error {
		var taskTrackerPayloads unit_links.AllPayloadToAddOrDeletePr
		for idx := range links {
			links[idx].IsActive = false
			if err := u.removeLinks(ctx, links[idx]); err != nil {
				return fmt.Errorf("remove inactive unit link: %w", err)
			}
			payloadToAddOrDeletePr := unit_links.PayloadToAddOrDeletePr{
				FromUnitID:   links[idx].FromUnitID,
				FromUnitType: links[idx].FromUnitType,
				ToUnitID:     links[idx].ToUnitID,
				IsActive:     links[idx].IsActive,
			}
			taskTrackerPayloads = append(taskTrackerPayloads, payloadToAddOrDeletePr)
		}

		if err := u.insertTasks(
			ctx,
			unit_links_sender.SendDeletePullRequestLinksAction,
			taskTrackerPayloads,
			fromUnitID,
			pullRequestURL,
			userName,
		); err != nil {
			return fmt.Errorf("insert add tasks: %w", err)
		}

		return nil
	}

	if err := db.WithTx(ctx, removeLinks); err != nil {
		return fmt.Errorf("remove unit links: %w", err)
	}

	return nil
}

func (u unitLinkDB) removeLinks(_ context.Context, link unit_links.UnitLinks) error {
	removeQuery := `
UPDATE unit_links
SET is_active = ?, updated_at = ?
WHERE from_unit_id = ? AND from_unit_type = ? AND to_unit_id = ?
`
	timeNow := time.Now().Unix()

	_, err := u.engine.Exec(
		removeQuery,
		link.IsActive,
		timeNow,
		link.FromUnitID,
		link.FromUnitType,
		link.ToUnitID,
	)

	if err != nil {
		return fmt.Errorf("remove from unit links: %w", err)
	}

	return nil
}
