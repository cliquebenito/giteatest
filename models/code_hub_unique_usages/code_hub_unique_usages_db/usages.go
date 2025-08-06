package code_hub_unique_usages_db

import (
	"xorm.io/xorm"
)

type dbEngine interface {
	Where(interface{}, ...interface{}) *xorm.Session
	Delete(beans ...interface{}) (int64, error)
	Insert(beans ...interface{}) (int64, error)
}

type codeHubUniqueUsagesDB struct {
	engine dbEngine
}

func New(engine dbEngine) codeHubUniqueUsagesDB {
	return codeHubUniqueUsagesDB{engine: engine}
}
