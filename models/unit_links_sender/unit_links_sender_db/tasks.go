package unit_links_sender_db

import (
	"database/sql"
	"xorm.io/xorm"
)

// //go:generate mockery --name=dbEngine --exported
type dbEngine interface {
	Exec(...interface{}) (sql.Result, error)
	Find(interface{}, ...interface{}) error
	Where(interface{}, ...interface{}) *xorm.Session
}

type unitLinksSenderDB struct {
	engine dbEngine
}

func New(db dbEngine) unitLinksSenderDB {
	return unitLinksSenderDB{engine: db}
}
