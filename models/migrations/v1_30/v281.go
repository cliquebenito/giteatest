package v1_30

import (
	"xorm.io/xorm"

	"code.gitea.io/gitea/models/repo_marks"
)

// CreateRepoMarksTable создание таблицы repo_marks
func CreateRepoMarksTable(x *xorm.Engine) error {
	return x.Sync(new(repo_marks.RepoMarks))
}
