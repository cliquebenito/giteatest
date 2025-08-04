package task_tracker_db

import (
	"database/sql"
	"xorm.io/xorm"
)

// //go:generate mockery --name=dbEngine --exported
type dbEngine interface {
	Exec(...interface{}) (sql.Result, error)
	Find(interface{}, ...interface{}) error
	Where(interface{}, ...interface{}) *xorm.Session
	Insert(beans ...interface{}) (int64, error)
}

// taskTrackerDB .
type taskTrackerDB struct {
	engine dbEngine
}

// NewTaskTrackerDB создаем taskTrackerDB
func NewTaskTrackerDB(engine dbEngine) taskTrackerDB {
	return taskTrackerDB{engine: engine}
}
