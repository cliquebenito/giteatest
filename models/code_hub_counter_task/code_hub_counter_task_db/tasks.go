package code_hub_counter_task_db

import (
	"xorm.io/xorm"
)

type dbEngine interface {
	Where(interface{}, ...interface{}) *xorm.Session
	Insert(beans ...interface{}) (int64, error)
	Delete(beans ...interface{}) (int64, error)
}

type codeHubCounterTasksDB struct {
	engine dbEngine
}

func New(db dbEngine) codeHubCounterTasksDB {
	return codeHubCounterTasksDB{engine: db}
}
