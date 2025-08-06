package update

import (
	"context"
	"fmt"

	"sc-gitaly-server-hooks/pkg/logger"
	"sc-gitaly-server-hooks/pkg/models"
	"sc-gitaly-server-hooks/pkg/readers/env"
)

type UpdateHook struct {
	hookLogger *logger.HookLogger
	envReader  env.Reader
}

func NewUpdateHook(logger *logger.HookLogger, envReader env.Reader) UpdateHook {
	return UpdateHook{
		hookLogger: logger,
		envReader:  envReader,
	}
}

func (h UpdateHook) Run(ctx context.Context) error {
	ownerName, err := h.envReader.GetByKey(models.EnvRepoUsername)
	if err != nil {
		h.hookLogger.Error("error getting owner name", err)
		return fmt.Errorf("run update hook: %w", err)
	}
	h.hookLogger.OwnerName = ownerName

	repoName, err := h.envReader.GetByKey(models.EnvRepoName)
	if err != nil {
		h.hookLogger.Error("error getting repo name", err)
		return fmt.Errorf("run update hook: %w", err)
	}
	h.hookLogger.RepoName = repoName

	pusherId, err := h.envReader.GetByKey(models.EnvPusherID)
	if err != nil {
		h.hookLogger.Error("error getting pusher id", err)
		return fmt.Errorf("run update hook: %w", err)
	}
	h.hookLogger.PusherId = pusherId

	h.hookLogger.Debug("Hook success finished")

	return nil
}
