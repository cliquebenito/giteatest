package default_reviewers_db

import (
	"xorm.io/xorm"
)

type dbEngine interface {
	Where(interface{}, ...interface{}) *xorm.Session
	Delete(beans ...interface{}) (int64, error)
	Insert(beans ...interface{}) (int64, error)
}

type defaultReviewersDB struct {
	engine dbEngine
}

func New(engine dbEngine) defaultReviewersDB {
	return defaultReviewersDB{engine: engine}
}
