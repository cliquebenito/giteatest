package main

import (
	"context"
	"os"
	"time"

	proc_receive "sc-gitaly-server-hooks/internal/proc-receive"
	"sc-gitaly-server-hooks/pkg/client/sc"
	"sc-gitaly-server-hooks/pkg/logger"
	"sc-gitaly-server-hooks/pkg/models"
	config_reader "sc-gitaly-server-hooks/pkg/readers/config"
	"sc-gitaly-server-hooks/pkg/readers/console"
	"sc-gitaly-server-hooks/pkg/readers/env"
	"sc-gitaly-server-hooks/pkg/writers/pkt_line"

	"github.com/spf13/afero"
)

const (
	timeout  = 60 * time.Second
	hookName = "proc-receive"
)

func main() {
	fs := afero.NewOsFs()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	configReader := config_reader.NewConfigReader()
	consoleReader := console.NewConsoleReader()
	pktLineWriter := pkt_line.NewPktLineWriter()
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

	scConfig := config.SourceControlConfig()
	scClient := sc.NewScClient(scConfig)

	preReceiveHook := proc_receive.NewProcReceiveHook(hookLogger, consoleReader, envReader, pktLineWriter, scClient)

	if err = preReceiveHook.Run(ctx); err != nil {
		os.Exit(1)
	}
}
