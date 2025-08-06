package explore

import (
	"context"

	"code.gitea.io/gitea/models/external_metric_counter"
	"code.gitea.io/gitea/models/internal_metric_counter"
	"code.gitea.io/gitea/models/repo_marks"
)

type internalCounterDB interface {
	GetInternalMetricCountersByRepoIDs(_ context.Context, repoIDs []int64) ([]*internal_metric_counter.InternalMetricCounter, error)
}

type externalCounterDB interface {
	GetExternalMetricCountersByRepoIDs(_ context.Context, repoIDs []int64) ([]*external_metric_counter.ExternalMetricCounter, error)
}

type repoMarksDB interface {
	GetRepoMarksByRepoIDs(_ context.Context, repoIDs []int64) ([]*repo_marks.RepoMarks, error)
}

type Server struct {
	internalCounterDB
	externalCounterDB
	repoMarksDB
	processedMarks []repo_marks.RepoMark
	counterEnabled bool
	marksEnabled   bool
}

func New(
	codeHubCounterDB internalCounterDB,
	externalCounterDB externalCounterDB,
	repoMarksDB repoMarksDB,
	processedMarks []repo_marks.RepoMark,
	counterEnabled bool,
	marksEnabled bool,
) Server {
	return Server{
		internalCounterDB: codeHubCounterDB,
		externalCounterDB: externalCounterDB,
		repoMarksDB:       repoMarksDB,
		processedMarks:    processedMarks,
		counterEnabled:    counterEnabled,
		marksEnabled:      marksEnabled,
	}
}
