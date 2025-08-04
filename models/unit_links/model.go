package unit_links

import (
	"code.gitea.io/gitea/models/pull_request_sender"
	"fmt"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/modules/timeutil"
)

func init() {
	db.RegisterModel(new(UnitLinks))
}

type FromUnitType int

const (
	PullRequestFromUnitType FromUnitType = iota + 1
)

type AllUnitLinks []UnitLinks

// UnitLinks структура полей для таблицы unit_links
type UnitLinks struct {
	ID           int64              `xorm:"PK AUTOINCR" json:"-"`
	FromUnitID   int64              `xorm:"NOT NULL UNIQUE(s)" json:"from_unit_id"`
	FromUnitType FromUnitType       `xorm:"NOT NULL UNIQUE(s)" json:"from_unit_type"`
	ToUnitID     string             `xorm:"NOT NULL UNIQUE(s)" json:"to_unit_id"`
	IsActive     bool               `xorm:"NOT NULL DEFAULT true" json:"is_active"`
	CreatedAt    timeutil.TimeStamp `xorm:"CREATED" json:"-"`
	UpdatedAt    timeutil.TimeStamp `xorm:"UPDATED" json:"-"`
}

type Diff struct {
	LinksToAdd    AllUnitLinks `json:"links_to_add"`
	LinksToDelete AllUnitLinks `json:"links_to_delete"`
}

func (d Diff) IsEmpty() bool {
	return len(d.LinksToAdd) == 0 && len(d.LinksToDelete) == 0
}

func (a AllUnitLinks) GetFromUnitID() (int64, error) {
	uniqIDs := make(map[int64]struct{})

	for _, link := range a {
		uniqIDs[link.FromUnitID] = struct{}{}
	}

	if len(uniqIDs) == 0 {
		return 0, fmt.Errorf("no unit links found")
	}

	if len(uniqIDs) > 1 {
		return 0, fmt.Errorf("too many unit links found: %v", uniqIDs)
	}

	var fromUnitID int64
	for id := range uniqIDs {
		fromUnitID = id
		break
	}

	return fromUnitID, nil
}

func (a AllUnitLinks) IsEmpty() bool {
	return len(a) == 0
}

// AllPayloadToAddOrDeletePr слайс PayloadToAddOrDeletePr
type AllPayloadToAddOrDeletePr []PayloadToAddOrDeletePr

// PayloadToAddOrDeletePr payload to add or delete pull request for task tracker tasks
type PayloadToAddOrDeletePr struct {
	FromUnitID     int64                                `json:"from_unit_id"`
	FromUnitType   FromUnitType                         `json:"from_unit_type"`
	ToUnitID       string                               `json:"to_unit_id"`
	IsActive       bool                                 `json:"is_active"`
	FromUnitStatus pull_request_sender.FromUnitStatusPr `json:"from_unit_status,omitempty"`
}
