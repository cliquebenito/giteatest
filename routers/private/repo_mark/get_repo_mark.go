package repo_mark

import (
	"context"
	"fmt"

	"code.gitea.io/gitea/models/repo_marks"
	"code.gitea.io/gitea/modules/log"
)

// //go:generate mockery --name=repoMarksDB --exported
type repoMarksDB interface {
	GetRepoMarks(ctx context.Context, repoID int64) ([]repo_marks.RepoMarks, error)
}

type RepoMarksGetter struct {
	repoMarksDB
	processedMarks []repo_marks.RepoMark
}

// NewRepoMarksGetter processedMarks - слайс всех обрабатываемых сервисом видов отметок репозитория
func NewRepoMarksGetter(repoMarksDB repoMarksDB, processedMarks []repo_marks.RepoMark) RepoMarksGetter {
	return RepoMarksGetter{repoMarksDB: repoMarksDB, processedMarks: processedMarks}
}

// GetRepoMarks метод для получения отметок репозитория, в ответе прилетает обогащенная модель с актуальными label'ами отметок
func (g RepoMarksGetter) GetRepoMarks(ctx context.Context, repoID int64) (GetRepoMarksResponse, error) {
	var response []Mark
	marksDef := make(map[string]string)

	for _, mark := range g.processedMarks {
		marksDef[mark.Key()] = mark.Label()
	}

	marks, err := g.repoMarksDB.GetRepoMarks(ctx, repoID)
	if err != nil {
		log.Error("error has occurred while getting repo marks for repo %d : %v", repoID, err)
		return GetRepoMarksResponse{}, fmt.Errorf("get repo marks: %w", err)
	}
	for _, mark := range marks {
		if val, ok := marksDef[mark.MarkKey]; ok {
			log.Debug("adding found mark with label: %s", val)
			response = append(response, Mark{Label: val, ExpertID: mark.ExpertID})
		}
	}
	return GetRepoMarksResponse{Marks: response}, nil
}
