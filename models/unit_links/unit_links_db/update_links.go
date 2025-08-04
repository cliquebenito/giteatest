package unit_links_db

import (
	"code.gitea.io/gitea/models/issues"
	"code.gitea.io/gitea/modules/log"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/pull_request_sender"
	"code.gitea.io/gitea/models/unit_links"
	"code.gitea.io/gitea/models/unit_links_sender"
)

// UpdateLinks метод обновляет список линков
func (u unitLinkDB) UpdateLinks(
	ctx context.Context,
	fromUnitID int64,
	links unit_links.AllUnitLinks,
	userName, pullRequestURL string,
) error {
	createLinks := func(ctx context.Context) error {
		diff, diffErr := u.calculateLinksDiff(ctx, links, fromUnitID)
		if diffErr != nil {
			return fmt.Errorf("calculate links diff: %w", diffErr)
		}

		if diff.IsEmpty() {
			return nil
		}
		var taskTrackerPayloadsAdd unit_links.AllPayloadToAddOrDeletePr
		pullRequest, err := issues.GetPullRequestByID(ctx, fromUnitID)
		if err != nil {
			log.Error("Error has occurred while getting pull request by id: %d: %v", fromUnitID, err)
			return fmt.Errorf("get pull request by id: %w", err)
		}
		issue, err := issues.GetIssueByID(ctx, pullRequest.IssueID)
		if err != nil {
			log.Error("Error has occurred while getting issue by id: %d: %v", issue.ID, err)
			return fmt.Errorf("get issue by id: %w", err)
		}
		fromUnitStatus := pull_request_sender.PRStatusOpen
		if pullRequest.HasMerged && issue.IsClosed {
			fromUnitStatus = pull_request_sender.PRStatusMerged
		} else if issue.IsClosed {
			fromUnitStatus = pull_request_sender.PRStatusClosed
		}
		for idx := range diff.LinksToAdd {
			diff.LinksToAdd[idx].IsActive = true
			if err := u.upsertLinks(ctx, diff.LinksToAdd[idx]); err != nil {
				return fmt.Errorf("upsert active unit link: %w", err)
			}
			payloadToAddOrDeletePr := unit_links.PayloadToAddOrDeletePr{
				FromUnitID:     diff.LinksToAdd[idx].FromUnitID,
				FromUnitType:   diff.LinksToAdd[idx].FromUnitType,
				ToUnitID:       diff.LinksToAdd[idx].ToUnitID,
				IsActive:       diff.LinksToAdd[idx].IsActive,
				FromUnitStatus: fromUnitStatus,
			}
			taskTrackerPayloadsAdd = append(taskTrackerPayloadsAdd, payloadToAddOrDeletePr)
		}

		if err := u.insertTasks(
			ctx,
			unit_links_sender.SendAddPullRequestLinksAction,
			taskTrackerPayloadsAdd,
			fromUnitID,
			pullRequestURL,
			userName,
		); err != nil {
			return fmt.Errorf("insert add tasks: %w", err)
		}

		var taskTrackerPayloadsDel unit_links.AllPayloadToAddOrDeletePr
		for idx := range diff.LinksToDelete {
			diff.LinksToDelete[idx].IsActive = false

			if err := u.upsertLinks(ctx, diff.LinksToDelete[idx]); err != nil {
				return fmt.Errorf("upsert inactive unit link: %w", err)
			}
			payloadToDeletePr := unit_links.PayloadToAddOrDeletePr{
				FromUnitID:   diff.LinksToDelete[idx].FromUnitID,
				FromUnitType: diff.LinksToDelete[idx].FromUnitType,
				ToUnitID:     diff.LinksToDelete[idx].ToUnitID,
				IsActive:     diff.LinksToDelete[idx].IsActive,
			}
			taskTrackerPayloadsDel = append(taskTrackerPayloadsDel, payloadToDeletePr)
		}

		if err := u.insertTasks(
			ctx,
			unit_links_sender.SendDeletePullRequestLinksAction,
			taskTrackerPayloadsDel,
			fromUnitID,
			pullRequestURL,
			userName,
		); err != nil {
			return fmt.Errorf("insert delete tasks: %w", err)
		}

		return nil
	}

	if err := db.WithTx(ctx, createLinks); err != nil {
		return fmt.Errorf("create unit links: %w", err)
	}

	return nil
}

func (u unitLinkDB) upsertLinks(_ context.Context, link unit_links.UnitLinks) error {
	upsertQuery := `
INSERT INTO unit_links
	(from_unit_id, from_unit_type, to_unit_id, is_active, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT
	(from_unit_id, from_unit_type, to_unit_id)
DO UPDATE SET
	is_active = EXCLUDED.is_active,
	created_at = EXCLUDED.created_at
`
	timeNow := time.Now().Unix()

	_, err := u.engine.Exec(
		upsertQuery,
		link.FromUnitID,
		link.FromUnitType,
		link.ToUnitID,
		link.IsActive,
		timeNow,
		timeNow,
	)

	if err != nil {
		return fmt.Errorf("upsert from unit links: %w", err)
	}

	return nil
}

func (u unitLinkDB) insertTasks(
	_ context.Context,
	action unit_links_sender.Action,
	links unit_links.AllPayloadToAddOrDeletePr,
	pullRequestID int64,
	userName, pullRequestURL string,
) error {
	if links == nil {
		return nil
	}

	insertQuery := `
INSERT INTO unit_links_sender_tasks
	(payload, action, pull_request_id, pull_request_url, user_name, status, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
`
	timeNow := time.Now().Unix()

	payload, err := json.Marshal(links)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	_, err = u.engine.Exec(
		insertQuery,
		string(payload),
		action,
		pullRequestID,
		userName,
		pullRequestURL,
		unit_links_sender.StatusUnlocked,
		timeNow,
		timeNow,
	)

	if err != nil {
		return fmt.Errorf("insert from unit links: %w", err)
	}

	return nil
}
