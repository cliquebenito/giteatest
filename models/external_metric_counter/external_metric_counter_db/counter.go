package external_metric_counter_db

import (
	"database/sql"

	"xorm.io/xorm"
)

type dbEngine interface {
	Where(interface{}, ...interface{}) *xorm.Session
	OrderBy(order interface{}, args ...interface{}) *xorm.Session
	Delete(beans ...interface{}) (int64, error)
	SQL(interface{}, ...interface{}) *xorm.Session
	Exec(...interface{}) (sql.Result, error)
	In(string, ...interface{}) *xorm.Session
}

type externalMetricCounterDB struct {
	engine dbEngine
}

func New(engine dbEngine) externalMetricCounterDB {
	return externalMetricCounterDB{engine: engine}
}
