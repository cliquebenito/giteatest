package logger

import (
	"go.uber.org/zap"
)

var zapLogger *zap.Logger

func NewLogger(path, errPath, level string) (*zap.Logger, error) {
	if zapLogger != nil {
		return zapLogger, nil
	}

	atomicLevel, err := zap.ParseAtomicLevel(level)
	if err != nil {
		return nil, err
	}
	outputPaths := make([]string, 0)
	errOutputPaths := make([]string, 0)

	outputPaths = append(outputPaths, path)
	errOutputPaths = append(errOutputPaths, path)
	errOutputPaths = append(errOutputPaths, errPath)

	cfg := zap.NewProductionConfig()
	cfg.OutputPaths = outputPaths
	cfg.ErrorOutputPaths = errOutputPaths
	cfg.Level = atomicLevel
	zapLogger, err := cfg.Build()
	if err != nil {
		return nil, err
	}
	return zapLogger, nil
}

type HookLogger struct {
	HookName  string
	OwnerName string
	RepoName  string
	PusherId  string

	logger *zap.Logger
}

func NewDefaultHookLogger(hookName string) (*HookLogger, error) {
	logger, err := NewLogger("/var/log/gitaly/hooks/hooks.log", "/var/log/gitaly/hooks/hooks_error.log", "info")
	if err != nil {
		return nil, err
	}
	return &HookLogger{
		HookName: hookName,
		logger:   logger,
	}, nil
}

func NewCustomHookLogger(hookName string, path, errPath, level string) (*HookLogger, error) {
	logger, err := NewLogger(path, errPath, level)
	if err != nil {
		return nil, err
	}
	return &HookLogger{
		HookName: hookName,
		logger:   logger,
	}, nil
}

func (l *HookLogger) Info(msg string) {
	l.logger.Info(msg,
		zap.String("hook", l.HookName),
		zap.String("owner_name", l.OwnerName),
		zap.String("repo_name", l.RepoName),
		zap.String("pusher_id", l.PusherId),
	)
}

func (l *HookLogger) Debug(msg string) {
	l.logger.Debug(msg,
		zap.String("hook", l.HookName),
		zap.String("owner_name", l.OwnerName),
		zap.String("repo_name", l.RepoName),
		zap.String("pusher_id", l.PusherId),
	)
}

func (l *HookLogger) Error(msg string, err error) {
	l.logger.Error(msg,
		zap.String("hook", l.HookName),
		zap.String("owner_name", l.OwnerName),
		zap.String("repo_name", l.RepoName),
		zap.String("pusher_id", l.PusherId),
		zap.Error(err),
	)
}
