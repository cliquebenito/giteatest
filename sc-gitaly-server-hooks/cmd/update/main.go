package main

import (
	"context"
	"os"
	"time"

	"sc-gitaly-server-hooks/internal/update"
	"sc-gitaly-server-hooks/pkg/logger"
	"sc-gitaly-server-hooks/pkg/models"
	config_reader "sc-gitaly-server-hooks/pkg/readers/config"
	"sc-gitaly-server-hooks/pkg/readers/env"

	"github.com/spf13/afero"
)

const (
	timeout  = 60 * time.Second
	hookName = "update"
)

func main() {
	fs := afero.NewOsFs()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	configReader := config_reader.NewConfigReader()
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

	preReceiveHook := update.NewUpdateHook(hookLogger, envReader)

	if err := preReceiveHook.Run(ctx); err != nil {
		os.Exit(1)
	}
}
