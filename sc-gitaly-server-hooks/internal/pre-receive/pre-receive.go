package pre_receive

import (
	"context"
	"fmt"
	"strconv"

	"sc-gitaly-server-hooks/pkg/client"
	"sc-gitaly-server-hooks/pkg/logger"
	"sc-gitaly-server-hooks/pkg/models"
	"sc-gitaly-server-hooks/pkg/readers/env"
)

type PreReceiveHook struct {
	hookLogger  *logger.HookLogger
	envReader   env.Reader
	client      client.HookClient
	commitDescs []models.CommitDescriptor
}

func NewPreReceiveHook(logger *logger.HookLogger, envReader env.Reader, client client.HookClient, commitDescs []models.CommitDescriptor) PreReceiveHook {
	return PreReceiveHook{
		hookLogger:  logger,
		envReader:   envReader,
		client:      client,
		commitDescs: commitDescs,
	}
}

func (h PreReceiveHook) Run(ctx context.Context) error {
	fmt.Printf("Checking %d references\n", len(h.commitDescs))
	var err error

	if h.hookLogger.OwnerName, err = h.envReader.GetByKey(models.EnvRepoUsername); err != nil {
		h.hookLogger.Error("error getting owner name", err)
		return fmt.Errorf("run pre-receive hook: %w", err)
	}

	repoName, err := h.envReader.GetByKey(models.EnvRepoName)
	if err != nil {
		h.hookLogger.Error("error getting repo name", err)
		return fmt.Errorf("run pre-receive hook: %w", err)
	}
	h.hookLogger.RepoName = repoName

	pusherId, err := h.envReader.GetByKey(models.EnvPusherID)
	if err != nil {
		h.hookLogger.Error("error getting pusher id", err)
		return fmt.Errorf("run pre-receive hook: %w", err)
	}
	h.hookLogger.PusherId = pusherId

	userID, err := strconv.ParseInt(pusherId, 10, 64)
	if err != nil {
		h.hookLogger.Error("error parsing pusher id", err)
		return fmt.Errorf("run pre-receive hook: %w", err)
	}

	hookOptions := models.NewHookOptionsWithCommitInfo(userID, h.commitDescs)
	err = hookOptions.SetGitOptionsFromEnv(h.envReader)
	if err != nil {
		h.hookLogger.Error("error getting git options from env", err)
		return fmt.Errorf("run pre-receive hook: %w", err)
	}

	requestOptions := models.NewHookRequestOptions(h.hookLogger.OwnerName, repoName, hookOptions)

	extra := h.client.PreReceive(ctx, requestOptions)
	if extra.HasError() {
		h.hookLogger.Error("error request hook on client", err)
		return fmt.Errorf("run pre-receive hook: %w", err)
	}

	fmt.Printf("Checked %d references in total\n", len(h.commitDescs))
	h.hookLogger.Debug("Hook success finished")

	return nil
}
