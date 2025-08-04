package repo_marks_db

import (
	"xorm.io/xorm"
)

type dbEngine interface {
	Where(interface{}, ...interface{}) *xorm.Session
	Insert(beans ...interface{}) (int64, error)
	Exist(...interface{}) (bool, error)
	Delete(...interface{}) (int64, error)
	SQL(interface{}, ...interface{}) *xorm.Session
	In(string, ...interface{}) *xorm.Session
}

type repoMarksDB struct {
	engine dbEngine
}

func NewRepoMarksDB(db dbEngine) repoMarksDB {
	return repoMarksDB{engine: db}
}
