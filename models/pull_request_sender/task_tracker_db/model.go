package task_tracker_db

import (
	"code.gitea.io/gitea/models/pull_request_sender"
	"code.gitea.io/gitea/models/unit_links"
)

type payloadStatusForUpdating struct {
	FromUnitID     int64                                `json:"from_unit_id"`
	FromUnitType   unit_links.FromUnitType              `json:"from_unit_type"`
	FromUnitStatus pull_request_sender.FromUnitStatusPr `json:"from_unit_status"`
}
