package internal_metric_counter_db

import (
	"xorm.io/xorm"
)

type dbEngine interface {
	Where(interface{}, ...interface{}) *xorm.Session
	OrderBy(order interface{}, args ...interface{}) *xorm.Session
	Delete(beans ...interface{}) (int64, error)
	SQL(interface{}, ...interface{}) *xorm.Session
	In(string, ...interface{}) *xorm.Session
}

type codeHubCounterDB struct {
	engine dbEngine
}

func New(engine dbEngine) codeHubCounterDB {
	return codeHubCounterDB{engine: engine}
}
