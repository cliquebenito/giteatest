package main

import (
	"context"
	"os"
	"time"

	post_receive "sc-gitaly-server-hooks/internal/post-receive"
	"sc-gitaly-server-hooks/pkg/client/sc"
	"sc-gitaly-server-hooks/pkg/logger"
	"sc-gitaly-server-hooks/pkg/models"
	config_reader "sc-gitaly-server-hooks/pkg/readers/config"
	"sc-gitaly-server-hooks/pkg/readers/console"
	"sc-gitaly-server-hooks/pkg/readers/env"

	"github.com/spf13/afero"
)

const (
	timeout  = 60 * time.Second
	hookName = "post-receive"
)

func main() {
	fs := afero.NewOsFs()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	configReader := config_reader.NewConfigReader()
	consoleReader := console.NewConsoleReader()
	envReader := env.NewEnvReader()

	setupHookConfig, err := configReader.Read(fs, models.HookConfigPath)
	if err != nil {
		os.Exit(1)
	}
	hooksConfig := setupHookConfig.GitalyHooksConfig()
	if hooksConfig == nil {
		hooksConfig = &models.HooksConfig{}
	}

	hookLogger, err := logger.NewCustomHookLogger(hookName, hooksConfig.GetLogPath(), hooksConfig.GetLogErrorPath(), hooksConfig.GetLogLevel())
	if err != nil {
		os.Exit(1)
	}
	hookLogger.Debug("Starting hook")

	config, err := configReader.Read(fs, hooksConfig.GetConfigPath())
	if err != nil {
		hookLogger.Error("Error reading gitaly config file", err)
		os.Exit(1)
	}

	commitDescriptors, err := consoleReader.Read(ctx, os.Stdin)
	if err != nil {
		hookLogger.Error("Error reading console", err)
		os.Exit(1)
	}

	scConfig := config.SourceControlConfig()
	scClient := sc.NewScClient(scConfig)

	postReceiveHook := post_receive.NewPostReceiveHook(hookLogger, envReader, scClient, commitDescriptors)

	if err = postReceiveHook.Run(ctx); err != nil {
		os.Exit(1)
	}
}
