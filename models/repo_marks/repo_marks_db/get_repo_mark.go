package repo_marks_db

import (
	"context"
	"fmt"

	"code.gitea.io/gitea/models/repo_marks"
	"code.gitea.io/gitea/modules/log"
)

// GetRepoMarks получение отметки codehub по repoID
func (m repoMarksDB) GetRepoMarks(ctx context.Context, repoID int64) ([]repo_marks.RepoMarks, error) {
	var marks []repo_marks.RepoMarks

	// тут нужен голый sql, так как ОРМка объединяет несколько запросов с помощью AND и ответ получается пустой
	sqlQuery := "SELECT * FROM repo_marks WHERE repo_id = ?"
	if err := m.engine.SQL(sqlQuery, repoID).Find(&marks); err != nil {
		log.Error("Error has occurred while getting repo marks: %v", err)
		return nil, fmt.Errorf("get repo marks: %w", err)
	}

	return marks, nil
}

// GetRepoMarksByRepoIDs получить отметки по слайсу id
func (m repoMarksDB) GetRepoMarksByRepoIDs(_ context.Context, repoIDs []int64) ([]*repo_marks.RepoMarks, error) {
	if len(repoIDs) == 0 {
		return []*repo_marks.RepoMarks{}, nil
	}

	marks := make([]*repo_marks.RepoMarks, 0)
	err := m.engine.In("repo_id", repoIDs).Find(&marks)
	if err != nil {
		return nil, fmt.Errorf("find repo marks: %w", err)
	}
	return marks, nil
}
