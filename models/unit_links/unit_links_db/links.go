package unit_links_db

import (
	"context"
	"database/sql"
	"fmt"

	"xorm.io/builder"
	"xorm.io/xorm"

	"code.gitea.io/gitea/models/unit_links"
	"code.gitea.io/gitea/modules/log"
)

// //go:generate mockery --name=dbEngine --exported
type dbEngine interface {
	Exec(...interface{}) (sql.Result, error)
	Find(interface{}, ...interface{}) error
	Where(interface{}, ...interface{}) *xorm.Session
}

type unitLinkDB struct {
	engine dbEngine
}

func NewUnitLinkDB(engine dbEngine) unitLinkDB {
	return unitLinkDB{engine: engine}
}

func (u unitLinkDB) calculateLinksDiff(_ context.Context, links unit_links.AllUnitLinks, fromUnitID int64) (unit_links.Diff, error) {
	oldLinks := make([]unit_links.UnitLinks, 0)

	if err := u.engine.Where(builder.Eq{"is_active": 1}, builder.Eq{"from_unit_id": fromUnitID}).
		Table("unit_links").
		Find(&oldLinks); err != nil {
		return unit_links.Diff{}, fmt.Errorf("delete unit links: %w", err)
	}

	linksDiff, err := unit_links.CalculateDiff(oldLinks, links)
	if err != nil {
		return unit_links.Diff{}, fmt.Errorf("calculate links: %w", err)
	}

	jsonDiff, err := linksDiff.JSON()
	if err != nil {
		log.Debug("Try to calculate links diff: %v", err)
	}

	log.Debug("Calculated links diff: %s", jsonDiff)

	return linksDiff, nil
}
