package main

import (
	"context"
	"fmt"
	"os"
	"time"

	pre_receive "sc-gitaly-server-hooks/internal/pre-receive"
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
	hookName = "pre-receive"
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
		fmt.Println(err)
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

	commitDescriptors, err := consoleReader.ReadBranchesAndTags(ctx, os.Stdin)
	if err != nil {
		hookLogger.Error("Error reading console", err)
		os.Exit(1)
	}

	scConfig := config.SourceControlConfig()
	scClient := sc.NewScClient(scConfig)

	preReceiveHook := pre_receive.NewPreReceiveHook(hookLogger, envReader, scClient, commitDescriptors)

	if err = preReceiveHook.Run(ctx); err != nil {
		os.Exit(1)
	}
}
