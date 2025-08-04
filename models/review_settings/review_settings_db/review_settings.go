package review_settings_db

import (
	"database/sql"

	"xorm.io/xorm"
)

type dbEngine interface {
	Where(interface{}, ...interface{}) *xorm.Session
	Exec(...interface{}) (sql.Result, error)
}

type reviewSettingsDB struct {
	engine dbEngine
}

func New(engine dbEngine) reviewSettingsDB {
	return reviewSettingsDB{engine: engine}
}
