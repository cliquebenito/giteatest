package post_receive

import (
	"context"
	"fmt"
	"strconv"

	"sc-gitaly-server-hooks/pkg/client"
	"sc-gitaly-server-hooks/pkg/logger"
	"sc-gitaly-server-hooks/pkg/models"
	"sc-gitaly-server-hooks/pkg/readers/env"
)

type PostReceiveHook struct {
	hookLogger  *logger.HookLogger
	envReader   env.Reader
	client      client.HookClient
	commitDescs []models.CommitDescriptor
}

func NewPostReceiveHook(logger *logger.HookLogger, envReader env.Reader, client client.HookClient, commitDescs []models.CommitDescriptor) PostReceiveHook {
	return PostReceiveHook{
		hookLogger:  logger,
		envReader:   envReader,
		client:      client,
		commitDescs: commitDescs,
	}
}

func (h PostReceiveHook) Run(ctx context.Context) error {
	ownerName, err := h.envReader.GetByKey(models.EnvRepoUsername)
	if err != nil {
		h.hookLogger.Error("error getting owner name", err)
		return fmt.Errorf("run post-receive hook: %w", err)
	}
	h.hookLogger.OwnerName = ownerName

	repoName, err := h.envReader.GetByKey(models.EnvRepoName)
	if err != nil {
		h.hookLogger.Error("error getting repo name", err)
		return fmt.Errorf("run post-receive hook: %w", err)
	}
	h.hookLogger.RepoName = repoName

	pusherId, err := h.envReader.GetByKey(models.EnvPusherID)
	if err != nil {
		h.hookLogger.Error("error getting pusher id", err)
		return fmt.Errorf("run post-receive hook: %w", err)
	}
	h.hookLogger.PusherId = pusherId

	pusherName, err := h.envReader.GetByKey(models.EnvPusherName)
	if err != nil {
		h.hookLogger.Error("error getting pusher name", err)
		return fmt.Errorf("run post-receive hook: %w", err)
	}

	userID, err := strconv.ParseInt(pusherId, 10, 64)
	if err != nil {
		h.hookLogger.Error("error parsing pusher id", err)
		return fmt.Errorf("run post-receive hook: %w", err)
	}

	hookOptions := models.NewHookOptionsWithCommitInfo(userID, h.commitDescs)
	hookOptions.UserName = pusherName
	err = hookOptions.SetGitOptionsFromEnv(h.envReader)
	if err != nil {
		h.hookLogger.Error("error getting git options from env", err)
		return fmt.Errorf("run post-receive hook: %w", err)
	}

	requestOptions := models.NewHookRequestOptions(ownerName, repoName, hookOptions)

	resp, extra := h.client.PostReceive(ctx, requestOptions)
	if extra.HasError() {
		h.hookLogger.Error("error request hook on client", err)
		for _, res := range resp.Results {
			err = res.Print()
			h.hookLogger.Error("error print response hook from client", err)
			return fmt.Errorf("run post-receive hook: %w", err)
		}
		return fmt.Errorf("run post-receive hook: %w", err)
	}

	fmt.Printf("Processed %d references in total\n", len(h.commitDescs))
	h.hookLogger.Debug("Hook success finished")

	return nil
}
