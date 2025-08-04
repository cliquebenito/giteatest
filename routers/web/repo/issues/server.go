package issues

import (
	"code.gitea.io/gitea/routers/private/pull_request_task_creator"
	"code.gitea.io/gitea/routers/private/unit_linker"
)

// Server .
type Server struct {
	unitLinker         unit_linker.UnitLinker
	pullRequestSender  pull_request_task_creator.PullRequestTaskCreator
	taskTrackerEnabled bool
}

// NewServer создаем новый объект issue server
func NewServer(
	unitLinker unit_linker.UnitLinker,
	pullRequestTaskCreator pull_request_task_creator.PullRequestTaskCreator,
	taskTrackerEnabled bool) Server {
	return Server{
		unitLinker:         unitLinker,
		pullRequestSender:  pullRequestTaskCreator,
		taskTrackerEnabled: taskTrackerEnabled,
	}
}
